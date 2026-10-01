package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20260929235620CreateIdentity{},
		&migrations.M20260929235634CreateUserAddressesTable{},
		&migrations.M20260929235644CreateCompaniesTable{},
		&migrations.M20260929235653CreateCompanyMembersTable{},
		&migrations.M20260929235702CreatePublicationsTable{},
		&migrations.M20260929235712CreatePublicationItemsTable{},
		&migrations.M20260929235722CreatePublicationApplicationsTable{},
		&migrations.M20260929235731CreatePublicationEventsTable{},
		&migrations.M20260929235741CreatePublicationTrackingLinksTable{},
		&migrations.M20260929235750CreateReviewsTable{},
		&migrations.M20260930231138DeliveryWorkflow{},
		&migrations.M20260930235204SessionGrants{},
		&migrations.M20261001153005PublicJobs{},
		&migrations.M20261001162141AdminManagement{},
	}
}
