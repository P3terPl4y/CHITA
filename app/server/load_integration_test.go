package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"goravel/app/facades"
	"goravel/app/server"
	"goravel/app/services"
	"goravel/bootstrap"
)

// TestLoadOneThousandUsers is an opt-in destructive load scenario. It only
// targets a local listener and a database explicitly named *_validation.
func TestLoadOneThousandUsers(t *testing.T) {
	if os.Getenv("CHITA_LOAD") != "1" || os.Getenv("CHITA_INTEGRATION") != "1" {
		t.Skip("set CHITA_LOAD=1 and CHITA_INTEGRATION=1 for isolated load test")
	}
	db := os.Getenv("DB_DATABASE")
	if !strings.HasSuffix(db, "_validation") || db == "" {
		t.Fatal("load test refuses any database not ending in _validation")
	}
	host := os.Getenv("DB_HOST")
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		t.Fatalf("load test requires a loopback PostgreSQL host; got %q", host)
	}
	bootstrap.Boot()
	facades.Config().Add("app.key", "0123456789abcdef0123456789abcdef")
	if _, err := facades.Orm().Query().Exec("TRUNCATE users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	remote, err := services.NewRemote("http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	app := server.New(nil, false, tracking)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://" + listener.Addr().String()
	serveErr := make(chan error, 1)
	go func() { serveErr <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() {
		_ = listener.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		select {
		case e := <-serveErr:
			if e != nil && !strings.Contains(e.Error(), "closed network connection") && !strings.Contains(e.Error(), "use of closed network connection") {
				t.Errorf("load server stopped: %v", e)
			}
		case <-ctx.Done():
			t.Error("load server did not stop")
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, e := http.Get(baseURL + "/api/session")
		if e == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("local server did not start: %v", e)
		}
		time.Sleep(20 * time.Millisecond)
	}

	const users, pairs, workers = 1000, 500, 20
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	companies, couriers := make([]*loadVU, pairs), make([]*loadVU, pairs)
	metrics := &loadMetrics{}
	var next uint32
	newVU := func() (*loadVU, error) {
		ipNum := atomicAdd(&next)
		ip := net.IPv4(127, byte(ipNum>>16), byte(ipNum>>8), byte(ipNum))
		jar, _ := cookiejar.New(nil)
		transport := &http.Transport{MaxIdleConns: 2, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, DialContext: (&net.Dialer{Timeout: 5 * time.Second, LocalAddr: &net.TCPAddr{IP: ip}}).DialContext}
		return &loadVU{http: &http.Client{Jar: jar, Transport: transport, Timeout: 30 * time.Second}, base: baseURL, run: stamp, metrics: metrics}, nil
	}

	start := time.Now()
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	errCh := make(chan error, users)
	for i := 0; i < pairs; i++ {
		for _, role := range []string{"company", "courier"} {
			index := i
			roleCopy := role
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				vu, e := newVU()
				email := fmt.Sprintf("load-%s-%s-%04d@chita.test", stamp, roleCopy, index)
				if e == nil {
					e = vu.register(email, roleCopy)
				}
				if e != nil {
					errCh <- fmt.Errorf("register %s %d: %w", roleCopy, index, e)
					return
				}
				if roleCopy == "company" {
					companies[index] = vu
				} else {
					couriers[index] = vu
				}
			}()
		}
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Error(e)
	}
	if t.Failed() {
		return
	}
	registerDuration := time.Since(start)
	t.Logf("registered %d simulated accounts in %s (workers=%d)", users, registerDuration.Round(time.Millisecond), workers)

	// Each pair performs the full consent, publication and delivery cycle for
	// three jobs so the real three-delivery rating threshold is exercised.
	start = time.Now()
	sem = make(chan struct{}, workers/2)
	errCh = make(chan error, pairs)
	for i := 0; i < pairs; i++ {
		index := i
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if e := runLoadPair(companies[index], couriers[index], couriers[(index+1)%pairs], companies[(index+1)%pairs], index, stamp); e != nil {
				errCh <- fmt.Errorf("pair %d: %w", index, e)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Error(e)
	}
	workflowDuration := time.Since(start)
	if t.Failed() {
		return
	}
	validToken := couriers[0].token
	couriers[0].token = "intentionally-invalid-token"
	if _, _, err := couriers[0].request("POST", "/api/location/stop", map[string]any{}, 403); err != nil {
		t.Error(err)
		return
	}
	if err := couriers[0].session(); err != nil {
		t.Fatalf("session did not recover after CSRF rejection: %v", err)
	}
	if couriers[0].token == validToken {
		t.Log("CSRF token remained stable after rejected request")
	}

	// Exercise 1000 distinct authenticated users at once, with read-only calls
	// and a bounded request timeout. This measures burst capacity without a
	// thundering write storm against the local validation database.
	start = time.Now()
	metrics = &loadMetrics{}
	for i := range companies {
		companies[i].metrics = metrics
		couriers[i].metrics = metrics
	}
	burstErrs := make(chan error, users*3)
	sem = make(chan struct{}, users)
	for i := 0; i < users; i++ {
		vu := companies[i/2]
		if i%2 == 1 {
			vu = couriers[i/2]
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(v *loadVU) {
			defer wg.Done()
			defer func() { <-sem }()
			for _, path := range []string{"/api/session", "/api/profile", "/api/jobs?total=false"} {
				if _, _, e := v.request("GET", path, nil, 200); e != nil {
					burstErrs <- e
				}
			}
		}(vu)
	}
	wg.Wait()
	close(burstErrs)
	var burstFailures int
	for e := range burstErrs {
		burstFailures++
		t.Error(e)
	}
	t.Logf("completed %d authenticated read requests from %d concurrent virtual users in %s; failures=%d", users*3, users, time.Since(start).Round(time.Millisecond), burstFailures)
	for _, endpoint := range []string{"/api/session", "/api/profile", "/api/jobs?total=false"} {
		p50, p95, maximum := metrics.summary(endpoint)
		t.Logf("burst %s latency p50=%s p95=%s max=%s", endpoint, p50, p95, maximum)
	}
	t.Logf("completed %d full company/courier workflows (3 deliveries and rating each) in %s", pairs, workflowDuration.Round(time.Millisecond))
}

func atomicAdd(p *uint32) uint32 {
	// Registration goroutines are concurrent; atomic increment keeps each VU
	// on a distinct loopback source address so per-IP auth limits stay real.
	return atomic.AddUint32(p, 1)
}

type loadVU struct {
	http    *http.Client
	base    string
	token   string
	run     string
	id      uint64
	metrics *loadMetrics
}

func (v *loadVU) request(method, path string, body any, want int) (map[string]any, []byte, error) {
	var input io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		input = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, v.base+path, input)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if method != "GET" {
		req.Header.Set("X-CSRF-Token", v.token)
	}
	started := time.Now()
	res, err := v.http.Do(req)
	if v.metrics != nil {
		v.metrics.add(path, time.Since(started))
	}
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, raw, err
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if res.StatusCode != want {
		return out, raw, fmt.Errorf("%s %s returned %d, want %d: %s", method, path, res.StatusCode, want, raw)
	}
	return out, raw, nil
}

type loadMetrics struct {
	mu    sync.Mutex
	items map[string][]time.Duration
}

func (m *loadMetrics) add(path string, d time.Duration) {
	m.mu.Lock()
	if m.items == nil {
		m.items = make(map[string][]time.Duration)
	}
	m.items[path] = append(m.items[path], d)
	m.mu.Unlock()
}

func (m *loadMetrics) summary(path string) (time.Duration, time.Duration, time.Duration) {
	m.mu.Lock()
	items := append([]time.Duration(nil), m.items[path]...)
	m.mu.Unlock()
	if len(items) == 0 {
		return 0, 0, 0
	}
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
	return items[(len(items)-1)*50/100], items[(len(items)-1)*95/100], items[len(items)-1]
}

func (v *loadVU) session() error {
	out, _, err := v.request("GET", "/api/session", nil, 200)
	if err != nil {
		return err
	}
	token, ok := out["csrf_token"].(string)
	if !ok || token == "" {
		return fmt.Errorf("session omitted CSRF token")
	}
	v.token = token
	return nil
}

func (v *loadVU) register(email, role string) error {
	if err := v.session(); err != nil {
		return err
	}
	data := map[string]any{"name": "Carga " + role, "email": email, "password": "LoadTestStrong123!", "phone": "12345678", "role": role, "address": "Calle carga 10", "latitude": 23.11345, "longitude": -82.3667, "company_name": "Empresa carga", "vehicle_type": "bicycle"}
	if _, _, err := v.request("POST", "/api/auth/register", data, 201); err != nil {
		return err
	}
	if err := v.session(); err != nil {
		return err
	}
	identity, _, err := v.request("GET", "/api/session", nil, 200)
	if err != nil {
		return err
	}
	user, ok := identity["user"].(map[string]any)
	if !ok {
		return fmt.Errorf("registered session omitted user")
	}
	v.id, err = loadID(user["id"])
	return err
}

func runLoadPair(company, courier, strangerCourier, strangerCompany *loadVU, pair int, run string) error {
	for _, path := range []string{"/api/profile", "/api/jobs", "/api/network", "/api/notifications"} {
		status := 200
		if _, _, err := courier.request("GET", path, nil, status); err != nil {
			return err
		}
	}
	if _, _, err := courier.request("POST", "/api/jobs", jobInput(), 403); err != nil {
		return err
	}
	if pair == 0 {
		consent, _, err := courier.request("PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23.11345, "longitude": -82.3667}, 200)
		if err != nil {
			return err
		}
		if token, ok := consent["token"].(string); !ok || token == "" {
			return fmt.Errorf("availability omitted consent token")
		}
		var nearby []map[string]any
		_, rawNearby, err := company.request("GET", "/api/couriers/nearby?lat=23.11345&lng=-82.3667&radius=2", nil, 200)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(rawNearby, &nearby); err != nil {
			return err
		}
		foundCourier := false
		for _, row := range nearby {
			id, _ := loadID(row["id"])
			if id == courier.id {
				foundCourier = true
			}
			for _, private := range []string{"email", "phone", "address", "password_hash", "session_ciphertext"} {
				if _, exists := row[private]; exists {
					return fmt.Errorf("nearby courier leaked field %s", private)
				}
			}
		}
		if !foundCourier {
			return fmt.Errorf("consenting courier %d missing from nearby results", courier.id)
		}
	}
	if _, _, err := company.request("GET", "/api/admin/companies", nil, 403); err != nil {
		return err
	}
	_, directory, err := company.request("GET", "/api/couriers/directory", nil, 200)
	if err != nil {
		return err
	}
	for _, private := range []string{`"email"`, `"phone"`, `"address"`, `"latitude"`, `"longitude"`} {
		if bytes.Contains(directory, []byte(private)) {
			return fmt.Errorf("courier directory leaked field %s", private)
		}
	}
	invite, _, err := company.request("POST", "/api/network", map[string]any{"email": fmt.Sprintf("load-%s-courier-%04d@chita.test", run, pair)}, 201)
	if err != nil {
		return err
	}
	memberID, err := loadID(invite["id"])
	if err != nil {
		return err
	}
	if _, _, err = strangerCourier.request("POST", fmt.Sprintf("/api/network/%d/accept", memberID), map[string]any{}, 404); err != nil {
		return err
	}
	if _, _, err = courier.request("POST", fmt.Sprintf("/api/network/%d/accept", memberID), map[string]any{}, 204); err != nil {
		return err
	}
	for n := 0; n < 3; n++ {
		if n == 2 && pair == 0 {
			if _, _, e := courier.request("PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23.11345, "longitude": -82.3667}, 200); e != nil {
				return e
			}
		}
		job, _, e := company.request("POST", "/api/jobs", jobInput(), 201)
		if e != nil {
			return e
		}
		id, e := loadID(job["id"])
		if e != nil {
			return e
		}
		path := fmt.Sprintf("/api/jobs/%d", id)
		if _, _, e = strangerCompany.request("GET", path, nil, 404); e != nil {
			return e
		}
		if _, _, e = strangerCourier.request("GET", path, nil, 404); e != nil {
			return e
		}
		if _, _, e = strangerCourier.request("POST", path+"/accept", map[string]any{}, 404); e != nil {
			return e
		}
		if n == 2 && pair == 0 {
			offer, _, offerErr := company.request("POST", path+"/offer", map[string]any{"courier_id": courier.id}, 201)
			if offerErr != nil {
				return offerErr
			}
			offerID, idErr := loadID(offer["id"])
			if idErr != nil {
				return idErr
			}
			if _, _, offerErr = strangerCourier.request("POST", fmt.Sprintf("/api/offers/%d/accept", offerID), map[string]any{}, 404); offerErr != nil {
				return offerErr
			}
			if _, _, offerErr = courier.request("GET", "/api/offers", nil, 200); offerErr != nil {
				return offerErr
			}
			if _, _, offerErr = courier.request("POST", fmt.Sprintf("/api/offers/%d/accept", offerID), map[string]any{}, 204); offerErr != nil {
				return offerErr
			}
			if _, _, offerErr = courier.request("PUT", "/api/availability", map[string]any{"enabled": false}, 204); offerErr != nil {
				return offerErr
			}
		} else if _, _, e = courier.request("POST", path+"/accept", map[string]any{}, 200); e != nil {
			return e
		}
		if _, _, e = strangerCourier.request("GET", path, nil, 404); e != nil {
			return e
		}
		if _, _, e = strangerCourier.request("GET", path+"/location", nil, 404); e != nil {
			return e
		}
		if _, _, e = strangerCourier.request("POST", path+"/location", map[string]any{"latitude": 23.11345, "longitude": -82.3667}, 404); e != nil {
			return e
		}
		for _, action := range []string{"pickup", "arrive", "report"} {
			body := map[string]any{}
			if action == "report" {
				body["note"] = "Entrega simulada"
			}
			if _, _, e = courier.request("POST", path+"/"+action, body, 200); e != nil {
				return e
			}
		}
		if _, _, e = company.request("POST", path+"/confirm", map[string]any{}, 200); e != nil {
			return e
		}
	}
	if _, _, err = company.request("PUT", fmt.Sprintf("/api/couriers/%d/rating", courier.id), map[string]any{"rating": 5, "comment": "Carga validada"}, 200); err != nil {
		return err
	}
	return nil
}

func loadID(value any) (uint64, error) {
	n, ok := value.(float64)
	if !ok || n <= 0 {
		return 0, fmt.Errorf("invalid numeric id: %v", value)
	}
	return uint64(n), nil
}
