package models
import ("github.com/goravel/framework/database/orm")
type PublicationEvent struct {
	orm.Model // solo ID y timestamps; sin SoftDeletes

	PublicationID uint   `gorm:"index;not null" json:"publication_id"`
	ActorUserID   *uint  `gorm:"index" json:"actor_user_id,omitempty"`
	EventType     string `gorm:"type:varchar(40);index;not null" json:"event_type"`
	FromStatus    string `gorm:"type:varchar(20)" json:"from_status,omitempty"`
	ToStatus      string `gorm:"type:varchar(20)" json:"to_status,omitempty"`
	Payload       string `gorm:"type:jsonb;default:'{}'" json:"payload"`
}

func (e *PublicationEvent) TableName() string { return "publication_events" }
