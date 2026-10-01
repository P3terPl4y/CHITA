package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestReverseGeocoderCacheAndValidation(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/reverse" || r.URL.Query().Get("lat") == "" || r.Header.Get("User-Agent") == "" {
			t.Error("invalid provider request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"features":[{"properties":{"street":"Calle Central","housenumber":"12","city":"Habana","state":"Habana","country":"Cuba","osm_id":1234}}]}`))
	}))
	defer server.Close()
	geo := NewGeocoder(server.URL)
	address, e := geo.Reverse(context.Background(), 23.1, -82.3)
	if e != nil || address != "Calle Central 12, Habana, Cuba" {
		t.Fatal(address, e)
	}
	if _, e = geo.Reverse(context.Background(), 23.1, -82.3); e != nil || requests.Load() != 1 {
		t.Fatal("cache failed", e)
	}
	if _, e = geo.Reverse(context.Background(), 24, -82); e == nil {
		t.Fatal("global rate limit missing")
	}
	if _, e = geo.Reverse(context.Background(), 100, 0); e == nil {
		t.Fatal("invalid coordinates accepted")
	}
}
func TestReverseGeocoderFailures(t *testing.T) {
	for _, body := range []string{`{"features":[]}`, `not-json`, `{"features":[{"properties":{"country":""}}]}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
			defer server.Close()
			if _, e := NewGeocoder(server.URL).Reverse(context.Background(), 0, 0); e == nil {
				t.Fatal("unusable address accepted")
			}
		})
	}
}
