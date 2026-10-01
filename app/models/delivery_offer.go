package models

import (
	"github.com/goravel/framework/database/orm"
	"time"
)

type DeliveryOffer struct {
	orm.Model
	PublicationID uint         `json:"publication_id"`
	CompanyID     uint         `json:"company_id"`
	CourierUserID uint         `json:"courier_user_id"`
	Status        string       `json:"status"`
	ExpiresAt     time.Time    `json:"expires_at"`
	RespondedAt   *time.Time   `json:"responded_at"`
	Publication   *Publication `gorm:"foreignKey:PublicationID" json:"-"`
	Courier       *User        `gorm:"foreignKey:CourierUserID" json:"-"`
}

func (*DeliveryOffer) TableName() string { return "delivery_offers" }
