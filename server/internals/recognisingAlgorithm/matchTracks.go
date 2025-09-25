package recognisingalgorithm

import (
	"errors"
	"fmt"

	"github.com/Pritam-deb/echo-sense/db"
	"github.com/Pritam-deb/echo-sense/db/models"
	"github.com/Pritam-deb/echo-sense/pkg"
	"github.com/Pritam-deb/echo-sense/utils"
)

func FindMatches(sampleRate int, trackDuration float64, trackSample []float64) {
	fmt.Println("the info received here", sampleRate)
	spectrogram, err := Spectrogram(trackSample, sampleRate)
	if err != nil {
		errors.New("Failed to compute spectrogram.")
		return
	}

	peaks := ExtractPeaks(spectrogram, trackDuration)
	fingerprints := Fingerprint(peaks, fmt.Sprint(utils.GenerateUniqueID()))
	audioFingerprints := make(map[uint32]uint32)
	for address, fp := range fingerprints {
		audioFingerprints[address] = fp.AnchorTime
	}
	MatchFingerPrints(audioFingerprints)
}

func MatchFingerPrints(inputFP map[uint32]uint32) error {

	addresses := make([]uint32, 0, len(inputFP))
	for address := range inputFP {
		addresses = append(addresses, address)
	}
	couples, err := GetCouples(addresses)
	if err != nil {
		fmt.Errorf("failed to fetch address from the database.")
	}
	fmt.Println("number of address matches are: ", len(couples))
	return nil
}

func GetCouples(fpAddress []uint32) (map[uint32][]pkg.Couple, error) {
	var fingerPrints []models.AudioFingerprint
	if err := db.DB.Where("address IN ?", fpAddress).Find(&fingerPrints).Error; err != nil {
		return nil, err
	}
	couples := make(map[uint32][]pkg.Couple)
	for _, fp := range fingerPrints {
		couples[uint32(fp.Address)] = append(couples[uint32(fp.Address)], pkg.Couple{
			SongID:     fp.SongID.String(),
			AnchorTime: uint32(fp.AnchorTime),
		})
	}
	return couples, nil
}
