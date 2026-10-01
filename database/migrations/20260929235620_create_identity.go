package migrations

import "goravel/app/facades"

type M20260929235620CreateIdentity struct{}

func (*M20260929235620CreateIdentity) Signature() string { return "20260929235620_create_identity" }
func (*M20260929235620CreateIdentity) Up() error {
	_, err := facades.Schema().Orm().Query().Exec(`CREATE TABLE IF NOT EXISTS users (
 id BIGSERIAL PRIMARY KEY, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ,
 uuid UUID NOT NULL UNIQUE, email VARCHAR(254) NOT NULL, password_hash VARCHAR(255) NOT NULL,
 phone VARCHAR(20) NOT NULL, first_name VARCHAR(80) NOT NULL, last_name VARCHAR(80) NOT NULL DEFAULT '',
 display_name VARCHAR(120) NOT NULL, avatar_url TEXT NOT NULL DEFAULT '',
 role VARCHAR(20) NOT NULL CHECK(role IN ('company','courier')), status BOOLEAN NOT NULL DEFAULT true,
 email_verified_at TIMESTAMPTZ, phone_verified_at TIMESTAMPTZ, last_login_at TIMESTAMPTZ,
 locale VARCHAR(10) NOT NULL DEFAULT 'es', timezone VARCHAR(50) NOT NULL DEFAULT 'UTC', metadata JSONB NOT NULL DEFAULT '{}');
 CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active ON users(LOWER(email)) WHERE deleted_at IS NULL`)
	return err
}
func (*M20260929235620CreateIdentity) Down() error { return facades.Schema().DropIfExists("users") }
