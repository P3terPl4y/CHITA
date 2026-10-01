package controllers

import (
	"github.com/gofiber/fiber/v3"
	"goravel/app/models"
	"goravel/app/services"
	"strconv"
)

type AdminController struct{ Tracking *services.Tracking }

func RequireAdmin(c fiber.Ctx) error {
	if e := services.Admin(current(c)); e != nil {
		return e
	}
	return c.Next()
}
func adminPage(c fiber.Ctx) (int, error) {
	n, e := strconv.Atoi(c.Query("page", "1"))
	if e != nil || n < 1 || n > 100000 {
		return 0, services.Fail(422, "Página inválida")
	}
	return n, nil
}
func adminJobDTO(p *models.Publication) any {
	m := jobDTO(p).(fiber.Map)
	m["version"] = p.AdminVersion
	m["archived"] = p.DeletedAt.Valid
	return m
}
func (d *AdminController) Overview(c fiber.Ctx) error {
	x, e := services.AdminOverview(current(c))
	if e != nil {
		return e
	}
	return c.JSON(x)
}
func (d *AdminController) Audits(c fiber.Ctx) error {
	page, e := adminPage(c)
	if e != nil {
		return e
	}
	rows, n, e := services.AdminAudits(current(c), page)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"items": rows, "total": n, "page": page})
}
func (d *AdminController) List(c fiber.Ctx) error {
	page, e := adminPage(c)
	if e != nil {
		return e
	}
	entity := c.Params("entity")
	if entity == "jobs" {
		rows, n, e := services.AdminJobs(current(c), c.Query("search"), c.Query("state"), page)
		if e != nil {
			return e
		}
		out := make([]any, 0, len(rows))
		for i := range rows {
			out = append(out, adminJobDTO(&rows[i]))
		}
		return c.JSON(fiber.Map{"items": out, "total": n, "page": page})
	}
	rows, n, e := services.AdminAccounts(current(c), entity, c.Query("search"), c.Query("state"), page)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"items": rows, "total": n, "page": page})
}
func (d *AdminController) Detail(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	if c.Params("entity") == "jobs" {
		p, e := services.AdminJobDetail(current(c), n)
		if e != nil {
			return e
		}
		return c.JSON(adminJobDTO(p))
	}
	p, e := services.AdminAccountDetail(current(c), c.Params("entity"), n)
	if e != nil {
		return e
	}
	return c.JSON(p)
}
func (d *AdminController) Save(c fiber.Ctx) error {
	var n uint
	var e error
	if c.Params("id") != "" {
		n, e = id(c)
		if e != nil {
			return e
		}
	}
	if c.Params("entity") == "jobs" {
		var input services.AdminJobInput
		if e := Decode(c, &input); e != nil {
			return e
		}
		p, e := services.AdminSaveJob(current(c), n, input)
		if e != nil {
			return e
		}
		if n == 0 {
			c.Status(201)
		}
		return c.JSON(adminJobDTO(p))
	}
	var input services.AdminAccountInput
	if e := Decode(c, &input); e != nil {
		return e
	}
	saved, e := services.AdminSaveAccount(current(c), c.Params("entity"), n, input)
	if e != nil {
		return e
	}
	p, e := services.AdminAccountDetail(current(c), c.Params("entity"), saved)
	if e != nil {
		return e
	}
	if !p.Enabled || input.Password != "" {
		d.Tracking.Stop(p.UserID)
	}
	if n == 0 {
		c.Status(201)
	}
	return c.JSON(p)
}
func (d *AdminController) Change(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	var input services.AdminChange
	if e := Decode(c, &input); e != nil {
		return e
	}
	action := c.Params("action")
	if c.Method() == "DELETE" {
		action = "archive"
	}
	entity := c.Params("entity")
	if entity == "jobs" {
		e = services.AdminChangeJob(current(c), n, input, action)
		if e != nil {
			return e
		}
		p, e := services.AdminJobDetail(current(c), n)
		if e != nil {
			return e
		}
		if p.AssignedCourierID != nil && !services.TrackingActive(p.Status) {
			d.Tracking.StopJob(*p.AssignedCourierID, p.ID)
		}
		return c.JSON(adminJobDTO(p))
	}
	if action != "archive" && action != "restore" {
		return services.Fail(404, "Acción no encontrada")
	}
	if e = services.AdminArchiveAccount(current(c), entity, n, input, action == "restore"); e != nil {
		return e
	}
	p, e := services.AdminAccountDetail(current(c), entity, n)
	if e != nil {
		return e
	}
	d.Tracking.Stop(p.UserID)
	return c.JSON(p)
}
func (d *AdminController) Password(c fiber.Ctx) error {
	var r struct {
		Current  string `json:"current_password"`
		Password string `json:"password"`
	}
	if e := Decode(c, &r); e != nil {
		return e
	}
	if e := services.AdminPassword(current(c), r.Current, r.Password); e != nil {
		return e
	}
	return c.SendStatus(204)
}
