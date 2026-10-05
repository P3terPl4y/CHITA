package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"goravel/app/services"
)

func TestPublicMapSearchRoute(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/" || r.URL.Query().Get("q") != "Calle Central" {
			t.Errorf("unexpected provider request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"features":[{"geometry":{"coordinates":[-82.3,23.1]},"properties":{"street":"Calle Central","city":"Habana","country":"Cuba"}}]}`))
	}))
	defer provider.Close()
	remote, err := services.NewRemote("http://127.0.0.1:3300")
	if err != nil {
		t.Fatal(err)
	}
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	app := New(nil, false, tracking, services.NewGeocoder(provider.URL))
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/maps/search?q=Calle%20Central", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("search route status=%d", response.StatusCode)
	}
	var results []services.GeocodeResult
	if err := json.NewDecoder(response.Body).Decode(&results); err != nil || len(results) != 1 {
		t.Fatalf("search route response=%+v, err=%v", results, err)
	}
	if results[0].Latitude != 23.1 || results[0].Longitude != -82.3 {
		t.Fatalf("search route returned invalid point: %+v", results[0])
	}
	invalid, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/maps/search?q=x", nil))
	if err != nil {
		t.Fatal(err)
	}
	invalid.Body.Close()
	if invalid.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("short search status=%d", invalid.StatusCode)
	}
}
