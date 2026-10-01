package migrations

import (
	"fmt"
	"goravel/app/facades"
)

type M20261001153005PublicJobs struct{}

// Signature The unique signature for the migration.
func (r *M20261001153005PublicJobs) Signature() string {
	return "20261001153005_public_jobs"
}

// Up Run the migrations.
func (r *M20261001153005PublicJobs) Up() error {
	_, err := facades.Schema().Orm().Query().Exec(`ALTER TABLE publications ADD COLUMN visibility VARCHAR(16) NOT NULL DEFAULT 'network' CHECK (visibility IN ('network','public')); CREATE INDEX idx_publications_public_available ON publications(expires_at) WHERE visibility='public' AND status='published' AND deleted_at IS NULL;`)
	return err
}

// Down Reverse the migrations.
func (r *M20261001153005PublicJobs) Down() error {
	var rows []struct{ Count int64 }
	if err := facades.Schema().Orm().Query().Raw("SELECT COUNT(*) AS count FROM publications WHERE visibility='public'").Scan(&rows); err != nil {
		return err
	}
	if len(rows) > 0 && rows[0].Count > 0 {
		return fmt.Errorf("cannot remove visibility while public publications exist")
	}
	_, err := facades.Schema().Orm().Query().Exec("DROP INDEX idx_publications_public_available; ALTER TABLE publications DROP COLUMN visibility")
	return err
}
