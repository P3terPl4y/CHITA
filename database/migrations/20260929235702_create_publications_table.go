package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235702CreatePublicationsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235702CreatePublicationsTable) Signature() string {
	return "20260929235702_create_publications_table"
}

// Up Run the migrations.
func (r *M20260929235702CreatePublicationsTable) Up() error {
	if !facades.Schema().HasTable("publications") {
		return facades.Schema().Create("publications", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.TimestampTz("deleted_at").Nullable()
			table.Uuid("uuid")
			table.String("reference_code", 20)
			table.UnsignedBigInteger("company_id")
			table.UnsignedBigInteger("created_by_user_id")
			table.String("title", 200)
			table.Text("description")
			table.UnsignedBigInteger("pickup_address_id").Nullable()
			table.String("pickup_contact_name", 120)
			table.String("pickup_contact_phone", 20)
			table.Decimal("pickup_lat")
			table.Decimal("pickup_lng")
			table.Text("dropoff_address_text")
			table.String("dropoff_contact_name", 120)
			table.String("dropoff_contact_phone", 20)
			table.Decimal("dropoff_lat")
			table.Decimal("dropoff_lng")
			table.Decimal("distance_km")
			table.BigInteger("package_count").Default(1)
			table.Decimal("total_weight_kg")
			table.Boolean("is_fragile").Default(false)
			table.Boolean("requires_signature").Default(false)
			table.BigInteger("declared_value_cents").Nullable()
			table.String("currency", 3).Default("USD")
			table.BigInteger("offered_price_cents")
			table.String("payment_method", 20).Default("cash")
			table.TimestampTz("scheduled_pickup_from")
			table.TimestampTz("scheduled_pickup_to")
			table.TimestampTz("scheduled_delivery_by").Nullable()
			table.TimestampTz("expires_at")
			table.String("status", 20).Default("draft")
			table.UnsignedBigInteger("assigned_courier_id").Nullable()
			table.TimestampTz("assigned_at").Nullable()
			table.TimestampTz("delivered_at").Nullable()
			table.TimestampTz("cancelled_at").Nullable()
			table.Text("cancellation_reason")
			table.Text("special_instructions")
			table.Json("metadata").Default("{}")

			table.Index("assigned_courier_id").Name("idx_publications_assigned_courier_id")
			table.Index("company_id").Name("idx_publications_company_id")
			table.Index("created_by_user_id").Name("idx_publications_created_by_user_id")
			table.Index("expires_at").Name("idx_publications_expires_at")
			table.Unique("reference_code").Name("idx_publications_reference_code")
			table.Index("status").Name("idx_publications_status")
			table.Unique("uuid").Name("idx_publications_uuid")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235702CreatePublicationsTable) Down() error {
	return facades.Schema().DropIfExists("publications")
}
