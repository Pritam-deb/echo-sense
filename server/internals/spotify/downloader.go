package spotify

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/Pritam-deb/echo-sense/db"
	"github.com/Pritam-deb/echo-sense/db/models"
	recognisingalgorithm "github.com/Pritam-deb/echo-sense/internals/recognisingAlgorithm"
	wavservice "github.com/Pritam-deb/echo-sense/internals/wavService"
	"github.com/Pritam-deb/echo-sense/utils"
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
	var downloadedCount int
	// var downloadedTracks []string

	noCPUs := runtime.NumCPU()
	sem := make(chan struct{}, noCPUs)
	logger := utils.GetLogger()
	results := make(chan int, len(tracks))
	errs := make(chan error, len(tracks))
	ctx := context.Background()

	for _, track := range tracks {

		wg.Add(1)
		go func(track Track) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			trackInfo := track.buildTrack()
			//get YT id of the track
			ytID, err := getYoutubeID(*trackInfo)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to get YT ID", slog.Any("error", err), slog.Any("track", trackInfo))
				errs <- err
				results <- 0
				return
			}
			//download the track from yt
			trackInfo.Title, trackInfo.Artist = changeFileName(trackInfo.Title, trackInfo.Artist)
			fileName := fmt.Sprintf("%s - %s", trackInfo.Artist, trackInfo.Title)
			filePath := filepath.Join(downloadPath, fileName+".m4a")

			if _, err := os.Stat(filePath); err == nil {
				logger.Info("File already exists, skipping download", "filePath", filePath)
				err = processAndSaveTrack(filePath, trackInfo.Title, trackInfo.Artist, track.Album, ytID)
				if err != nil {
					logger.ErrorContext(ctx, "Failed to process and save track", slog.Any("error", err), slog.Any("filePath", filePath))
					errs <- err
					results <- 0
					return
				}
				results <- 1
				return
			}

			err = downloadAudioFromYT(ytID, downloadPath, filePath)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to download audio from YT", slog.Any("error", err), slog.Any("ytID", ytID), slog.Any("filePath", filePath))
				errs <- err
				results <- 0
				return
			}
			err = processAndSaveTrack(filePath, trackInfo.Title, trackInfo.Artist, track.Album, ytID)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to process and save track", slog.Any("error", err), slog.Any("filePath", filePath))
				errs <- err
				results <- 0
				return
			}
			results <- 1
		}(track)
	}
	wg.Wait()
	close(results)
	close(errs)
	for r := range results {
		downloadedCount += r
	}
	var firstErr error
	for err := range errs {
		if err != nil {
			firstErr = err
			break
		}
	}
	if downloadedCount == 0 && firstErr != nil {
		return 0, firstErr
	}
	return downloadedCount, nil
}

