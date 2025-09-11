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

	"github.com/Pritam-deb/echo-sense/db"
	"github.com/Pritam-deb/echo-sense/db/models"
	recognisingalgorithm "github.com/Pritam-deb/echo-sense/internals/recognisingAlgorithm"
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
			err = processAndSaveTrack(filePath, trackInfo.Title, trackInfo.Artist, track.Album, ytID)
		}(track)
	}
	wg.Wait()
	close(results)
	return downloadedCount, nil
}

func processAndSaveTrack(audioFilePath, songTitle, songArtist, songAlbum, ytID string) error {

	logger := utils.GetLogger()
	wavFilePath, err := wavservice.ConvertToWav(audioFilePath, 1)
	if err != nil {
		logger.Error("Failed to convert to WAV", "error", err, "audioFilePath", audioFilePath)
		return err
	}
	fmt.Println("Converted to WAV:", wavFilePath)
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
	fmt.Println("First 10 samples:", samples[:10])
	spectrogram, err := recognisingalgorithm.Spectrogram(samples, int(wavInfo.SampleRate))
	if err != nil {
		logger.Error("Failed to compute spectrogram", "error", err, "wavFilePath", wavFilePath)
		return fmt.Errorf("Failed to compute spectrogram: %v", err)
	}
	// fmt.Println("Spectrogram computed with", len(spectrogram), "time frames")
	// //to view the spectrogram image, uncomment the lines below
	// err = utils.SaveSpectrogramImage(spectrogram, int(wavInfo.SampleRate), fmt.Sprintf("%s_spectrogram.png", songTitle))
	const (
		DSPratio    = 4
		freqBinSize = 1024
		hopSize     = freqBinSize / 32
	)
	utils.SaveSpectrogramWithLabels(spectrogram, fmt.Sprintf("%s_spectrogram.png", songTitle), int(wavInfo.SampleRate), hopSize, true)

	if err != nil {
		logger.Error("Failed to save spectrogram image", "error", err, "ytID", ytID)
		return fmt.Errorf("Failed to save spectrogram image: %v", err)
	}

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
	fmt.Println("Generated", len(fingerprints), "fingerprints")

	// peaks := recognisingalgorithm.ExtractPeaks(spectrogram, wavInfo.Duration, int(wavInfo.SampleRate))
	// pairs := recognisingalgorithm.BuildConstellationMap(peaks, 3.0)
	// fingerprints := recognisingalgorithm.GenerateFingerprints(pairs)

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
	err = os.Remove(audioFilePath)
	if err != nil {
		logger.Warn("Failed to remove audio file", "error", err, "audioFilePath", audioFilePath)
	}
	// err = os.Remove(wavFilePath)
	// if err != nil {
	// 	logger.Warn("Failed to remove WAV file", "error", err, "wavFilePath", wavFilePath)
	// }
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
	//try not to use busy wait here
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
