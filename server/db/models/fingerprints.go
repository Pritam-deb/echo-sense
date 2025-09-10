package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AudioFingerprint struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Address    int
	AnchorTime int
	SongID     uuid.UUID `gorm:"type:uuid"`
	Song       Song      `gorm:"foreignKey:SongID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Generate UUID before inserting
func (af *AudioFingerprint) BeforeCreate(tx *gorm.DB) (err error) {
	if af.ID == uuid.Nil {
		af.ID = uuid.New()
	}
	return
}
