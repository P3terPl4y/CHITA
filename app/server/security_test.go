package server_test

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"goravel/app/server"
	"goravel/app/services"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSessionAndSecurityHeaders(t *testing.T) {
	for _, prod := range []bool{false, true} {
		r, _ := services.NewRemote("http://127.0.0.1:3300")
		tr := services.NewTracking(r)
		a := server.New(nil, prod, tr)
		req, _ := http.NewRequest("GET", "https://chita.test/api/session", nil)
		res, e := a.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 || !strings.Contains(string(body), "csrf_token") {
			t.Fatal(string(body))
		}
		if res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("X-Frame-Options") != "DENY" || res.Header.Get("Content-Security-Policy") == "" {
			t.Fatal("security headers missing")
		}
		if res.Header.Get("Cross-Origin-Embedder-Policy") == "require-corp" {
			t.Fatal("map tiles would be blocked")
		}
		sessionCookie := false
		for _, c := range res.Cookies() {
			if strings.Contains(c.Name, "session") {
				sessionCookie = true
				if !c.HttpOnly || c.Secure != prod || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
					t.Fatalf("cookie policy %+v", c)
				}
				if prod && c.Name != "__Host-chita-session" {
					t.Fatal(c.Name)
				}
			}
		}
		if !sessionCookie {
			t.Fatal("session cookie missing")
		}
		req, _ = http.NewRequest("POST", "https://chita.test/api/auth/login", strings.NewReader(`{"email":"x@example.test","password":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		res, e = a.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != 403 {
			t.Fatal("CSRF missing accepted", res.StatusCode)
		}
		tr.Close()
	}
}

// Cloudflare appends the real visitor to X-Forwarded-For. A client-controlled
// prefix must not create a different rate-limit bucket for every request.
func TestForwardedHeadersCannotBypassMapRateLimit(t *testing.T) {
	remote, _ := services.NewRemote("http://127.0.0.1:9")
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	app := server.New(nil, false, tracking)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Error(err)
		}
		if err := <-finished; err != nil {
			t.Error(err)
		}
	}()
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < 22; i++ {
		req, err := http.NewRequest("GET", "http://"+listener.Addr().String()+"/api/maps/reverse?lat=invalid&lng=invalid", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d, 203.0.113.10", i+1))
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		want := 422
		if i >= 20 {
			want = 429
		}
		if res.StatusCode != want {
			t.Fatalf("request %d: got %d want %d; forged prefix bypassed the limiter", i+1, res.StatusCode, want)
		}
	}
}
