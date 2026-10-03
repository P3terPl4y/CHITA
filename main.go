package main

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/storage/redis/v3"
	"goravel/app/facades"
	"goravel/app/server"
	"goravel/app/services"
	"goravel/bootstrap"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	if isArtisanCommand(os.Args) {
		bootstrap.Boot()
		if err := facades.Artisan().Run(os.Args, true); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := serve(); err != nil {
		log.Fatal(err)
	}
}

func isArtisanCommand(args []string) bool {
	return len(args) > 1 && args[1] == "artisan"
}

func serve() error {
	bootstrap.Boot()
	c := facades.Config()
	integer := func(key string, fallback int) int {
		raw := fmt.Sprint(c.Env(key, fallback))
		n, e := strconv.Atoi(raw)
		if e != nil {
			log.Fatalf("%s debe ser un número", key)
		}
		return n
	}
	remote, e := services.NewRemote(fmt.Sprint(c.Env("HALCON_URL", "http://127.0.0.1:3300")))
	if e != nil {
		log.Fatal(e)
	}
	if n := len(c.GetString("app.key")); n != 16 && n != 24 && n != 32 {
		log.Fatal("Genera una APP_KEY válida antes de iniciar CHITA")
	}
	tracking := services.NewTracking(remote)
	defer tracking.Close()
	store := redis.New(redis.Config{Host: fmt.Sprint(c.Env("REDIS_HOST", "127.0.0.1")), Port: integer("REDIS_PORT", 6379), Username: fmt.Sprint(c.Env("REDIS_USERNAME", "")), Password: fmt.Sprint(c.Env("REDIS_PASSWORD", "")), Database: integer("REDIS_DB", 0)})
	defer store.Close()
	app := server.New(store, c.GetString("app.env") == "production", tracking, services.NewGeocoder(fmt.Sprint(c.Env("GEOCODER_URL", "https://photon.komoot.io"))))
	host := fmt.Sprint(c.Env("APP_HOST", "127.0.0.1"))
	address := net.JoinHostPort(host, strconv.Itoa(integer("APP_PORT", 3330)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("CHITA escuchando en %s", address)
	return app.Listen(address, fiber.ListenConfig{GracefulContext: ctx, ShutdownTimeout: 10 * time.Second})
}
