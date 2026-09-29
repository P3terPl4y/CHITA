package models
import (
	"github.com/goravel/framework/database/orm"
)

type PublicationItem struct {
	orm.Model

	PublicationID       uint    `gorm:"index;not null" json:"publication_id"`
	Description         string  `gorm:"type:varchar(255);not null" json:"description"`
	Quantity            int     `gorm:"not null;default:1" json:"quantity"`
	WeightKg            float64 `gorm:"type:numeric(8,2)" json:"weight_kg,omitempty"`
	LengthCm            *int    `json:"length_cm,omitempty"`
	WidthCm             *int    `json:"width_cm,omitempty"`
	HeightCm            *int    `json:"height_cm,omitempty"`
	DeclaredValueCents  *int64  `json:"declared_value_cents,omitempty"`
	IsFragile           bool    `gorm:"not null;default:false" json:"is_fragile"`
	PhotoURLs           string  `gorm:"type:jsonb;default:'[]'" json:"photo_urls"`
}

func (i *PublicationItem) TableName() string { return "publication_items" }
