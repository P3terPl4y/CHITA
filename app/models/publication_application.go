package models
import (
"github.com/goravel/framework/database/orm"
	"time"
	)

type PublicationApplication struct {
	orm.Model

	PublicationID      uint       `gorm:"uniqueIndex:idx_pub_courier_active,where:status IN ('pending','accepted');not null" json:"publication_id"`
	CourierUserID      uint       `gorm:"uniqueIndex:idx_pub_courier_active,where:status IN ('pending','accepted');not null" json:"courier_user_id"`
	Message            string     `gorm:"type:text" json:"message,omitempty"`
	OfferedPriceCents  *int64     `json:"offered_price_cents,omitempty"`
	EstimatedPickupAt  *time.Time `json:"estimated_pickup_at,omitempty"`
	Status             string     `gorm:"type:varchar(20);index;not null;default:'pending'" json:"status"`
	RespondedAt        *time.Time `json:"responded_at,omitempty"`
}

func (a *PublicationApplication) TableName() string { return "publication_applications" }
