package flac

import (
	"github.com/mewkiz/flac/frame"
	iobits "github.com/mewkiz/flac/internal/bits"
)

// analyseFixed selects the best fixed predictor (order 0-4) for the given
// subframe and fills the fields required by the existing writer so that a
// compressed SUBFRAME_FIXED is emitted instead of a verbatim subframe.
//
// The algorithm is a very small subset of libFLAC's encoder analysis:
//  1. For each order 0..4 compute residuals using the fixed coefficients
//     defined in frame.FixedCoeffs.
//  2. For those residuals, choose the Rice parameter k (0..14) that minimises
//     the encoded bit-length assuming partition order 0.
//  3. Pick the order with the overall fewest bits.
//
// ignoring partition orders >0 and Rice2 for now.
func analyseFixed(sf *frame.Subframe, bps uint) {
	// Guard against degenerate inputs. If there are fewer than two samples we
	// simply keep verbatim encoding.
	if len(sf.Samples) < 2 {
		return
	}

	bestBits := int(^uint(0) >> 1) // max int
	bestOrder := 0
	bestK := uint(0)

	// Try predictor orders 0 through 4.
	for order := 0; order <= 4 && order < len(sf.Samples); order++ {
		residuals := computeFixedResiduals(sf.Samples, order)
		k := chooseRice(residuals)
		bits := costFixed(order, bps, residuals, k)
		if bits < bestBits {
			bestBits = bits
			bestOrder = order
			bestK = k
		}
	}

	// Populate subframe fields so the existing encode* routines can do their
	// job. Warm-up samples are already present in sf.Samples.
	sf.Pred = frame.PredFixed
	sf.Order = bestOrder
	sf.ResidualCodingMethod = frame.ResidualCodingMethodRice1
	sf.RiceSubframe = &frame.RiceSubframe{
		PartOrder:  0,
		Partitions: []frame.RicePartition{{Param: bestK}},
	}

	// Note: We do NOT mutate sf.Samples. The encoder expects original samples
	// because it recomputes residuals internally. The metadata we filled in is
	// enough for encodeFixedSamples to reproduce the exact same residuals.
}

// computeFixedResiduals returns the residual signal for a given fixed
// predictor order. The returned slice has length len(samples)-order.
func computeFixedResiduals(samples []int32, order int) []int32 {
	n := len(samples)
	res := make([]int32, 0, n-order)

	switch order {
	case 0:
		for i := 0; i < n; i++ {
			res = append(res, samples[i])
		}
	case 1:
		for i := 1; i < n; i++ {
			res = append(res, samples[i]-samples[i-1])
		}
	case 2:
		for i := 2; i < n; i++ {
			predicted := 2*samples[i-1] - samples[i-2]
			res = append(res, samples[i]-predicted)
		}
	case 3:
		for i := 3; i < n; i++ {
			predicted := 3*samples[i-1] - 3*samples[i-2] + samples[i-3]
			res = append(res, samples[i]-predicted)
		}
	case 4:
		for i := 4; i < n; i++ {
			predicted := 4*samples[i-1] - 6*samples[i-2] + 4*samples[i-3] - samples[i-4]
			res = append(res, samples[i]-predicted)
		}
	}
	return res
}

// chooseRice returns the Rice parameter k (0..14) that minimises the encoded
// length of residuals when using Rice coding with paramSize=4 (Rice1).
func chooseRice(residuals []int32) uint {
	bestK := uint(0)
	bestBits := int(^uint(0) >> 1)

	for k := uint(0); k < 15; k++ { // 15 is escape code, so evaluate 0..14
		bits := 0
		for _, r := range residuals {
			folded := iobits.EncodeZigZag(r)
			quo := folded >> k
			bits += int(quo) + 1 + int(k) // unary + stop bit + k LSBs
		}
		if bits < bestBits {
			bestBits = bits
			bestK = k
		}
	}
	return bestK
}

// costFixed returns the number of bits needed to code the subframe with the
// given parameters. 6 bits for the subframe header are included so orders with
// more warm-up samples are fairly compared.
func costFixed(order int, bps uint, residuals []int32, k uint) int {
	warmUpBits := order * int(bps)

	// residual bits for chosen k
	residBits := 0
	for _, r := range residuals {
		folded := iobits.EncodeZigZag(r)
		quo := folded >> k
		residBits += int(quo) + 1 + int(k)
	}

	// Subframe header is 6 bits + 1 wasted flag bit (always 0 here)
	return 6 + warmUpBits + residBits
}

// analyseSubframe decides on the best prediction method (constant, verbatim, or fixed)
// for a subframe that is currently marked PredVerbatim. It will update the Subframe
// fields to use the chosen method. The heuristic is simple: it picks the encoding
// that yields the fewest estimated bits when assuming a single Rice partition.
func analyseSubframe(sf *frame.Subframe, bps uint) {
	// Only analyse when the caller has not chosen a prediction method yet.
	if sf.Pred != frame.PredVerbatim {
		return
	}

	samples := sf.Samples
	n := len(samples)
	if n == 0 {
		return
	}

	// --- Constant predictor cost.
	allEqual := true
	for i := 1; i < n; i++ {
		if samples[i] != samples[0] {
			allEqual = false
			break
		}
	}
	constBits := int(^uint(0) >> 1) // infinity
	if allEqual {
		// 6-bit header + one sample.
		constBits = 6 + int(bps)
	}

	// --- Verbatim predictor cost.
	verbatimBits := 6 + n*int(bps) // 6-bit header + raw samples

	// --- Fixed predictor: reuse existing helper to find best order/k.
	analyseFixed(sf, bps) // fills Order, RiceSubframe, etc.
	// Cost of that choice
	fixedBits := costFixed(sf.Order, bps, computeFixedResiduals(samples, sf.Order), sf.RiceSubframe.Partitions[0].Param)

	// Choose the smallest.
	if constBits < verbatimBits && constBits < fixedBits {
		// Use constant encoding.
		sf.Pred = frame.PredConstant
		// No other metadata needed.
	} else if fixedBits < verbatimBits {
		// Keep fixed settings filled in by analyseFixed.
		sf.Pred = frame.PredFixed
	} else {
		// Stick with verbatim – restore defaults that analyseFixed may have overwritten.
		sf.Pred = frame.PredVerbatim
		sf.Order = 0
		sf.RiceSubframe = nil
	}
}
