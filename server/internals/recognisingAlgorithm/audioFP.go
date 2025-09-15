package recognisingalgorithm

import (
	"math/cmplx"

	"github.com/Pritam-deb/echo-sense/pkg"
)

// Peak represents a local maximum in the spectrogram.
type Peak struct {
	Time float64    // Time in seconds of the peak
	Freq complex128 // Frequency bin (complex value)
	Mag  float64    // Magnitude at the peak
	Bin  int        // Frequency bin index
}

// BuildConstellationMap processes the spectrogram to extract a constellation map,
// which is a set of significant peaks representing local maxima in time-frequency space.
// This map is used as the basis for generating fingerprints.
func BuildConstellationMap(spectrogram [][]complex128, audioDuration float64) []Peak {
	return ExtractPeaks(spectrogram, audioDuration)
}

// ExtractPeaks analyzes a spectrogram and extracts significant local maxima peaks in each frequency band over time.
// It collects the top N peaks per band per time bin, using local maxima detection and adaptive thresholding.
func ExtractPeaks(spectrogram [][]complex128, audioDuration float64) []Peak {
	if len(spectrogram) < 1 {
		return []Peak{}
	}

	// Define frequency bands as (min, max) bin indices (exclusive).
	bands := []struct{ min, max int }{
		{0, 10}, {10, 20}, {20, 40}, {40, 80}, {80, 160}, {160, 512},
	}
	topN := 3 // Number of top peaks to collect per band per time bin.

	var peaks []Peak
	binDuration := audioDuration / float64(len(spectrogram))

	// Helper: find local maxima in a slice of magnitudes.
	findLocalMaxima := func(mags []float64, window int) []int {
		var idxs []int
		for i := window; i < len(mags)-window; i++ {
			isMax := true
			for j := i - window; j <= i+window; j++ {
				if j == i {
					continue
				}
				if mags[j] >= mags[i] {
					isMax = false
					break
				}
			}
			if isMax {
				idxs = append(idxs, i)
			}
		}
		return idxs
	}

	for binIdx, bin := range spectrogram {
		for _, band := range bands {
			// Compute magnitudes for this band.
			bandSize := band.max - band.min
			if band.max > len(bin) {
				continue
			}
			mags := make([]float64, bandSize)
			for i := 0; i < bandSize; i++ {
				mags[i] = cmplx.Abs(bin[band.min+i])
			}
			// Find local maxima in this band.
			localMaxIdxs := findLocalMaxima(mags, 1)
			// Collect (idx, mag) pairs for sorting.
			type idxMag struct {
				idx int
				mag float64
			}
			var maxima []idxMag
			for _, idx := range localMaxIdxs {
				maxima = append(maxima, idxMag{idx: idx, mag: mags[idx]})
			}
			// Sort by descending magnitude.
			for i := 0; i < len(maxima); i++ {
				for j := i + 1; j < len(maxima); j++ {
					if maxima[j].mag > maxima[i].mag {
						maxima[i], maxima[j] = maxima[j], maxima[i]
					}
				}
			}
			// Adaptive threshold: use mean + stddev of band.
			var sum, sumSq float64
			for _, v := range mags {
				sum += v
				sumSq += v * v
			}
			mean := sum / float64(len(mags))
			std := 0.0
			if len(mags) > 1 {
				std = (sumSq/float64(len(mags)) - mean*mean)
				if std > 0 {
					std = sqrt(std)
				} else {
					std = 0
				}
			}
			threshold := mean + std*0.5
			// Take up to topN maxima above threshold.
			count := 0
			for _, m := range maxima {
				if m.mag < threshold {
					break
				}
				freqIdx := band.min + m.idx
				// Calculate the time for this peak.
				peakTime := float64(binIdx) * binDuration
				peaks = append(peaks, Peak{
					Time: peakTime,
					Freq: bin[freqIdx],
					Mag:  m.mag,
					Bin:  freqIdx,
				})
				count++
				if count >= topN {
					break
				}
			}
		}
	}

	return peaks
}

const (
	maxFreqBits    = 9  // Number of bits for quantized frequency bins
	maxDeltaBits   = 14 // Number of bits for time delta
	targetZoneSize = 5  // Number of target peaks per anchor
	freqQuant      = 2  // Quantization step for frequency bins (robustness)
)

// Fingerprint generates robust hashes from the extracted peaks.
// It uses quantized frequency bins and time deltas to create addresses for matching.
func Fingerprint(peaks []Peak, songID string) map[uint32]pkg.Couple {
	fingerprints := map[uint32]pkg.Couple{}

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]
			// Quantize frequency bins for robustness.
			anchorFreqQ := quantizeFreqBin(anchor.Bin, freqQuant)
			targetFreqQ := quantizeFreqBin(target.Bin, freqQuant)
			address := createAddressQuant(anchorFreqQ, targetFreqQ, anchor.Time, target.Time)
			anchorTimeMs := uint32(anchor.Time * 1000)
			fingerprints[address] = pkg.Couple{SongID: songID, AnchorTime: anchorTimeMs}
		}
	}
	return fingerprints
}

// quantizeFreqBin returns the quantized frequency bin index.
func quantizeFreqBin(binIdx int, step int) int {
	return (binIdx / step) * step
}

// createAddressQuant creates a robust address by combining quantized anchor/target bins and time delta.
// Layout: [anchorFreq (9 bits)][targetFreq (9 bits)][deltaMs (14 bits)]
func createAddressQuant(anchorFreq, targetFreq int, anchorTime, targetTime float64) uint32 {
	// Clamp bins to 9 bits.
	anchorFreq &= (1<<maxFreqBits - 1)
	targetFreq &= (1<<maxFreqBits - 1)
	deltaMs := uint32(targetTime-anchorTime) * 1000
	if deltaMs > (1<<maxDeltaBits - 1) {
		deltaMs = (1<<maxDeltaBits - 1)
	}
	address := (uint32(anchorFreq) << (maxFreqBits + maxDeltaBits)) |
		(uint32(targetFreq) << maxDeltaBits) |
		deltaMs
	return address
}

// sqrt is a helper for square root of float64.
func sqrt(x float64) float64 {
	// Use Newton's method for sqrt.
	z := x
	if z == 0 {
		return 0
	}
	for i := 0; i < 8; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
