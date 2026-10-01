package models

import (
	"github.com/goravel/framework/database/orm"
	"time"
)

type Publication struct {
	orm.Model
	orm.SoftDeletes

	UUID            string `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	ReferenceCode   string `gorm:"type:varchar(20);uniqueIndex;not null" json:"reference_code"`
	CompanyID       uint   `gorm:"index;not null" json:"company_id"`
	CreatedByUserID uint   `gorm:"index;not null" json:"created_by_user_id"`

	Title            string   `gorm:"type:varchar(200);not null" json:"title"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
	Visibility       string   `gorm:"type:varchar(16);not null;default:network" json:"visibility"`
	PickupDistanceKm *float64 `gorm:"-" json:"pickup_distance_km,omitempty"`

	// Pickup
	PickupAddressID    *uint   `json:"pickup_address_id,omitempty"`
	PickupAddressText  string  `json:"pickup_address_text"`
	PickupContactName  string  `gorm:"type:varchar(120);not null" json:"pickup_contact_name"`
	PickupContactPhone string  `gorm:"type:varchar(20);not null" json:"pickup_contact_phone"`
	PickupLat          float64 `gorm:"type:numeric(10,7);not null" json:"pickup_lat"`
	PickupLng          float64 `gorm:"type:numeric(10,7);not null" json:"pickup_lng"`

	// Dropoff
	DropoffAddressText  string  `gorm:"type:text;not null" json:"dropoff_address_text"`
	DropoffContactName  string  `gorm:"type:varchar(120);not null" json:"dropoff_contact_name"`
	DropoffContactPhone string  `gorm:"type:varchar(20);not null" json:"dropoff_contact_phone"`
	DropoffLat          float64 `gorm:"type:numeric(10,7);not null" json:"dropoff_lat"`
	DropoffLng          float64 `gorm:"type:numeric(10,7);not null" json:"dropoff_lng"`

	DistanceKm         float64 `gorm:"type:numeric(8,2)" json:"distance_km,omitempty"`
	PackageCount       int     `gorm:"not null;default:1" json:"package_count"`
	TotalWeightKg      float64 `gorm:"type:numeric(8,2)" json:"total_weight_kg,omitempty"`
	IsFragile          bool    `gorm:"not null;default:false" json:"is_fragile"`
	RequiresSignature  bool    `gorm:"not null;default:false" json:"requires_signature"`
	DeclaredValueCents *int64  `json:"declared_value_cents,omitempty"`
	Currency           string  `gorm:"type:char(3);not null;default:'USD'" json:"currency"`
	OfferedPriceCents  int64   `gorm:"not null" json:"offered_price_cents"`
	PaymentMethod      string  `gorm:"type:varchar(20);not null;default:'cash'" json:"payment_method"`

	ScheduledPickupFrom   time.Time  `gorm:"not null" json:"scheduled_pickup_from"`
	ScheduledPickupTo     time.Time  `gorm:"not null" json:"scheduled_pickup_to"`
	ScheduledDeliveryBy   *time.Time `json:"scheduled_delivery_by,omitempty"`
	ScheduledDeliveryFrom time.Time  `json:"scheduled_delivery_from"`
	ScheduledDeliveryTo   time.Time  `json:"scheduled_delivery_to"`
	ReportedAt            *time.Time `json:"reported_at,omitempty"`
	ConfirmedAt           *time.Time `json:"confirmed_at,omitempty"`
	Company               *Company   `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
	Courier               *User      `gorm:"foreignKey:AssignedCourierID" json:"courier,omitempty"`
	ExpiresAt             time.Time  `gorm:"index;not null" json:"expires_at"`

	Status              string     `gorm:"type:varchar(20);index;not null;default:'draft'" json:"status"`
	AssignedCourierID   *uint      `gorm:"index" json:"assigned_courier_id,omitempty"`
	AssignedAt          *time.Time `json:"assigned_at,omitempty"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
	CancelledAt         *time.Time `json:"cancelled_at,omitempty"`
	CancellationReason  string     `gorm:"type:text" json:"cancellation_reason,omitempty"`
	SpecialInstructions string     `gorm:"type:text" json:"special_instructions,omitempty"`
	Metadata            string     `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

func (p *Publication) TableName() string { return "publications" }
