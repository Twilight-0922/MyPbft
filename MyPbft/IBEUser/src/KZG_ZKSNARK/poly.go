package KZG_ZKSNARK

import "IBEUser/src/KZG_ZKSNARK/kzg_bls"

// invert the divisor, then multiply
func polyFactorDiv(dst *kzg_bls.Fr, a *kzg_bls.Fr, b *kzg_bls.Fr) {
	// TODO: use divmod instead.
	var tmp kzg_bls.Fr
	kzg_bls.InvModFr(&tmp, b)
	kzg_bls.MulModFr(dst, &tmp, a)
}

// Long polynomial division for two polynomials in coefficient form
func polyLongDiv(dividend []kzg_bls.Fr, divisor []kzg_bls.Fr) []kzg_bls.Fr {
	a := make([]kzg_bls.Fr, len(dividend), len(dividend))
	for i := 0; i < len(a); i++ {
		kzg_bls.CopyFr(&a[i], &dividend[i])
	}
	aPos := len(a) - 1
	bPos := len(divisor) - 1
	diff := aPos - bPos
	out := make([]kzg_bls.Fr, diff+1, diff+1)
	for diff >= 0 {
		quot := &out[diff]
		polyFactorDiv(quot, &a[aPos], &divisor[bPos])
		var tmp, tmp2 kzg_bls.Fr
		for i := bPos; i >= 0; i-- {
			// In steps: a[diff + i] -= b[i] * quot
			// tmp =  b[i] * quot
			kzg_bls.MulModFr(&tmp, quot, &divisor[i])
			// tmp2 = a[diff + i] - tmp
			kzg_bls.SubModFr(&tmp2, &a[diff+i], &tmp)
			// a[diff + i] = tmp2
			kzg_bls.CopyFr(&a[diff+i], &tmp2)
		}
		aPos -= 1
		diff -= 1
	}
	return out
}
