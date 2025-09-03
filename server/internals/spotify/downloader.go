package spotify

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	wavservice "github.com/Pritam-deb/echo-sense/internals/wavService"
	"github.com/Pritam-deb/echo-sense/utils"
	"github.com/kkdai/youtube/v2"
)

var wg sync.WaitGroup

func DownloadSingleTrack(url string, downloadPath string) {
	logger := utils.GetLogger()
	logger.Info("Starting download for single track", "url", url, "path", downloadPath)
	track, err := GetTrackInfo(url)
	if err != nil {
		logger.Error("Failed to get track info", "error", err)
		return
	}
	logger.Info("Track info retrieved", "track", track)
	tracks := []Track{*track}
	count, err := TracksDownloader(tracks, downloadPath)
	if err != nil {
		logger.Error("Failed to download track", "error", err)
		return
	}
	logger.Info("Download completed", "count", count)
}

func TracksDownloader(tracks []Track, downloadPath string) (int, error) {
	fmt.Println("Starting download of", len(tracks), "tracks to", downloadPath)
	var downloadedCount int
	// var downloadedTracks []string

	noCPUs := runtime.NumCPU()
	sem := make(chan struct{}, noCPUs)
	logger := utils.GetLogger()
	results := make(chan int, len(tracks))
	ctx := context.Background()

	for _, track := range tracks {

		wg.Add(1)
		go func(track Track) {
			fmt.Println("Inside goroutine for track!")
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			trackInfo := track.buildTrack()
			//get YT id of the track
			ytID, err := getYoutubeID(*trackInfo)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to get YT ID", slog.Any("error", err), slog.Any("track", trackInfo))
				results <- 0
				return
			}
			//download the track from yt
			trackInfo.Title, trackInfo.Artist = changeFileName(trackInfo.Title, trackInfo.Artist)
			fileName := fmt.Sprintf("%s - %s", trackInfo.Artist, trackInfo.Title)
			filePath := filepath.Join(downloadPath, fileName+".m4a")
			err = downloadAudioFromYT(ytID, downloadPath, filePath)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to download audio from YT", slog.Any("error", err), slog.Any("ytID", ytID), slog.Any("filePath", filePath))
				results <- 0
				return
			}
			err = processAndSaveTrack(filePath, trackInfo.Title, trackInfo.Artist, ytID)
		}(track)
	}
	wg.Wait()
	close(results)
	return downloadedCount, nil
}

func processAndSaveTrack(audioFilePath, songTitle, songArtist, ytID string) error {
	logger := utils.GetLogger()
	wavFilePath, err := wavservice.ConvertToWav(audioFilePath, 1)
	if err != nil {
		logger.Error("Failed to convert to WAV", "error", err, "audioFilePath", audioFilePath)
		return err
	}
	fmt.Println("Converted to WAV:", wavFilePath)

	return nil
}

func downloadAudioFromYT(id, path, filepath string) error {
	fmt.Println("Downloading to:", filepath, " in directory:", path)
	logger := utils.GetLogger()
	dir, err := os.Stat(path)
	if err != nil {
		logger.Error("Failed to get directory info", "error", err, "path", path)
		return err
	}

	if !dir.IsDir() {
		err := fmt.Errorf("provided path is not a directory: %s", path)
		logger.Error("Invalid directory path", "error", err, "path", path)
		return err
	}
	youtubeClient := youtube.Client{}
	video, err := youtubeClient.GetVideo(id)
	if err != nil {
		logger.Error("Failed to get YouTube video", "error", err, "id", id)
		return err
	}

	formats := video.Formats.Itag(140) // m4a format
	if len(formats) == 0 {
		err := fmt.Errorf("no suitable format found for video ID: %s", id)
		logger.Error("No suitable format", "error", err, "id", id)
		return err
	}
	file, err := os.Create(filepath)
	if err != nil {
		logger.Error("Failed to create file", "error", err, "filepath", filepath)
		return err
	}
	var fileSize int64
	//to make sure file size is not 0 and file is completely written
	for fileSize == 0 {
		stream, _, err := youtubeClient.GetStream(video, &formats[0])
		if err != nil {
			logger.Error("Failed to get video stream", "error", err, "id", id)
			file.Close()
			return err
		}
		if _, err = io.Copy(file, stream); err != nil {
			logger.Error("Failed to copy stream to file", "error", err, "filepath", filepath)
			file.Close()
			return err
		}
		fileSize, _ = GetFileSize(filepath)
	}
	defer file.Close()
	return nil
}
