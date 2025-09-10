package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Song struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title     string
	Artist    string
	Album     string
	YoutubeID string
	SongKey   string
	Duration  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Generate UUID before inserting
func (s *Song) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}
