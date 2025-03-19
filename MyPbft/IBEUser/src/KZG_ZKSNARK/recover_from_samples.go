package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
)

// unshift poly, in-place. Multiplies each coeff with 1/shift_factor**i
func (fs *FFTSettings) ShiftPoly(poly []kzg_bls.Fr) {
	var shiftFactor kzg_bls.Fr
	kzg_bls.AsFr(&shiftFactor, 5) // primitive root of unity
	var factorPower kzg_bls.Fr
	kzg_bls.CopyFr(&factorPower, &kzg_bls.ONE)
	var invFactor kzg_bls.Fr
	kzg_bls.InvModFr(&invFactor, &shiftFactor)
	var tmp kzg_bls.Fr
	for i := 0; i < len(poly); i++ {
		kzg_bls.CopyFr(&tmp, &poly[i])
		kzg_bls.MulModFr(&poly[i], &tmp, &factorPower)
		// TODO: pre-compute all these shift scalars
		kzg_bls.CopyFr(&tmp, &factorPower)
		kzg_bls.MulModFr(&factorPower, &tmp, &invFactor)
	}
}

// unshift poly, in-place. Multiplies each coeff with shift_factor**i
func (fs *FFTSettings) UnshiftPoly(poly []kzg_bls.Fr) {
	var shiftFactor kzg_bls.Fr
	kzg_bls.AsFr(&shiftFactor, 5) // primitive root of unity
	var factorPower kzg_bls.Fr
	kzg_bls.CopyFr(&factorPower, &kzg_bls.ONE)
	var tmp kzg_bls.Fr
	for i := 0; i < len(poly); i++ {
		kzg_bls.CopyFr(&tmp, &poly[i])
		kzg_bls.MulModFr(&poly[i], &tmp, &factorPower)
		// TODO: pre-compute all these shift scalars
		kzg_bls.CopyFr(&tmp, &factorPower)
		kzg_bls.MulModFr(&factorPower, &tmp, &shiftFactor)
	}
}

func (fs *FFTSettings) RecoverPolyFromSamples(samples []*kzg_bls.Fr, zeroPolyFn ZeroPolyFn) ([]kzg_bls.Fr, error) {
	// TODO: using a single additional temporary array, all the FFTs can run in-place.

	missingIndices := make([]uint64, 0, len(samples))
	for i, s := range samples {
		if s == nil {
			missingIndices = append(missingIndices, uint64(i))
		}
	}

	zeroEval, zeroPoly := zeroPolyFn(missingIndices, uint64(len(samples)))

	for i, s := range samples {
		if (s == nil) != kzg_bls.EqualZero(&zeroEval[i]) {
			panic("bad zero eval")
		}
	}

	polyEvaluationsWithZero := make([]kzg_bls.Fr, len(samples), len(samples))
	for i, s := range samples {
		if s == nil {
			kzg_bls.CopyFr(&polyEvaluationsWithZero[i], &kzg_bls.ZERO)
		} else {
			kzg_bls.MulModFr(&polyEvaluationsWithZero[i], s, &zeroEval[i])
		}
	}
	polyWithZero, err := fs.FFT(polyEvaluationsWithZero, true)
	if err != nil {
		return nil, err
	}
	// shift in-place
	fs.ShiftPoly(polyWithZero)
	shiftedPolyWithZero := polyWithZero

	fs.ShiftPoly(zeroPoly)
	shiftedZeroPoly := zeroPoly

	evalShiftedPolyWithZero, err := fs.FFT(shiftedPolyWithZero, false)
	if err != nil {
		return nil, err
	}
	evalShiftedZeroPoly, err := fs.FFT(shiftedZeroPoly, false)
	if err != nil {
		return nil, err
	}

	evalShiftedReconstructedPoly := evalShiftedPolyWithZero
	for i := 0; i < len(evalShiftedReconstructedPoly); i++ {
		kzg_bls.DivModFr(&evalShiftedReconstructedPoly[i], &evalShiftedPolyWithZero[i], &evalShiftedZeroPoly[i])
	}
	shiftedReconstructedPoly, err := fs.FFT(evalShiftedReconstructedPoly, true)
	if err != nil {
		return nil, err
	}
	fs.UnshiftPoly(shiftedReconstructedPoly)
	reconstructedPoly := shiftedReconstructedPoly

	reconstructedData, err := fs.FFT(reconstructedPoly, false)
	if err != nil {
		return nil, err
	}
	for i, s := range samples {
		if s != nil && !kzg_bls.EqualFr(&reconstructedData[i], s) {
			return nil, fmt.Errorf("failed to reconstruct data correctly, changed value at index %d. Expected: %s, got: %s", i, kzg_bls.FrStr(s), kzg_bls.FrStr(&reconstructedData[i]))
		}
	}
	return reconstructedData, nil
}
