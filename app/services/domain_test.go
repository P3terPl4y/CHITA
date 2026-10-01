package services

import (
	"math"
	"testing"
	"time"
)

func TestCoordinates(t *testing.T) {
	for _, x := range []struct {
		a, b float64
		ok   bool
	}{{0, 0, true}, {90, 180, true}, {-90, -180, true}, {90.1, 0, false}, {0, 180.1, false}, {math.NaN(), 0, false}, {0, math.Inf(1), false}} {
		if ValidPoint(x.a, x.b) != x.ok {
			t.Errorf("point %v %v", x.a, x.b)
		}
	}
}
func TestTransitions(t *testing.T) {
	states := []string{"published", "accepted", "picked_up", "arrived", "delivery_reported", "completed", "cancelled"}
	actions := []string{"accept", "pickup", "arrive", "report", "confirm", "reject", "cancel", "unknown"}
	for _, s := range states {
		for _, a := range actions {
			for _, r := range []string{"company", "courier"} {
				n, e := NextState(s, a, r)
				expected := (s == "published" && a == "accept" && r == "courier") || (s == "accepted" && a == "pickup" && r == "courier") || (s == "picked_up" && a == "arrive" && r == "courier") || (s == "arrived" && a == "report" && r == "courier") || (s == "delivery_reported" && (a == "confirm" || a == "reject") && r == "company") || ((s == "published" || s == "accepted") && a == "cancel" && r == "company")
				if (e == nil) != expected {
					t.Fatalf("%s %s %s: %s %v", s, a, r, n, e)
				}
			}
		}
	}
}
func TestInput(t *testing.T) {
	for _, v := range []string{"x", "Name <a@b.com>", "a@@b.com"} {
		if _, e := Email(v); e == nil {
			t.Fatal(v)
		}
	}
	if x, e := Email(" TEST@example.com "); e != nil || x != "test@example.com" {
		t.Fatal(x, e)
	}
	for _, v := range []string{"short", ""} {
		if Password(v) == nil {
			t.Fatal(v)
		}
	}
	if _, e := ParseTime("2026-10-01T12:00"); e == nil {
		t.Fatal("timezone required")
	}
	if _, e := NewRemote("http://example.com"); e == nil {
		t.Fatal("insecure remote allowed")
	}
	if _, e := NewRemote("https://user:password@example.com"); e == nil {
		t.Fatal("userinfo allowed")
	}
}
func TestWindows(t *testing.T) {
	now := time.Now().UTC()
	p := 0.
	r := JobInput{Title: "Entrega", PickupAddress: "Calle uno", DropoffAddress: "Calle dos", PickupLat: &p, PickupLng: &p, DropoffLat: &p, DropoffLng: &p, PickupFrom: now.Add(time.Hour).Format(time.RFC3339), PickupTo: now.Add(2 * time.Hour).Format(time.RFC3339), DeliveryFrom: now.Add(2 * time.Hour).Format(time.RFC3339), DeliveryTo: now.Add(3 * time.Hour).Format(time.RFC3339), PriceCents: 123, Currency: "USD"}
	if _, e := r.Validate(now); e != nil {
		t.Fatal(e)
	}
	r.DeliveryTo = r.PickupFrom
	if _, e := r.Validate(now); e == nil {
		t.Fatal("invalid window")
	}
}
