package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

type User struct {
	orm.Model
	orm.SoftDeletes

	UUID              string     `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	Email             string     `gorm:"type:citext;uniqueIndex:idx_users_email_active,where:deleted_at IS NULL;not null" json:"email"`
	EmailVerifiedAt   *time.Time `json:"email_verified_at,omitempty"`
	Phone             string     `gorm:"type:varchar(20);index;not null" json:"phone"`
	PhoneVerifiedAt   *time.Time `json:"phone_verified_at,omitempty"`
	PasswordHash      string     `gorm:"type:varchar(255);not null" json:"-"`
	FirstName         string     `gorm:"type:varchar(80);not null" json:"first_name"`
	LastName          string     `gorm:"type:varchar(80);not null" json:"last_name"`
	DisplayName       string     `gorm:"type:varchar(120);not null" json:"display_name"`
	AvatarURL         string     `gorm:"type:text" json:"avatar_url,omitempty"`
	Role              string     `gorm:"type:varchar(20);index;not null;default:'courier'" json:"role"`
	Status            string     `gorm:"type:varchar(20);index;not null;default:'pending'" json:"status"`
	Locale            string     `gorm:"type:varchar(10);not null;default:'es'" json:"locale"`
	Timezone          string     `gorm:"type:varchar(50);not null;default:'UTC'" json:"timezone"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	Metadata          string     `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

func (u *User) TableName() string { return "users" }
