package models

import (
	"github.com/goravel/framework/database/orm"
	"time"
)

type Notification struct {
	orm.Model
	UserID        uint       `json:"-"`
	PublicationID *uint      `json:"publication_id,omitempty"`
	Message       string     `json:"message"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
}
