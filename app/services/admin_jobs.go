package services

import (
	"github.com/goravel/framework/contracts/database/orm"
	"golang.org/x/crypto/bcrypt"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

type AdminJobInput struct {
	JobInput
	CompanyID uint   `json:"company_id"`
	Version   int    `json:"version"`
	Reason    string `json:"reason"`
}

func AdminJobs(u *models.User, search, state string, page int) ([]models.Publication, int64, error) {
	if e := Admin(u); e != nil {
		return nil, 0, e
	}
	if _, e := Text(search, 0, 100, "Búsqueda"); e != nil {
		return nil, 0, e
	}
	q := facades.Orm().Query().Model(&models.Publication{}).With("Company").With("Courier")
	switch state {
	case "", "all":
	case "archived":
		q = q.WithTrashed().Where("deleted_at IS NOT NULL")
	case "published", "accepted", "picked_up", "arrived", "delivery_reported", "completed", "cancelled":
		q = q.Where("status=?", state)
	default:
		return nil, 0, Fail(422, "Filtro inválido")
	}
	if search != "" {
		q = q.Where("title ILIKE ? OR reference_code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	rows := make([]models.Publication, 0)
	var n int64
	e := q.OrderByDesc("id").Paginate(page, 20, &rows, &n)
	return rows, n, e
}
func AdminJobDetail(u *models.User, id uint) (*models.Publication, error) {
	if e := Admin(u); e != nil {
		return nil, e
	}
	var p models.Publication
	e := facades.Orm().Query().WithTrashed().Where("id=?", id).With("Company").With("Courier").First(&p)
	if e != nil {
		return nil, e
	}
	if p.ID == 0 {
		return nil, Fail(404, "Trabajo no encontrado")
	}
	return &p, nil
}
func AdminSaveJob(u *models.User, id uint, input AdminJobInput) (*models.Publication, error) {
	if e := Admin(u); e != nil {
		return nil, e
	}
	reason, e := adminReason(input.Reason)
	if e != nil {
		return nil, e
	}
	p, e := input.JobInput.Validate(time.Now().UTC())
	if e != nil {
		return nil, e
	}
	e = facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, "admin"); e != nil {
			return e
		}
		action := "create"
		if id != 0 {
			var old models.Publication
			if e := tx.Where("id=?", id).LockForUpdate().First(&old); e != nil {
				return e
			}
			if old.ID == 0 {
				return Fail(404, "Trabajo no encontrado")
			}
			if old.AdminVersion != input.Version {
				return Fail(409, "El trabajo cambió. Actualiza la lista")
			}
			if old.Status != "published" || old.AssignedCourierID != nil {
				return Fail(409, "Sólo se pueden editar trabajos disponibles sin asignación")
			}
			p.ID = old.ID
			p.UUID = old.UUID
			p.ReferenceCode = old.ReferenceCode
			p.CreatedByUserID = old.CreatedByUserID
			p.AdminVersion = old.AdminVersion + 1
			action = "update"
		} else {
			p.CreatedByUserID = u.ID
			p.AdminVersion = 1
		}
		var c models.Company
		if e := tx.Where("id=?", input.CompanyID).First(&c); e != nil {
			return e
		}
		if c.ID == 0 {
			return Fail(422, "Elige una empresa activa")
		}
		if e := activeUser(tx, c.OwnerUserID, "company"); e != nil {
			return e
		}
		if e := tx.Where("id=?", c.ID).LockForUpdate().First(&c); e != nil {
			return e
		}
		if c.Status != "active" {
			return Fail(422, "Elige una empresa activa")
		}
		p.CompanyID = c.ID
		if id == 0 {
			if e := tx.Create(p); e != nil {
				return e
			}
		} else {
			fields := map[string]any{"company_id": p.CompanyID, "title": p.Title, "description": p.Description, "visibility": p.Visibility, "pickup_address_text": p.PickupAddressText, "pickup_lat": p.PickupLat, "pickup_lng": p.PickupLng, "dropoff_address_text": p.DropoffAddressText, "dropoff_lat": p.DropoffLat, "dropoff_lng": p.DropoffLng, "scheduled_pickup_from": p.ScheduledPickupFrom, "scheduled_pickup_to": p.ScheduledPickupTo, "scheduled_delivery_from": p.ScheduledDeliveryFrom, "scheduled_delivery_to": p.ScheduledDeliveryTo, "scheduled_delivery_by": p.ScheduledDeliveryBy, "expires_at": p.ExpiresAt, "offered_price_cents": p.OfferedPriceCents, "currency": p.Currency, "admin_version": p.AdminVersion}
			if _, e := tx.Model(&models.Publication{}).Where("id=?", id).Update(fields); e != nil {
				return e
			}
		}
		if e := tx.Create(&models.PublicationEvent{PublicationID: p.ID, ActorUserID: &u.ID, EventType: "admin_" + action, ToStatus: p.Status, Payload: "{}"}); e != nil {
			return e
		}
		if e := tx.Create(&models.Notification{UserID: c.OwnerUserID, PublicationID: &p.ID, Message: p.Title + ": administración actualizó la publicación. Motivo: " + reason}); e != nil {
			return e
		}
		return audit(tx, u, "jobs", p.ID, action, reason)
	})
	return p, e
}
func AdminChangeJob(u *models.User, id uint, input AdminChange, action string) error {
	if e := Admin(u); e != nil {
		return e
	}
	reason, e := adminReason(input.Reason)
	if e != nil {
		return e
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, "admin"); e != nil {
			return e
		}
		var p models.Publication
		if e := tx.WithTrashed().Where("id=?", id).LockForUpdate().First(&p); e != nil {
			return e
		}
		if p.ID == 0 {
			return Fail(404, "Trabajo no encontrado")
		}
		if p.AdminVersion != input.Version {
			return Fail(409, "El trabajo cambió. Actualiza la lista")
		}
		verbs := map[string]string{"archive": "el archivo del trabajo", "restore": "la restauración del trabajo", "cancel": "la cancelación del trabajo"}
		fields := map[string]any{"admin_version": p.AdminVersion + 1}
		previous := p.Status
		switch action {
		case "archive":
			if p.DeletedAt.Valid || !(p.Status == "published" || p.Status == "cancelled" || p.Status == "completed") {
				return Fail(409, "No se puede archivar una entrega en curso")
			}
			fields["deleted_at"] = time.Now().UTC()
		case "restore":
			if !p.DeletedAt.Valid {
				return Fail(409, "El trabajo no está archivado")
			}
			var c models.Company
			if e := tx.Where("id=?", p.CompanyID).Where("status=?", "active").First(&c); e != nil {
				return e
			}
			if c.ID == 0 {
				return Fail(409, "Activa primero la empresa")
			}
			fields["deleted_at"] = nil
		case "cancel":
			if p.DeletedAt.Valid || (p.Status != "published" && p.Status != "accepted") {
				return Fail(409, "Sólo se puede cancelar antes de la recogida")
			}
			p.Status = "cancelled"
			fields["status"] = p.Status
			fields["cancelled_at"] = time.Now().UTC()
			fields["cancellation_reason"] = reason
		default:
			return Fail(404, "Acción no encontrada")
		}
		if _, e := tx.WithTrashed().Model(&models.Publication{}).Where("id=?", id).Update(fields); e != nil {
			return e
		}
		if e := tx.Create(&models.PublicationEvent{PublicationID: id, ActorUserID: &u.ID, EventType: "admin_" + action, FromStatus: previous, ToStatus: p.Status, Payload: "{}"}); e != nil {
			return e
		}
		var c models.Company
		if e := tx.WithTrashed().Where("id=?", p.CompanyID).First(&c); e != nil {
			return e
		}
		if e := tx.Create(&models.Notification{UserID: c.OwnerUserID, PublicationID: &id, Message: p.Title + ": administración realizó " + verbs[action] + ". Motivo: " + reason}); e != nil {
			return e
		}
		if p.AssignedCourierID != nil {
			if e := tx.Create(&models.Notification{UserID: *p.AssignedCourierID, PublicationID: &id, Message: p.Title + ": administración realizó " + verbs[action] + ". Motivo: " + reason}); e != nil {
				return e
			}
		}
		return audit(tx, u, "jobs", id, action, reason)
	})
}

