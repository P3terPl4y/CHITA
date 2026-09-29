package models
import ("github.com/goravel/framework/database/orm")
type UserAddress struct {
	orm.Model

	UserID      uint    `gorm:"index;not null" json:"user_id"`
	Label       string  `gorm:"type:varchar(50);not null" json:"label"`
	IsPrimary   bool    `gorm:"not null;default:false" json:"is_primary"`
	Line1       string  `gorm:"type:varchar(255);not null" json:"line1"`
	Line2       string  `gorm:"type:varchar(255)" json:"line2,omitempty"`
	City        string  `gorm:"type:varchar(120);not null" json:"city"`
	State       string  `gorm:"type:varchar(120);not null" json:"state"`
	CountryCode string  `gorm:"type:char(2);not null" json:"country_code"`
	PostalCode  string  `gorm:"type:varchar(20)" json:"postal_code,omitempty"`
	Latitude    float64 `gorm:"type:numeric(10,7)" json:"latitude,omitempty"`
	Longitude   float64 `gorm:"type:numeric(10,7)" json:"longitude,omitempty"`
}

func (a *UserAddress) TableName() string { return "user_addresses" }
