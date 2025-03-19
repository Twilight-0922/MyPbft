package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
)

func (fs *FFTSettings) simpleFTG1(vals []kzg_bls.G1Point, valsOffset uint64, valsStride uint64, rootsOfUnity []kzg_bls.Fr, rootsOfUnityStride uint64, out []kzg_bls.G1Point) {
	l := uint64(len(out))
	var v kzg_bls.G1Point
	var tmp kzg_bls.G1Point
	var last kzg_bls.G1Point
	for i := uint64(0); i < l; i++ {
		jv := &vals[valsOffset]
		r := &rootsOfUnity[0]
		kzg_bls.MulG1(&v, jv, r)
		kzg_bls.CopyG1(&last, &v)

		for j := uint64(1); j < l; j++ {
			jv := &vals[valsOffset+j*valsStride]
			r := &rootsOfUnity[((i*j)%l)*rootsOfUnityStride]
			kzg_bls.MulG1(&v, jv, r)
			kzg_bls.CopyG1(&tmp, &last)
			kzg_bls.AddG1(&last, &tmp, &v)
		}
		kzg_bls.CopyG1(&out[i], &last)
	}
}

func (fs *FFTSettings) _fftG1(vals []kzg_bls.G1Point, valsOffset uint64, valsStride uint64, rootsOfUnity []kzg_bls.Fr, rootsOfUnityStride uint64, out []kzg_bls.G1Point) {
	if len(out) <= 4 { // if the value count is small, run the unoptimized version instead. // TODO tune threshold. (can be different for G1)
		fs.simpleFTG1(vals, valsOffset, valsStride, rootsOfUnity, rootsOfUnityStride, out)
		return
	}

	half := uint64(len(out)) >> 1
	// L will be the left half of out
	fs._fftG1(vals, valsOffset, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[:half])
	// R will be the right half of out
	fs._fftG1(vals, valsOffset+valsStride, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[half:]) // just take even again

	var yTimesRoot kzg_bls.G1Point
	var x, y kzg_bls.G1Point
	for i := uint64(0); i < half; i++ {
		// temporary copies, so that writing to output doesn't conflict with input
		kzg_bls.CopyG1(&x, &out[i])
		kzg_bls.CopyG1(&y, &out[i+half])
		root := &rootsOfUnity[i*rootsOfUnityStride]
		kzg_bls.MulG1(&yTimesRoot, &y, root)
		kzg_bls.AddG1(&out[i], &x, &yTimesRoot)
		kzg_bls.SubG1(&out[i+half], &x, &yTimesRoot)
	}
}

func (fs *FFTSettings) FFTG1(vals []kzg_bls.G1Point, inv bool) ([]kzg_bls.G1Point, error) {
	n := uint64(len(vals))
	if n > fs.MaxWidth {
		return nil, fmt.Errorf("got %d values but only have %d roots of unity", n, fs.MaxWidth)
	}
	if !kzg_bls.IsPowerOfTwo(n) {
		return nil, fmt.Errorf("got %d values but not a power of two", n)
	}
	// We make a copy so we can mutate it during the work.
	valsCopy := make([]kzg_bls.G1Point, n, n)
	for i := 0; i < len(vals); i++ { // TODO: maybe optimize this away, and write back to original input array?
		kzg_bls.CopyG1(&valsCopy[i], &vals[i])
	}
	if inv {
		var invLen kzg_bls.Fr
		kzg_bls.AsFr(&invLen, n)
		kzg_bls.InvModFr(&invLen, &invLen)
		rootz := fs.ReverseRootsOfUnity[:fs.MaxWidth]
		stride := fs.MaxWidth / n

		out := make([]kzg_bls.G1Point, n, n)
		fs._fftG1(valsCopy, 0, 1, rootz, stride, out)
		var tmp kzg_bls.G1Point
		for i := 0; i < len(out); i++ {
			kzg_bls.MulG1(&tmp, &out[i], &invLen)
			kzg_bls.CopyG1(&out[i], &tmp)
		}
		return out, nil
	} else {
		out := make([]kzg_bls.G1Point, n, n)
		rootz := fs.ExpandedRootsOfUnity[:fs.MaxWidth]
		stride := fs.MaxWidth / n
		// Regular FFT
		fs._fftG1(valsCopy, 0, 1, rootz, stride, out)
		return out, nil
	}
}

// rearrange G1 elements in reverse bit order. Supports 2**31 max element count.
func reverseBitOrderG1(values []kzg_bls.G1Point) {
	if len(values) > (1 << 31) {
		panic("list too large")
	}
	var tmp kzg_bls.G1Point
	reverseBitOrder(uint32(len(values)), func(i, j uint32) {
		kzg_bls.CopyG1(&tmp, &values[i])
		kzg_bls.CopyG1(&values[i], &values[j])
		kzg_bls.CopyG1(&values[j], &tmp)
	})
}
