// Original: https://github.com/ethereum/research/blob/master/mimc_stark/fft.py

package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"math/bits"
)

// if not already a power of 2, return the next power of 2
func nextPowOf2(v uint64) uint64 {
	if v == 0 {
		return 1
	}
	return uint64(1) << bits.Len64(v-1)
}

// Expands the power circle for a given root of unity to WIDTH+1 values.
// The first entry will be 1, the last entry will also be 1,
// for convenience when reversing the array (useful for inverses)
func expandRootOfUnity(rootOfUnity *kzg_bls.Fr) []kzg_bls.Fr {
	rootz := make([]kzg_bls.Fr, 2)
	rootz[0] = kzg_bls.ONE // some unused number in py code
	rootz[1] = *rootOfUnity
	for i := 1; !kzg_bls.EqualOne(&rootz[i]); {
		rootz = append(rootz, kzg_bls.Fr{})
		this := &rootz[i]
		i++
		kzg_bls.MulModFr(&rootz[i], this, rootOfUnity)
	}
	return rootz
}

type FFTSettings struct {
	MaxWidth uint64
	// the generator used to get all roots of unity
	RootOfUnity *kzg_bls.Fr
	// domain, starting and ending with 1 (duplicate!)
	ExpandedRootsOfUnity []kzg_bls.Fr
	// reverse domain, same as inverse values of domain. Also starting and ending with 1.
	ReverseRootsOfUnity []kzg_bls.Fr
}

func NewFFTSettings(maxScale uint8) *FFTSettings {
	width := uint64(1) << maxScale
	root := &kzg_bls.Scale2RootOfUnity[maxScale]
	rootz := expandRootOfUnity(&kzg_bls.Scale2RootOfUnity[maxScale])
	// reverse roots of unity
	rootzReverse := make([]kzg_bls.Fr, len(rootz), len(rootz))
	copy(rootzReverse, rootz)
	for i, j := uint64(0), uint64(len(rootz)-1); i < j; i, j = i+1, j-1 {
		rootzReverse[i], rootzReverse[j] = rootzReverse[j], rootzReverse[i]
	}

	return &FFTSettings{
		MaxWidth:             width,
		RootOfUnity:          root,
		ExpandedRootsOfUnity: rootz,
		ReverseRootsOfUnity:  rootzReverse,
	}
}
