package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235731CreatePublicationEventsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235731CreatePublicationEventsTable) Signature() string {
	return "20260929235731_create_publication_events_table"
}

// Up Run the migrations.
func (r *M20260929235731CreatePublicationEventsTable) Up() error {
	if !facades.Schema().HasTable("publication_events") {
		return facades.Schema().Create("publication_events", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("publication_id")
			table.UnsignedBigInteger("actor_user_id").Nullable()
			table.String("event_type", 40)
			table.String("from_status", 20)
			table.String("to_status", 20)
			table.Jsonb("payload").Default("{}")

			table.Index("actor_user_id").Name("idx_publication_events_actor_user_id")
			table.Index("event_type").Name("idx_publication_events_event_type")
			table.Index("publication_id").Name("idx_publication_events_publication_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235731CreatePublicationEventsTable) Down() error {
	return facades.Schema().DropIfExists("publication_events")
}
