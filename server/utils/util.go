package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateSongKey(artist, title string) string {
	return fmt.Sprintf("%s-%s", artist, title)
}

func GenerateUniqueID() uint32 {
	rand.NewSource(time.Now().UnixNano())
	randomNumber := rand.Uint32()

	return randomNumber
}
