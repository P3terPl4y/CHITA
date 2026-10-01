package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"goravel/app/http/controllers"
	"goravel/app/services"
	"strconv"
	"time"
)

func Web(app *fiber.App, d *controllers.DeliveryController) {
	api := app.Group("/api")
	api.Get("/session", d.Session)
	reached := func(c fiber.Ctx) error {
		return services.Fail(429, "Demasiadas solicitudes; espera un minuto antes de volver a intentarlo")
	}
	authLimit := limiter.New(limiter.Config{Max: 12, Expiration: time.Minute, LimitReached: reached})
	api.Post("/auth/register", authLimit, d.Register)
	api.Post("/auth/login", authLimit, d.Login)
	protected := api.Group("", controllers.Require)
	userKey := func(c fiber.Ctx) string {
		return strconv.FormatUint(uint64(controllers.UserID(c)), 10)
	}
	protected.Use(limiter.New(limiter.Config{Max: 240, Expiration: time.Minute, KeyGenerator: userKey, LimitReached: reached}))
	protected.Post("/auth/logout", d.Logout)
	protected.Get("/profile", d.Profile)
	protected.Get("/jobs", d.Jobs)
	protected.Post("/jobs", d.Create)
	protected.Get("/jobs/:id/location", d.Position)
	protected.Post("/jobs/:id/location", limiter.New(limiter.Config{Max: 60, Expiration: time.Minute, KeyGenerator: userKey, LimitReached: reached}), d.Position)
	protected.Get("/jobs/:id", d.Job)
	protected.Post("/jobs/:id/:action", d.Action)
	protected.Get("/network", d.Networks)
	protected.Post("/network", d.Invite)
	protected.Post("/network/:id/:action", d.Membership)
	protected.Get("/notifications", d.Notifications)
	protected.Post("/notifications/:id/read", d.Read)
	protected.Get("/halcon", d.Halcon)
	protected.Post("/halcon", limiter.New(limiter.Config{Max: 6, Expiration: time.Minute, KeyGenerator: userKey, LimitReached: reached}), d.Link)
	protected.Delete("/halcon", d.Unlink)
	protected.Post("/location/stop", d.Stop)
	api.All("/*", func(c fiber.Ctx) error { return fiber.ErrNotFound })
}
