package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/server"
	"goravel/app/services"
	"goravel/bootstrap"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type client struct {
	app     *fiber.App
	cookies map[string]*http.Cookie
	token   string
	t       *testing.T
}

var requestCount atomic.Uint64

func (c *client) req(method, path string, data any) (int, map[string]any, []byte) {
	requestCount.Add(1)
	var b bytes.Buffer
	if data != nil {
		json.NewEncoder(&b).Encode(data)
	}
	r, _ := http.NewRequest(method, "http://localhost"+path, &b)
	r.Header.Set("Content-Type", "application/json")
	if method != "GET" {
		r.Header.Set("X-CSRF-Token", c.token)
	}
	for _, v := range c.cookies {
		r.AddCookie(v)
	}
	res, e := c.app.Test(r, fiber.TestConfig{Timeout: 20 * time.Second})
	if e != nil {
		c.t.Fatal(e)
	}
	defer res.Body.Close()
	for _, v := range res.Cookies() {
		c.cookies[v.Name] = v
	}
	raw, _ := io.ReadAll(res.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out, raw
}
func (c *client) want(status int, method, path string, data any) map[string]any {
	s, out, raw := c.req(method, path, data)
	if s != status {
		c.t.Fatalf("%s %s: got %d want %d: %s", method, path, s, status, raw)
	}
	return out
}
func (c *client) session() {
	out := c.want(200, "GET", "/api/session", nil)
	c.token = out["csrf_token"].(string)
}
func newClient(t *testing.T, a *fiber.App) *client {
	c := &client{a, map[string]*http.Cookie{}, "", t}
	c.session()
	return c
}
func (c *client) register(email, role string) uint {
	out := c.want(201, "POST", "/api/auth/register", map[string]any{"name": "Prueba " + role, "email": email, "password": "StrongTest123!", "phone": "12345678", "role": role, "address": "Calle prueba 10", "latitude": 23.11345, "longitude": -82.3667, "company_name": "Empresa de prueba", "vehicle_type": "bicycle"})
	c.session()
	return uint(out["user"].(map[string]any)["id"].(float64))
}
func jobInput() map[string]any {
	n := time.Now().UTC()
	return map[string]any{"title": "Entrega de prueba", "description": "Paquete de prueba", "pickup_address": "Calle origen 1", "pickup_lat": 23.1134567, "pickup_lng": -82.3667891, "dropoff_address": "Calle destino 2", "dropoff_lat": 23.12, "dropoff_lng": -82.37, "pickup_from": n.Add(time.Hour).Format(time.RFC3339), "pickup_to": n.Add(2 * time.Hour).Format(time.RFC3339), "delivery_from": n.Add(2 * time.Hour).Format(time.RFC3339), "delivery_to": n.Add(3 * time.Hour).Format(time.RFC3339), "price_cents": 1234, "currency": "USD"}
}
func TestDeliveryAndTenantSecurity(t *testing.T) {
	if os.Getenv("CHITA_INTEGRATION") != "1" {
		t.Skip("requires isolated PostgreSQL; see README")
	}
	if !strings.HasSuffix(os.Getenv("DB_DATABASE"), "_validation") {
		t.Fatal("test database name must end _validation")
	}
	bootstrap.Boot()
	requestCount.Store(0)
	facades.Config().Add("app.key", "0123456789abcdef0123456789abcdef")
	if _, e := facades.Orm().Query().Exec("TRUNCATE users RESTART IDENTITY CASCADE"); e != nil {
		t.Fatal(e)
	}
	remoteURL := os.Getenv("CHITA_HALCON_URL")
	if remoteURL == "" {
		remoteURL = "http://127.0.0.1:3300"
	}
	r, _ := services.NewRemote(remoteURL)
	tracking := services.NewTracking(r)
	defer tracking.Close()
	a := server.New(nil, false, tracking)
	anon := newClient(t, a)
	anon.want(401, "GET", "/api/jobs", nil)
	anon.want(401, "GET", "/api/nonexistent", nil)
	company := newClient(t, a)
	company.register("company@chita.test", "company")
	company.want(404, "GET", "/api/nonexistent", nil)
	other := newClient(t, a)
	other.register("other@chita.test", "company")
	courier := newClient(t, a)
	uid := courier.register("courier@chita.test", "courier")
	stranger := newClient(t, a)
	stranger.register("stranger@chita.test", "courier")
	courier.want(403, "POST", "/api/jobs", jobInput())
	bad := jobInput()
	bad["assigned_courier_id"] = uid
	company.want(422, "POST", "/api/jobs", bad)
	bad = jobInput()
	bad["price_cents"] = 0
	company.want(422, "POST", "/api/jobs", bad)
	j := company.want(201, "POST", "/api/jobs", jobInput())
	id := uint(j["id"].(float64))
	path := fmt.Sprintf("/api/jobs/%d", id)
	other.want(404, "GET", path, nil)
	courier.want(404, "GET", path, nil)
	other.want(404, "POST", path+"/cancel", map[string]any{"note": "Intento ajeno"})
	inv := company.want(201, "POST", "/api/network", map[string]any{"email": "courier@chita.test"})
	mid := uint(inv["id"].(float64))
	stranger.want(404, "POST", fmt.Sprintf("/api/network/%d/accept", mid), map[string]any{})
	courier.want(404, "GET", path, nil)
	courier.want(204, "POST", fmt.Sprintf("/api/network/%d/accept", mid), map[string]any{})
	courier.want(200, "GET", path, nil)
	stranger.want(404, "GET", path, nil)
	// Invalid CSRF must not mutate even when authenticated.
	saved := courier.token
	courier.token = "wrong"
	courier.want(403, "POST", path+"/accept", map[string]any{})
	courier.token = saved
	courier.want(200, "POST", path+"/accept", map[string]any{})
	courier.want(409, "POST", path+"/accept", map[string]any{})
	stranger.want(404, "POST", path+"/pickup", map[string]any{})
	courier.want(409, "POST", path+"/report", map[string]any{})
	company.want(409, "POST", path+"/confirm", map[string]any{})
	company.want(204, "POST", fmt.Sprintf("/api/network/%d/remove", mid), map[string]any{})
	courier.want(200, "GET", path, nil)
	newJob := company.want(201, "POST", "/api/jobs", jobInput())
	courier.want(404, "GET", fmt.Sprintf("/api/jobs/%v", newJob["id"]), nil)
	// Revoked membership does not strand accepted jobs.
	courier.want(200, "POST", path+"/pickup", map[string]any{})
	company.want(409, "POST", path+"/cancel", map[string]any{"note": "Tarde"})
	courier.want(200, "POST", path+"/arrive", map[string]any{})
	other.want(404, "GET", path+"/location", nil)
	stranger.want(404, "POST", path+"/location", map[string]any{"latitude": 23, "longitude": -82})
	courier.want(422, "POST", path+"/location", map[string]any{"latitude": 91, "longitude": 0})
	courier.want(409, "POST", path+"/location", map[string]any{"latitude": 23, "longitude": -82})
	if os.Getenv("CHITA_HALCON_URL") != "" {
		email := halconFixture(t, remoteURL)
		courier.want(200, "POST", "/api/halcon", map[string]any{"email": email, "password": "StrongTest123!"})
		stranger.want(409, "POST", "/api/halcon", map[string]any{"email": email, "password": "StrongTest123!"})
		var linked models.HalconAccount
		if err := facades.Orm().Query().Where("user_id=?", uid).First(&linked); err != nil {
			t.Fatal(err)
		}
		if linked.SessionCiphertext == "" || strings.Contains(linked.SessionCiphertext, "StrongTest123!") {
			t.Fatal("remote session was not encrypted")
		}
		courier.want(204, "POST", path+"/location", map[string]any{"latitude": 23.1134567, "longitude": -82.3667891})
		deadline := time.Now().Add(5 * time.Second)
		ok := false
		for time.Now().Before(deadline) {
			s, x, _ := company.req("GET", path+"/location", nil)
			if s == 200 && x["active"] == true && x["latitude"] == 23.1134567 && x["longitude"] == -82.3667891 {
				ok = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !ok {
			t.Fatal("company never received actual HALCON coordinates")
		}
		courier.want(204, "DELETE", "/api/halcon", nil)
		company.want(409, "GET", path+"/location", nil)
		courier.want(200, "POST", "/api/halcon", map[string]any{"email": email, "password": "StrongTest123!"})
		courier.want(204, "POST", path+"/location", map[string]any{"latitude": 23.1134567, "longitude": -82.3667891})
	}
	courier.want(200, "POST", path+"/report", map[string]any{"note": "Entregado"})
	company.want(409, "GET", path+"/location", nil)
	company.want(422, "POST", path+"/reject", map[string]any{})
	company.want(200, "POST", path+"/reject", map[string]any{"note": "Verificar recepción"})
	courier.want(200, "POST", path+"/report", map[string]any{})
	company.want(200, "POST", path+"/confirm", map[string]any{})
	courier.want(204, "DELETE", "/api/halcon", nil)
	courier.want(409, "POST", path+"/report", map[string]any{})
	company.want(409, "POST", path+"/confirm", map[string]any{})
	// Coordinates retain seven decimals.
	var p models.Publication
	if e := facades.Orm().Query().Where("id=?", id).First(&p); e != nil {
		t.Fatal(e)
	}
	if p.PickupLat != 23.1134567 || p.PickupLng != -82.3667891 || p.ConfirmedAt == nil || p.DeliveredAt == nil {
		t.Fatalf("persisted publication: %+v", p)
	}
	count, e := facades.Orm().Query().Model(&models.PublicationEvent{}).Where("publication_id=?", id).Count()
	if e != nil || count != 8 {
		t.Fatalf("events %d %v", count, e)
	}
	_, _, raw := courier.req("GET", "/api/notifications", nil)
	var nn []models.Notification
	if e := json.Unmarshal(raw, &nn); e != nil || len(nn) == 0 {
		t.Fatal("notifications missing", string(raw))
	}
	other.want(404, "POST", fmt.Sprintf("/api/notifications/%d/read", nn[0].ID), map[string]any{})
	courier.want(204, "POST", fmt.Sprintf("/api/notifications/%d/read", nn[0].ID), map[string]any{})
	// Foreign selected job must stay hidden even when another assigned job exists (SQL OR scoping).
	stranger.want(404, "GET", path, nil)
	courier.want(404, "GET", fmt.Sprintf("/api/jobs/%v", newJob["id"]), nil)
	// Two eligible couriers race; exactly one gets assignment.
	company.want(201, "POST", "/api/network", map[string]any{"email": "courier@chita.test"})
	_, _, raw = courier.req("GET", "/api/network", nil)
	var members []models.CompanyMember
	json.Unmarshal(raw, &members)
	courier.want(204, "POST", fmt.Sprintf("/api/network/%d/accept", members[0].ID), map[string]any{})
	inv2 := company.want(201, "POST", "/api/network", map[string]any{"email": "stranger@chita.test"})
	stranger.want(204, "POST", fmt.Sprintf("/api/network/%v/accept", inv2["id"]), map[string]any{})
	j = company.want(201, "POST", "/api/jobs", jobInput())
	racePath := fmt.Sprintf("/api/jobs/%v/accept", j["id"])
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, c := range []*client{courier, stranger} {
		wg.Add(1)
		go func(c *client) { defer wg.Done(); s, _, _ := c.req("POST", racePath, map[string]any{}); results <- s }(c)
	}
	wg.Wait()
	close(results)
	wins := 0
	for s := range results {
		if s == 200 {
			wins++
		} else if s != 409 {
			t.Fatalf("unexpected race status %d", s)
		}
	}
	if wins != 1 {
		t.Fatal("assignment race", wins)
	}
	cancelled := company.want(201, "POST", "/api/jobs", jobInput())
	cancelPath := fmt.Sprintf("/api/jobs/%v", cancelled["id"])
	company.want(200, "POST", cancelPath+"/cancel", map[string]any{"note": "Cambio de horario"})
	courier.want(404, "GET", cancelPath, nil)
	expired := company.want(201, "POST", "/api/jobs", jobInput())
	_, err := facades.Orm().Query().Model(&models.Publication{}).Where("id=?", expired["id"]).Update("expires_at", time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	courier.want(409, "POST", fmt.Sprintf("/api/jobs/%v/accept", expired["id"]), map[string]any{})
	// API never exposes password hashes or private HALCON sessions.
	_, _, raw = company.req("GET", "/api/jobs", nil)
	if bytes.Contains(raw, []byte("password_hash")) || bytes.Contains(raw, []byte("session_ciphertext")) {
		t.Fatal("private fields leaked")
	}
	company.want(403, "GET", "/api/halcon", nil)
	courier.want(200, "GET", "/api/profile", nil)
	company.want(200, "GET", "/api/profile", nil)
	courier.want(204, "POST", "/api/auth/logout", map[string]any{})
	courier.session()
	courier.want(401, "GET", "/api/jobs", nil)
	courier.want(200, "POST", "/api/auth/login", map[string]any{"email": "courier@chita.test", "password": "StrongTest123!"})
	courier.session()
	courier.want(200, "GET", "/api/profile", nil)
	var u models.User
	facades.Orm().Query().Where("id=?", uid).First(&u)
	facades.Orm().Query().Model(&models.User{}).Where("id=?", uid).Update("status", false)
	courier.want(401, "GET", "/api/jobs", nil)
	limitApp := server.New(nil, false, tracking)
	limited := newClient(t, limitApp)
	for i := 0; i < 12; i++ {
		limited.want(422, "POST", "/api/auth/register", map[string]any{"role": "admin"})
	}
	limited.want(429, "POST", "/api/auth/register", map[string]any{"role": "admin"})
	t.Logf("%d HTTP requests checked: role isolation, invitation consent, lifecycle, notifications, CSRF, GPS bounds, throttling and assignment race passed", requestCount.Load())
}

func halconFixture(t *testing.T, base string) string {
	if base != "http://127.0.0.1:3331" {
		t.Fatal("isolated HALCON must use port 3331")
	}
	h := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := h.Get(base + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	var info struct {
		Token string `json:"csrf_token"`
	}
	json.NewDecoder(res.Body).Decode(&info)
	res.Body.Close()
	cookies := res.Cookies()
	email := fmt.Sprintf("chita-api-%d@example.test", time.Now().UnixNano())
	form := url.Values{"name": {"Repartidor API test"}, "email": {email}, "phone": {fmt.Sprint(time.Now().UnixNano() % 1000000000000)}, "password": {"StrongTest123!"}, "_csrf": {info.Token}}
	req, _ := http.NewRequest("POST", base+"/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", base)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	res, err = h.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 302 && res.StatusCode != 303 {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("HALCON fixture %d %.500s", res.StatusCode, raw)
	}
	return email
}
