package services

import (
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

const OfferLifetime = 2 * time.Minute

func pendingOffer(tx orm.Query, jobID uint) (*models.DeliveryOffer, error) {
	var o models.DeliveryOffer
	e := tx.Where("publication_id=?", jobID).Where("status=?", "pending").Where("expires_at>?", time.Now().UTC()).First(&o)
	return &o, e
}
func withdrawOffers(tx orm.Query, jobID uint, message string) error {
	var offers []models.DeliveryOffer
	if e := tx.Where("publication_id=?", jobID).Where("status=?", "pending").Find(&offers); e != nil {
		return e
	}
	for _, o := range offers {
		if e := tx.Create(&models.Notification{UserID: o.CourierUserID, PublicationID: &jobID, Message: message}); e != nil {
			return e
		}
	}
	_, e := tx.Model(&models.DeliveryOffer{}).Where("publication_id=?", jobID).Where("status=?", "pending").Update(map[string]any{"status": "withdrawn", "responded_at": time.Now().UTC()})
	return e
}
func OfferJob(u *models.User, jobID, courierID uint) (*models.DeliveryOffer, error) {
	if u.Role != "company" {
		return nil, Fail(403, "Sólo la empresa puede proponer un trabajo")
	}
	var o models.DeliveryOffer
	e := facades.Orm().Transaction(func(tx orm.Query) error {
		var p models.Publication
		if e := tx.Where("id=?", jobID).LockForUpdate().First(&p); e != nil {
			return e
		}
		if p.ID == 0 {
			return Fail(404, "Trabajo no encontrado")
		}
		if e := activeUser(tx, u.ID, "company"); e != nil {
			return e
		}
		c, e := Company(tx, u.ID)
		if e != nil {
			return e
		}
		if c.ID != p.CompanyID {
			return Fail(404, "Trabajo no encontrado")
		}
		if p.Status != "published" || p.AssignedCourierID != nil || !p.ExpiresAt.After(time.Now().UTC()) {
			return Fail(409, "El trabajo ya no está disponible")
		}
		if e := activeUser(tx, courierID, "courier"); e != nil {
			return e
		}
		if e := noActiveJobs(tx, "couriers", courierID, false); e != nil {
			return e
		}
		var profile models.CourierProfile
		if e := tx.Where("user_id=?", courierID).Where("available_until>?", time.Now().UTC()).First(&profile); e != nil {
			return e
		}
		if profile.ID == 0 || profile.DiscoveryLat == nil || profile.DiscoveryLng == nil {
			return Fail(409, "Este repartidor ya no está disponible")
		}
		if PickupDistance(JobLocation{p.PickupLat, p.PickupLng}, JobLocation{*profile.DiscoveryLat, *profile.DiscoveryLng}) > 50 {
			return Fail(422, "El repartidor está fuera del radio de 50 km desde la recogida")
		}
		if _, e := tx.Model(&models.DeliveryOffer{}).Where("status=?", "pending").Where("expires_at<=?", time.Now().UTC()).Where("publication_id=? OR courier_user_id=?", jobID, courierID).Update(map[string]any{"status": "expired", "responded_at": time.Now().UTC()}); e != nil {
			return e
		}
		n, e := tx.Model(&models.DeliveryOffer{}).Where("status=?", "pending").Where("publication_id=? OR courier_user_id=?", jobID, courierID).Count()
		if e != nil {
			return e
		}
		if n > 0 {
			return Fail(409, "El trabajo o el repartidor ya tiene una propuesta pendiente")
		}
		expires := time.Now().UTC().Add(OfferLifetime)
		if p.ExpiresAt.Before(expires) {
			expires = p.ExpiresAt
		}
		o = models.DeliveryOffer{PublicationID: jobID, CompanyID: c.ID, CourierUserID: courierID, Status: "pending", ExpiresAt: expires}
		if e := tx.Create(&o); e != nil {
			return e
		}
		if _, e = tx.Model(&models.Publication{}).Where("id=?", jobID).Update("admin_version", p.AdminVersion+1); e != nil {
			return e
		}
		if e := tx.Create(&models.PublicationEvent{PublicationID: jobID, ActorUserID: &u.ID, EventType: "offer", ToStatus: p.Status, Payload: "{}"}); e != nil {
			return e
		}
		return tx.Create(&models.Notification{UserID: courierID, PublicationID: &jobID, Message: p.Title + ": tienes una propuesta de " + c.TradeName + ". Puedes aceptarla o rechazarla antes de " + expires.Format(time.RFC3339)})
	})
	return &o, e
}
func Offers(u *models.User) ([]models.DeliveryOffer, error) {
	q := facades.Orm().Query().Model(&models.DeliveryOffer{}).With("Publication.Company").With("Courier")
	if u.Role == "company" {
		c, e := Company(facades.Orm().Query(), u.ID)
		if e != nil {
			return nil, e
		}
		q = q.Where("company_id=?", c.ID)
	} else if u.Role == "courier" {
		q = q.Where("courier_user_id=?", u.ID)
	} else {
		return nil, Fail(403, "Rol no permitido")
	}
	rows := make([]models.DeliveryOffer, 0)
	e := q.OrderByDesc("id").Limit(50).Find(&rows)
	return rows, e
}
func RespondOffer(u *models.User, id uint, action string) error {
	if action == "accept" {
		if u.Role != "courier" {
			return Fail(403, "Sólo el repartidor puede aceptar")
		}
		var o models.DeliveryOffer
		if e := facades.Orm().Query().Where("id=? AND courier_user_id=?", id, u.ID).First(&o); e != nil {
			return e
		}
		if o.ID == 0 {
			return Fail(404, "Propuesta no encontrada")
		}
		_, e := transition(u, o.PublicationID, "accept", "", o.ID)
		return e
	}
	if action != "decline" && action != "withdraw" {
		return Fail(422, "Acción desconocida")
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		var o models.DeliveryOffer
		if e := tx.Where("id=?", id).First(&o); e != nil {
			return e
		}
		if o.ID == 0 {
			return Fail(404, "Propuesta no encontrada")
		}
		var p models.Publication
		if e := tx.Where("id=?", o.PublicationID).LockForUpdate().First(&p); e != nil {
			return e
		}
		if p.ID == 0 {
			return Fail(409, "El trabajo no está disponible")
		}
		if e := activeUser(tx, u.ID, u.Role); e != nil {
			return e
		}
		if action == "decline" {
			if u.Role != "courier" || o.CourierUserID != u.ID {
				return Fail(404, "Propuesta no encontrada")
			}
		} else {
			if u.Role != "company" {
				return Fail(403, "Sólo la empresa puede retirar la propuesta")
			}
			c, e := Company(tx, u.ID)
			if e != nil {
				return e
			}
			if c.ID != o.CompanyID {
				return Fail(404, "Propuesta no encontrada")
			}
		}
		if e := tx.Where("id=?", id).LockForUpdate().First(&o); e != nil {
			return e
		}
		if o.Status != "pending" || !o.ExpiresAt.After(time.Now().UTC()) {
			return Fail(409, "La propuesta ya cambió de estado o venció")
		}
		status := "declined"
		recipient := uint(0)
		var c models.Company
		if e := tx.Where("id=?", o.CompanyID).First(&c); e != nil {
			return e
		}
		recipient = c.OwnerUserID
		if action == "withdraw" {
			status = "withdrawn"
			recipient = o.CourierUserID
		}
		if _, e := tx.Model(&models.DeliveryOffer{}).Where("id=?", id).Update(map[string]any{"status": status, "responded_at": time.Now().UTC()}); e != nil {
			return e
		}
		if _, e := tx.Model(&models.Publication{}).Where("id=?", p.ID).Update("admin_version", p.AdminVersion+1); e != nil {
			return e
		}
		message := p.Title + ": el repartidor rechazó la propuesta"
		if action == "withdraw" {
			message = p.Title + ": la empresa retiró la propuesta"
		}
		return tx.Create(&models.Notification{UserID: recipient, PublicationID: &p.ID, Message: message})
	})
}
func attachOffers(q orm.Query, rows []models.Publication) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]any, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var offers []models.DeliveryOffer
	if e := q.WhereIn("publication_id", ids).Where("status=?", "pending").Where("expires_at>?", time.Now().UTC()).Find(&offers); e != nil {
		return e
	}
	byID := map[uint]*models.DeliveryOffer{}
	for i := range offers {
		byID[offers[i].PublicationID] = &offers[i]
	}
	for i := range rows {
		rows[i].PendingOffer = byID[rows[i].ID]
	}
	return nil
}

// Call only after locking the account, consistently with proposal acceptance.
func stopAccountDiscovery(tx orm.Query, entity string, id, userID uint) error {
	if entity == "couriers" {
		if _, e := tx.Model(&models.CourierProfile{}).Where("user_id=?", userID).Update(map[string]any{"available_until": nil, "discovery_token": nil}); e != nil {
			return e
		}
	}
	column := "courier_user_id"
	target := userID
	if entity == "companies" {
		column = "company_id"
		target = id
	}
	_, e := tx.Model(&models.DeliveryOffer{}).Where(column+"=?", target).Where("status=?", "pending").Update(map[string]any{"status": "withdrawn", "responded_at": time.Now().UTC()})
	return e
}
