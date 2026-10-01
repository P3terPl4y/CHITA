package commands

import (
	"fmt"
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"goravel/app/services"
	"os"
	"strings"
)

type CreateAdmin struct{}

func (*CreateAdmin) Signature() string { return "admin:create" }
func (*CreateAdmin) Description() string {
	return "Crea una cuenta administrativa separada (sin acceso público al alta)"
}
func (*CreateAdmin) Extend() command.Extend {
	return command.Extend{Category: "admin", Flags: []command.Flag{&command.StringFlag{Name: "email", Required: true}, &command.StringFlag{Name: "name", Value: "Administrador"}, &command.StringFlag{Name: "password-file", Usage: "Archivo privado con contraseña; alternativa al ingreso interactivo"}}}
}
func (*CreateAdmin) Handle(ctx console.Context) error {
	var password string
	var e error
	if path := ctx.Option("password-file"); path != "" {
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		info, e := f.Stat()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 100 {
			return fmt.Errorf("el archivo debe ser regular, privado (0600) y pequeño")
		}
		b := make([]byte, 100)
		n, e := f.Read(b)
		if e != nil {
			return e
		}
		password = strings.TrimSuffix(strings.TrimSuffix(string(b[:n]), "\n"), "\r")
	} else {
		password, e = ctx.Secret("Contraseña de administración")
		if e != nil {
			return e
		}
		confirmation, e := ctx.Secret("Confirma la contraseña")
		if e != nil {
			return e
		}
		if password != confirmation {
			return fmt.Errorf("las contraseñas no coinciden")
		}
	}
	if e = services.CreateAdministrator(ctx.Option("name"), ctx.Option("email"), password); e != nil {
		return e
	}
	ctx.Info("Cuenta administrativa creada. Entra desde /entrar.")
	return nil
}
