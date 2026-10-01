package services

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"math"
	"time"
)

type JobInput struct {
	Visibility     string   `json:"visibility"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	PickupAddress  string   `json:"pickup_address"`
	PickupLat      *float64 `json:"pickup_lat"`
	PickupLng      *float64 `json:"pickup_lng"`
	DropoffAddress string   `json:"dropoff_address"`
	DropoffLat     *float64 `json:"dropoff_lat"`
	DropoffLng     *float64 `json:"dropoff_lng"`
	PickupFrom     string   `json:"pickup_from"`
	PickupTo       string   `json:"pickup_to"`
	DeliveryFrom   string   `json:"delivery_from"`
	DeliveryTo     string   `json:"delivery_to"`
	PriceCents     int64    `json:"price_cents"`
	Currency       string   `json:"currency"`
}

func (r JobInput) Validate(now time.Time) (*models.Publication, error) {
	if r.Visibility == "" {
		r.Visibility = "network"
	}
	if r.Visibility != "network" && r.Visibility != "public" {
		return nil, Fail(422, "Elige visibilidad pública o exclusiva de tu red")
	}
	p := &models.Publication{UUID: uuid.NewString(), ReferenceCode: uuid.NewString()[:18], Status: "published", OfferedPriceCents: r.PriceCents, Currency: r.Currency, PackageCount: 1, PaymentMethod: "external", Metadata: "{}"}
	p.Visibility = r.Visibility
	var err error
	for _, field := range []struct {
		raw      string
		target   *string
		min, max int
		label    string
	}{{r.Title, &p.Title, 3, 200, "Título"}, {r.Description, &p.Description, 0, 3000, "Descripción"}, {r.PickupAddress, &p.PickupAddressText, 3, 255, "Dirección de recogida"}, {r.DropoffAddress, &p.DropoffAddressText, 3, 255, "Dirección de entrega"}} {
		if *field.target, err = Text(field.raw, field.min, field.max, field.label); err != nil {
			return nil, err
		}
	}
	if r.PickupLat == nil || r.PickupLng == nil || r.DropoffLat == nil || r.DropoffLng == nil || !ValidPoint(*r.PickupLat, *r.PickupLng) || !ValidPoint(*r.DropoffLat, *r.DropoffLng) {
		return nil, Fail(422, "Indica coordenadas válidas de recogida y entrega")
	}
	p.PickupLat = *r.PickupLat
	p.PickupLng = *r.PickupLng
	p.DropoffLat = *r.DropoffLat
	p.DropoffLng = *r.DropoffLng
	if p.OfferedPriceCents <= 0 || p.OfferedPriceCents > 1_000_000_000 || !Currency(p.Currency) {
		return nil, Fail(422, "Indica una tarifa positiva en USD, CUP o EUR")
	}
	if p.ScheduledPickupFrom, err = ParseTime(r.PickupFrom); err != nil {
		return nil, err
	}
	if p.ScheduledPickupTo, err = ParseTime(r.PickupTo); err != nil {
		return nil, err
	}
	if p.ScheduledDeliveryFrom, err = ParseTime(r.DeliveryFrom); err != nil {
		return nil, err
	}
	if p.ScheduledDeliveryTo, err = ParseTime(r.DeliveryTo); err != nil {
		return nil, err
	}
	if p.ScheduledPickupFrom.After(p.ScheduledPickupTo) || p.ScheduledDeliveryFrom.After(p.ScheduledDeliveryTo) || p.ScheduledDeliveryFrom.Before(p.ScheduledPickupFrom) || p.ScheduledDeliveryTo.Before(p.ScheduledPickupTo) || !p.ScheduledPickupTo.After(now) {
		return nil, Fail(422, "Revisa el orden de los horarios y que la recogida no haya vencido")
	}
	p.ExpiresAt = p.ScheduledPickupTo
	p.ScheduledDeliveryBy = &p.ScheduledDeliveryTo
	return p, nil
}

func CreateJob(u *models.User, input JobInput) (*models.Publication, error) {
	if u.Role != "company" {
		return nil, Fail(403, "Sólo una empresa puede publicar trabajos")
	}
	p, err := input.Validate(time.Now().UTC())
	if err != nil {
		return nil, err
	}
	err = facades.Orm().Transaction(func(tx orm.Query) error {
		c, err := Company(tx, u.ID)
		if err != nil {
			return err
		}
		p.CompanyID = c.ID
		p.CreatedByUserID = u.ID
		if err = tx.Create(p); err != nil {
			return err
		}
		actor := u.ID
		return tx.Create(&models.PublicationEvent{PublicationID: p.ID, ActorUserID: &actor, EventType: "published", ToStatus: p.Status, Payload: "{}"})
	})
	return p, err
}
func JobScope(q orm.Query, u *models.User) orm.Query {
	if u.Role == "company" {
		return q.Where("EXISTS(SELECT 1 FROM companies c WHERE c.id=publications.company_id AND c.owner_user_id=? AND c.status='active' AND c.deleted_at IS NULL)", u.ID)
	}
	return q.Where("assigned_courier_id=? OR (status='published' AND expires_at>? AND EXISTS(SELECT 1 FROM companies c WHERE c.id=publications.company_id AND c.status='active' AND c.deleted_at IS NULL) AND (visibility='public' OR EXISTS(SELECT 1 FROM company_members m WHERE m.company_id=publications.company_id AND m.user_id=? AND m.status='accepted')))", u.ID, time.Now().UTC(), u.ID)
}

type JobLocation struct{ Latitude, Longitude float64 }

// Great-circle distance, not a road route or an estimated travel time.
func PickupDistance(a, b JobLocation) float64 {
	const radians = math.Pi / 180
	lat := (b.Latitude - a.Latitude) * radians / 2
	lng := (b.Longitude - a.Longitude) * radians / 2
	h := math.Sin(lat)*math.Sin(lat) + math.Cos(a.Latitude*radians)*math.Cos(b.Latitude*radians)*math.Sin(lng)*math.Sin(lng)
	return 6371 * 2 * math.Asin(math.Sqrt(math.Max(0, math.Min(1, h))))
}

func Jobs(u *models.User, page int, locations ...JobLocation) ([]models.Publication, int64, error) {
	if page < 1 || page > 100000 {
		page = 1
	}
	out := make([]models.Publication, 0)
	var total int64
	q := JobScope(facades.Orm().Query().Model(&models.Publication{}), u).With("Company").With("Courier")
	var origin *JobLocation
	if u.Role == "courier" {
		if len(locations) > 0 {
			if !ValidPoint(locations[0].Latitude, locations[0].Longitude) {
				return nil, 0, Fail(422, "Indica coordenadas válidas")
			}
			origin = &locations[0]
		} else {
			var profile models.CourierProfile
			if err := facades.Orm().Query().Where("user_id=?", u.ID).First(&profile); err != nil {
				return nil, 0, err
			}
			if profile.ID > 0 && ValidPoint(profile.Latitude, profile.Longitude) {
				origin = &JobLocation{profile.Latitude, profile.Longitude}
			}
		}
		q = q.OrderByRaw("CASE WHEN status IN ('accepted','picked_up','arrived','delivery_reported') THEN 0 WHEN status='published' THEN 1 ELSE 2 END")
		if origin != nil {
			// Only finite, validated numbers enter this fixed SQL expression.
			q = q.OrderByRaw(fmt.Sprintf("CASE WHEN status='published' THEN acos(least(1.0,greatest(-1.0,sin(radians(%.10f))*sin(radians(pickup_lat))+cos(radians(%.10f))*cos(radians(pickup_lat))*cos(radians(pickup_lng-(%.10f)))))) ELSE 0 END", origin.Latitude, origin.Latitude, origin.Longitude))
		}
		q = q.OrderBy("scheduled_pickup_to")
	}
	err := q.OrderByDesc("id").Paginate(page, 30, &out, &total)
	if origin != nil {
		for i := range out {
			d := PickupDistance(*origin, JobLocation{out[i].PickupLat, out[i].PickupLng})
			out[i].PickupDistanceKm = &d
		}
	}
	return out, total, err
}
func GetJob(u *models.User, id uint) (*models.Publication, error) {
	var p models.Publication
	err := JobScope(facades.Orm().Query().Model(&models.Publication{}), u).Where("id=?", id).With("Company").With("Courier").First(&p)
	if err != nil {
		return nil, err
	}
	if p.ID == 0 {
		return nil, Fail(404, "Trabajo no encontrado")
	}
	return &p, nil
}

func Transition(u *models.User, id uint, action, note string) (*models.Publication, error) {
	var p models.Publication
	note, err := Text(note, 0, 2000, "Nota")
	if err != nil {
		return nil, err
	}
	if (action == "reject" || action == "cancel") && note == "" {
		return nil, Fail(422, "Indica el motivo")
	}
	err = facades.Orm().Transaction(func(tx orm.Query) error {
		if err := tx.Where("id=?", id).LockForUpdate().First(&p); err != nil {
			return err
		}
		if p.ID == 0 {
			return Fail(404, "Trabajo no encontrado")
		}
		var company models.Company
		if err := tx.Where("id=?", p.CompanyID).Where("status=?", "active").First(&company); err != nil {
			return err
		}
		if company.ID == 0 {
			return Fail(404, "Trabajo no disponible")
		}
		if u.Role == "company" {
			if company.OwnerUserID != u.ID {
				return Fail(404, "Trabajo no encontrado")
			}
		} else if action == "accept" {
			if p.Visibility != "public" {
				var m models.CompanyMember
				if err := tx.Where("company_id=?", p.CompanyID).Where("user_id=?", u.ID).Where("status=?", "accepted").LockForUpdate().First(&m); err != nil {
					return err
				}
				if m.ID == 0 {
					return Fail(404, "Trabajo no encontrado")
				}
			}
			if !p.ExpiresAt.After(time.Now().UTC()) {
				return Fail(409, "La recogida de este trabajo ha vencido")
			}
		} else if p.AssignedCourierID == nil || *p.AssignedCourierID != u.ID {
			return Fail(404, "Trabajo no encontrado")
		}
		next, err := NextState(p.Status, action, u.Role)
		if err != nil {
			return err
		}
		previous := p.Status
		now := time.Now().UTC()
		updates := map[string]any{"status": next}
		switch action {
		case "accept":
			p.AssignedCourierID = &u.ID
			p.AssignedAt = &now
			updates["assigned_courier_id"] = u.ID
			updates["assigned_at"] = now
		case "report":
			p.ReportedAt = &now
			updates["reported_at"] = now
		case "confirm":
			p.ConfirmedAt = &now
			p.DeliveredAt = &now
			updates["confirmed_at"] = now
			updates["delivered_at"] = now
		case "reject":
			p.ReportedAt = nil
			updates["reported_at"] = nil
		case "cancel":
			p.CancelledAt = &now
			p.CancellationReason = note
			updates["cancelled_at"] = now
			updates["cancellation_reason"] = note
		}
		if _, err = tx.Model(&models.Publication{}).Where("id=?", id).Update(updates); err != nil {
			return err
		}
		p.Status = next
		payload, _ := json.Marshal(map[string]string{"note": note})
		if err = tx.Create(&models.PublicationEvent{PublicationID: id, ActorUserID: &u.ID, EventType: action, FromStatus: previous, ToStatus: next, Payload: string(payload)}); err != nil {
			return err
		}
		recipient := company.OwnerUserID
		if u.Role == "company" {
			if p.AssignedCourierID == nil {
				return nil
			}
			recipient = *p.AssignedCourierID
		}
		messages := map[string]string{
			"accept":  "Trabajo aceptado por el repartidor",
			"pickup":  "Recogida confirmada",
			"arrive":  "El repartidor llegó al punto de entrega",
			"report":  "El repartidor notificó la entrega. Revisa y confirma el resultado",
			"confirm": "Entrega confirmada por la empresa",
			"reject":  "La empresa solicitó revisar la entrega",
			"cancel":  "Trabajo cancelado",
		}
		message := p.Title + ": " + messages[action]
		if note != "" {
			message += ". Nota: " + note
		}
		return tx.Create(&models.Notification{UserID: recipient, PublicationID: &id, Message: message})
	})
	return &p, err
}
