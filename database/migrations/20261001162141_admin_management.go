package migrations

import (
	"fmt"
	"goravel/app/facades"
)

type M20261001162141AdminManagement struct{}

func (*M20261001162141AdminManagement) Signature() string { return "20261001162141_admin_management" }
func (*M20261001162141AdminManagement) Up() error {
	_, e := facades.Schema().Orm().Query().Exec(`ALTER TABLE users DROP CONSTRAINT users_role_check;
 ALTER TABLE users ADD CONSTRAINT users_role_check CHECK(role IN ('company','courier','admin'));
 ALTER TABLE users ADD COLUMN admin_version INTEGER NOT NULL DEFAULT 1;
 ALTER TABLE companies ADD COLUMN admin_version INTEGER NOT NULL DEFAULT 1;
 ALTER TABLE publications ADD COLUMN admin_version INTEGER NOT NULL DEFAULT 1;
 CREATE TABLE admin_audits(id BIGSERIAL PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), actor_user_id BIGINT NOT NULL REFERENCES users(id), entity VARCHAR(20) NOT NULL, entity_id BIGINT NOT NULL, action VARCHAR(30) NOT NULL, reason TEXT NOT NULL);
 CREATE INDEX idx_admin_audits_entity ON admin_audits(entity,entity_id,id);`)
	return e
}
func (*M20261001162141AdminManagement) Down() error {
	var r []struct{ Count int64 }
	q := facades.Schema().Orm().Query()
	if e := q.Raw("SELECT (SELECT count(*) FROM users WHERE role='admin')+(SELECT count(*) FROM admin_audits) AS count").Scan(&r); e != nil {
		return e
	}
	if len(r) > 0 && r[0].Count > 0 {
		return fmt.Errorf("cannot remove admin management while administrators or audit records exist")
	}
	_, e := q.Exec(`DROP TABLE admin_audits; ALTER TABLE users DROP COLUMN admin_version; ALTER TABLE companies DROP COLUMN admin_version; ALTER TABLE publications DROP COLUMN admin_version; ALTER TABLE users DROP CONSTRAINT users_role_check; ALTER TABLE users ADD CONSTRAINT users_role_check CHECK(role IN ('company','courier'));`)
	return e
}
