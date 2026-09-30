package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235524CreateUserTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235524CreateUserTable) Signature() string {
	return "20260929235524_create_user_table"
}

// Up Run the migrations.
func (r *M20260929235524CreateUserTable) Up() error {
	if !facades.Schema().HasTable("users") {
		return facades.Schema().Create("users", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.TimestampTz("deleted_at").Nullable()
			table.Uuid("uuid")
			table.Text("email")
			table.TimestampTz("email_verified_at").Nullable()
			table.String("phone", 20)
			table.TimestampTz("phone_verified_at").Nullable()
			table.String("password_hash", 255)
			table.String("first_name", 80)
			table.String("last_name", 80)
			table.String("display_name", 120)
			table.Text("avatar_url")
			table.String("role", 20).Default("courier")
			table.String("status", 20).Default("pending")
			table.String("locale", 10).Default("es")
			table.String("timezone", 50).Default("UTC")
			table.TimestampTz("last_login_at").Nullable()
			table.Json("metadata").Default("{}")

			table.Unique("email").Name("idx_users_email_active")
			table.Index("phone").Name("idx_users_phone")
			table.Index("role").Name("idx_users_role")
			table.Index("status").Name("idx_users_status")
			table.Unique("uuid").Name("idx_users_uuid")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235524CreateUserTable) Down() error {
	return facades.Schema().DropIfExists("users")
}
