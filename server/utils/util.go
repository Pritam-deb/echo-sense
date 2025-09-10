package utils

import "fmt"

func GenerateSongKey(artist, title string) string {
	return fmt.Sprintf("%s-%s", artist, title)
}
