package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"goravel/app/services"
)

func TestHealthzIsAvailableWithoutSession(t *testing.T) {
	app := New(nil, false, services.NewTracking(nil))
	response, err := app.Test(httptest.NewRequest("GET", "/healthz", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("GET /healthz returned %d, want %d", response.StatusCode, fiber.StatusNoContent)
	}
}
