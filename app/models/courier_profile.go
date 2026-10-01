package models

import (
	"github.com/goravel/framework/database/orm"
)

type CourierProfile struct {
	orm.Model
	UserID      uint    `json:"user_id"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	VehicleType string  `json:"vehicle_type"`
}
