package services

import (
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

func Networks(u *models.User) ([]models.CompanyMember, error) {
	q := facades.Orm().Query().Model(&models.CompanyMember{})
	if u.Role == "company" {
		c, err := Company(facades.Orm().Query(), u.ID)
		if err != nil {
			return nil, err
		}
		q = q.Where("company_id=?", c.ID).With("Courier")
	} else {
		q = q.Where("user_id=?", u.ID).With("Company")
	}
	out := make([]models.CompanyMember, 0)
	err := q.OrderByDesc("id").Limit(100).Find(&out)
	return out, err
}
func Invite(u *models.User, email string) (*models.CompanyMember, error) {
	if u.Role != "company" {
		return nil, Fail(403, "Sólo la empresa puede invitar")
	}
	email, err := Email(email)
	if err != nil {
		return nil, err
	}
	var m models.CompanyMember
	err = facades.Orm().Transaction(func(tx orm.Query) error {
		c, err := Company(tx.LockForUpdate(), u.ID)
		if err != nil {
			return err
		}
		var courier models.User
		if err = tx.Where("email=?", email).Where("role=?", "courier").Where("status=?", true).First(&courier); err != nil {
			return err
		}
		if courier.ID == 0 {
			return Fail(422, "Indica el correo de una cuenta de repartidor activa")
		}
		if err = tx.Where("company_id=?", c.ID).Where("user_id=?", courier.ID).First(&m); err != nil {
			return err
		}
		if m.ID != 0 && m.Status != "revoked" {
			return Fail(409, "Este repartidor ya tiene una invitación o pertenece a tu red")
		}
		m.CompanyID = c.ID
		m.UserID = courier.ID
		m.RoleInCompany = "courier"
		m.InvitedByUserID = &u.ID
		m.Status = "pending"
		m.AcceptedAt = nil
		if m.ID == 0 {
			err = tx.Create(&m)
		} else {
			_, err = tx.Model(&models.CompanyMember{}).Where("id=?", m.ID).Update(map[string]any{"status": "pending", "accepted_at": nil, "invited_by_user_id": u.ID})
		}
		if err != nil {
			return err
		}
		return tx.Create(&models.Notification{UserID: courier.ID, Message: c.TradeName + " te invita a su red de repartidores"})
	})
	return &m, err
}
func Membership(u *models.User, id uint, action string) error {
	return facades.Orm().Transaction(func(tx orm.Query) error {
		var m models.CompanyMember
		if err := tx.Where("id=?", id).LockForUpdate().First(&m); err != nil {
			return err
		}
		if m.ID == 0 {
			return Fail(404, "Invitación no encontrada")
		}
		if action == "accept" {
			if u.Role != "courier" || m.UserID != u.ID {
				return Fail(404, "Invitación no encontrada")
			}
			if m.Status != "pending" {
				return Fail(409, "La invitación ya cambió de estado")
			}
			_, err := tx.Model(&models.CompanyMember{}).Where("id=?", id).Update(map[string]any{"status": "accepted", "accepted_at": time.Now().UTC()})
			return err
		}
		if action != "remove" {
			return Fail(422, "Acción desconocida")
		}
		c, err := Company(tx, u.ID)
		if err != nil || u.Role != "company" || c.ID != m.CompanyID {
			return Fail(404, "Invitación no encontrada")
		}
		_, err = tx.Model(&models.CompanyMember{}).Where("id=?", id).Update("status", "revoked")
		return err
	})
}
