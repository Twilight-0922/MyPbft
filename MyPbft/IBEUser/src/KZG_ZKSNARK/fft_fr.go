package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
)

func (fs *FFTSettings) simpleFT(vals []kzg_bls.Fr, valsOffset uint64, valsStride uint64, rootsOfUnity []kzg_bls.Fr, rootsOfUnityStride uint64, out []kzg_bls.Fr) {
	l := uint64(len(out))
	var v kzg_bls.Fr
	var tmp kzg_bls.Fr
	var last kzg_bls.Fr
	for i := uint64(0); i < l; i++ {
		jv := &vals[valsOffset]
		r := &rootsOfUnity[0]
		kzg_bls.MulModFr(&v, jv, r)
		kzg_bls.CopyFr(&last, &v)

		for j := uint64(1); j < l; j++ {
			jv := &vals[valsOffset+j*valsStride]
			r := &rootsOfUnity[((i*j)%l)*rootsOfUnityStride]
			kzg_bls.MulModFr(&v, jv, r)
			kzg_bls.CopyFr(&tmp, &last)
			kzg_bls.AddModFr(&last, &tmp, &v)
		}
		kzg_bls.CopyFr(&out[i], &last)
	}
}

func (fs *FFTSettings) _fft(vals []kzg_bls.Fr, valsOffset uint64, valsStride uint64, rootsOfUnity []kzg_bls.Fr, rootsOfUnityStride uint64, out []kzg_bls.Fr) {
	if len(out) <= 4 { // if the value count is small, run the unoptimized version instead. // TODO tune threshold.
		fs.simpleFT(vals, valsOffset, valsStride, rootsOfUnity, rootsOfUnityStride, out)
		return
	}

	half := uint64(len(out)) >> 1
	// L will be the left half of out
	fs._fft(vals, valsOffset, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[:half])
	// R will be the right half of out
	fs._fft(vals, valsOffset+valsStride, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[half:]) // just take even again

	var yTimesRoot kzg_bls.Fr
	var x, y kzg_bls.Fr
	for i := uint64(0); i < half; i++ {
		// temporary copies, so that writing to output doesn't conflict with input
		kzg_bls.CopyFr(&x, &out[i])
		kzg_bls.CopyFr(&y, &out[i+half])
		root := &rootsOfUnity[i*rootsOfUnityStride]
		kzg_bls.MulModFr(&yTimesRoot, &y, root)
		kzg_bls.AddModFr(&out[i], &x, &yTimesRoot)
		kzg_bls.SubModFr(&out[i+half], &x, &yTimesRoot)
	}
}

func (fs *FFTSettings) FFT(vals []kzg_bls.Fr, inv bool) ([]kzg_bls.Fr, error) {
	n := uint64(len(vals))
	if n > fs.MaxWidth {
		return nil, fmt.Errorf("got %d values but only have %d roots of unity", n, fs.MaxWidth)
	}
	n = nextPowOf2(n)
	// We make a copy so we can mutate it during the work.
	valsCopy := make([]kzg_bls.Fr, n, n)
	for i := 0; i < len(vals); i++ {
		kzg_bls.CopyFr(&valsCopy[i], &vals[i])
	}
	for i := uint64(len(vals)); i < n; i++ {
		kzg_bls.CopyFr(&valsCopy[i], &kzg_bls.ZERO)
	}
	out := make([]kzg_bls.Fr, n, n)
	if err := fs.InplaceFFT(valsCopy, out, inv); err != nil {
		return nil, err
	}
	return out, nil
}

func (fs *FFTSettings) InplaceFFT(vals []kzg_bls.Fr, out []kzg_bls.Fr, inv bool) error {
	n := uint64(len(vals))
	if n > fs.MaxWidth {
		return fmt.Errorf("got %d values but only have %d roots of unity", n, fs.MaxWidth)
	}
	if !kzg_bls.IsPowerOfTwo(n) {
		return fmt.Errorf("got %d values but not a power of two", n)
	}
	if inv {
		var invLen kzg_bls.Fr
		kzg_bls.AsFr(&invLen, n)
		kzg_bls.InvModFr(&invLen, &invLen)
		rootz := fs.ReverseRootsOfUnity[:fs.MaxWidth]
		stride := fs.MaxWidth / n

		fs._fft(vals, 0, 1, rootz, stride, out)
		var tmp kzg_bls.Fr
		for i := 0; i < len(out); i++ {
			kzg_bls.MulModFr(&tmp, &out[i], &invLen)
			kzg_bls.CopyFr(&out[i], &tmp) // TODO: depending on Fr implementation, allow to directly write back to an input
		}
		return nil
	} else {
		rootz := fs.ExpandedRootsOfUnity[:fs.MaxWidth]
		stride := fs.MaxWidth / n
		// Regular FFT
		fs._fft(vals, 0, 1, rootz, stride, out)
		return nil
	}
}

// rearrange Fr elements in reverse bit order. Supports 2**31 max element count.
func reverseBitOrderFr(values []kzg_bls.Fr) {
	if len(values) > (1 << 31) {
		panic("list too large")
	}
	var tmp kzg_bls.Fr
	reverseBitOrder(uint32(len(values)), func(i, j uint32) {
		kzg_bls.CopyFr(&tmp, &values[i])
		kzg_bls.CopyFr(&values[i], &values[j])
		kzg_bls.CopyFr(&values[j], &tmp)
	})
}

// rearrange Fr ptr elements in reverse bit order. Supports 2**31 max element count.
func reverseBitOrderFrPtr(values []*kzg_bls.Fr) {
	if len(values) > (1 << 31) {
		panic("list too large")
	}
	reverseBitOrder(uint32(len(values)), func(i, j uint32) {
		values[i], values[j] = values[j], values[i]
	})
}
