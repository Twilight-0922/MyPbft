package common

import (
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"math/big"
)

type SigncryptedOutput struct {
	NodePubKey       Point
	NodeIndex        int
	SigncryptedShare Signcryption
}

type Signcryption struct {
	Ciphertext []byte
	R          Point
	Signature  big.Int
}

type PrimaryPolynomial struct {
	Coeff     []big.Int //系数
	Threshold int       //门限
}

type BlsPrimaryPolynomial struct {
	Coeff     []bls.Scalar //系数
	Threshold int          //门限
}

type PrimaryShare struct {
	Index int
	Value big.Int
}

type BlsPrimaryShare struct {
	Index int
	Value bls.Scalar
}

type BlsG2PrimaryShare struct {
	Index int
	Value bls.G2
}

type BlsG1PrimaryShare struct {
	Index int
	Value bls.G1
}

type Point struct {
	X big.Int
	Y big.Int
}

//type Hash struct {
//	cmn.HexBytes
//}

type Node struct {
	Index  int
	PubKey Point
}
