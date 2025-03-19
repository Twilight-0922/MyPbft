package ibecommon

import (
	"crypto/sha256"
	bls "github.com/cloudflare/circl/ecc/bls12381"
)

func H1(in []byte) *bls.G1 {
	out := new(bls.G1)
	out.Hash(in, make([]byte, 0))
	return out
}

func H11(in []byte) *bls.G1 {
	out := new(bls.G1)
	out.Hash(in, make([]byte, 0))
	return out
}

func H12(in []byte) *bls.G2 {
	out := new(bls.G2)
	out.Hash(in, make([]byte, 0))
	return out
}

func H2(in *bls.Gt) []byte {
	bytes, err := in.MarshalBinary()
	if err != nil {
		panic(err)
	}
	md := sha256.New()
	md.Write(bytes)

	return md.Sum(nil)
}

func XorBytes(a []byte, b []byte) []byte {
	res := make([]byte, len(a))
	for i, elem := range a {
		res[i] = elem ^ b[i]
	}
	return res
}
