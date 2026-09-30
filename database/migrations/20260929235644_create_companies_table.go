package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235644CreateCompaniesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235644CreateCompaniesTable) Signature() string {
	return "20260929235644_create_companies_table"
}

// Up Run the migrations.
func (r *M20260929235644CreateCompaniesTable) Up() error {
	if !facades.Schema().HasTable("companies") {
		return facades.Schema().Create("companies", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.TimestampTz("deleted_at").Nullable()
			table.Uuid("uuid")
			table.UnsignedBigInteger("owner_user_id")
			table.String("legal_name", 200)
			table.String("trade_name", 200)
			table.String("tax_id", 50)
			table.Text("email")
			table.String("phone", 20)
			table.Text("logo_url")
			table.Text("website")
			table.Text("description")
			table.String("status", 20).Default("pending")
			table.TimestampTz("verified_at").Nullable()
			table.UnsignedBigInteger("verified_by_user_id").Nullable()
			table.Jsonb("metadata").Default("{}")

			table.Index("owner_user_id").Name("idx_companies_owner_user_id")
			table.Index("status").Name("idx_companies_status")
			table.Unique("tax_id").Name("idx_companies_taxid_active")
			table.Unique("uuid").Name("idx_companies_uuid")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235644CreateCompaniesTable) Down() error {
	return facades.Schema().DropIfExists("companies")
}
