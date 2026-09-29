package models
import (
	"github.com/goravel/framework/database/orm"
 "time"
)
type PublicationTrackingLink struct {
	orm.Model

	PublicationID  uint       `gorm:"index;not null" json:"publication_id"`
	Provider       string     `gorm:"type:varchar(50);uniqueIndex:idx_tracking_provider_ext;not null" json:"provider"`
	ExternalID     string     `gorm:"type:varchar(120);uniqueIndex:idx_tracking_provider_ext;not null" json:"external_id"`
	ExternalStatus string     `gorm:"type:varchar(50)" json:"external_status,omitempty"`
	LastSyncedAt   *time.Time `json:"last_synced_at,omitempty"`
	LastPayload    string     `gorm:"type:jsonb" json:"last_payload,omitempty"`
}

func (t *PublicationTrackingLink) TableName() string { return "publication_tracking_links" }
