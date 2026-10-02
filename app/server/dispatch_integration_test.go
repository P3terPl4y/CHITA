package server_test

import (
	"encoding/json"
	"fmt"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/server"
	"goravel/app/services"
	"goravel/bootstrap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDispatchLocationsOffersAndRatings(t *testing.T) {
	if os.Getenv("CHITA_INTEGRATION") != "1" {
		t.Skip("requires isolated PostgreSQL")
	}
	if !strings.HasSuffix(os.Getenv("DB_DATABASE"), "_validation") {
		t.Fatal("isolated database required")
	}
	bootstrap.Boot()
	if facades.Config().GetString("telemetry.exporters.otlplog.protocol") != "http/protobuf" {
		t.Fatal("the incompatible gRPC log exporter must remain disabled")
	}
	facades.Config().Add("app.key", "0123456789abcdef0123456789abcdef")
	if _, e := facades.Orm().Query().Exec("TRUNCATE users RESTART IDENTITY CASCADE"); e != nil {
		t.Fatal(e)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"features":[{"properties":{"street":"Calle del mapa","housenumber":"15","city":"Habana","country":"Cuba","osm_id":123}}]}`)
	}))
	defer provider.Close()
	remote, _ := services.NewRemote("http://127.0.0.1:9")
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	app := server.New(nil, false, tracking, services.NewGeocoder(provider.URL))
	anon := newClient(t, app)
	geocoded := anon.want(200, "GET", "/api/maps/reverse?lat=23.1&lng=-82.3", nil)
	if geocoded["address"] != "Calle del mapa 15, Habana, Cuba" {
		t.Fatal("geocoder failed", geocoded)
	}
	anon.want(422, "GET", "/api/maps/reverse?lat=NaN&lng=0", nil)
	anon.want(401, "GET", "/api/couriers/nearby?lat=23&lng=-82", nil)
	company := newClient(t, app)
	company.register("dispatch-company@chita.test", "company")
	otherCompany := newClient(t, app)
	otherCompany.register("dispatch-other@chita.test", "company")
	courier := newClient(t, app)
	uid := courier.register("dispatch-courier@chita.test", "courier")
	other := newClient(t, app)
	otherID := other.register("dispatch-other-courier@chita.test", "courier")
	hidden := newClient(t, app)
	hidden.register("dispatch-hidden@chita.test", "courier")
	for _, c := range []*client{company, courier} {
		c.want(204, "PUT", "/api/profile/location", map[string]any{"address": "Dirección actualizada", "latitude": 23.11345, "longitude": -82.3667})
		profile := c.want(200, "GET", "/api/profile", nil)["profile"].(map[string]any)
		if profile["address"] != "Dirección actualizada" {
			t.Fatal("profile update failed")
		}
		c.want(422, "PUT", "/api/profile/location", map[string]any{"address": "No válida", "latitude": 91, "longitude": 0})
		c.want(422, "PUT", "/api/profile/location", map[string]any{"address": "No válida", "latitude": 23, "longitude": -82, "role": "admin"})
	}
	company.want(403, "PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23, "longitude": -82})
	courier.want(422, "PUT", "/api/availability", map[string]any{"enabled": true})
	consent := courier.want(200, "PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23.11345, "longitude": -82.3667})["token"]
	courier.want(200, "PUT", "/api/availability", map[string]any{"enabled": true, "token": consent, "latitude": 23.11345, "longitude": -82.3667})
	courier.want(204, "PUT", "/api/availability", map[string]any{"enabled": false})
	courier.want(409, "PUT", "/api/availability", map[string]any{"enabled": true, "token": consent, "latitude": 23.11345, "longitude": -82.3667})
	renewed := courier.want(200, "PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23.11345, "longitude": -82.3667})["token"]
	if renewed == consent {
		t.Fatal("revoked consent reused")
	}
	other.want(200, "PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23.115, "longitude": -82.3667})
	company.want(422, "GET", "/api/couriers/nearby?lat=23&lng=-82&radius=NaN", nil)
	courier.want(403, "GET", "/api/couriers/nearby?lat=23&lng=-82", nil)
	list := func() []map[string]any {
		status, _, raw := company.req("GET", "/api/couriers/nearby?lat=23.11345&lng=-82.3667&radius=2", nil)
		if status != 200 {
			t.Fatal(status, string(raw))
		}
		var rows []map[string]any
		if e := json.Unmarshal(raw, &rows); e != nil {
			t.Fatal(e)
		}
		return rows
	}
	rows := list()
	if len(rows) != 2 || uint(rows[0]["id"].(float64)) != uid {
		t.Fatal("nearby order or consent failed", rows)
	}
	for _, row := range rows {
		for _, secret := range []string{"email", "phone", "password_hash", "address", "session_ciphertext"} {
			if _, ok := row[secret]; ok {
				t.Fatal("discovery leaked private data")
			}
		}
	}
	hidden.want(403, "PUT", fmt.Sprintf("/api/couriers/%d/rating", uid), map[string]any{"rating": 5})
	company.want(403, "PUT", fmt.Sprintf("/api/couriers/%d/rating", uid), map[string]any{"rating": 5})
	// An exclusive-network publication may be proposed to a consenting outside courier.
	ji := jobInput()
	job := company.want(201, "POST", "/api/jobs", ji)
	path := fmt.Sprintf("/api/jobs/%.0f", job["id"])
	courier.want(404, "GET", path, nil)
	offer := company.want(201, "POST", path+"/offer", map[string]any{"courier_id": uid})
	offerPath := fmt.Sprintf("/api/offers/%.0f", offer["id"])
	courier.want(200, "GET", path, nil)
	other.want(404, "GET", path, nil)
	other.want(404, "POST", offerPath+"/accept", map[string]any{})
	courier.want(403, "POST", path+"/offer", map[string]any{"courier_id": otherID})
	otherCompany.want(404, "POST", path+"/offer", map[string]any{"courier_id": otherID})
	courier.want(200, "GET", "/api/offers", nil)
	company.want(200, "GET", "/api/offers", nil)
	// Recipient consent, not invitation to a network, authorizes this assignment.
	courier.want(204, "POST", offerPath+"/accept", map[string]any{})
	accepted := company.want(200, "GET", path, nil)
	if accepted["status"] != "accepted" || uint(accepted["assigned_courier_id"].(float64)) != uid {
		t.Fatal("proposal did not assign")
	}
	courier.want(409, "POST", offerPath+"/accept", map[string]any{})
	company.want(403, "PUT", fmt.Sprintf("/api/couriers/%d/rating", uid), map[string]any{"rating": 5})
	if len(list()) != 1 {
		t.Fatal("busy courier remained available")
	}
	company.want(200, "POST", path+"/cancel", map[string]any{"note": "Cancelar trabajo de prueba"})
	courier.want(200, "PUT", "/api/availability", map[string]any{"enabled": true, "latitude": 23.11345, "longitude": -82.3667})
	ji = jobInput()
	ji["visibility"] = "public"
	job = company.want(201, "POST", "/api/jobs", ji)
	path = fmt.Sprintf("/api/jobs/%.0f", job["id"])
	offer = company.want(201, "POST", path+"/offer", map[string]any{"courier_id": uid})
	offerPath = fmt.Sprintf("/api/offers/%.0f", offer["id"])
	other.want(404, "GET", path, nil)
	other.want(404, "POST", path+"/accept", map[string]any{})
	courier.want(204, "POST", offerPath+"/decline", map[string]any{})
	other.want(200, "GET", path, nil)
	offer = company.want(201, "POST", path+"/offer", map[string]any{"courier_id": uid})
	offerPath = fmt.Sprintf("/api/offers/%.0f", offer["id"])
	if _, e := facades.Orm().Query().Model(&models.DeliveryOffer{}).Where("id=?", offer["id"]).Update("expires_at", time.Now().UTC().Add(-time.Second)); e != nil {
		t.Fatal(e)
	}
	courier.want(409, "POST", offerPath+"/accept", map[string]any{})
	other.want(200, "GET", path, nil)
	offer = company.want(201, "POST", path+"/offer", map[string]any{"courier_id": uid})
	offerPath = fmt.Sprintf("/api/offers/%.0f", offer["id"])
	otherCompany.want(404, "POST", offerPath+"/withdraw", map[string]any{})
	company.want(204, "POST", offerPath+"/withdraw", map[string]any{})
	// Multiple callers cannot reserve the same courier twice.
	var owner models.User
	if e := facades.Orm().Query().Where("email=?", "dispatch-company@chita.test").First(&owner); e != nil {
		t.Fatal(e)
	}
	job2 := company.want(201, "POST", "/api/jobs", ji)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, id := range []uint{uint(job["id"].(float64)), uint(job2["id"].(float64))} {
		wait.Add(1)
		go func(id uint) { defer wait.Done(); <-start; _, e := services.OfferJob(&owner, id, uid); results <- e }(id)
	}
	close(start)
	wait.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else if p, ok := e.(*services.Problem); !ok || p.Status != 409 {
			t.Fatal(e)
		}
	}
	if success != 1 {
		t.Fatal("double reservation", success)
	}
	var pending models.DeliveryOffer
	if e := facades.Orm().Query().Where("courier_user_id=? AND status='pending'", uid).First(&pending); e != nil {
		t.Fatal(e)
	}
	company.want(204, "POST", fmt.Sprintf("/api/offers/%d/withdraw", pending.ID), map[string]any{})
	// Three distinct confirmed deliveries for this company, including archived history.
	ratingPath := fmt.Sprintf("/api/couriers/%d/rating", uid)
	completed := []uint{}
	for i := 0; i < 3; i++ {
		j := company.want(201, "POST", "/api/jobs", ji)
		jp := fmt.Sprintf("/api/jobs/%.0f", j["id"])
		courier.want(200, "POST", jp+"/accept", map[string]any{})
		for _, action := range []string{"pickup", "arrive", "report"} {
			courier.want(200, "POST", jp+"/"+action, map[string]any{})
		}
		company.want(200, "POST", jp+"/confirm", map[string]any{})
		completed = append(completed, uint(j["id"].(float64)))
		summary := company.want(200, "GET", ratingPath, nil)
		if summary["completed_jobs"].(float64) != float64(i+1) {
			t.Fatal("wrong completed count", summary)
		}
		if i < 2 {
			company.want(403, "PUT", ratingPath, map[string]any{"rating": 5})
		}
	}
	otherCompany.want(403, "PUT", ratingPath, map[string]any{"rating": 5})
	company.want(422, "PUT", ratingPath, map[string]any{"rating": 6})
	company.want(422, "PUT", ratingPath, map[string]any{"rating": 5, "company_id": 2})
	if _, e := facades.Orm().Query().Where("id=?", completed[0]).Delete(&models.Publication{}); e != nil {
		t.Fatal(e)
	}
	summary := company.want(200, "PUT", ratingPath, map[string]any{"rating": 5, "comment": "Tres entregas confirmadas"})
	if summary["count"].(float64) != 1 || summary["average"].(float64) != 5 {
		t.Fatal("rating creation failed", summary)
	}
	summary = company.want(200, "PUT", ratingPath, map[string]any{"rating": 3, "comment": "Actualización"})
	if summary["count"].(float64) != 1 || summary["average"].(float64) != 3 || summary["completed_jobs"].(float64) != 3 {
		t.Fatal("rating weighting or archived count failed", summary)
	}
	directory := company.want(200, "GET", "/api/couriers/directory", nil)
	foundRated := false
	for _, item := range directory["items"].([]any) {
		entry := item.(map[string]any)
		if entry["id"] == float64(uid) {
			foundRated = true
			if entry["average_rating"] != float64(3) || entry["rating_count"] != float64(1) {
				t.Fatal("directory rating differs from real rating", entry)
			}
		}
	}
	if !foundRated {
		t.Fatal("rated courier missing from directory")
	}
	courier.want(200, "GET", ratingPath, nil)
	other.want(403, "GET", ratingPath, nil)
	courier.want(204, "PUT", "/api/availability", map[string]any{"enabled": false})
	if len(list()) != 1 {
		t.Fatal("consent revocation failed")
	}
	if _, e := facades.Orm().Query().Model(&models.CourierProfile{}).Where("user_id=?", otherID).Update("available_until", time.Now().UTC().Add(-time.Second)); e != nil {
		t.Fatal(e)
	}
	if len(list()) != 0 {
		t.Fatal("stale GPS location leaked")
	}
	// A request authorized before logout cannot reopen discovery afterwards.
	var grant struct{ Token string }
	if e := facades.Orm().Query().Table("auth_grants").Where("user_id=?", uid).First(&grant); e != nil {
		t.Fatal(e)
	}
	staleActor, e := services.User(uid)
	if e != nil {
		t.Fatal(e)
	}
	courier.want(204, "POST", "/api/auth/logout", map[string]any{})
	enabled := true
	lat, lng := 23.11345, -82.3667
	if _, e := services.SetAvailability(staleActor, services.AvailabilityInput{Enabled: &enabled, Latitude: &lat, Longitude: &lng}, grant.Token); e == nil {
		t.Fatal("stale session reopened discovery")
	}
	if e := services.SaveLocation(staleActor, services.LocationInput{Address: "Cambio con sesión revocada", Latitude: &lat, Longitude: &lng}, grant.Token); e == nil {
		t.Fatal("stale session changed reference location after logout")
	}
	t.Log("Maps, profile locations, voluntary discovery, offer consent/expiry/reservation races and three-delivery rating eligibility passed")
}
