package models
import("github.com/goravel/framework/database/orm")
type Review struct {
	orm.Model

	PublicationID    uint   `gorm:"uniqueIndex:idx_review_pub_author;not null" json:"publication_id"`
	AuthorUserID     uint   `gorm:"uniqueIndex:idx_review_pub_author;not null" json:"author_user_id"`
	TargetUserID     *uint  `gorm:"index" json:"target_user_id,omitempty"`
	TargetCompanyID  *uint  `gorm:"index" json:"target_company_id,omitempty"`
	Rating           int    `gorm:"type:smallint;not null" json:"rating"`
	Comment          string `gorm:"type:text" json:"comment,omitempty"`
}

func (r *Review) TableName() string { return "reviews" }
