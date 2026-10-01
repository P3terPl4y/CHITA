package services

import (
	"github.com/google/uuid"
	"github.com/goravel/framework/contracts/database/orm"
	"golang.org/x/crypto/bcrypt"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

func Admin(u *models.User) error {
	if u == nil || u.Role != "admin" {
		return Fail(403, "Acceso exclusivo de administración")
	}
	fresh, e := User(u.ID)
	if e != nil {
		return e
	}
	if fresh.Role != "admin" {
		return Fail(403, "Acceso exclusivo de administración")
	}
	return nil
}
func activeUser(tx orm.Query, id uint, role string) error {
	var u models.User
	if e := tx.Where("id=?", id).LockForUpdate().First(&u); e != nil {
		return e
	}
	if u.ID == 0 || !u.Status || u.Role != role {
		return Fail(403, "La cuenta no está activa para esta operación")
	}
	return nil
}
func audit(tx orm.Query, u *models.User, entity string, id uint, action, reason string) error {
	_, e := tx.Exec("INSERT INTO admin_audits(actor_user_id,entity,entity_id,action,reason) VALUES(?,?,?,?,?)", u.ID, entity, id, action, reason)
	return e
}
func adminReason(raw string) (string, error) { return Text(raw, 5, 1000, "Motivo del cambio") }
func CreateAdministrator(name, email, password string) error {
	var e error
	name, e = Text(name, 2, 80, "Nombre")
	if e != nil {
		return e
	}
	email, e = Email(email)
	if e != nil {
		return e
	}
	if e = Password(password); e != nil {
		return e
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	return facades.Orm().Transaction(func(tx orm.Query) error {
		n, e := tx.Model(&models.User{}).Where("LOWER(email)=?", email).Count()
		if e != nil {
			return e
		}
		if n > 0 {
			return Fail(409, "Este correo ya tiene una cuenta; usa una cuenta separada")
		}
		u := models.User{UUID: uuid.NewString(), Email: email, FirstName: name, DisplayName: name, PasswordHash: string(h), Role: "admin", Status: true, Locale: "es", Timezone: "UTC", Metadata: "{}"}
		if e = tx.Create(&u); e != nil {
			return e
		}
		return audit(tx, &u, "admin", u.ID, "create", "Alta local mediante Artisan")
	})
}

type AdminAccountInput struct {
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Password    string   `json:"password"`
	Address     string   `json:"address"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	CompanyName string   `json:"company_name"`
	VehicleType string   `json:"vehicle_type"`
	Enabled     *bool    `json:"enabled"`
	Version     int      `json:"version"`
	Reason      string   `json:"reason"`
}
type AdminChange struct {
	Version int    `json:"version"`
	Reason  string `json:"reason"`
}
type AdminAccount struct {
	ID          uint    `json:"id"`
	UserID      uint    `json:"user_id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Phone       string  `json:"phone"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CompanyName string  `json:"company_name"`
	VehicleType string  `json:"vehicle_type"`
	Enabled     bool    `json:"enabled"`
	Archived    bool    `json:"archived"`
	Version     int     `json:"version"`
}

func accountQuery(tx orm.Query, entity string) (orm.Query, error) {
	switch entity {
	case "companies":
		return tx.Table("companies AS c").Join("JOIN users AS u ON u.id=c.owner_user_id").SelectRaw("c.id, u.id AS user_id,u.display_name AS name,u.email,u.phone,c.address,c.latitude,c.longitude,c.trade_name AS company_name,'' AS vehicle_type,(u.status AND u.deleted_at IS NULL AND c.status='active') AS enabled,(c.deleted_at IS NOT NULL) AS archived,c.admin_version AS version"), nil
	case "couriers":
		return tx.Table("users AS u").Join("JOIN courier_profiles AS p ON p.user_id=u.id").Where("u.role=?", "courier").SelectRaw("u.id,u.id AS user_id,u.display_name AS name,u.email,u.phone,p.address,p.latitude,p.longitude,'' AS company_name,p.vehicle_type,u.status AS enabled,(u.deleted_at IS NOT NULL) AS archived,u.admin_version AS version"), nil
	}
	return nil, Fail(404, "Entidad no encontrada")
}
func accountID(entity string) string {
	if entity == "companies" {
		return "c.id"
	}
	return "u.id"
}
func accountDeleted(entity string) string {
	if entity == "companies" {
		return "c.deleted_at"
	}
	return "u.deleted_at"
}
func AdminAccounts(u *models.User, entity, search, state string, page int) ([]AdminAccount, int64, error) {
	if e := Admin(u); e != nil {
		return nil, 0, e
	}
	if _, e := Text(search, 0, 100, "Búsqueda"); e != nil {
		return nil, 0, e
	}
	q, e := accountQuery(facades.Orm().Query(), entity)
	if e != nil {
		return nil, 0, e
	}
	if state == "archived" {
		q = q.Where(accountDeleted(entity) + " IS NOT NULL")
	} else {
		q = q.Where(accountDeleted(entity) + " IS NULL")
		switch state {
		case "", "all":
		case "active", "disabled":
			value := state == "active"
			if entity == "companies" {
				if value {
					q = q.Where("u.status AND u.deleted_at IS NULL AND c.status='active'")
				} else {
					q = q.Where("NOT u.status OR u.deleted_at IS NOT NULL OR c.status<>'active'")
				}
			} else {
				q = q.Where("u.status=?", value)
			}
		default:
			return nil, 0, Fail(422, "Filtro inválido")
		}
	}
	if search != "" {
		pattern := "%" + search + "%"
		if entity == "companies" {
			q = q.Where("u.email ILIKE ? OR u.display_name ILIKE ? OR c.trade_name ILIKE ?", pattern, pattern, pattern)
		} else {
			q = q.Where("u.email ILIKE ? OR u.display_name ILIKE ?", pattern, pattern)
		}
	}
	rows := make([]AdminAccount, 0)
	var n int64
	e = q.OrderByDesc(accountID(entity)).Paginate(page, 20, &rows, &n)
	return rows, n, e
}
func AdminAccountDetail(u *models.User, entity string, id uint) (*AdminAccount, error) {
	if e := Admin(u); e != nil {
		return nil, e
	}
	q, e := accountQuery(facades.Orm().Query(), entity)
	if e != nil {
		return nil, e
	}
	var a AdminAccount
	e = q.Where(accountID(entity)+"=?", id).First(&a)
	if e != nil {
		return nil, e
	}
	if a.ID == 0 {
		return nil, Fail(404, "Registro no encontrado")
	}
	return &a, nil
}
func accountRole(entity string) (string, error) {
	switch entity {
	case "companies":
		return "company", nil
	case "couriers":
		return "courier", nil
	}
	return "", Fail(404, "Entidad no encontrada")
}
func AdminSaveAccount(u *models.User, entity string, id uint, input AdminAccountInput) (uint, error) {
	if e := Admin(u); e != nil {
		return 0, e
	}
	role, e := accountRole(entity)
	if e != nil {
		return 0, e
	}
	reason, e := adminReason(input.Reason)
	if e != nil {
		return 0, e
	}
	if input.Enabled == nil {
		return 0, Fail(422, "Indica si la cuenta está activa")
	}
	password := input.Password
	if id != 0 && password == "" {
		password = "validation-placeholder"
	}
	r, e := validateRegistration(Registration{Name: input.Name, Email: input.Email, Phone: input.Phone, Password: password, Role: role, Address: input.Address, Latitude: input.Latitude, Longitude: input.Longitude, CompanyName: input.CompanyName, VehicleType: input.VehicleType})
	if e != nil {
		return 0, e
	}
	var saved uint
	e = facades.Orm().Transaction(func(tx orm.Query) error {
		if e := activeUser(tx, u.ID, "admin"); e != nil {
			return e
		}
		if id == 0 {
			a, e := newRegisteredUser(tx, r)
			if e != nil {
				return e
			}
			saved = a.ID
			if entity == "companies" {
				var c models.Company
				if e = tx.Where("owner_user_id=?", a.ID).First(&c); e != nil {
					return e
				}
				saved = c.ID
				if !*input.Enabled {
					if _, e = tx.Model(&models.Company{}).Where("id=?", c.ID).Update("status", "inactive"); e != nil {
						return e
					}
				}
			}
			if !*input.Enabled {
				if _, e = tx.Model(&models.User{}).Where("id=?", a.ID).Update("status", false); e != nil {
					return e
				}
			}
			return audit(tx, u, entity, saved, "create", reason)
		}
		uid := id
		if entity == "companies" {
			var c models.Company
			if e := tx.Where("id=?", id).First(&c); e != nil {
				return e
			}
			if c.ID == 0 {
				return Fail(404, "Empresa no encontrada")
			}
			uid = c.OwnerUserID
		}
		var a models.User
		if e := tx.Where("id=?", uid).LockForUpdate().First(&a); e != nil {
			return e
		}
		if a.ID == 0 || a.Role != role {
			return Fail(404, "Registro no encontrado")
		}
		version := a.AdminVersion
		if entity == "companies" {
			var c models.Company
			if e := tx.Where("id=?", id).LockForUpdate().First(&c); e != nil {
				return e
			}
			if c.ID == 0 {
				return Fail(404, "Empresa no encontrada")
			}
			version = c.AdminVersion
		}
		if version != input.Version {
			return Fail(409, "El registro cambió. Actualiza la lista antes de editar")
		}
		if !*input.Enabled {
			if e := noActiveJobs(tx, entity, id, false); e != nil {
				return e
			}
		}
		n, e := tx.Model(&models.User{}).Where("LOWER(email)=? AND id<>?", r.Email, a.ID).Count()
		if e != nil {
			return e
		}
		if n > 0 {
			return Fail(409, "Este correo ya tiene una cuenta")
		}
		fields := map[string]any{"display_name": r.Name, "first_name": r.Name, "email": r.Email, "phone": r.Phone, "status": *input.Enabled, "admin_version": a.AdminVersion + 1}
		if input.Password != "" {
			h, e := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
			if e != nil {
				return e
			}
			fields["password_hash"] = string(h)
		}
		if _, e = tx.Model(&models.User{}).Where("id=?", a.ID).Update(fields); e != nil {
			return e
		}
		if entity == "companies" {
			status := "inactive"
			if *input.Enabled {
				status = "active"
			}
			_, e = tx.Model(&models.Company{}).Where("id=?", id).Update(map[string]any{"trade_name": r.CompanyName, "email": r.Email, "phone": r.Phone, "address": r.Address, "latitude": *r.Latitude, "longitude": *r.Longitude, "status": status, "admin_version": version + 1})
		} else {
			_, e = tx.Model(&models.CourierProfile{}).Where("user_id=?", a.ID).Update(map[string]any{"address": r.Address, "latitude": *r.Latitude, "longitude": *r.Longitude, "vehicle_type": r.VehicleType})
		}
		if e != nil {
			return e
		}
		if !*input.Enabled || input.Password != "" {
			if _, e = tx.Exec("DELETE FROM auth_grants WHERE user_id=?", a.ID); e != nil {
				return e
			}
			if _, e = tx.Exec("DELETE FROM halcon_accounts WHERE user_id=?", a.ID); e != nil {
				return e
			}
		}
		saved = id
		return audit(tx, u, entity, id, "update", reason)
	})
	return saved, e
}
func noActiveJobs(tx orm.Query, entity string, id uint, archive bool) error {
	q := tx.Model(&models.Publication{})
	if entity == "companies" {
		q = q.Where("company_id=?", id)
	} else {
		q = q.Where("assigned_courier_id=?", id)
	}
	states := []any{"accepted", "picked_up", "arrived", "delivery_reported"}
	if archive && entity == "companies" {
		states = append(states, "published")
	}
	n, e := q.WhereIn("status", states).Count()
	if e != nil {
		return e
	}
	if n > 0 {
		return Fail(409, "Tiene trabajos pendientes. Resuélvelos antes de desactivar o archivar")
	}
	return nil
}
func AdminArchiveAccount(u *models.User, entity string, id uint, input AdminChange, restore bool) error {
	if e := Admin(u); e != nil {
		return e
	}
	role, e := accountRole(entity)
	if e != nil {
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
		uid := id
		if entity == "companies" {
			var c models.Company
			if e := tx.WithTrashed().Where("id=?", id).First(&c); e != nil {
				return e
			}
			if c.ID == 0 {
				return Fail(404, "Empresa no encontrada")
			}
			uid = c.OwnerUserID
		}
		var a models.User
		if e := tx.WithTrashed().Where("id=?", uid).LockForUpdate().First(&a); e != nil {
			return e
		}
		if a.ID == 0 || a.Role != role {
			return Fail(404, "Registro no encontrado")
		}
		version := a.AdminVersion
		archived := a.DeletedAt.Valid
		if entity == "companies" {
			var c models.Company
			if e := tx.WithTrashed().Where("id=?", id).LockForUpdate().First(&c); e != nil {
				return e
			}
			version = c.AdminVersion
			archived = c.DeletedAt.Valid
		}
		if version != input.Version || archived != restore {
			return Fail(409, "El registro cambió. Actualiza la lista")
		}
		if restore {
			n, e := tx.Model(&models.User{}).Where("LOWER(email)=? AND id<>?", a.Email, a.ID).Count()
			if e != nil {
				return e
			}
			if n > 0 {
				return Fail(409, "El correo está en uso; resuelve el conflicto antes de restaurar")
			}
		} else {
			if e := noActiveJobs(tx, entity, id, true); e != nil {
				return e
			}
		}
		var deleted any = time.Now().UTC()
		action := "archive"
		if restore {
			deleted = nil
			action = "restore"
		}
		if _, e := tx.WithTrashed().Model(&models.User{}).Where("id=?", a.ID).Update(map[string]any{"deleted_at": deleted, "status": false, "admin_version": a.AdminVersion + 1}); e != nil {
			return e
		}
		if entity == "companies" {
			if _, e := tx.WithTrashed().Model(&models.Company{}).Where("id=?", id).Update(map[string]any{"deleted_at": deleted, "status": "inactive", "admin_version": version + 1}); e != nil {
				return e
			}
		}
		if !restore {
			key := "user_id"
			if entity == "companies" {
				key = "company_id"
			}
			if _, e := tx.Model(&models.CompanyMember{}).Where(key+"=?", id).Update("status", "revoked"); e != nil {
				return e
			}
		}
		if _, e := tx.Exec("DELETE FROM auth_grants WHERE user_id=?", a.ID); e != nil {
			return e
		}
		if _, e := tx.Exec("DELETE FROM halcon_accounts WHERE user_id=?", a.ID); e != nil {
			return e
		}
		return audit(tx, u, entity, id, action, reason)
	})
}
