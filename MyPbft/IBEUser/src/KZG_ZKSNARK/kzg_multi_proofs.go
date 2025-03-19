// Original: https://github.com/ethereum/research/blob/master/kzg_data_availability/kzg_proofs.py

package KZG_ZKSNARK

import "IBEUser/src/KZG_ZKSNARK/kzg_bls"

// Compute KZG proof for polynomial in coefficient form at positions x * w^y where w is
// an n-th root of unity (this is the proof for one data availability sample, which consists
// of several polynomial evaluations)
func (ks *KZGSettings) ComputeProofMulti(poly []kzg_bls.Fr, x uint64, n uint64) *kzg_bls.G1Point {
	// divisor = [-pow(x, n, MODULUS)] + [0] * (n - 1) + [1]
	divisor := make([]kzg_bls.Fr, n+1, n+1)
	var xFr kzg_bls.Fr
	kzg_bls.AsFr(&xFr, x)
	// TODO: inefficient, could use squaring, or maybe kzg_bls lib offers a power method?
	// TODO: for small ranges, maybe compute pow(x, n, mod) in uint64?
	var xPowN, tmp kzg_bls.Fr
	for i := uint64(0); i < n; i++ {
		kzg_bls.MulModFr(&tmp, &xPowN, &xFr)
		kzg_bls.CopyFr(&xPowN, &tmp)
	}

	// -pow(x, n, MODULUS)
	kzg_bls.SubModFr(&divisor[0], &kzg_bls.ZERO, &xPowN)
	// [0] * (n - 1)
	for i := uint64(1); i < n; i++ {
		kzg_bls.CopyFr(&divisor[i], &kzg_bls.ZERO)
	}
	// 1
	kzg_bls.CopyFr(&divisor[n], &kzg_bls.ONE)

	// quot = poly / divisor
	quotientPolynomial := polyLongDiv(poly, divisor[:])
	//for i := 0; i < len(quotientPolynomial); i++ {
	//	fmt.Printf("quot poly %d: %s\n", i, FrStr(&quotientPolynomial[i]))
	//}

	// evaluate quotient poly at shared secret, in G1
	return kzg_bls.LinCombG1(ks.SecretG1[:len(quotientPolynomial)], quotientPolynomial)
}

// Check a proof for a KZG commitment for an evaluation f(x w^i) = y_i
// The ys must have a power of 2 length
func (ks *KZGSettings) CheckProofMulti(commitment *kzg_bls.G1Point, proof *kzg_bls.G1Point, x *kzg_bls.Fr, ys []kzg_bls.Fr) bool {
	// Interpolate at a coset. Note because it is a coset, not the subgroup, we have to multiply the
	// polynomial coefficients by x^i
	interpolationPoly, err := ks.FFT(ys, true)
	if err != nil {
		panic("ys is bad, cannot compute FFT")
	}
	// TODO: can probably be optimized
	// apply div(c, pow(x, i, MODULUS)) to every coeff c in interpolationPoly
	// x^0 at first, then up to x^n
	var xPow kzg_bls.Fr
	kzg_bls.CopyFr(&xPow, &kzg_bls.ONE)
	var tmp, tmp2 kzg_bls.Fr
	for i := 0; i < len(interpolationPoly); i++ {
		kzg_bls.InvModFr(&tmp, &xPow)
		kzg_bls.MulModFr(&tmp2, &interpolationPoly[i], &tmp)
		kzg_bls.CopyFr(&interpolationPoly[i], &tmp2)
		kzg_bls.MulModFr(&tmp, &xPow, x)
		kzg_bls.CopyFr(&xPow, &tmp)
	}
	// [x^n]_2
	var xn2 kzg_bls.G2Point
	kzg_bls.MulG2(&xn2, &kzg_bls.GenG2, &xPow)
	// [s^n - x^n]_2
	var xnMinusYn kzg_bls.G2Point
	kzg_bls.SubG2(&xnMinusYn, &ks.SecretG2[len(ys)], &xn2)

	// [interpolation_polynomial(s)]_1
	is1 := kzg_bls.LinCombG1(ks.SecretG1[:len(interpolationPoly)], interpolationPoly)
	// [commitment - interpolation_polynomial(s)]_1 = [commit]_1 - [interpolation_polynomial(s)]_1
	var commitMinusInterpolation kzg_bls.G1Point
	kzg_bls.SubG1(&commitMinusInterpolation, commitment, is1)

	// Verify the pairing equation
	//
	// e([commitment - interpolation_polynomial(s)], [1]) = e([proof],  [s^n - x^n])
	//    equivalent to
	// e([commitment - interpolation_polynomial]^(-1), [1]) * e([proof],  [s^n - x^n]) = 1_T
	//

	return kzg_bls.PairingsVerify(&commitMinusInterpolation, &kzg_bls.GenG2, proof, &xnMinusYn)
}
