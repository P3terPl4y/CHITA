package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235722CreatePublicationApplicationsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235722CreatePublicationApplicationsTable) Signature() string {
	return "20260929235722_create_publication_applications_table"
}

// Up Run the migrations.
func (r *M20260929235722CreatePublicationApplicationsTable) Up() error {
	if !facades.Schema().HasTable("publication_applications") {
		return facades.Schema().Create("publication_applications", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("publication_id")
			table.UnsignedBigInteger("courier_user_id")
			table.Text("message")
			table.BigInteger("offered_price_cents").Nullable()
			table.TimestampTz("estimated_pickup_at").Nullable()
			table.String("status", 20).Default("pending")
			table.TimestampTz("responded_at").Nullable()

			table.Unique("publication_id", "courier_user_id").Name("idx_pub_courier_active")
			table.Index("status").Name("idx_publication_applications_status")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235722CreatePublicationApplicationsTable) Down() error {
	return facades.Schema().DropIfExists("publication_applications")
}
