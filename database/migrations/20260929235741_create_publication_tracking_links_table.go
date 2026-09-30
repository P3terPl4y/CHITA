package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235741CreatePublicationTrackingLinksTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235741CreatePublicationTrackingLinksTable) Signature() string {
	return "20260929235741_create_publication_tracking_links_table"
}

// Up Run the migrations.
func (r *M20260929235741CreatePublicationTrackingLinksTable) Up() error {
	if !facades.Schema().HasTable("publication_tracking_links") {
		return facades.Schema().Create("publication_tracking_links", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("publication_id")
			table.String("provider", 50)
			table.String("external_id", 120)
			table.String("external_status", 50)
			table.TimestampTz("last_synced_at").Nullable()
			table.Json("last_payload")

			table.Index("publication_id").Name("idx_publication_tracking_links_publication_id")
			table.Unique("provider", "external_id").Name("idx_tracking_provider_ext")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235741CreatePublicationTrackingLinksTable) Down() error {
	return facades.Schema().DropIfExists("publication_tracking_links")
}
