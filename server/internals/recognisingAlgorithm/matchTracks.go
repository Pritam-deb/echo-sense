package recognisingalgorithm

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Pritam-deb/echo-sense/db"
	"github.com/Pritam-deb/echo-sense/db/models"
	"github.com/Pritam-deb/echo-sense/pkg"
	"github.com/Pritam-deb/echo-sense/utils"
)

type Match struct {
	SongID     string
	SongTitle  string
	SongArtist string
	YouTubeID  string
	Timestamp  uint32
	Score      float64
}

func FindMatches(sampleRate int, trackDuration float64, trackSample []float64) ([]Match, time.Duration, error) {
	fmt.Println("the info received here", sampleRate)
	startTime := time.Now()
	spectrogram, err := Spectrogram(trackSample, sampleRate)
	if err != nil {
		errors.New("Failed to compute spectrogram.")
		return nil, time.Since(startTime), nil
	}

	peaks := ExtractPeaks(spectrogram, trackDuration)
	fingerprints := Fingerprint(peaks, fmt.Sprint(utils.GenerateUniqueID()))
	audioFingerprints := make(map[uint32]uint32)
	for address, fp := range fingerprints {
		audioFingerprints[address] = fp.AnchorTime
	}
	matches, _, _ := MatchFingerPrints(audioFingerprints)
	return matches, time.Since(startTime), nil
}

func MatchFingerPrints(inputFP map[uint32]uint32) ([]Match, time.Duration, error) {
	start := time.Now()

	addresses := make([]uint32, 0, len(inputFP))
	for address := range inputFP {
		addresses = append(addresses, address)
	}
	audioFP, err := GetCouples(addresses)
	if err != nil {
		return nil, time.Since(start), fmt.Errorf("failed to fetch addresses from the database: %w", err)
	}

	matches := map[string][][2]uint32{}
	timeStamps := map[string]uint32{}
	targetZones := map[string]map[uint32]int{}
	for address, couples := range audioFP {
		for _, couple := range couples {
			matches[couple.SongID] = append(matches[couple.SongID], [2]uint32{inputFP[address], couple.AnchorTime})
			if timeExists, ok := timeStamps[couple.SongID]; !ok || couple.AnchorTime < timeExists {
				timeStamps[couple.SongID] = couple.AnchorTime
			}
			if _, ok := targetZones[couple.SongID]; !ok {
				targetZones[couple.SongID] = make(map[uint32]int)
			}
			targetZones[couple.SongID][couple.AnchorTime]++
		}
	}

	scores := getRelativeTiming(matches)
	results := make([]Match, 0, len(scores))
	for songID, score := range scores {
		song, err := db.GetSongByIDString(songID)
		if err != nil {
			continue
		}
		results = append(results, Match{
			SongID:     songID,
			SongTitle:  song.Title,
			SongArtist: song.Artist,
			YouTubeID:  song.YoutubeID,
			Timestamp:  timeStamps[songID],
			Score:      score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, time.Since(start), nil
}

func getRelativeTiming(matches map[string][][2]uint32) map[string]float64 {
	scores := make(map[string]float64)

	for songID, times := range matches {
		offsetCounts := make(map[int32]int)

		for _, timePair := range times {
			sampleTime := int32(timePair[0])
			dbTime := int32(timePair[1])
			offset := dbTime - sampleTime
			offsetBucket := offset / 100
			offsetCounts[offsetBucket]++
		}

		maxCount := 0
		for _, count := range offsetCounts {
			if count > maxCount {
				maxCount = count
			}
		}

		scores[songID] = float64(maxCount)
	}

	return scores
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
