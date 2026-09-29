package models
import (
"github.com/goravel/framework/database/orm"
"time"
)

type CompanyMember struct {
	orm.Model

	CompanyID       uint       `gorm:"uniqueIndex:idx_company_user;not null" json:"company_id"`
	UserID          uint       `gorm:"uniqueIndex:idx_company_user;not null" json:"user_id"`
	RoleInCompany   string     `gorm:"type:varchar(30);not null;default:'viewer'" json:"role_in_company"`
	InvitedByUserID *uint      `json:"invited_by_user_id,omitempty"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
}

func (m *CompanyMember) TableName() string { return "company_members" }

