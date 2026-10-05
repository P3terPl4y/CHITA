package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/server"
	"goravel/app/services"
	"goravel/bootstrap"
	"image"
	"image/png"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDirectoryInvitationsAndOwnAvatar(t *testing.T) {
	if os.Getenv("CHITA_INTEGRATION") != "1" {
		t.Skip("requires isolated PostgreSQL")
	}
	if !strings.HasSuffix(os.Getenv("DB_DATABASE"), "_validation") {
		t.Fatal("isolated database required")
	}
	bootstrap.Boot()
	facades.Config().Add("app.key", "0123456789abcdef0123456789abcdef")
	if _, err := facades.Orm().Query().Exec("TRUNCATE users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	remote, _ := services.NewRemote("http://127.0.0.1:9")
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	app := server.New(nil, false, tracking)
	anon := newClient(t, app)
	anon.want(401, "GET", "/api/couriers/directory", nil)
	anon.want(401, "PUT", "/api/profile/avatar", map[string]any{"avatar": ""})
	anon.want(401, "POST", "/api/profile/avatar", map[string]any{"avatar": ""})
	company := newClient(t, app)
	companyID := company.register("directory-company@chita.test", "company")
	otherCompany := newClient(t, app)
	otherCompany.register("directory-company2@chita.test", "company")
	courier := newClient(t, app)
	courierID := courier.register("directory-courier@chita.test", "courier")
	other := newClient(t, app)
	otherID := other.register("directory-other@chita.test", "courier")
	courier.want(403, "GET", "/api/couriers/directory", nil)
	for _, path := range []string{"/api/couriers/directory?page=0", "/api/couriers/directory?page=no", "/api/couriers/directory?search=" + strings.Repeat("a", 101)} {
		company.want(422, "GET", path, nil)
	}
	directory := company.want(200, "GET", "/api/couriers/directory", nil)
	if directory["total"] != float64(2) {
		t.Fatal("wrong directory count", directory)
	}
	for _, row := range directory["items"].([]any) {
		entry := row.(map[string]any)
		for _, secret := range []string{"email", "phone", "address", "latitude", "longitude", "discovery_lat", "password_hash"} {
			if _, exists := entry[secret]; exists {
				t.Fatal("directory leaked private field", secret)
			}
		}
		if entry["membership"] != "none" || entry["rating_count"] != float64(0) {
			t.Fatal("invalid membership/rating", entry)
		}
	}
	if company.want(200, "GET", "/api/couriers/directory?search=%27%20OR%201=1--", nil)["total"] != float64(0) {
		t.Fatal("search injection")
	}
	invitation := company.want(201, "POST", "/api/network", map[string]any{"courier_id": courierID})
	company.want(409, "POST", "/api/network", map[string]any{"courier_id": courierID})
	company.want(422, "POST", "/api/network", map[string]any{"courier_id": courierID, "email": "directory-courier@chita.test"})
	company.want(404, "POST", "/api/network", map[string]any{"courier_id": companyID})
	courier.want(403, "POST", "/api/network", map[string]any{"courier_id": otherID})
	accept := fmt.Sprintf("/api/network/%v/accept", invitation["id"])
	other.want(404, "POST", accept, map[string]any{})
	courier.want(204, "POST", accept, map[string]any{})
	// Membership remains specific to the requesting company.
	for _, check := range []struct {
		c        *client
		expected string
	}{{company, "accepted"}, {otherCompany, "none"}} {
		rows := check.c.want(200, "GET", "/api/couriers/directory", nil)["items"].([]any)
		for _, row := range rows {
			entry := row.(map[string]any)
			if entry["id"] == float64(courierID) && entry["membership"] != check.expected {
				t.Fatal("membership crossed companies", entry)
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 128, 128)))
	photo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	normalized := courier.want(200, "POST", "/api/profile/avatar", map[string]any{"avatar": photo})["avatar_url"].(string)
	if !strings.HasPrefix(normalized, "data:image/jpeg;base64,") {
		t.Fatal("photo was not normalized")
	}
	courier.want(200, "PUT", "/api/profile/avatar", map[string]any{"avatar": photo})
	courier.want(422, "POST", "/api/profile/avatar", map[string]any{"avatar": photo, "user_id": otherID})
	courier.want(422, "PUT", "/api/profile/avatar", map[string]any{"avatar": photo, "user_id": otherID})
	courier.want(422, "PUT", "/api/profile/avatar", map[string]any{})
	courier.want(422, "PUT", "/api/profile/avatar", map[string]any{"avatar": "https://example.com/photo.svg"})
	// app.Test rejects oversized bodies before returning a response. Verify the
	// externally observable status through the actual HTTP listener instead.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	defer func() {
		if err := app.Shutdown(); err != nil {
			t.Error(err)
		}
		if err := <-served; err != nil {
			t.Error(err)
		}
	}()
	request, _ := http.NewRequest("PUT", "http://"+listener.Addr().String()+"/api/profile/avatar", strings.NewReader(`{"avatar":"`+strings.Repeat("a", 33000)+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", courier.token)
	for _, cookie := range courier.cookies {
		request.AddCookie(cookie)
	}
	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 413 {
		t.Fatalf("oversized body status = %d", response.StatusCode)
	}
	token := courier.token
	courier.token = ""
	courier.want(403, "PUT", "/api/profile/avatar", map[string]any{"avatar": ""})
	courier.token = token
	own := courier.want(200, "GET", "/api/profile", nil)["user"].(map[string]any)
	untouched := other.want(200, "GET", "/api/profile", nil)["user"].(map[string]any)
	if own["avatar_url"] != normalized || untouched["avatar_url"] != "" {
		t.Fatal("avatar ownership/persistence failed")
	}
	directory = company.want(200, "GET", "/api/couriers/directory", nil)
	raw, _ := json.Marshal(directory)
	if !strings.Contains(string(raw), "data:image/jpeg;base64,") {
		t.Fatal("directory missing avatar")
	}
	courier.want(200, "PUT", "/api/profile/avatar", map[string]any{"avatar": ""})
	if _, err := facades.Orm().Query().Model(&models.User{}).Where("id=?", otherID).Update("status", false); err != nil {
		t.Fatal(err)
	}
	if company.want(200, "GET", "/api/couriers/directory", nil)["total"] != float64(1) {
		t.Fatal("inactive courier exposed")
	}
	t.Log("Directory privacy, role checks, invitation consent, CSRF, image limits and avatar ownership passed")
}
