package recognisingalgorithm

// import (
// 	"math"
// 	"math/cmplx"
// )

// type Peak struct {
// 	Time float64
// 	Freq float64
// }

// func hzToMel(hz float64) float64 {
// 	return 2595 * math.Log10(1+hz/700)
// }

// func melToHz(mel float64) float64 {
// 	return 700 * (math.Pow(10, mel/2595) - 1)
// }

// // ExtractPeaks analyzes a spectrogram and extracts significant peaks in the frequency domain over time.
// func ExtractPeaks(spectrogram [][]complex128, audioDuration float64, sampleRate int) []Peak {
// 	if len(spectrogram) < 1 {
// 		return []Peak{}
// 	}

// 	var peaks []Peak
// 	binDuration := audioDuration / float64(len(spectrogram))
// 	nFreqBins := len(spectrogram[0])

// 	// Define number of mel bands
// 	nMelBands := 40

// 	// Frequency range in Hz
// 	fMin := 0.0
// 	fMax := float64(sampleRate) / 2

// 	// Compute mel scale boundaries for bands
// 	melMin := hzToMel(fMin)
// 	melMax := hzToMel(fMax)
// 	melBandEdges := make([]float64, nMelBands+1)
// 	for i := 0; i <= nMelBands; i++ {
// 		melBandEdges[i] = melMin + (float64(i)/float64(nMelBands))*(melMax-melMin)
// 	}

// 	// Convert mel band edges back to Hz
// 	freqBandEdgesHz := make([]float64, nMelBands+1)
// 	for i := 0; i <= nMelBands; i++ {
// 		freqBandEdgesHz[i] = melToHz(melBandEdges[i])
// 	}

// 	// Convert freqBandEdgesHz to corresponding bin indices
// 	binEdges := make([]int, nMelBands+1)
// 	for i := 0; i <= nMelBands; i++ {
// 		// bin index = freq * (nFreqBins*2) / sampleRate
// 		bin := int(freqBandEdgesHz[i] * float64(nFreqBins*2) / float64(sampleRate))
// 		if bin < 0 {
// 			bin = 0
// 		}
// 		if bin > nFreqBins-1 {
// 			bin = nFreqBins - 1
// 		}
// 		binEdges[i] = bin
// 	}

// 	for binIdx, bin := range spectrogram {
// 		magnitudes := make([]float64, nFreqBins)
// 		for i, freq := range bin {
// 			magnitudes[i] = cmplx.Abs(freq)
// 		}

// 		// Calculate mean and standard deviation for dynamic threshold
// 		var sum float64
// 		for _, mag := range magnitudes {
// 			sum += mag
// 		}
// 		mean := sum / float64(nFreqBins)

// 		var varianceSum float64
// 		for _, mag := range magnitudes {
// 			diff := mag - mean
// 			varianceSum += diff * diff
// 		}
// 		stdDev := math.Sqrt(varianceSum / float64(nFreqBins))

// 		threshold := mean + stdDev

// 		// For each mel band, find local maxima above threshold
// 		for band := 0; band < nMelBands; band++ {
// 			startBin := binEdges[band]
// 			endBin := binEdges[band+1]

// 			if endBin <= startBin {
// 				continue
// 			}

// 			// Find max magnitude and its index in this mel band
// 			maxMag := 0.0
// 			maxIdx := -1
// 			for i := startBin; i <= endBin; i++ {
// 				if magnitudes[i] > maxMag {
// 					maxMag = magnitudes[i]
// 					maxIdx = i
// 				}
// 			}

// 			if maxIdx == -1 || maxIdx == 0 || maxIdx == nFreqBins-1 {
// 				continue
// 			}

// 			// Check if local maximum and above threshold
// 			if magnitudes[maxIdx] > magnitudes[maxIdx-1] && magnitudes[maxIdx] > magnitudes[maxIdx+1] && magnitudes[maxIdx] > threshold {
// 				// Convert bin index to frequency in Hz
// 				freqHz := float64(maxIdx) * float64(sampleRate) / float64(nFreqBins*2)

// 				peakTime := float64(binIdx) * binDuration

// 				peaks = append(peaks, Peak{Time: peakTime, Freq: freqHz})
// 			}
// 		}
// 	}

// 	return peaks
// }

// // PeakPair represents a pair of peaks within a time window.
// type PeakPair struct {
// 	Time1 float64
// 	Time2 float64
// 	Freq1 float64
// 	Freq2 float64
// }

// // BuildConstellationMap builds a list of PeakPairs from peaks within the given time window.
// func BuildConstellationMap(peaks []Peak, timeWindow float64) []PeakPair {
// 	var pairs []PeakPair
// 	n := len(peaks)
// 	for i := 0; i < n; i++ {
// 		for j := i + 1; j < n; j++ {
// 			if peaks[j].Time-peaks[i].Time > timeWindow {
// 				break
// 			}
// 			if peaks[j].Time > peaks[i].Time && peaks[j].Time-peaks[i].Time <= timeWindow {
// 				pair := PeakPair{
// 					Time1: peaks[i].Time,
// 					Time2: peaks[j].Time,
// 					Freq1: peaks[i].Freq,
// 					Freq2: peaks[j].Freq,
// 				}
// 				pairs = append(pairs, pair)
// 			}
// 		}
// 	}
// 	return pairs
// }

// // Fingerprint represents a compact hash of a peak pair with associated reference time.
// type Fingerprint struct {
// 	Hash uint64
// 	Time float64
// }

// // GenerateFingerprints creates fingerprints from peak pairs by hashing quantized frequencies and time delta.
// func GenerateFingerprints(pairs []PeakPair) []Fingerprint {
// 	var fingerprints []Fingerprint

// 	for _, pair := range pairs {
// 		// Quantize frequencies to nearest 10 Hz
// 		qFreq1 := uint64(math.Round(pair.Freq1/10) * 10)
// 		qFreq2 := uint64(math.Round(pair.Freq2/10) * 10)

// 		// Quantize time delta to nearest 0.01 seconds (10 ms)
// 		timeDelta := pair.Time2 - pair.Time1
// 		qTimeDelta := uint64(math.Round(timeDelta / 0.01))

// 		// Combine quantized values into a single 64-bit hash
// 		// For example: 20 bits for freq1, 20 bits for freq2, 24 bits for time delta
// 		// Shift and combine accordingly
// 		hash := (qFreq1 & 0xFFFFF) << 44 // top 20 bits
// 		hash |= (qFreq2 & 0xFFFFF) << 24 // middle 20 bits
// 		hash |= (qTimeDelta & 0xFFFFFF)  // lower 24 bits

// 		fingerprints = append(fingerprints, Fingerprint{
// 			Hash: hash,
// 			Time: pair.Time1,
// 		})
// 	}

// 	return fingerprints
// }
