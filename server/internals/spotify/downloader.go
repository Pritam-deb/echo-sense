package spotify

import (
	"fmt"
	"sync"

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

	// noCPUs := runtime.NumCPU()
	// sem := make(chan struct{}, noCPUs)
	// logger := utils.GetLogger()
	// results := make(chan int, len(tracks))
	// ctx := context.Background()

	for _, track := range tracks {
		trackInfo := track.buildTrack()
		//get YT id of the track
		ytID, _ := getYTID(trackInfo)
		fmt.Println("YT ID: ", ytID)

		// wg.Add(1)
		// go func(t Track) {
		// 	defer wg.Done()
		// 	sem <- struct{}{}
		// 	defer func() { <-sem }()
		// 	trackInfo := t.buildTrack()
		// 	//get YT id of the track
		// 	ytID, err := getYTID(trackInfo)
		// 	if err != nil {
		// 		logger.ErrorContext(ctx, "Failed to get YT ID", slog.Any("error", err), slog.Any("track", trackInfo))
		// 		results <- 0
		// 		return
		// 	}
		// 	//download the track from yt
		// }
	}
	return downloadedCount, nil
}

func getYTID(track *Track) (string, error) {
	getYoutubeID(*track)
	return "", nil
}
