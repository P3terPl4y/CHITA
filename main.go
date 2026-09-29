package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/goravel/framework/facades"
	"goravel/bootstrap"
)

func main() {
	// Inicializa el framework (config, ORM, providers, etc.)
	bootstrap.Boot()

	// Canal para escuchar señales del sistema operativo
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Arranca el servidor HTTP (Fiber, porque así lo configuramos)
	go func() {
		if err := facades.Route().Run(); err != nil {
			facades.Log().Errorf("error al ejecutar el servidor HTTP: %v", err)
		}
	}()

	// Espera señal de apagado para cerrar limpiamente
	<-quit
	if err := facades.Route().Shutdown(); err != nil {
		facades.Log().Errorf("error al apagar el servidor: %v", err)
	}
}
