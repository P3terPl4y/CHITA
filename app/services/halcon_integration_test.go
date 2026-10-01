package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// This contract test talks to a real, separate HALCON instance with its own test database.
func TestRealHalconContract(t *testing.T) {
	base := os.Getenv("CHITA_HALCON_URL")
	if base == "" {
		t.Skip("requires isolated HALCON instance")
	}
	if base != "http://127.0.0.1:3331" {
		t.Fatal("test HALCON must use isolated loopback port 3331")
	}
	r, e := NewRemote(base)
	if e != nil {
		t.Fatal(e)
	}
	var s RemoteSession
	var a struct {
		CSRF string `json:"csrf_token"`
	}
	if e = r.call(&s, "GET", "/api/session", nil, &a); e != nil {
		t.Fatal(e)
	}
	email := fmt.Sprintf("chita-contract-%d@example.test", time.Now().UnixNano())
	form := url.Values{"name": {"Courier CHITA test"}, "email": {email}, "phone": {fmt.Sprintf("%d", time.Now().UnixNano()%1000000000000)}, "password": {"StrongTest123!"}, "password_confirmation": {"StrongTest123!"}, "_csrf": {a.CSRF}}
	req, _ := http.NewRequest("POST", base+"/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", base)
	for _, c := range s.Cookies {
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
	}
	res, e := r.Client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 302 && res.StatusCode != 303 {
		t.Fatalf("HALCON fixture registration %d %.500s", res.StatusCode, raw)
	}
	s, uid, pid, e := r.Login(email, "StrongTest123!")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Logout(s)
	if uid == 0 || pid == 0 {
		t.Fatal("missing remote identities")
	}
	initial, err := r.Location(s, pid)
	if err != nil || initial.Latitude != nil || initial.Longitude != nil || initial.Active {
		t.Fatal("uninitialized HALCON must not appear at zero coordinates", initial, err)
	}
	c, e := r.Dial(s)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	lat, lng := 23.1134567, -82.3667891
	data, _ := json.Marshal(Point{&lat, &lng})
	if e = c.WriteMessage(1, data); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(5 * time.Second)
	var loc Location
	for time.Now().Before(deadline) {
		loc, e = r.Location(s, pid)
		if e == nil && loc.Active && loc.Latitude != nil && *loc.Latitude == lat && loc.Longitude != nil && *loc.Longitude == lng {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if e != nil || !loc.Active || loc.Latitude == nil || *loc.Latitude != lat || loc.Longitude == nil || *loc.Longitude != lng {
		t.Fatalf("HALCON did not persist real GPS: %+v %v", loc, e)
	}
	c.Close()
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		loc, e = r.Location(s, pid)
		if e == nil && !loc.Active {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if e != nil || loc.Active {
		t.Fatal("HALCON remained active after disconnect", e)
	}
	t.Log("real HALCON login, identity, WebSocket, GPS persistence and disconnect passed")
}
