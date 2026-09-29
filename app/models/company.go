package models
import (
"github.com/goravel/framework/database/orm"
	"time"
	)
type Company struct {
	orm.Model
	orm.SoftDeletes

	UUID             string     `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	OwnerUserID      uint       `gorm:"index;not null" json:"owner_user_id"`
	LegalName        string     `gorm:"type:varchar(200);not null" json:"legal_name"`
	TradeName        string     `gorm:"type:varchar(200);not null" json:"trade_name"`
	TaxID            string     `gorm:"type:varchar(50);uniqueIndex:idx_companies_taxid_active,where:deleted_at IS NULL" json:"tax_id,omitempty"`
	Email            string     `gorm:"type:citext;not null" json:"email"`
	Phone            string     `gorm:"type:varchar(20);not null" json:"phone"`
	LogoURL          string     `gorm:"type:text" json:"logo_url,omitempty"`
	Website          string     `gorm:"type:text" json:"website,omitempty"`
	Description      string     `gorm:"type:text" json:"description,omitempty"`
	Status           string     `gorm:"type:varchar(20);index;not null;default:'pending'" json:"status"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	VerifiedByUserID *uint      `json:"verified_by_user_id,omitempty"`
	Metadata         string     `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

func (c *Company) TableName() string { return "companies" }
