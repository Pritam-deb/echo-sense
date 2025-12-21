package db

import (
	"errors"
	"fmt"

	"github.com/Pritam-deb/echo-sense/db/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrDBNotConnected = errors.New("db not connected")

func GetSongByID(id uuid.UUID) (*models.Song, error) {
	if DB == nil {
		return nil, ErrDBNotConnected
	}
	var song models.Song
	if err := DB.First(&song, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &song, nil
}

func GetSongByIDString(id string) (*models.Song, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid song id: %w", err)
	}
	return GetSongByID(parsed)
}

func GetSongByYoutubeID(youtubeID string) (*models.Song, error) {
	if DB == nil {
		return nil, ErrDBNotConnected
	}
	var song models.Song
	if err := DB.First(&song, "youtube_id = ?", youtubeID).Error; err != nil {
		return nil, err
	}
	return &song, nil
}

func GetSongBySongKey(songKey string) (*models.Song, error) {
	if DB == nil {
		return nil, ErrDBNotConnected
	}
	var song models.Song
	if err := DB.First(&song, "song_key = ?", songKey).Error; err != nil {
		return nil, err
	}
	return &song, nil
}

func GetSongByArtistTitle(artist, title string) (*models.Song, error) {
	if DB == nil {
		return nil, ErrDBNotConnected
	}
	var song models.Song
	if err := DB.First(&song, "artist = ? AND title = ?", artist, title).Error; err != nil {
		return nil, err
	}
	return &song, nil
}

func ListSongs(limit int) ([]models.Song, error) {
	if DB == nil {
		return nil, ErrDBNotConnected
	}
	if limit <= 0 {
		limit = 100
	}
	var songs []models.Song
	if err := DB.Order("created_at DESC").Limit(limit).Find(&songs).Error; err != nil {
		return nil, err
	}
	return songs, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
