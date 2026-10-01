package services

import (
	"goravel/app/models"
	"testing"
)

func TestAdminRejectsNonAdministrativeActors(t *testing.T) {
	for _, u := range []*models.User{nil, {Role: "company"}, {Role: "courier"}, {Role: "Admin"}, {Role: ""}} {
		e := Admin(u)
		p, ok := e.(*Problem)
		if !ok || p.Status != 403 {
			t.Fatalf("unexpected guard result: %v", e)
		}
	}
}
func TestAdminReasonsAndEntityAllowlist(t *testing.T) {
	for _, s := range []string{"", "x", "    ", string(make([]byte, 1001))} {
		if _, e := adminReason(s); e == nil {
			t.Fatalf("invalid reason accepted")
		}
	}
	r, e := adminReason("  Corrección solicitada  ")
	if e != nil || r != "Corrección solicitada" {
		t.Fatalf("invalid normalized reason %q: %v", r, e)
	}
	for _, entity := range []string{"users", "admin", "auth_grants", "companies; DROP TABLE users", ""} {
		if _, e := accountRole(entity); e == nil {
			t.Fatalf("unsupported entity accepted: %s", entity)
		}
	}
	for entity, role := range map[string]string{"companies": "company", "couriers": "courier"} {
		if got, e := accountRole(entity); e != nil || got != role {
			t.Fatalf("wrong immutable role")
		}
	}
}
