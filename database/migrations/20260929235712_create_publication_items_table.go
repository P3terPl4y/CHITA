package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235712CreatePublicationItemsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235712CreatePublicationItemsTable) Signature() string {
	return "20260929235712_create_publication_items_table"
}

// Up Run the migrations.
func (r *M20260929235712CreatePublicationItemsTable) Up() error {
	if !facades.Schema().HasTable("publication_items") {
		return facades.Schema().Create("publication_items", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("publication_id")
			table.String("description", 255)
			table.BigInteger("quantity").Default(1)
			table.Decimal("weight_kg")
			table.BigInteger("length_cm").Nullable()
			table.BigInteger("width_cm").Nullable()
			table.BigInteger("height_cm").Nullable()
			table.BigInteger("declared_value_cents").Nullable()
			table.Boolean("is_fragile").Default(false)
			table.Jsonb("photo_urls").Default("[]")

			table.Index("publication_id").Name("idx_publication_items_publication_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235712CreatePublicationItemsTable) Down() error {
	return facades.Schema().DropIfExists("publication_items")
}
