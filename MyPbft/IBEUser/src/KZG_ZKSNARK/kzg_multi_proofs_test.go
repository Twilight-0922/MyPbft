package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
	"testing"
)

func TestKZGSettings_CheckProofMulti(t *testing.T) {
	fs := NewFFTSettings(4)
	s1, s2 := GenerateTestingSetup("1927409816240961209460912649124", 16+1)
	ks := NewKZGSettings(fs, s1, s2)
	for i := 0; i < len(ks.SecretG1); i++ {
		t.Logf("secret g1 %d: %s", i, kzg_bls.StrG1(&ks.SecretG1[i]))
	}

	polynomial := testPoly(1, 2, 3, 4, 7, 7, 7, 7, 13, 13, 13, 13, 13, 13, 13, 13)
	for i := 0; i < len(polynomial); i++ {
		t.Logf("poly coeff %d: %s", i, kzg_bls.FrStr(&polynomial[i]))
	}

	commitment := ks.CommitToPoly(polynomial)
	t.Log("commitment\n", kzg_bls.StrG1(commitment))

	x := uint64(5431)
	var xFr kzg_bls.Fr
	kzg_bls.AsFr(&xFr, x)
	cosetScale := uint8(3)
	coset := make([]kzg_bls.Fr, 1<<cosetScale, 1<<cosetScale)
	s1, s2 = GenerateTestingSetup("1927409816240961209460912649124", 8+1)
	ks = NewKZGSettings(NewFFTSettings(cosetScale), s1, s2)
	for i := 0; i < len(coset); i++ {
		fmt.Printf("rootz %d: %s\n", i, kzg_bls.FrStr(&ks.ExpandedRootsOfUnity[i]))
		kzg_bls.MulModFr(&coset[i], &xFr, &ks.ExpandedRootsOfUnity[i])
		fmt.Printf("coset %d: %s\n", i, kzg_bls.FrStr(&coset[i]))
	}
	ys := make([]kzg_bls.Fr, len(coset), len(coset))
	for i := 0; i < len(coset); i++ {
		kzg_bls.EvalPolyAt(&ys[i], polynomial, &coset[i])
		fmt.Printf("ys %d: %s\n", i, kzg_bls.FrStr(&ys[i]))
	}

	proof := ks.ComputeProofMulti(polynomial, x, uint64(len(coset)))
	fmt.Printf("proof: %s\n", kzg_bls.StrG1(proof))
	if !ks.CheckProofMulti(commitment, proof, &xFr, ys) {
		t.Fatal("could not verify proof")
	}
}
