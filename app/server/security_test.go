package server_test

import (
	"goravel/app/server"
	"goravel/app/services"
	"io"
	"net/http"
	"strings"
	"testing"
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
