package migrations

import "goravel/app/facades"

type M20260930235204SessionGrants struct{}

func (*M20260930235204SessionGrants) Signature() string { return "20260930235204_session_grants" }
func (*M20260930235204SessionGrants) Up() error {
	_, e := facades.Schema().Orm().Query().Exec(`CREATE TABLE auth_grants(token VARCHAR(36) PRIMARY KEY,user_id BIGINT NOT NULL REFERENCES users(id),expires_at TIMESTAMPTZ NOT NULL);CREATE INDEX idx_auth_grants_expiration ON auth_grants(expires_at);ALTER TABLE companies ADD CONSTRAINT ck_companies_coordinates CHECK(latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180);`)
	return e
}
func (*M20260930235204SessionGrants) Down() error {
	_, e := facades.Schema().Orm().Query().Exec(`DROP TABLE auth_grants;ALTER TABLE companies DROP CONSTRAINT ck_companies_coordinates;`)
	return e
}
