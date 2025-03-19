package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"testing"
)

func TestKZGSettings_DAUsingFK20(t *testing.T) {
	fs := NewFFTSettings(5)
	s1, s2 := GenerateTestingSetup("1927409816240961209460912649124", 32+1)
	ks := NewKZGSettings(fs, s1, s2)
	fk := NewFK20SingleSettings(ks, 32)

	polynomial := testPoly(1, 2, 3, 4, 7, 7, 7, 7, 13, 13, 13, 13, 13, 13, 13, 13)

	commitment := ks.CommitToPoly(polynomial)
	t.Log("commitment\n", kzg_bls.StrG1(commitment))

	allProofs := fk.DAUsingFK20(polynomial)
	t.Log("All KZG proofs computed")
	for i := 0; i < len(allProofs); i++ {
		t.Logf("%d: %s", i, kzg_bls.StrG1(&allProofs[i]))
	}

	// Now check a random position
	pos := uint64(9)
	var posFr kzg_bls.Fr
	kzg_bls.AsFr(&posFr, pos)
	var x kzg_bls.Fr
	kzg_bls.CopyFr(&x, &ks.ExpandedRootsOfUnity[pos])
	t.Log("x:\n", kzg_bls.FrStr(&x))
	var y kzg_bls.Fr
	kzg_bls.EvalPolyAt(&y, polynomial, &x)
	t.Log("y:\n", kzg_bls.FrStr(&y))

	proof := &allProofs[reverseBitsLimited(uint32(2*16), uint32(pos))]

	if !ks.CheckProofSingle(commitment, proof, &x, &y) {
		t.Fatal("could not verify proof")
	}
}
