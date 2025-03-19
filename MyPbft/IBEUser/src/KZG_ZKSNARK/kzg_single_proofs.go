// Original: https://github.com/ethereum/research/blob/master/kzg_data_availability/kzg_proofs.py

package KZG_ZKSNARK

import "IBEUser/src/KZG_ZKSNARK/kzg_bls"

// KZG commitment to polynomial in evaluation form, i.e. eval = FFT(coeffs).
// The eval length must match the prepared KZG settings width.
func CommitToEvalPoly(secretG1IFFT []kzg_bls.G1Point, eval []kzg_bls.Fr) *kzg_bls.G1Point {
	return kzg_bls.LinCombG1(secretG1IFFT, eval)
}

// KZG commitment to polynomial in coefficient form
func (ks *KZGSettings) CommitToPoly(coeffs []kzg_bls.Fr) *kzg_bls.G1Point {
	return kzg_bls.LinCombG1(ks.SecretG1[:len(coeffs)], coeffs)
}

// KZG commitment to polynomial in coefficient form
func (ks *KZGSettings) CommitToPolyG2(coeffs []kzg_bls.Fr) *kzg_bls.G2Point {
	return kzg_bls.LinCombG2(ks.SecretG2[:len(coeffs)], coeffs)
}

// KZG commitment to polynomial in coefficient form, unoptimized version
func (ks *KZGSettings) CommitToPolyUnoptimized(coeffs []kzg_bls.Fr) *kzg_bls.G1Point {
	// Do so by computing the linear combination with the shared secret.
	var out kzg_bls.G1Point
	kzg_bls.ClearG1(&out)
	var tmp, tmp2 kzg_bls.G1Point
	for i := 0; i < len(coeffs); i++ {
		kzg_bls.MulG1(&tmp, &ks.SecretG1[i], &coeffs[i])
		kzg_bls.AddG1(&tmp2, &out, &tmp)
		kzg_bls.CopyG1(&out, &tmp2)
	}
	return &out
}

// Compute KZG proof for polynomial in coefficient form at position x
func (ks *KZGSettings) ComputeProofSingle(poly []kzg_bls.Fr, x uint64) *kzg_bls.G1Point {
	// divisor = [-x, 1]
	divisor := [2]kzg_bls.Fr{}
	var tmp kzg_bls.Fr
	kzg_bls.AsFr(&tmp, x)
	kzg_bls.SubModFr(&divisor[0], &kzg_bls.ZERO, &tmp)
	kzg_bls.CopyFr(&divisor[1], &kzg_bls.ONE)
	//for i := 0; i < 2; i++ {
	//	fmt.Printf("div poly %d: %s\n", i, FrStr(&divisor[i]))
	//}
	// quot = poly / divisor
	quotientPolynomial := polyLongDiv(poly, divisor[:])
	//for i := 0; i < len(quotientPolynomial); i++ {
	//	fmt.Printf("quot poly %d: %s\n", i, FrStr(&quotientPolynomial[i]))
	//}

	// evaluate quotient poly at shared secret, in G1
	return kzg_bls.LinCombG1(ks.SecretG1[:len(quotientPolynomial)], quotientPolynomial)
}

func (ks *KZGSettings) ComputeProofSingleG2(poly []kzg_bls.Fr, x uint64) *kzg_bls.G2Point {
	// divisor = [-x, 1]
	divisor := [2]kzg_bls.Fr{}
	var tmp kzg_bls.Fr
	kzg_bls.AsFr(&tmp, x)
	kzg_bls.SubModFr(&divisor[0], &kzg_bls.ZERO, &tmp)
	kzg_bls.CopyFr(&divisor[1], &kzg_bls.ONE)
	//for i := 0; i < 2; i++ {
	//	fmt.Printf("div poly %d: %s\n", i, FrStr(&divisor[i]))
	//}
	// quot = poly / divisor
	quotientPolynomial := polyLongDiv(poly, divisor[:])
	//for i := 0; i < len(quotientPolynomial); i++ {
	//	fmt.Printf("quot poly %d: %s\n", i, FrStr(&quotientPolynomial[i]))
	//}

	// evaluate quotient poly at shared secret, in G1
	return kzg_bls.LinCombG2(ks.SecretG2[:len(quotientPolynomial)], quotientPolynomial)
}

// Check a proof for a KZG commitment for an evaluation f(x) = y
func (ks *KZGSettings) CheckProofSingle(commitment *kzg_bls.G1Point, proof *kzg_bls.G1Point, x *kzg_bls.Fr, y *kzg_bls.Fr) bool {
	// Verify the pairing equation
	var xG2 kzg_bls.G2Point
	kzg_bls.MulG2(&xG2, &kzg_bls.GenG2, x)
	var sMinuxX kzg_bls.G2Point
	kzg_bls.SubG2(&sMinuxX, &ks.SecretG2[1], &xG2)
	var yG1 kzg_bls.G1Point
	kzg_bls.MulG1(&yG1, &kzg_bls.GenG1, y)
	var commitmentMinusY kzg_bls.G1Point
	kzg_bls.SubG1(&commitmentMinusY, commitment, &yG1)

	// This trick may be applied in the kzg_bls-lib specific code:
	//
	// e([commitment - y], [1]) = e([proof],  [s - x])
	//    equivalent to
	// e([commitment - y]^(-1), [1]) * e([proof],  [s - x]) = 1_T
	//
	return kzg_bls.PairingsVerify(&commitmentMinusY, &kzg_bls.GenG2, proof, &sMinuxX)
}

func (ks *KZGSettings) CheckProofSingleG1(commitment *kzg_bls.G1Point, proof *kzg_bls.G1Point, x *kzg_bls.Fr, yG1 *kzg_bls.G1Point) bool {
	// Verify the pairing equation
	var xG2 kzg_bls.G2Point
	kzg_bls.MulG2(&xG2, &kzg_bls.GenG2, x)
	var sMinuxX kzg_bls.G2Point
	kzg_bls.SubG2(&sMinuxX, &ks.SecretG2[1], &xG2)
	var commitmentMinusY kzg_bls.G1Point
	kzg_bls.SubG1(&commitmentMinusY, commitment, yG1)

	// This trick may be applied in the kzg_bls-lib specific code:
	//
	// e([commitment - y], [1]) = e([proof],  [s - x])
	//    equivalent to
	// e([commitment - y]^(-1), [1]) * e([proof],  [s - x]) = 1_T
	//
	return kzg_bls.PairingsVerify(&commitmentMinusY, &kzg_bls.GenG2, proof, &sMinuxX)
}
