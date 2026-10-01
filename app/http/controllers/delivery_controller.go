package controllers

import (
	"bytes"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/google/uuid"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"io"
	"mime"
	"strconv"
	"time"
)

type DeliveryController struct{ Tracking *services.Tracking }

func NewDeliveryController(t *services.Tracking) *DeliveryController {
	return &DeliveryController{Tracking: t}
}
func Decode(c fiber.Ctx, v any) error {
	media, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return services.Fail(415, "Envía JSON con Content-Type application/json")
	}
	d := json.NewDecoder(bytes.NewReader(c.Body()))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return services.Fail(422, "Datos inválidos")
	}
	if d.Decode(new(any)) != io.EOF {
		return services.Fail(422, "Datos inválidos")
	}
	return nil
}
func Identity(c fiber.Ctx) (*models.User, error) {
	id, ok := session.FromContext(c).Get("user_id").(uint)
	if !ok || id == 0 {
		return nil, services.Fail(401, "Inicia sesión")
	}
	token, _ := session.FromContext(c).Get("auth_grant").(string)
	if err := services.CheckGrant(id, token); err != nil {
		return nil, err
	}
	return services.User(id)
}
func Require(c fiber.Ctx) error {
	u, e := Identity(c)
	if e != nil {
		return e
	}
	c.Locals("user", u)
	return c.Next()
}
func current(c fiber.Ctx) *models.User { return c.Locals("user").(*models.User) }
func UserID(c fiber.Ctx) uint          { return current(c).ID }
func id(c fiber.Ctx) (uint, error) {
	n, e := strconv.ParseInt(c.Params("id"), 10, 64)
	if e != nil || n <= 0 {
		return 0, services.Fail(404, "Recurso no encontrado")
	}
	return uint(n), nil
}
func userDTO(u *models.User) any {
	if u == nil {
		return nil
	}
	return fiber.Map{"id": u.ID, "name": u.DisplayName, "email": u.Email, "phone": u.Phone, "role": u.Role}
}
func jobDTO(p *models.Publication) any {
	m := fiber.Map{"id": p.ID, "title": p.Title, "description": p.Description, "status": p.Status, "company_id": p.CompanyID, "assigned_courier_id": p.AssignedCourierID, "pickup_address": p.PickupAddressText, "pickup_lat": p.PickupLat, "pickup_lng": p.PickupLng, "dropoff_address": p.DropoffAddressText, "dropoff_lat": p.DropoffLat, "dropoff_lng": p.DropoffLng, "pickup_from": p.ScheduledPickupFrom, "pickup_to": p.ScheduledPickupTo, "delivery_from": p.ScheduledDeliveryFrom, "delivery_to": p.ScheduledDeliveryTo, "price_cents": p.OfferedPriceCents, "currency": p.Currency, "reported_at": p.ReportedAt, "confirmed_at": p.ConfirmedAt, "cancellation_reason": p.CancellationReason}
	if p.Company != nil {
		m["company"] = fiber.Map{"id": p.Company.ID, "name": p.Company.TradeName}
	}
	m["visibility"] = p.Visibility
	m["pickup_distance_km"] = p.PickupDistanceKm
	if p.Courier != nil {
		m["courier"] = fiber.Map{"id": p.Courier.ID, "name": p.Courier.DisplayName}
	}
	return m
}
func (d *DeliveryController) Session(c fiber.Ctx) error {
	u, e := Identity(c)
	if e != nil {
		if p, ok := e.(*services.Problem); !ok || p.Status != 401 {
			return e
		}
		u = nil
	}
	return c.JSON(fiber.Map{"user": userDTO(u), "csrf_token": csrf.TokenFromContext(c)})
}
func signIn(c fiber.Ctx, u *models.User) error {
	s := session.FromContext(c)
	if oldID, ok := s.Get("user_id").(uint); ok {
		oldToken, _ := s.Get("auth_grant").(string)
		if err := services.Revoke(oldID, oldToken); err != nil {
			return err
		}
	}
	if e := s.Regenerate(); e != nil {
		return e
	}
	s.Set("user_id", u.ID)
	token := uuid.NewString()
	if err := services.Grant(u.ID, token, u.PasswordHash); err != nil {
		_ = s.Destroy()
		return err
	}
	s.Set("auth_grant", token)
	return c.JSON(fiber.Map{"user": userDTO(u)})
}
func (d *DeliveryController) Register(c fiber.Ctx) error {
	var r services.Registration
	if e := Decode(c, &r); e != nil {
		return e
	}
	u, e := services.Register(r)
	if e != nil {
		return e
	}
	c.Status(201)
	return signIn(c, u)
}
func (d *DeliveryController) Login(c fiber.Ctx) error {
	var r struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if e := Decode(c, &r); e != nil {
		return e
	}
	u, e := services.Login(r.Email, r.Password)
	if e != nil {
		return e
	}
	return signIn(c, u)
}
func (d *DeliveryController) Logout(c fiber.Ctx) error {
	token, _ := session.FromContext(c).Get("auth_grant").(string)
	if err := services.Revoke(current(c).ID, token); err != nil {
		return err
	}
	d.Tracking.Stop(current(c).ID)
	if e := session.FromContext(c).Destroy(); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (d *DeliveryController) Profile(c fiber.Ctx) error {
	u := current(c)
	var p any
	if u.Role == "company" {
		x, e := services.Company(facades.Orm().Query(), u.ID)
		if e != nil {
			return e
		}
		p = fiber.Map{"name": x.TradeName, "address": x.Address, "latitude": x.Latitude, "longitude": x.Longitude}
	} else {
		var x models.CourierProfile
		if e := facades.Orm().Query().Where("user_id=?", u.ID).First(&x); e != nil {
			return e
		}
		p = x
	}
	return c.JSON(fiber.Map{"user": userDTO(u), "profile": p})
}
func (d *DeliveryController) Jobs(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 || page > 100000 {
		page = 1
	}
	var locations []services.JobLocation
	if c.Query("lat") != "" || c.Query("lng") != "" {
		lat, e1 := strconv.ParseFloat(c.Query("lat"), 64)
		lng, e2 := strconv.ParseFloat(c.Query("lng"), 64)
		if e1 != nil || e2 != nil || !services.ValidPoint(lat, lng) {
			return services.Fail(422, "Indica latitud y longitud válidas")
		}
		locations = append(locations, services.JobLocation{Latitude: lat, Longitude: lng})
	}
	p, n, e := services.Jobs(current(c), page, locations...)
	if e != nil {
		return e
	}
	out := make([]any, 0, len(p))
	for i := range p {
		out = append(out, jobDTO(&p[i]))
	}
	return c.JSON(fiber.Map{"items": out, "total": n, "page": page})
}
func (d *DeliveryController) Job(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	p, e := services.GetJob(current(c), n)
	if e != nil {
		return e
	}
	return c.JSON(jobDTO(p))
}
func (d *DeliveryController) Create(c fiber.Ctx) error {
	var r services.JobInput
	if e := Decode(c, &r); e != nil {
		return e
	}
	p, e := services.CreateJob(current(c), r)
	if e != nil {
		return e
	}
	return c.Status(201).JSON(jobDTO(p))
}
func (d *DeliveryController) Action(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	var r struct {
		Note string `json:"note"`
	}
	if e := Decode(c, &r); e != nil {
		return e
	}
	p, e := services.Transition(current(c), n, c.Params("action"), r.Note)
	if e != nil {
		return e
	}
	if !services.TrackingActive(p.Status) && p.AssignedCourierID != nil {
		d.Tracking.StopJob(*p.AssignedCourierID, p.ID)
	}
	return c.JSON(jobDTO(p))
}
func (d *DeliveryController) Networks(c fiber.Ctx) error {
	rows, e := services.Networks(current(c))
	if e != nil {
		return e
	}
	out := make([]any, 0, len(rows))
	for _, m := range rows {
		x := fiber.Map{"id": m.ID, "status": m.Status, "user_id": m.UserID, "company_id": m.CompanyID}
		if m.Courier != nil {
			x["name"] = m.Courier.DisplayName
			x["email"] = m.Courier.Email
		}
		if m.Company != nil {
			x["name"] = m.Company.TradeName
		}
		out = append(out, x)
	}
	return c.JSON(out)
}
func (d *DeliveryController) Invite(c fiber.Ctx) error {
	var r struct {
		Email string `json:"email"`
	}
	if e := Decode(c, &r); e != nil {
		return e
	}
	m, e := services.Invite(current(c), r.Email)
	if e != nil {
		return e
	}
	return c.Status(201).JSON(fiber.Map{"id": m.ID, "status": m.Status})
}
func (d *DeliveryController) Membership(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	if e = services.Membership(current(c), n, c.Params("action")); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (d *DeliveryController) Notifications(c fiber.Ctx) error {
	out := make([]models.Notification, 0)
	e := facades.Orm().Query().Where("user_id=?", current(c).ID).OrderByDesc("id").Limit(100).Find(&out)
	if e != nil {
		return e
	}
	return c.JSON(out)
}
func (d *DeliveryController) Read(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	result, e := facades.Orm().Query().Model(&models.Notification{}).Where("id=?", n).Where("user_id=?", current(c).ID).Update("read_at", time.Now().UTC())
	if e != nil {
		return e
	}
	if result.RowsAffected == 0 {
		return services.Fail(404, "Aviso no encontrado")
	}
	return c.SendStatus(204)
}
func (d *DeliveryController) Halcon(c fiber.Ctx) error {
	x, e := d.Tracking.Info(current(c))
	if e != nil {
		return e
	}
	return c.JSON(x)
}
func (d *DeliveryController) Link(c fiber.Ctx) error {
	var r struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if e := Decode(c, &r); e != nil {
		return e
	}
	if e := d.Tracking.Link(current(c), r.Email, r.Password); e != nil {
		return e
	}
	return d.Halcon(c)
}
func (d *DeliveryController) Unlink(c fiber.Ctx) error {
	if e := d.Tracking.Unlink(current(c)); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (d *DeliveryController) Position(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	if c.Method() == "GET" {
		p, e := d.Tracking.Position(current(c), n)
		if e != nil {
			return e
		}
		return c.JSON(p)
	}
	var p services.Point
	if e := Decode(c, &p); e != nil {
		return e
	}
	grant, _ := session.FromContext(c).Get("auth_grant").(string)
	if e := d.Tracking.Publish(current(c), n, p, grant); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (d *DeliveryController) Stop(c fiber.Ctx) error {
	d.Tracking.Stop(current(c).ID)
	return c.SendStatus(204)
}
