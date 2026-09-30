package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235634CreateUserAddressesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235634CreateUserAddressesTable) Signature() string {
	return "20260929235634_create_user_addresses_table"
}

// Up Run the migrations.
func (r *M20260929235634CreateUserAddressesTable) Up() error {
	if !facades.Schema().HasTable("user_addresses") {
		return facades.Schema().Create("user_addresses", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("user_id")
			table.String("label", 50)
			table.Boolean("is_primary").Default(false)
			table.String("line1", 255)
			table.String("line2", 255)
			table.String("city", 120)
			table.String("state", 120)
			table.String("country_code", 2)
			table.String("postal_code", 20)
			table.Decimal("latitude")
			table.Decimal("longitude")

			table.Index("user_id").Name("idx_user_addresses_user_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235634CreateUserAddressesTable) Down() error {
	return facades.Schema().DropIfExists("user_addresses")
}