func processAndSaveTrack(audioFilePath, songTitle, songArtist, songAlbum, ytID string) error {

	logger := utils.GetLogger()
	wavFilePath, err := wavservice.ConvertToWav(audioFilePath, 1)
	if err != nil {
		logger.Error("Failed to convert to WAV", "error", err, "audioFilePath", audioFilePath)
		return err
	}
	// fmt.Println("Converted to WAV:", wavFilePath)
	wavInfo, err := wavservice.ReadWavFile(wavFilePath)
	if err != nil {
		logger.Error("Failed to read WAV file", "error", err, "wavFilePath", wavFilePath)
	}
	fmt.Println("wav duration:", wavInfo.Duration, "seconds")

	samples, err := wavservice.ConvertWavDataToSamples(wavInfo.Data)
	if err != nil {
		logger.Error("Failed to convert WAV data to samples", "error", err, "wavFilePath", wavFilePath)
		return fmt.Errorf("Failed to convert WAV data to samples: %v", err)
	}
	fmt.Println("Number of samples:", len(samples))
	// fmt.Println("First 64 samples:", samples[:64])
	spectrogram, err := recognisingalgorithm.Spectrogram(samples, int(wavInfo.SampleRate))
	if err != nil {
		logger.Error("Failed to compute spectrogram", "error", err, "wavFilePath", wavFilePath)
		return fmt.Errorf("Failed to compute spectrogram: %v", err)
	}
	// fmt.Println("Spectrogram computed with", len(spectrogram), "time frames")
	// //to view the spectrogram image, uncomment the lines below
	// err = utils.SaveSpectrogramImage(spectrogram, int(wavInfo.SampleRate), fmt.Sprintf("%s_spectrogram.png", songTitle))
	// const (
	// 	DSPratio    = 4
	// 	freqBinSize = 1024
	// 	hopSize     = freqBinSize / 32
	// )
	// utils.SaveSpectrogramWithLabels(spectrogram, fmt.Sprintf("%s_spectrogram.png", songTitle), int(wavInfo.SampleRate), hopSize, wavInfo.Duration, true)
	// utils.VerifySpectrogramCompleteness(spectrogram, samples, int(wavInfo.SampleRate), hopSize)
	// if err != nil {
	// 	logger.Error("Failed to save spectrogram image", "error", err, "ytID", ytID)
	// 	return fmt.Errorf("Failed to save spectrogram image: %v", err)
	// }

	song := models.Song{
		Title:     songTitle,
		Artist:    songArtist,
		Album:     songAlbum,
		YoutubeID: ytID,
		SongKey:   utils.GenerateSongKey(songArtist, songTitle),
		Duration:  int(wavInfo.Duration),
	}

	if err := db.DB.Create(&song).Error; err != nil {
		logger.Error("Failed to save song to DB", "error", err)
		return err
	}
	logger.Info("Song saved to DB", "youtube_id", ytID)
	peaks := recognisingalgorithm.ExtractPeaks(spectrogram, wavInfo.Duration)
	fingerprints := recognisingalgorithm.Fingerprint(peaks, song.ID.String())

	// peaks := recognisingalgorithm.ExtractPeaks(spectrogram, wavInfo.Duration, int(wavInfo.SampleRate))
	// pairs := recognisingalgorithm.BuildConstellationMap(peaks, 3.0)
	// fingerprints := recognisingalgorithm.GenerateFingerprints(pairs)
	fmt.Println("Generated", len(fingerprints), "fingerprints")
	var audioFingerprints []models.AudioFingerprint
	for address, fp := range fingerprints {
		audioFingerprints = append(audioFingerprints, models.AudioFingerprint{
			Address:    int(address),
			AnchorTime: int(fp.AnchorTime),
			SongID:     song.ID,
		})
	}

	if len(audioFingerprints) > 0 {
		batchSize := 1000
		for i := 0; i < len(audioFingerprints); i += batchSize {
			end := i + batchSize
			if end > len(audioFingerprints) {
				end = len(audioFingerprints)
			}
			if err := db.DB.Create(audioFingerprints[i:end]).Error; err != nil {
				logger.Error("Failed to save batch of fingerprints", "error", err)
				return err
			}
		}
	} else {
		fmt.Println("No fingerprints generated for song:", songTitle)
	}

	// clean up temp files
	// err = os.Remove(audioFilePath)
	// if err != nil {
	// 	logger.Warn("Failed to remove audio file", "error", err, "audioFilePath", audioFilePath)
	// }
	err = os.Remove(wavFilePath)
	if err != nil {
		logger.Warn("Failed to remove WAV file", "error", err, "wavFilePath", wavFilePath)
	}
	return nil
}

func downloadAudioFromYT(id, dirPath, filePath string) error {
	logger := utils.GetLogger()

	if id == "" {
		return fmt.Errorf("empty youtube id")
	}

	if _, err := os.Stat(dirPath); err != nil {
		return err
	}

	ytURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)
	tmpFile := filePath + ".part"

	cmd := exec.Command(
		"yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio",
		"-o", tmpFile,
		"--no-playlist",
		"--quiet",
		ytURL,
	)

	if err := cmd.Run(); err != nil {
		logger.Error("yt-dlp failed", "error", err, "id", id)
		_ = os.Remove(tmpFile)
		return err
	}

	if err := utils.MoveFile(tmpFile, filePath); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}

	size, err := GetFileSize(filePath)
	if err != nil || size <= 0 {
		_ = os.Remove(filePath)
		return fmt.Errorf("yt-dlp produced empty file")
	}

	return nil
}
