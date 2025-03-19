package kzg_bls

import (
	"crypto/rand"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"math/big"
)

type Fr bls.Scalar

var _modulus big.Int

func init() {
	initGlobals()
	ClearG1(&ZERO_G1)
	initG1G2()
}

func SetFr(dst *Fr, v string) {
	if err := (*bls.Scalar)(dst).SetString(v); err != nil {
		panic(err)
	}
}

// FrFrom32 mutates the fr num. The value v is little-endian 32-bytes.
// Returns false, without modifying dst, if the value is out of range.
func FrFrom32(dst *Fr, v [32]byte) (ok bool) {

	if !ValidFr(v) {
		return false
	}
	b := make([]byte, 32)
	for i, last := 0, len(b)-1; i < 16; i++ {
		b[i], b[last-i] = v[last-i], v[i]
	}
	(*bls.Scalar)(dst).SetBytes(b[:])
	return true
}

// FrTo32 serializes a fr number to 32 bytes. Encoded little-endian.
func FrTo32(src *Fr) (v [32]byte) {
	b, _ := (*bls.Scalar)(src).MarshalBinary()
	last := len(b) - 1
	// reverse endianness, Herumi outputs big-endian bytes
	for i := 0; i < 16; i++ {
		b[i], b[last-i] = b[last-i], b[i]
	}
	copy(v[:], b)
	return
}

func CopyFr(dst *Fr, v *Fr) {
	*dst = *v
}

func AsFr(dst *Fr, i uint64) {
	(*bls.Scalar)(dst).SetUint64(i)
}

func FrStr(b *Fr) string {
	if b == nil {
		return "<nil>"
	}
	temp := new(big.Int)
	temp.SetString((*bls.Scalar)(b).String()[2:], 16)
	return temp.Text(10)
}

func EqualOne(v *Fr) bool {
	one := new(bls.Scalar)
	one.SetOne()
	return (*bls.Scalar)(v).IsEqual(one) == 1
}

func EqualZero(v *Fr) bool {
	return (*bls.Scalar)(v).IsZero() == 1
}

func EqualFr(a *Fr, b *Fr) bool {
	return (*bls.Scalar)(a).IsEqual((*bls.Scalar)(b)) == 1
}

func RandomFr() *Fr {
	var out bls.Scalar
	_ = out.Random(rand.Reader)
	return (*Fr)(&out)
}

func SubModFr(dst *Fr, a, b *Fr) {
	(*bls.Scalar)(dst).Sub((*bls.Scalar)(a), (*bls.Scalar)(b))
}

func AddModFr(dst *Fr, a, b *Fr) {
	(*bls.Scalar)(dst).Add((*bls.Scalar)(a), (*bls.Scalar)(b))
}

func DivModFr(dst *Fr, a, b *Fr) {
	bInv := new(bls.Scalar)
	bInv.Set((*bls.Scalar)(b))
	bInv.Inv(bInv)
	MulModFr(dst, a, (*Fr)(bInv))
}

func MulModFr(dst *Fr, a, b *Fr) {
	(*bls.Scalar)(dst).Mul((*bls.Scalar)(a), (*bls.Scalar)(b))
}

func InvModFr(dst *Fr, v *Fr) {
	(*bls.Scalar)(dst).Set((*bls.Scalar)(v))
	(*bls.Scalar)(dst).Inv((*bls.Scalar)(dst))
}

// BatchInvModFr computes the inverse for each input.
// Warning: this does not actually batch, this is just here for compatibility with other BLS backends that do.
func BatchInvModFr(f []Fr) {
	for i := 0; i < len(f); i++ {
		(*bls.Scalar)(&f[i]).Inv((*bls.Scalar)(&f[i]))
	}
}

//func SqrModFr(dst *Fr, v *Fr) {
//	kzg_bls.FrSqr((*kzg_bls.Fr)(dst), (*kzg_bls.Fr)(v))
//}

func EvalPolyAt(dst *Fr, p []Fr, x *Fr) {
	EvalPolyAtUnoptimized(dst, p, x)
}

// ExpModFr computes v**e in Fr. Warning: this is a slow fallback on big int math.
func ExpModFr(dst *Fr, v *Fr, e *big.Int) {
	vBig, ok := new(big.Int).SetString(FrStr(v), 10)
	if !ok {
		panic("failed string hack")
	}
	MOD := new(big.Int)
	MOD.SetString(ModulusStr, 10)
	res := new(big.Int).Exp(vBig, e, MOD)
	SetFr(dst, res.String())
}
