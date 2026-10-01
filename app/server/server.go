package server

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/jackc/pgx/v5/pgconn"
	"goravel/app/http/controllers"
	"goravel/app/services"
	"goravel/routes"
	"log"
	"time"
)

func New(storage fiber.Storage, production bool, t *services.Tracking, geocoders ...*services.Geocoder) *fiber.App {
	app := fiber.New(fiber.Config{TrustProxy: true, ProxyHeader: fiber.HeaderXForwardedFor, TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true}, BodyLimit: 32 * 1024, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, ErrorHandler: func(c fiber.Ctx, e error) error {
		code := 500
		msg := "No se pudo completar la operación"
		var p *services.Problem
		var f *fiber.Error
		var db *pgconn.PgError
		if errors.As(e, &p) {
			code = p.Status
			msg = p.Message
		} else if errors.As(e, &f) {
			code = f.Code
			msg = f.Message
		} else if errors.As(e, &db) && db.Code == "23505" {
			code = 409
			msg = "El recurso ya existe; actualiza antes de volver a intentarlo"
		} else {
			log.Printf("CHITA error: %v", e)
		}
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}})
	app.Use(recover.New())
	app.Use(helmet.New(helmet.Config{ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://tile.openstreetmap.org; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'", XFrameOptions: "DENY", ReferrerPolicy: "strict-origin-when-cross-origin", CrossOriginEmbedderPolicy: "unsafe-none"}))
	app.Use(func(c fiber.Ctx) error {
		c.Set("Permissions-Policy", "geolocation=(self), camera=(), microphone=()")
		if len(c.Path()) >= 4 && c.Path()[:4] == "/api" {
			c.Set("Cache-Control", "no-store")
		}
		return c.Next()
	})
	cookie := "chita_session"
	csrfCookie := "chita_csrf"
	if production {
		cookie = "__Host-chita-session"
		csrfCookie = "__Host-chita-csrf"
	}
	sm, store := session.NewWithStore(session.Config{Storage: sessionStorage(storage), Extractor: extractors.FromCookie(cookie), CookieSecure: production, CookieHTTPOnly: true, CookieSameSite: "Lax", IdleTimeout: 30 * time.Minute, AbsoluteTimeout: 24 * time.Hour})
	app.Use("/api", sm)
	app.Use("/api", csrf.New(csrf.Config{CookieName: csrfCookie, CookieSecure: production, CookieHTTPOnly: true, CookieSameSite: "Lax", Session: store, Extractor: extractors.FromHeader("X-CSRF-Token"), ErrorHandler: func(c fiber.Ctx, e error) error {
		return services.Fail(403, "La sesión de seguridad cambió; actualiza la página")
	}}))
	geo := services.NewGeocoder("")
	if len(geocoders) > 0 && geocoders[0] != nil {
		geo = geocoders[0]
	}
	routes.Web(app, controllers.NewDeliveryController(t), geo)
	app.Use(static.New("./react/dist", static.Config{IndexNames: []string{"index.html"}}))
	app.Get("/*", func(c fiber.Ctx) error { return c.SendFile("./react/dist/index.html") })
	return app
}
