package spotify

import (
	"os"
	"runtime"
	"strings"
)

func changeFileName(title, artist string) (string, string) {
	if runtime.GOOS == "windows" {
		invalidChars := []byte{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}
		for _, invalidChar := range invalidChars {
			title = strings.ReplaceAll(title, string(invalidChar), "")
			artist = strings.ReplaceAll(artist, string(invalidChar), "")
		}
	} else {
		title = strings.ReplaceAll(title, "/", "\\")
		artist = strings.ReplaceAll(artist, "/", "\\")
	}
	return title, artist
}

func GetFileSize(file string) (int64, error) {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return 0, err
	}

	size := int64(fileInfo.Size())
	return size, nil
}
