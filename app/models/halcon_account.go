package models

import (
	"github.com/goravel/framework/database/orm"
	"time"
)

type HalconAccount struct {
	orm.Model
	UserID            uint      `json:"-"`
	HalconUserID      uint      `json:"-"`
	PersonalID        uint      `json:"-"`
	SessionCiphertext string    `json:"-"`
	ExpiresAt         time.Time `json:"expires_at"`
}
