package migrations

import (
	"fmt"
	"goravel/app/facades"
)

type M20261001170518DispatchAndRatings struct{}

func (*M20261001170518DispatchAndRatings) Signature() string {
	return "20261001170518_dispatch_and_ratings"
}
func (*M20261001170518DispatchAndRatings) Up() error {
	_, e := facades.Schema().Orm().Query().Exec(`ALTER TABLE courier_profiles ADD COLUMN discovery_lat NUMERIC(10,7),ADD COLUMN discovery_lng NUMERIC(10,7),ADD COLUMN discovery_at TIMESTAMPTZ,ADD COLUMN available_until TIMESTAMPTZ,ADD CONSTRAINT ck_discovery_coordinates CHECK((discovery_lat IS NULL AND discovery_lng IS NULL) OR (discovery_lat BETWEEN -90 AND 90 AND discovery_lng BETWEEN -180 AND 180));
 CREATE INDEX idx_courier_available ON courier_profiles(available_until);
 CREATE TABLE delivery_offers(id BIGSERIAL PRIMARY KEY,created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,publication_id BIGINT NOT NULL REFERENCES publications(id),company_id BIGINT NOT NULL REFERENCES companies(id),courier_user_id BIGINT NOT NULL REFERENCES users(id),status VARCHAR(20) NOT NULL CHECK(status IN ('pending','accepted','declined','expired','withdrawn')),expires_at TIMESTAMPTZ NOT NULL,responded_at TIMESTAMPTZ);
 CREATE UNIQUE INDEX idx_offer_publication_pending ON delivery_offers(publication_id) WHERE status='pending';
 CREATE UNIQUE INDEX idx_offer_courier_pending ON delivery_offers(courier_user_id) WHERE status='pending';
 CREATE INDEX idx_offers_company ON delivery_offers(company_id,id DESC);CREATE INDEX idx_offers_courier ON delivery_offers(courier_user_id,id DESC);
 CREATE TABLE courier_ratings(id BIGSERIAL PRIMARY KEY,created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,company_id BIGINT NOT NULL REFERENCES companies(id),courier_user_id BIGINT NOT NULL REFERENCES users(id),author_user_id BIGINT NOT NULL REFERENCES users(id),rating SMALLINT NOT NULL CHECK(rating BETWEEN 1 AND 5),comment TEXT NOT NULL DEFAULT '',UNIQUE(company_id,courier_user_id));
 CREATE INDEX idx_ratings_courier ON courier_ratings(courier_user_id);
 CREATE INDEX idx_completed_company_courier ON publications(company_id,assigned_courier_id) WHERE status='completed' AND confirmed_at IS NOT NULL;`)
	return e
}
func (*M20261001170518DispatchAndRatings) Down() error {
	var r []struct{ Count int64 }
	q := facades.Schema().Orm().Query()
	if e := q.Raw("SELECT (SELECT count(*) FROM delivery_offers)+(SELECT count(*) FROM courier_ratings) AS count").Scan(&r); e != nil {
		return e
	}
	if len(r) > 0 && r[0].Count > 0 {
		return fmt.Errorf("cannot discard offers or courier ratings")
	}
	_, e := q.Exec(`DROP INDEX idx_completed_company_courier;DROP TABLE courier_ratings,delivery_offers;DROP INDEX idx_courier_available;ALTER TABLE courier_profiles DROP CONSTRAINT ck_discovery_coordinates,DROP COLUMN discovery_lat,DROP COLUMN discovery_lng,DROP COLUMN discovery_at,DROP COLUMN available_until;`)
	return e
}
