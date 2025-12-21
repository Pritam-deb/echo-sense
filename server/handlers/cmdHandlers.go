package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	recognisingalgorithm "github.com/Pritam-deb/echo-sense/internals/recognisingAlgorithm"
	"github.com/Pritam-deb/echo-sense/internals/spotify"
	wavservice "github.com/Pritam-deb/echo-sense/internals/wavService"
	"github.com/Pritam-deb/echo-sense/utils"
)

const SONGS_DIR = "songs"

var logger = utils.GetLogger()

func Download(url string) {
	//Song download logic will be here

	err := utils.CreateDirIfNotExist(SONGS_DIR)
	if err != nil {

		ctx := context.Background()
		logger.ErrorContext(ctx, "Failed to create directory for songs", slog.Any("error", err))
	}

	if strings.Contains(url, "track") {
		spotify.DownloadSingleTrack(url, SONGS_DIR)
	}
}

func Search(track string) {
	filePath := filepath.Join(SONGS_DIR, track+".m4a")
	if _, err := os.Stat(filePath); err != nil {
		logger.Info("File doesnt exist.", "filepath", filePath)
		return
	}
	wavFilePath, err := wavservice.ConvertToWav(filePath, 1)
	if err != nil {
		logger.Error("Failed to convert to WAV", "error", err, "audioFilePath", filePath)
		return
	}
	wavInfo, err := wavservice.ReadWavFile(wavFilePath)
	if err != nil {
		logger.Error("Failed to read WAV file", "error", err, "wavFilePath", wavFilePath)
	}
	samples, err := wavservice.ConvertWavDataToSamples(wavInfo.Data)
	if err != nil {
		logger.Error("Failed to convert WAV data to samples", "error", err, "wavFilePath", wavFilePath)
		return
	}
	fmt.Println("Number of samples:", len(samples))
	matches, searchDuration, err := recognisingalgorithm.FindMatches(int(wavInfo.SampleRate), wavInfo.Duration, samples)

	if err != nil {
		fmt.Println("Error finding matches:", err)
		return
	}

	if len(matches) == 0 {
		fmt.Println("\nNo match found.")
		fmt.Printf("\nSearch took: %s\n", searchDuration)
		return
	}

	msg := "Matches:"
	topMatches := matches
	if len(matches) >= 5 {
		msg = "Top 5 matches:"
		topMatches = matches[:5]
	}

	fmt.Println(msg)
	for _, match := range topMatches {
		fmt.Printf("\t- %s by %s, score: %.2f\n",
			match.SongTitle, match.SongArtist, match.Score)
	}

	fmt.Printf("\nSearch took: %s\n", searchDuration)
	topMatch := topMatches[0]
	fmt.Printf("\nFinal prediction: %s by %s , score: %.2f\n", topMatch.SongTitle, topMatch.SongArtist, topMatch.Score)
}
