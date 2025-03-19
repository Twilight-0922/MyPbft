package KZG_ZKSNARK

import "IBEUser/src/KZG_ZKSNARK/kzg_bls"

// GenerateTestingSetup creates a setup of n values from the given secret. **for testing purposes only**
func GenerateTestingSetup(secret string, n uint64) ([]kzg_bls.G1Point, []kzg_bls.G2Point) {
	var s kzg_bls.Fr
	kzg_bls.SetFr(&s, secret)

	var sPow kzg_bls.Fr
	kzg_bls.CopyFr(&sPow, &kzg_bls.ONE)

	s1Out := make([]kzg_bls.G1Point, n, n)
	s2Out := make([]kzg_bls.G2Point, n, n)
	for i := uint64(0); i < n; i++ {
		kzg_bls.MulG1(&s1Out[i], &kzg_bls.GenG1, &sPow)
		kzg_bls.MulG2(&s2Out[i], &kzg_bls.GenG2, &sPow)
		var tmp kzg_bls.Fr
		kzg_bls.CopyFr(&tmp, &sPow)
		kzg_bls.MulModFr(&sPow, &tmp, &s)
	}
	return s1Out, s2Out
}
