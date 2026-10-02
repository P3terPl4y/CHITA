package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"goravel/app/services"
	"strconv"
)

func (d *DispatchController) Directory(c fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil {
		return services.Fail(422, "Página no válida")
	}
	result, err := services.Directory(current(c), c.Query("search"), page)
	if err != nil {
		return err
	}
	return c.JSON(result)
}
func (d *DispatchController) Avatar(c fiber.Ctx) error {
	var input struct {
		Avatar *string `json:"avatar"`
	}
	if err := Decode(c, &input); err != nil {
		return err
	}
	if input.Avatar == nil {
		return services.Fail(422, "Selecciona una foto o indica que deseas quitarla")
	}
	grant, _ := session.FromContext(c).Get("auth_grant").(string)
	avatar, err := services.SaveAvatar(current(c), *input.Avatar, grant)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"avatar_url": avatar})
}
