package services

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

type LocationInput struct {
	Address   string   `json:"address"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func SaveLocation(u *models.User, r LocationInput, grants ...string) error {
	if u.Role != "company" && u.Role != "courier" {
		return Fail(403, "Esta cuenta no tiene un perfil de ubicación")
	}
	address, e := Text(r.Address, 3, 255, "Dirección")
	if e != nil {
		return e
	}
	if r.Latitude == nil || r.Longitude == nil || !ValidPoint(*r.Latitude, *r.Longitude) {
		return Fail(422, "Indica coordenadas válidas")
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, u.Role); e != nil {
			return e
		}
		if len(grants) > 0 {
			if e := LockGrant(tx, u.ID, grants[0]); e != nil {
				return e
			}
		}
		fields := map[string]any{"address": address, "latitude": *r.Latitude, "longitude": *r.Longitude}
		if u.Role == "company" {
			c, e := Company(tx.LockForUpdate(), u.ID)
			if e != nil {
				return e
			}
			fields["admin_version"] = c.AdminVersion + 1
			if _, e = tx.Model(&models.Company{}).Where("id=?", c.ID).Update(fields); e != nil {
				return e
			}
		} else {
			if _, e := tx.Model(&models.CourierProfile{}).Where("user_id=?", u.ID).Update(fields); e != nil {
				return e
			}
		}
		_, e := tx.Exec("UPDATE users SET admin_version=admin_version+1 WHERE id=?", u.ID)
		return e
	})
}

type AvailabilityInput struct {
	Token     string   `json:"token"`
	Enabled   *bool    `json:"enabled"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func Availability(u *models.User, r AvailabilityInput) error { _, e := SetAvailability(u, r); return e }
func SetAvailability(u *models.User, r AvailabilityInput, grants ...string) (string, error) {
	if u.Role != "courier" {
		return "", Fail(403, "Sólo un repartidor puede activar su disponibilidad")
	}
	if r.Enabled == nil {
		return "", Fail(422, "Indica tu disponibilidad")
	}
	if *r.Enabled && (r.Latitude == nil || r.Longitude == nil || !ValidPoint(*r.Latitude, *r.Longitude)) {
		return "", Fail(422, "Indica coordenadas GPS válidas")
	}
	token := ""
	e := facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, "courier"); e != nil {
			return e
		}
		if len(grants) > 0 {
			if e := LockGrant(tx, u.ID, grants[0]); e != nil {
				return e
			}
		}
		fields := map[string]any{"available_until": nil, "discovery_token": nil}
		if *r.Enabled {
			if e := noActiveJobs(tx, "couriers", u.ID, false); e != nil {
				return e
			}
			var profile models.CourierProfile
			if e := tx.Where("user_id=?", u.ID).First(&profile); e != nil {
				return e
			}
			if profile.ID == 0 {
				return Fail(404, "Perfil no encontrado")
			}
			if r.Token != "" {
				if profile.DiscoveryToken == nil || *profile.DiscoveryToken != r.Token || profile.AvailableUntil == nil || !profile.AvailableUntil.After(time.Now().UTC()) {
					return Fail(409, "La disponibilidad terminó; actívala otra vez para mostrar tu ubicación")
				}
				token = r.Token
			} else {
				token = uuid.NewString()
			}
			fields["discovery_token"] = token
			fields["discovery_lat"] = *r.Latitude
			fields["discovery_lng"] = *r.Longitude
			fields["discovery_at"] = time.Now().UTC()
			fields["available_until"] = time.Now().UTC().Add(5 * time.Minute)
		}
		_, e := tx.Model(&models.CourierProfile{}).Where("user_id=?", u.ID).Update(fields)
		return e
	})
	return token, e
}

type NearbyCourier struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	VehicleType   string    `json:"vehicle_type"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	LastSeen      time.Time `json:"last_seen"`
	DistanceKm    float64   `json:"distance_km"`
	AverageRating float64   `json:"average_rating"`
	RatingCount   int64     `json:"rating_count"`
	CompletedJobs int64     `json:"completed_jobs"`
	PendingOffer  bool      `json:"pending_offer"`
}

func Nearby(u *models.User, lat, lng, radius float64) ([]NearbyCourier, error) {
	if u.Role != "company" {
		return nil, Fail(403, "Sólo una empresa puede buscar repartidores disponibles")
	}
	if !ValidPoint(lat, lng) || !(radius > 0 && radius <= 50) {
		return nil, Fail(422, "Indica un punto válido y un radio entre 0 y 50 km")
	}
	c, e := Company(facades.Orm().Query(), u.ID)
	if e != nil {
		return nil, e
	}
	distance := fmt.Sprintf("6371*acos(least(1.0,greatest(-1.0,sin(radians(%.10f))*sin(radians(p.discovery_lat))+cos(radians(%.10f))*cos(radians(p.discovery_lat))*cos(radians(p.discovery_lng-(%.10f))))))", lat, lat, lng)
	var rows []NearbyCourier
	e = facades.Orm().Query().Raw(`SELECT u.id,u.display_name AS name,p.vehicle_type,p.discovery_lat AS latitude,p.discovery_lng AS longitude,p.discovery_at AS last_seen,`+distance+` AS distance_km,COALESCE((SELECT AVG(r.rating) FROM courier_ratings r WHERE r.courier_user_id=u.id),0) AS average_rating,(SELECT COUNT(*) FROM courier_ratings r WHERE r.courier_user_id=u.id) AS rating_count,(SELECT COUNT(*) FROM publications j WHERE j.company_id=? AND j.assigned_courier_id=u.id AND j.status='completed' AND j.confirmed_at IS NOT NULL) AS completed_jobs,EXISTS(SELECT 1 FROM delivery_offers o WHERE o.courier_user_id=u.id AND o.status='pending' AND o.expires_at>?) AS pending_offer FROM users u JOIN courier_profiles p ON p.user_id=u.id WHERE u.role='courier' AND u.status AND u.deleted_at IS NULL AND p.available_until>? AND NOT EXISTS(SELECT 1 FROM publications j WHERE j.assigned_courier_id=u.id AND j.deleted_at IS NULL AND j.status IN ('accepted','picked_up','arrived','delivery_reported')) AND `+distance+`<=? ORDER BY distance_km,u.id LIMIT 50`, c.ID, time.Now().UTC(), time.Now().UTC(), radius).Scan(&rows)
	if rows == nil {
		rows = []NearbyCourier{}
	}
	return rows, e
}
