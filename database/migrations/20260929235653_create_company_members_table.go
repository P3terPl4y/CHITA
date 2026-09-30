package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235653CreateCompanyMembersTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235653CreateCompanyMembersTable) Signature() string {
	return "20260929235653_create_company_members_table"
}

// Up Run the migrations.
func (r *M20260929235653CreateCompanyMembersTable) Up() error {
	if !facades.Schema().HasTable("company_members") {
		return facades.Schema().Create("company_members", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("company_id")
			table.UnsignedBigInteger("user_id")
			table.String("role_in_company", 30).Default("viewer")
			table.UnsignedBigInteger("invited_by_user_id").Nullable()
			table.TimestampTz("accepted_at").Nullable()

			table.Unique("company_id", "user_id").Name("idx_company_user")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235653CreateCompanyMembersTable) Down() error {
	return facades.Schema().DropIfExists("company_members")
}
