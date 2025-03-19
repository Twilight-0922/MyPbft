package kzg_bls

import (
	"fmt"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"strings"
)

// TODO types file, swap BLS with build args
type G1Point bls.G1
type G2Point bls.G2

var ZERO_G1 G1Point

var GenG1 G1Point
var GenG2 G2Point

var ZeroG1 G1Point
var ZeroG2 G2Point

// Herumi BLS doesn't offer these points to us, so we have to work around it by declaring them ourselves.
func initG1G2() {
	GenG1 = G1Point(*(bls.G1Generator()))
	GenG2 = G2Point(*(bls.G2Generator()))

	ZeroG1 = G1Point(*new(bls.G1))
	(*bls.G1)(&ZeroG1).SetIdentity()
	ZeroG2 = G2Point(*new(bls.G2))
	(*bls.G2)(&ZeroG2).SetIdentity()
}

func ClearG1(x *G1Point) {
	(*bls.G1)(x).SetIdentity()
}

func CopyG1(dst *G1Point, v *G1Point) {
	*dst = *v
}

func MulG1(dst *G1Point, a *G1Point, b *Fr) {
	(*bls.G1)(dst).ScalarMult((*bls.Scalar)(b), (*bls.G1)(a))
}

func AddG1(dst *G1Point, a *G1Point, b *G1Point) {
	(*bls.G1)(dst).Add((*bls.G1)(a), (*bls.G1)(b))
}

func SubG1(dst *G1Point, a *G1Point, b *G1Point) {
	(*bls.G1)(b).Neg()
	AddG1(dst, a, b)
	(*bls.G1)(b).Neg()
}

func StrG1(v *G1Point) string {
	return (*bls.G1)(v).String()
}

func NegG1(dst *G1Point) {
	// in-place should be safe here (TODO double check)
	(*bls.G1)(dst).Neg()
}

func ClearG2(x *G2Point) {
	(*bls.G2)(x).SetIdentity()
}

func CopyG2(dst *G2Point, v *G2Point) {
	*dst = *v
}

func MulG2(dst *G2Point, a *G2Point, b *Fr) {
	(*bls.G2)(dst).ScalarMult((*bls.Scalar)(b), (*bls.G2)(a))
}

func AddG2(dst *G2Point, a *G2Point, b *G2Point) {
	(*bls.G2)(dst).Add((*bls.G2)(a), (*bls.G2)(b))
}

func SubG2(dst *G2Point, a *G2Point, b *G2Point) {
	(*bls.G2)(b).Neg()
	AddG2(dst, a, b)
	(*bls.G2)(b).Neg()
}

func NegG2(dst *G2Point) {
	// in-place should be safe here (TODO double check)
	(*bls.G2)(dst).Neg()
}

func StrG2(v *G2Point) string {
	return (*bls.G2)(v).String()
}

func EqualG1(a *G1Point, b *G1Point) bool {
	return (*bls.G1)(a).IsEqual((*bls.G1)(b))
}

func EqualG2(a *G2Point, b *G2Point) bool {
	return (*bls.G2)(a).IsEqual((*bls.G2)(b))
}

func ToCompressedG1(p *G1Point) []byte {
	return (*bls.G1)(p).BytesCompressed()
}

func FromCompressedG1(v []byte) (*G1Point, error) {
	p := new(bls.G1)
	err := p.SetBytes(v)
	return (*G1Point)(p), err
}

func ToCompressedG2(p *G2Point) []byte {
	return (*bls.G2)(p).BytesCompressed()
}

func FromCompressedG2(v []byte) (*G2Point, error) {
	p := new(bls.G2)
	err := p.SetBytes(v)
	return (*G2Point)(p), err
}

func LinCombG1(numbers []G1Point, factors []Fr) *G1Point {
	if len(numbers) != len(factors) {
		panic("got LinCombG1 numbers/factors length mismatch")
	}
	n := len(numbers)
	var out G1Point
	(*bls.G1)(&out).SetIdentity()
	for i := 0; i < n; i++ {
		temp := new(bls.G1)
		temp.ScalarMult((*bls.Scalar)(&factors[i]), (*bls.G1)(&numbers[i]))
		(*bls.G1)(&out).Add((*bls.G1)(&out), temp)
	}

	//kzg_bls.G1MulVec((*kzg_bls.G1)(&out), *(*[]kzg_bls.G1)(unsafe.Pointer(&numbers)), *(*[]kzg_bls.Fr)(unsafe.Pointer(&factors)))
	return &out
}

func LinCombG2(numbers []G2Point, factors []Fr) *G2Point {
	if len(numbers) != len(factors) {
		panic("got LinCombG1 numbers/factors length mismatch")
	}
	n := len(numbers)
	var out G2Point
	(*bls.G2)(&out).SetIdentity()
	for i := 0; i < n; i++ {
		temp := new(bls.G2)
		temp.ScalarMult((*bls.Scalar)(&factors[i]), (*bls.G2)(&numbers[i]))
		(*bls.G2)(&out).Add((*bls.G2)(&out), temp)
	}

	//kzg_bls.G1MulVec((*kzg_bls.G1)(&out), *(*[]kzg_bls.G1)(unsafe.Pointer(&numbers)), *(*[]kzg_bls.Fr)(unsafe.Pointer(&factors)))
	return &out
}

// e(a1^(-1), a2) * e(b1,  b2) = 1_T
func PairingsVerify(a1 *G1Point, a2 *G2Point, b1 *G1Point, b2 *G2Point) bool {
	a1a2 := bls.Pair((*bls.G1)(a1), (*bls.G2)(a2))
	a1a2.Inv(a1a2)

	b1b2 := bls.Pair((*bls.G1)(b1), (*bls.G2)(b2))

	a1a2.Mul(a1a2, b1b2)
	return a1a2.IsIdentity()
	// TODO, alternatively use the equal check (faster or slower?):
	////fmt.Println("tmp2", tmp2.GetString(10))
	//return tmp.IsEqual(&tmp2)
}

func DebugG1s(msg string, values []G1Point) {
	var out strings.Builder
	for i := range values {
		out.WriteString(fmt.Sprintf("%s %d: %s\n", msg, i, StrG1(&values[i])))
	}
	fmt.Println(out.String())
}
