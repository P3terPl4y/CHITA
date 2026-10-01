package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"

	"goravel/config"

	"goravel/app/facades"
	"goravel/app/models"
)

// bootstrap/app.go
func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithCommands(Commands).
		WithMigrations(Migrations).
		WithProviders(Providers).
		WithConfig(config.Boot).
		WithCallback(func() {
			facades.Schema().Extend(schema.Extension{
				Models: []any{
					&models.User{},
					&models.CourierProfile{},
					&models.Notification{},
					&models.HalconAccount{},
					&models.UserAddress{},
					&models.Company{},
					&models.CompanyMember{},
					&models.Publication{},
					&models.PublicationItem{},
					&models.PublicationApplication{},
					&models.PublicationEvent{},
					&models.PublicationTrackingLink{},
					&models.Review{},
				},
			})
		}).
		Create()
}
