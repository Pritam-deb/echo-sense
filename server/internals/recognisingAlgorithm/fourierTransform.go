package recognisingalgorithm

import (
	"math"
	"math/cmplx"
)

func fftRealToComplex(x []float64) []complex128 {
	N := len(x)
	// Convert real input to complex
	c := make([]complex128, N)
	for i := range x {
		c[i] = complex(x[i], 0)
	}

	return fft(c)
}

func fft(x []complex128) []complex128 {
	N := len(x)
	if N <= 1 {
		return x
	}

	// Split even and odd
	even := make([]complex128, N/2)
	odd := make([]complex128, N/2)
	for i := 0; i < N/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	// Recursive FFT
	Feven := fft(even)
	Fodd := fft(odd)

	// Combine
	combined := make([]complex128, N)
	for k := 0; k < N/2; k++ {
		twiddle := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(N)))
		combined[k] = Feven[k] + twiddle*Fodd[k]
		combined[k+N/2] = Feven[k] - twiddle*Fodd[k]
	}
	return combined
}
