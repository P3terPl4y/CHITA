package models

import (
	"github.com/goravel/framework/database/orm"
)

type CourierRating struct {
	orm.Model
	CompanyID     uint     `json:"company_id"`
	CourierUserID uint     `json:"courier_user_id"`
	AuthorUserID  uint     `json:"-"`
	Rating        int      `json:"rating"`
	Comment       string   `json:"comment"`
	Company       *Company `gorm:"foreignKey:CompanyID" json:"-"`
}

func (*CourierRating) TableName() string { return "courier_ratings" }
