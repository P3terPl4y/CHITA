package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"goravel/app/models"
	"goravel/app/services"
	"strconv"
	"time"
)

type DispatchController struct{ Geocoder *services.Geocoder }

func (d *DispatchController) Reverse(c fiber.Ctx) error {
	lat, e1 := strconv.ParseFloat(c.Query("lat"), 64)
	lng, e2 := strconv.ParseFloat(c.Query("lng"), 64)
	if e1 != nil || e2 != nil {
		return services.Fail(422, "Indica latitud y longitud")
	}
	address, e := d.Geocoder.Reverse(c.Context(), lat, lng)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"address": address, "latitude": lat, "longitude": lng})
}
func (d *DispatchController) Location(c fiber.Ctx) error {
	var r services.LocationInput
	if e := Decode(c, &r); e != nil {
		return e
	}
	if e := services.SaveLocation(current(c), r); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (d *DispatchController) Availability(c fiber.Ctx) error {
	var r services.AvailabilityInput
	if e := Decode(c, &r); e != nil {
		return e
	}
	grant, _ := session.FromContext(c).Get("auth_grant").(string)
	token, e := services.SetAvailability(current(c), r, grant)
	if e != nil {
		return e
	}
	if token != "" {
		return c.JSON(fiber.Map{"token": token})
	}
	return c.SendStatus(204)
}
func (d *DispatchController) Nearby(c fiber.Ctx) error {
	lat, e1 := strconv.ParseFloat(c.Query("lat"), 64)
	lng, e2 := strconv.ParseFloat(c.Query("lng"), 64)
	radius, e3 := strconv.ParseFloat(c.Query("radius", "10"), 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return services.Fail(422, "Indica un punto y un radio válidos")
	}
	rows, e := services.Nearby(current(c), lat, lng, radius)
	if e != nil {
		return e
	}
	return c.JSON(rows)
}
func offerDTO(o *models.DeliveryOffer) any {
	status := o.Status
	if status == "pending" && !o.ExpiresAt.After(time.Now().UTC()) {
		status = "expired"
	}
	m := fiber.Map{"id": o.ID, "publication_id": o.PublicationID, "courier_user_id": o.CourierUserID, "status": status, "expires_at": o.ExpiresAt, "responded_at": o.RespondedAt}
	if o.Publication != nil {
		m["job"] = jobDTO(o.Publication)
	}
	if o.Courier != nil {
		m["courier"] = fiber.Map{"id": o.Courier.ID, "name": o.Courier.DisplayName}
	}
	return m
}
func (d *DispatchController) Offers(c fiber.Ctx) error {
	rows, e := services.Offers(current(c))
	if e != nil {
		return e
	}
	out := make([]any, 0, len(rows))
	for i := range rows {
		out = append(out, offerDTO(&rows[i]))
	}
	return c.JSON(out)
}
func (d *DispatchController) Propose(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	var r struct {
		CourierID uint `json:"courier_id"`
	}
	if e := Decode(c, &r); e != nil {
		return e
	}
	o, e := services.OfferJob(current(c), n, r.CourierID)
	if e != nil {
		return e
	}
	return c.Status(201).JSON(offerDTO(o))
}
func (d *DispatchController) Respond(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	var r struct{}
	if e := Decode(c, &r); e != nil {
		return e
	}
	if e := services.RespondOffer(current(c), n, c.Params("action")); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (d *DispatchController) Rating(c fiber.Ctx) error {
	n, e := id(c)
	if e != nil {
		return e
	}
	if c.Method() == "PUT" {
		var r services.RatingInput
		if e := Decode(c, &r); e != nil {
			return e
		}
		if e := services.RateCourier(current(c), n, r); e != nil {
			return e
		}
	}
	summary, e := services.RatingInfo(current(c), n)
	if e != nil {
		return e
	}
	return c.JSON(summary)
}
