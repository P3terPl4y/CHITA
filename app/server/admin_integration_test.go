package server_test

import (
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/server"
	"goravel/app/services"
	"goravel/bootstrap"
	"goravel/database/migrations"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestAdminCRUDAndSecurity(t *testing.T) {
	if os.Getenv("CHITA_INTEGRATION") != "1" {
		t.Skip("requires isolated PostgreSQL")
	}
	if !strings.HasSuffix(os.Getenv("DB_DATABASE"), "_validation") {
		t.Fatal("isolated database required")
	}
	bootstrap.Boot()
	facades.Config().Add("app.key", "0123456789abcdef0123456789abcdef")
	if _, e := facades.Orm().Query().Exec("TRUNCATE users RESTART IDENTITY CASCADE"); e != nil {
		t.Fatal(e)
	}
	remote, _ := services.NewRemote("http://127.0.0.1:9")
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	app := server.New(nil, false, tracking)
	anon := newClient(t, app)
	anon.want(401, "GET", "/api/admin/overview", nil)
	company := newClient(t, app)
	company.register("regular-company@chita.test", "company")
	courier := newClient(t, app)
	courier.register("regular-courier@chita.test", "courier")
	for _, c := range []*client{company, courier} {
		for _, path := range []string{"overview", "audits", "companies", "couriers", "jobs"} {
			c.want(403, "GET", "/api/admin/"+path, nil)
		}
		for _, entity := range []string{"companies", "couriers", "jobs"} {
			base := "/api/admin/" + entity
			c.want(403, "POST", base, map[string]any{})
			c.want(403, "GET", base+"/1", nil)
			c.want(403, "PUT", base+"/1", map[string]any{})
			c.want(403, "DELETE", base+"/1", map[string]any{})
			c.want(403, "POST", base+"/1/restore", map[string]any{})
		}
	}
	anon.want(422, "POST", "/api/auth/register", map[string]any{"role": "admin"})
	if e := services.CreateAdministrator("Admin prueba", "admin@chita.test", "StrongAdmin123!"); e != nil {
		t.Fatal(e)
	}
	if e := services.CreateAdministrator("Conflict", "regular-company@chita.test", "StrongAdmin123!"); e == nil {
		t.Fatal("converted existing account")
	}
	admin := newClient(t, app)
	admin.want(200, "POST", "/api/auth/login", map[string]any{"email": "admin@chita.test", "password": "StrongAdmin123!"})
	admin.session()
	admin.want(200, "GET", "/api/admin/overview", nil)
	input := func(email string) map[string]any {
		return map[string]any{"name": "Cuenta prueba", "email": email, "phone": "12345678", "password": "StrongTest123!", "address": "Calle prueba 15", "latitude": 23.1, "longitude": -82.3, "company_name": "Empresa Admin", "vehicle_type": "bicycle", "enabled": true, "reason": "Alta administrativa de prueba"}
	}
	for _, entity := range []string{"companies", "couriers"} {
		t.Run(entity, func(t *testing.T) {
			admin.t = t
			r := input(entity + "@chita.test")
			bad := input("injection@chita.test")
			bad["role"] = "admin"
			admin.want(422, "POST", "/api/admin/"+entity, bad)
			csrf := admin.token
			admin.token = "invalid"
			admin.want(403, "POST", "/api/admin/"+entity, r)
			admin.token = csrf
			x := admin.want(201, "POST", "/api/admin/"+entity, r)
			id := uint(x["id"].(float64))
			path := fmt.Sprintf("/api/admin/%s/%d", entity, id)
			_, _, raw := admin.req("GET", path, nil)
			for _, secret := range []string{"password_hash", "password", "session", "metadata"} {
				if strings.Contains(string(raw), secret) {
					t.Fatal("private field leaked", secret)
				}
			}
			list := admin.want(200, "GET", "/api/admin/"+entity+"?search="+entity+"@chita.test", nil)
			if list["total"].(float64) != 1 {
				t.Fatal("search failed", list)
			}
			admin.want(422, "GET", "/api/admin/"+entity+"?state=invalid", nil)
			admin.want(422, "GET", "/api/admin/"+entity+"?page=-1", nil)
			r["password"] = ""
			r["version"] = x["version"]
			r["name"] = "Nombre editado"
			x = admin.want(200, "PUT", path, r)
			admin.want(409, "PUT", path, r)
			if x["name"] != "Nombre editado" {
				t.Fatal("edit failed")
			}
			login := newClient(t, app)
			login.want(200, "POST", "/api/auth/login", map[string]any{"email": r["email"], "password": "StrongTest123!"})
			login.session()
			r["version"] = x["version"]
			r["enabled"] = false
			x = admin.want(200, "PUT", path, r)
			login.want(401, "GET", "/api/profile", nil)
			disabled := admin.want(200, "GET", "/api/admin/"+entity+"?state=disabled&search="+entity+"@chita.test", nil)
			if disabled["total"].(float64) != 1 {
				t.Fatal("disabled filter failed", disabled)
			}
			active := admin.want(200, "GET", "/api/admin/"+entity+"?state=active&search="+entity+"@chita.test", nil)
			if active["total"].(float64) != 0 {
				t.Fatal("active filter included disabled account", active)
			}
			r["version"] = x["version"]
			r["enabled"] = true
			x = admin.want(200, "PUT", path, r)
			login.want(401, "GET", "/api/profile", nil)
			x = admin.want(200, "DELETE", path, map[string]any{"version": x["version"], "reason": "Archivar cuenta de prueba"})
			if x["archived"] != true {
				t.Fatal("archive failed")
			}
			login.want(401, "POST", "/api/auth/login", map[string]any{"email": r["email"], "password": "StrongTest123!"})
			list = admin.want(200, "GET", "/api/admin/"+entity+"?state=archived", nil)
			if list["total"].(float64) != 1 {
				t.Fatal("archive filter failed", list)
			}
			x = admin.want(200, "POST", path+"/restore", map[string]any{"version": x["version"], "reason": "Restaurar cuenta de prueba"})
			if x["enabled"] != false || x["archived"] != false {
				t.Fatal("restore must remain disabled")
			}
			r["version"] = x["version"]
			x = admin.want(200, "PUT", path, r)
			login.want(200, "POST", "/api/auth/login", map[string]any{"email": r["email"], "password": "StrongTest123!"})
		})
	}
	admin.t = t
	// Full job CRUD and business constraints with real courier acceptance.
	deliveryApp := server.New(nil, false, tracking)
	owner := newClient(t, deliveryApp)
	owner.want(200, "POST", "/api/auth/login", map[string]any{"email": "companies@chita.test", "password": "StrongTest123!"})
	owner.session()
	rider := newClient(t, deliveryApp)
	rider.want(200, "POST", "/api/auth/login", map[string]any{"email": "couriers@chita.test", "password": "StrongTest123!"})
	rider.session()
	companies := admin.want(200, "GET", "/api/admin/companies?search=companies@chita.test", nil)
	co := companies["items"].([]any)[0].(map[string]any)
	couriers := admin.want(200, "GET", "/api/admin/couriers?search=couriers@chita.test", nil)
	cu := couriers["items"].([]any)[0].(map[string]any)
	r := jobInput()
	r["company_id"] = co["id"]
	r["reason"] = "Crear trabajo público"
	r["visibility"] = "public"
	job := admin.want(201, "POST", "/api/admin/jobs", r)
	path := fmt.Sprintf("/api/admin/jobs/%.0f", job["id"])
	normal := fmt.Sprintf("/api/jobs/%.0f", job["id"])
	r["version"] = job["version"]
	r["title"] = "Trabajo editado"
	job = admin.want(200, "PUT", path, r)
	admin.want(409, "PUT", path, r)
	job = admin.want(200, "DELETE", path, map[string]any{"version": job["version"], "reason": "Archivar publicación disponible"})
	rider.want(404, "GET", normal, nil)
	job = admin.want(200, "POST", path+"/restore", map[string]any{"version": job["version"], "reason": "Restaurar publicación disponible"})
	rider.want(200, "POST", normal+"/accept", map[string]any{})
	job = admin.want(200, "GET", path, nil)
	r["version"] = job["version"]
	admin.want(409, "PUT", path, r)
	admin.want(409, "DELETE", path, map[string]any{"version": job["version"], "reason": "Intento archivar entrega activa"})
	for _, pair := range []struct {
		entity string
		record map[string]any
	}{{"companies", co}, {"couriers", cu}} {
		entityPath := fmt.Sprintf("/api/admin/%s/%.0f", pair.entity, pair.record["id"])
		x := input(pair.entity + "@chita.test")
		x["password"] = ""
		x["version"] = pair.record["version"]
		x["enabled"] = false
		admin.want(409, "PUT", entityPath, x)
		admin.want(409, "DELETE", entityPath, map[string]any{"version": pair.record["version"], "reason": "Intento archivar cuenta ocupada"})
	}
	rider.want(200, "POST", normal+"/pickup", map[string]any{})
	job = admin.want(200, "GET", path, nil)
	admin.want(409, "POST", path+"/cancel", map[string]any{"version": job["version"], "reason": "Cancelar después de recogida"})
	rider.want(200, "POST", normal+"/arrive", map[string]any{})
	rider.want(200, "POST", normal+"/report", map[string]any{})
	owner.want(200, "POST", normal+"/confirm", map[string]any{})
	job = admin.want(200, "GET", path, nil)
	job = admin.want(200, "DELETE", path, map[string]any{"version": job["version"], "reason": "Archivar entrega completada"})
	if job["status"] != "completed" {
		t.Fatal("history destroyed")
	}
	admin.want(200, "POST", path+"/restore", map[string]any{"version": job["version"], "reason": "Restaurar historial de entrega"})
	r = jobInput()
	r["company_id"] = co["id"]
	r["reason"] = "Trabajo para cancelación"
	job = admin.want(201, "POST", "/api/admin/jobs", r)
	path = fmt.Sprintf("/api/admin/jobs/%.0f", job["id"])
	job = admin.want(200, "POST", path+"/cancel", map[string]any{"version": job["version"], "reason": "Cancelación administrativa válida"})
	if job["status"] != "cancelled" {
		t.Fatal("cancel failed")
	}
	// Concurrent administrative edits must not silently overwrite one another.
	var actor models.User
	if e := facades.Orm().Query().Where("email=?", "admin@chita.test").First(&actor); e != nil {
		t.Fatal(e)
	}
	target, e := services.AdminAccountDetail(&actor, "couriers", uint(cu["id"].(float64)))
	if e != nil {
		t.Fatal(e)
	}
	before, e := facades.Orm().Query().Table("admin_audits").Count()
	if e != nil {
		t.Fatal(e)
	}
	enabled := true
	lat, lng := target.Latitude, target.Longitude
	update := services.AdminAccountInput{Name: target.Name, Email: target.Email, Phone: target.Phone, Address: target.Address, Latitude: &lat, Longitude: &lng, VehicleType: target.VehicleType, Enabled: &enabled, Version: target.Version, Reason: "Edición concurrente de prueba"}
	results := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, e := services.AdminSaveAccount(&actor, "couriers", target.ID, update)
			results <- e
		}()
	}
	group.Wait()
	close(results)
	success, conflict := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if p, ok := e.(*services.Problem); ok && p.Status == 409 {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("concurrent edit lost update", success, conflict)
	}
	after, e := facades.Orm().Query().Table("admin_audits").Count()
	if e != nil || after != before+1 {
		t.Fatal("failed edit committed audit", before, after, e)
	}
	// Deactivation racing an already authorized acceptance must never assign
	// a delivery to an inactive courier.
	for attempt := 0; attempt < 3; attempt++ {
		enabled := true
		cr := services.AdminAccountInput{Name: "Repartidor concurrente", Email: fmt.Sprintf("race-admin-%d@chita.test", attempt), Phone: "12345678", Password: "RaceCourier123!", Address: "Calle concurrencia 1", Latitude: &lat, Longitude: &lng, VehicleType: "bicycle", Enabled: &enabled, Reason: "Alta para concurrencia"}
		uid, e := services.AdminSaveAccount(&actor, "couriers", 0, cr)
		if e != nil {
			t.Fatal(e)
		}
		rider, e := services.User(uid)
		if e != nil {
			t.Fatal(e)
		}
		details, e := services.AdminAccountDetail(&actor, "couriers", uid)
		if e != nil {
			t.Fatal(e)
		}
		ji := jobInput()
		ji["company_id"] = co["id"]
		ji["reason"] = "Trabajo para concurrencia"
		ji["visibility"] = "public"
		row := admin.want(201, "POST", "/api/admin/jobs", ji)
		jobID := uint(row["id"].(float64))
		disabled := false
		cr.Enabled = &disabled
		cr.Password = ""
		cr.Version = details.Version
		start := make(chan struct{})
		var acceptError, disableError error
		var wait sync.WaitGroup
		wait.Add(2)
		go func() { defer wait.Done(); <-start; _, acceptError = services.Transition(rider, jobID, "accept", "") }()
		go func() {
			defer wait.Done()
			<-start
			_, disableError = services.AdminSaveAccount(&actor, "couriers", uid, cr)
		}()
		close(start)
		wait.Wait()
		var account models.User
		var delivery models.Publication
		if e := facades.Orm().Query().Where("id=?", uid).First(&account); e != nil {
			t.Fatal(e)
		}
		if e := facades.Orm().Query().Where("id=?", jobID).First(&delivery); e != nil {
			t.Fatal(e)
		}
		if delivery.AssignedCourierID != nil && !account.Status {
			t.Fatal("inactive courier accepted a delivery")
		}
		if acceptError == nil {
			if p, ok := disableError.(*services.Problem); !ok || p.Status != 409 {
				t.Fatal("deactivation did not detect assignment", disableError)
			}
		} else {
			if disableError != nil {
				t.Fatal("both race operations failed", acceptError, disableError)
			}
			if p, ok := acceptError.(*services.Problem); !ok || p.Status != 403 {
				t.Fatal("acceptance did not reject disabled account", acceptError)
			}
		}
	}
	// Database history may not be discarded by reverting the admin migration.
	if e := (&migrations.M20261001162141AdminManagement{}).Down(); e == nil {
		t.Fatal("migration discarded administrators or audits")
	}
	audits := admin.want(200, "GET", "/api/admin/audits", nil)
	if audits["total"].(float64) < 20 {
		t.Fatal("missing audits", audits)
	}
	admin.want(422, "POST", "/api/admin/password", map[string]any{"current_password": "incorrect", "password": "ChangedAdmin123!"})
	admin.want(204, "POST", "/api/admin/password", map[string]any{"current_password": "StrongAdmin123!", "password": "ChangedAdmin123!"})
	admin.want(401, "GET", "/api/admin/overview", nil)
	admin.want(200, "POST", "/api/auth/login", map[string]any{"email": "admin@chita.test", "password": "ChangedAdmin123!"})
	t.Log("Admin CRUD, role isolation, CSRF, mass-assignment, archive/restore, session revocation and active delivery constraints verified")
}
