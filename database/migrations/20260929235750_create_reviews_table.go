package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260929235750CreateReviewsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260929235750CreateReviewsTable) Signature() string {
	return "20260929235750_create_reviews_table"
}

// Up Run the migrations.
func (r *M20260929235750CreateReviewsTable) Up() error {
	if !facades.Schema().HasTable("reviews") {
		return facades.Schema().Create("reviews", func(table schema.Blueprint) {
			table.TimestampTz("created_at").Nullable()
			table.TimestampTz("updated_at").Nullable()
			table.BigIncrements("id")
			table.UnsignedBigInteger("publication_id")
			table.UnsignedBigInteger("author_user_id")
			table.UnsignedBigInteger("target_user_id").Nullable()
			table.UnsignedBigInteger("target_company_id").Nullable()
			table.Text("rating")
			table.Text("comment")

			table.Unique("publication_id", "author_user_id").Name("idx_review_pub_author")
			table.Index("target_company_id").Name("idx_reviews_target_company_id")
			table.Index("target_user_id").Name("idx_reviews_target_user_id")
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260929235750CreateReviewsTable) Down() error {
	return facades.Schema().DropIfExists("reviews")
}
