package migrations

import (
	"fmt"
	"goravel/app/facades"
)

type M20261001172419DiscoveryConsent struct{}

func (*M20261001172419DiscoveryConsent) Signature() string { return "20261001172419_discovery_consent" }
func (*M20261001172419DiscoveryConsent) Up() error {
	_, e := facades.Schema().Orm().Query().Exec("ALTER TABLE courier_profiles ADD COLUMN discovery_token UUID;UPDATE courier_profiles SET available_until=NULL;ALTER TABLE courier_profiles ADD CONSTRAINT ck_discovery_consent CHECK(available_until IS NULL OR (discovery_token IS NOT NULL AND discovery_lat IS NOT NULL AND discovery_lng IS NOT NULL AND discovery_at IS NOT NULL))")
	return e
}
func (*M20261001172419DiscoveryConsent) Down() error {
	var rows []struct{ Count int64 }
	q := facades.Schema().Orm().Query()
	if e := q.Raw("SELECT count(*) AS count FROM courier_profiles WHERE available_until IS NOT NULL").Scan(&rows); e != nil {
		return e
	}
	if len(rows) > 0 && rows[0].Count > 0 {
		return fmt.Errorf("cannot remove consent protection while availability is enabled")
	}
	_, e := q.Exec("ALTER TABLE courier_profiles DROP CONSTRAINT ck_discovery_consent,DROP COLUMN discovery_token")
	return e
}
