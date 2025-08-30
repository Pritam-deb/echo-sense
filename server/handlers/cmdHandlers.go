package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Pritam-deb/echo-sense/utils"
)

const SONGS_DIR = "songs"

func Download(url string) {
	//Song download logic will be here

	err := utils.CreateDirIfNotExist(SONGS_DIR)
	if err != nil {
		logger := utils.GetLogger()
		ctx := context.Background()
		logger.ErrorContext(ctx, "Failed to create directory for songs", slog.Any("error", err))
	}

	if strings.Contains(url, "track") {
	}
}
