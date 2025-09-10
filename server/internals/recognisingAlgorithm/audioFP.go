package recognisingalgorithm

import (
	"math/cmplx"

	"github.com/Pritam-deb/echo-sense/pkg"
)

type Peak struct {
	Time float64
	Freq complex128
}

// ExtractPeaks analyzes a spectrogram and extracts significant peaks in the frequency domain over time.
func ExtractPeaks(spectrogram [][]complex128, audioDuration float64) []Peak {
	if len(spectrogram) < 1 {
		return []Peak{}
	}

	type maxies struct {
		maxMag  float64
		maxFreq complex128
		freqIdx int
	}

	bands := []struct{ min, max int }{{0, 10}, {10, 20}, {20, 40}, {40, 80}, {80, 160}, {160, 512}}

	var peaks []Peak
	binDuration := audioDuration / float64(len(spectrogram))

	for binIdx, bin := range spectrogram {
		var maxMags []float64
		var maxFreqs []complex128
		var freqIndices []float64

		binBandMaxies := []maxies{}
		for _, band := range bands {
			var maxx maxies
			var maxMag float64
			for idx, freq := range bin[band.min:band.max] {
				magnitude := cmplx.Abs(freq)
				if magnitude > maxMag {
					maxMag = magnitude
					freqIdx := band.min + idx
					maxx = maxies{magnitude, freq, freqIdx}
				}
			}
			binBandMaxies = append(binBandMaxies, maxx)
		}

		for _, value := range binBandMaxies {
			maxMags = append(maxMags, value.maxMag)
			maxFreqs = append(maxFreqs, value.maxFreq)
			freqIndices = append(freqIndices, float64(value.freqIdx))
		}

		var maxMagsSum float64
		for _, max := range maxMags {
			maxMagsSum += max
		}
		avg := maxMagsSum / float64(len(maxFreqs))

		for i, value := range maxMags {
			if value > avg {
				peakTimeInBin := freqIndices[i] * binDuration / float64(len(bin))
				peakTime := float64(binIdx)*binDuration + peakTimeInBin

				peaks = append(peaks, Peak{Time: peakTime, Freq: maxFreqs[i]})
			}
		}
	}

	return peaks
}

const (
	maxFreqBits    = 9
	maxDeltaBits   = 14
	targetZoneSize = 5
)

func Fingerprint(peaks []Peak, songID string) map[uint32]pkg.Couple {
	fingerprints := map[uint32]pkg.Couple{}

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			address := createAddress(anchor, target)
			anchorTimeMs := uint32(anchor.Time * 1000)

			fingerprints[address] = pkg.Couple{SongID: songID, AnchorTime: anchorTimeMs}
		}
	}

	return fingerprints
}

func createAddress(anchor, target Peak) uint32 {
	anchorFreq := int(real(anchor.Freq))
	targetFreq := int(real(target.Freq))
	deltaMs := uint32((target.Time - anchor.Time) * 1000)

	address := uint32(anchorFreq<<23) | uint32(targetFreq<<14) | deltaMs

	return address
}
