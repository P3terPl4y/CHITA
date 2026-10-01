package models

import (
	"github.com/goravel/framework/database/orm"
	"time"
)

type CourierProfile struct {
	orm.Model
	DiscoveryToken *string    `json:"-"`
	DiscoveryLat   *float64   `json:"-"`
	DiscoveryLng   *float64   `json:"-"`
	DiscoveryAt    *time.Time `json:"-"`
	AvailableUntil *time.Time `json:"available_until"`
	UserID         uint       `json:"user_id"`
	Address        string     `json:"address"`
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	VehicleType    string     `json:"vehicle_type"`
}
