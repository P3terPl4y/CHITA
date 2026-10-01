package migrations

import "goravel/app/facades"

type M20260930231138DeliveryWorkflow struct{}

func (*M20260930231138DeliveryWorkflow) Signature() string { return "20260930231138_delivery_workflow" }
func (*M20260930231138DeliveryWorkflow) Up() error {
	statements := []string{
		`ALTER TABLE companies ADD COLUMN address TEXT NOT NULL DEFAULT '', ADD COLUMN latitude NUMERIC(10,7) NOT NULL DEFAULT 0, ADD COLUMN longitude NUMERIC(10,7) NOT NULL DEFAULT 0;
ALTER TABLE companies ALTER COLUMN tax_id DROP NOT NULL; UPDATE companies SET tax_id=NULL WHERE tax_id='';
CREATE UNIQUE INDEX idx_companies_owner ON companies(owner_user_id) WHERE deleted_at IS NULL;
ALTER TABLE companies ADD CONSTRAINT fk_companies_owner FOREIGN KEY(owner_user_id) REFERENCES users(id);
CREATE TABLE courier_profiles (id BIGSERIAL PRIMARY KEY, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ,
 user_id BIGINT NOT NULL UNIQUE REFERENCES users(id), address TEXT NOT NULL,
 latitude NUMERIC(10,7) NOT NULL CHECK(latitude BETWEEN -90 AND 90), longitude NUMERIC(10,7) NOT NULL CHECK(longitude BETWEEN -180 AND 180),
 vehicle_type VARCHAR(20) NOT NULL CHECK(vehicle_type IN ('foot','bicycle','motorbike','car','van')));
ALTER TABLE company_members ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','accepted','revoked'));
UPDATE company_members SET status='accepted' WHERE accepted_at IS NOT NULL;
ALTER TABLE company_members ADD CONSTRAINT fk_members_company FOREIGN KEY(company_id) REFERENCES companies(id), ADD CONSTRAINT fk_members_user FOREIGN KEY(user_id) REFERENCES users(id);
CREATE INDEX idx_members_courier_status ON company_members(user_id,status);`,
		`ALTER TABLE publications ADD COLUMN pickup_address_text TEXT NOT NULL DEFAULT '', ADD COLUMN scheduled_delivery_from TIMESTAMPTZ, ADD COLUMN scheduled_delivery_to TIMESTAMPTZ, ADD COLUMN reported_at TIMESTAMPTZ, ADD COLUMN confirmed_at TIMESTAMPTZ;
UPDATE publications SET scheduled_delivery_from=scheduled_pickup_from,scheduled_delivery_to=COALESCE(scheduled_delivery_by,scheduled_pickup_to);
ALTER TABLE publications ALTER COLUMN scheduled_delivery_from SET NOT NULL, ALTER COLUMN scheduled_delivery_to SET NOT NULL;
ALTER TABLE publications ALTER COLUMN pickup_lat TYPE NUMERIC(10,7), ALTER COLUMN pickup_lng TYPE NUMERIC(10,7), ALTER COLUMN dropoff_lat TYPE NUMERIC(10,7), ALTER COLUMN dropoff_lng TYPE NUMERIC(10,7);
ALTER TABLE publications ADD CONSTRAINT fk_publications_company FOREIGN KEY(company_id) REFERENCES companies(id), ADD CONSTRAINT fk_publications_creator FOREIGN KEY(created_by_user_id) REFERENCES users(id), ADD CONSTRAINT fk_publications_courier FOREIGN KEY(assigned_courier_id) REFERENCES users(id);
ALTER TABLE publications ADD CONSTRAINT ck_publications_fee CHECK(offered_price_cents>0), ADD CONSTRAINT ck_publications_windows CHECK(scheduled_pickup_from<=scheduled_pickup_to AND scheduled_delivery_from<=scheduled_delivery_to AND scheduled_delivery_to>=scheduled_pickup_to AND scheduled_delivery_from>=scheduled_pickup_from);
ALTER TABLE publications ADD CONSTRAINT ck_publications_coordinates CHECK(pickup_lat BETWEEN -90 AND 90 AND pickup_lng BETWEEN -180 AND 180 AND dropoff_lat BETWEEN -90 AND 90 AND dropoff_lng BETWEEN -180 AND 180);
CREATE INDEX idx_publications_feed ON publications(company_id,status,expires_at);
ALTER TABLE publication_events ADD CONSTRAINT fk_events_publication FOREIGN KEY(publication_id) REFERENCES publications(id);
CREATE TABLE notifications(id BIGSERIAL PRIMARY KEY,created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,user_id BIGINT NOT NULL REFERENCES users(id),publication_id BIGINT REFERENCES publications(id),message TEXT NOT NULL,read_at TIMESTAMPTZ);
CREATE INDEX idx_notifications_inbox ON notifications(user_id,id DESC);
CREATE TABLE halcon_accounts(id BIGSERIAL PRIMARY KEY,created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,user_id BIGINT NOT NULL UNIQUE REFERENCES users(id),halcon_user_id BIGINT NOT NULL UNIQUE,personal_id BIGINT NOT NULL,session_ciphertext TEXT NOT NULL,expires_at TIMESTAMPTZ NOT NULL);`,
	}
	for _, sql := range statements {
		if _, err := facades.Schema().Orm().Query().Exec(sql); err != nil {
			return err
		}
	}
	return nil
}
func (*M20260930231138DeliveryWorkflow) Down() error {
	_, err := facades.Schema().Orm().Query().Exec(`DROP TABLE halcon_accounts,notifications;
ALTER TABLE publication_events DROP CONSTRAINT fk_events_publication;
DROP INDEX idx_publications_feed;
ALTER TABLE publications DROP CONSTRAINT fk_publications_company,DROP CONSTRAINT fk_publications_creator,DROP CONSTRAINT fk_publications_courier,DROP CONSTRAINT ck_publications_fee,DROP CONSTRAINT ck_publications_windows,DROP CONSTRAINT ck_publications_coordinates;
ALTER TABLE publications DROP COLUMN pickup_address_text,DROP COLUMN scheduled_delivery_from,DROP COLUMN scheduled_delivery_to,DROP COLUMN reported_at,DROP COLUMN confirmed_at;
DROP INDEX idx_members_courier_status;
ALTER TABLE company_members DROP CONSTRAINT fk_members_company,DROP CONSTRAINT fk_members_user,DROP COLUMN status;
DROP TABLE courier_profiles;
ALTER TABLE companies DROP CONSTRAINT fk_companies_owner,DROP COLUMN address,DROP COLUMN latitude,DROP COLUMN longitude;
DROP INDEX idx_companies_owner`)
	return err
}