type AdminAuditRow struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	ActorUserID uint      `json:"actor_user_id"`
	Entity      string    `json:"entity"`
	EntityID    uint      `json:"entity_id"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
}

func AdminAudits(u *models.User, page int) ([]AdminAuditRow, int64, error) {
	if e := Admin(u); e != nil {
		return nil, 0, e
	}
	rows := make([]AdminAuditRow, 0)
	var n int64
	e := facades.Orm().Query().Table("admin_audits").OrderByDesc("id").Paginate(page, 20, &rows, &n)
	return rows, n, e
}
func AdminOverview(u *models.User) (map[string]int64, error) {
	if e := Admin(u); e != nil {
		return nil, e
	}
	out := map[string]int64{}
	for _, entity := range []string{"companies", "couriers", "jobs"} {
		var e error
		if entity == "companies" {
			out[entity], e = facades.Orm().Query().Model(&models.Company{}).Count()
		} else if entity == "couriers" {
			out[entity], e = facades.Orm().Query().Model(&models.User{}).Where("role=?", "courier").Count()
		} else {
			out[entity], e = facades.Orm().Query().Model(&models.Publication{}).Count()
		}
		if e != nil {
			return nil, e
		}
	}
	n, e := facades.Orm().Query().Model(&models.Publication{}).Where("status=?", "delivery_reported").Count()
	out["pending_confirmation"] = n
	return out, e
}
func AdminPassword(u *models.User, current, password string) error {
	if e := Admin(u); e != nil {
		return e
	}
	if e := Password(password); e != nil {
		return e
	}
	if len(current) > 72 {
		return Fail(422, "Contraseña actual incorrecta")
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		var a models.User
		if e := tx.Where("id=?", u.ID).LockForUpdate().First(&a); e != nil {
			return e
		}
		if a.ID == 0 || !a.Status || a.Role != "admin" {
			return Fail(403, "Acceso exclusivo de administración")
		}
		if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(current)) != nil {
			return Fail(422, "Contraseña actual incorrecta")
		}
		h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if e != nil {
			return e
		}
		if _, e = tx.Model(&models.User{}).Where("id=?", a.ID).Update("password_hash", string(h)); e != nil {
			return e
		}
		if _, e = tx.Exec("DELETE FROM auth_grants WHERE user_id=?", a.ID); e != nil {
			return e
		}
		return audit(tx, u, "admin", u.ID, "password", "Cambio de contraseña propia")
	})
}
