package spotify

import "github.com/Pritam-deb/echo-sense/utils"

func DownloadSingleTrack(url string, downloadPath string) {
	logger := utils.GetLogger()
	logger.Info("Starting download for single track", "url", url, "path", downloadPath)
	GetTrackInfo(url)
}
