package services

import (
	"github.com/google/uuid"
	"github.com/goravel/framework/contracts/database/orm"
	"golang.org/x/crypto/bcrypt"
	"goravel/app/facades"
	"goravel/app/models"
	"strings"
	"time"
)

type Registration struct {
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Phone       string   `json:"phone"`
	Role        string   `json:"role"`
	Address     string   `json:"address"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	CompanyName string   `json:"company_name"`
	VehicleType string   `json:"vehicle_type"`
}

func validateRegistration(r Registration) (Registration, error) {
	var err error
	if r.Role != "company" && r.Role != "courier" {
		return r, Fail(422, "Elige empresa o repartidor")
	}
	if r.Name, err = Text(r.Name, 2, 80, "Nombre"); err != nil {
		return r, err
	}
	if r.Email, err = Email(r.Email); err != nil {
		return r, err
	}
	if r.Phone, err = Text(r.Phone, 3, 20, "Teléfono"); err != nil {
		return r, err
	}
	if r.Address, err = Text(r.Address, 3, 255, "Dirección"); err != nil {
		return r, err
	}
	if err = Password(r.Password); err != nil {
		return r, err
	}
	if r.Latitude == nil || r.Longitude == nil || !ValidPoint(*r.Latitude, *r.Longitude) {
		return r, Fail(422, "Indica coordenadas válidas para tu dirección")
	}
	if r.Role == "company" {
		if r.CompanyName, err = Text(r.CompanyName, 2, 200, "Empresa"); err != nil {
			return r, err
		}
	}
	if r.Role == "courier" {
		switch r.VehicleType {
		case "foot", "bicycle", "motorbike", "car", "van":
		default:
			return r, Fail(422, "Elige un medio de transporte")
		}
	}
	return r, nil
}

func Register(r Registration) (*models.User, error) {
	r, err := validateRegistration(r)
	if err != nil {
		return nil, err
	}
	var u *models.User
	err = facades.Orm().Transaction(func(tx orm.Query) error { var e error; u, e = newRegisteredUser(tx, r); return e })
	return u, err
}

func newRegisteredUser(tx orm.Query, r Registration) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := models.User{UUID: uuid.NewString(), Email: r.Email, Phone: r.Phone, FirstName: r.Name, DisplayName: r.Name, PasswordHash: string(hash), Role: r.Role, Status: true, Locale: "es", Timezone: "UTC", Metadata: "{}"}
	err = createRegisteredUser(tx, r, &u)
	return &u, err
}

func createRegisteredUser(tx orm.Query, r Registration, u *models.User) error {
	count, err := tx.Model(&models.User{}).Where("LOWER(email)=?", r.Email).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return Fail(409, "Este correo ya tiene una cuenta")
	}
	if err := tx.Create(u); err != nil {
		return err
	}
	if r.Role == "company" {
		return tx.Create(&models.Company{UUID: uuid.NewString(), OwnerUserID: u.ID, LegalName: r.CompanyName, TradeName: r.CompanyName, Email: r.Email, Phone: r.Phone, Status: "active", Address: r.Address, Latitude: *r.Latitude, Longitude: *r.Longitude, Metadata: "{}"})
	}
	return tx.Create(&models.CourierProfile{UserID: u.ID, Address: r.Address, Latitude: *r.Latitude, Longitude: *r.Longitude, VehicleType: r.VehicleType})
}

func Login(email, password string) (*models.User, error) {
	var u models.User
	err := facades.Orm().Query().Where("LOWER(email)=?", strings.ToLower(strings.TrimSpace(email))).First(&u)
	if err != nil {
		return nil, err
	}
	// Equal bcrypt work for unknown accounts reduces user enumeration by timing.
	hash := u.PasswordHash
	if u.ID == 0 {
		hash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	}
	if len(password) > 72 || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || u.ID == 0 || !u.Status {
		return nil, Fail(401, "Correo o contraseña incorrectos")
	}
	now := time.Now().UTC()
	_, err = facades.Orm().Query().Model(&models.User{}).Where("id=?", u.ID).Update("last_login_at", now)
	return &u, err
}

func User(id uint) (*models.User, error) {
	var u models.User
	err := facades.Orm().Query().Where("id=?", id).First(&u)
	if err != nil {
		return nil, err
	}
	if u.ID == 0 || !u.Status {
		return nil, Fail(401, "Tu sesión terminó")
	}
	return &u, nil
}
func Company(q orm.Query, userID uint) (*models.Company, error) {
	var c models.Company
	if err := q.Where("owner_user_id=?", userID).Where("status=?", "active").First(&c); err != nil {
		return nil, err
	}
	if c.ID == 0 {
		return nil, Fail(403, "No tienes una empresa activa")
	}
	return &c, nil
}

// Grants make logout final even if an overlapping HTTP request saves an old
// copy of its session after Redis deletion.
func Grant(userID uint, token string, expectedHash ...string) error {
	return facades.Orm().Transaction(func(tx orm.Query) error {
		var u models.User
		if e := tx.Where("id=?", userID).LockForUpdate().First(&u); e != nil {
			return e
		}
		if u.ID == 0 || !u.Status {
			return Fail(401, "La cuenta no está activa")
		}
		if len(expectedHash) > 0 && u.PasswordHash != expectedHash[0] {
			return Fail(401, "La contraseña cambió. Vuelve a entrar")
		}
		if _, e := tx.Exec("DELETE FROM auth_grants WHERE expires_at<?", time.Now().UTC()); e != nil {
			return e
		}
		_, e := tx.Exec("INSERT INTO auth_grants(token,user_id,expires_at) VALUES(?,?,?)", token, userID, time.Now().UTC().Add(24*time.Hour))
		return e
	})
}

func CheckGrant(userID uint, token string) error {
	if token == "" {
		return Fail(401, "Inicia sesión")
	}
	n, e := facades.Orm().Query().Table("auth_grants").Where("token=?", token).Where("user_id=?", userID).Where("expires_at>?", time.Now().UTC()).Count()
	if e != nil {
		return e
	}
	if n != 1 {
		return Fail(401, "Tu sesión terminó")
	}
	return nil
}
func Revoke(userID uint, token string) error {
	_, e := facades.Orm().Query().Exec("DELETE FROM auth_grants WHERE user_id=? AND token=?", userID, token)
	return e
}

// LockGrant serializes GPS publication with logout revocation, so an already
// authorized but delayed request cannot reopen tracking after logout finishes.
func LockGrant(q orm.Query, userID uint, token string) error {
	var grant struct {
		Token     string
		UserID    uint
		ExpiresAt time.Time
	}
	if err := q.Table("auth_grants").Where("user_id=?", userID).Where("token=?", token).Where("expires_at>?", time.Now().UTC()).LockForUpdate().First(&grant); err != nil {
		return err
	}
	if grant.Token == "" {
		return Fail(401, "Tu sesión terminó")
	}
	return nil
}
