package ibecommon

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"IBEUser/src/THSecretShare/common"
	_ "embed"
	"encoding/json"
	"errors"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"math/big"
	"math/bits"
)

type Polynomial []kzg_bls.Fr
type Polynomials [][]kzg_bls.Fr
type KZGCommitment [48]byte
type KZGProof [48]byte
type VersionedHash [32]byte
type Root [32]byte
type Slot uint64

type JSONKZGTrustedSetup struct {
	SetupG1 [][]byte `json:"setup_G1"`
	SetupG2 [][]byte `json:"setup_G2"`
}

// 离线公共参数，可通过 helpers.go 中的 GeneratePrams 生成
//
//go:embed trusted_setup.json
var kzgSetupStr string

func bigFromHex(hex string) (*big.Int, error) {
	b, ok := new(big.Int).SetString(hex, 16)
	if !ok {
		return nil, errors.New("invalid hex string")
	}
	return b, nil
}

func bitReversalPermutation(l []kzg_bls.G1Point) []kzg_bls.G1Point {
	out := make([]kzg_bls.G1Point, len(l))

	order := uint64(len(l))

	for i := range l {
		out[i] = l[reverseBits(uint64(i), order)]
	}

	return out
}

func reverseBits(n, order uint64) uint64 {
	if !isPowerOfTwo(order) {
		panic("Order must be a power of two.")
	}

	return bits.Reverse64(n) >> (65 - bits.Len64(order))
}

func isPowerOfTwo(value uint64) bool {
	return value > 0 && (value&(value-1) == 0)
}

func PolynomialToKZGCommitment(kzgSetupLagrange []kzg_bls.G1Point, eval Polynomial) KZGCommitment {
	g1 := kzg_bls.LinCombG1(kzgSetupLagrange[:len(eval)], []kzg_bls.Fr(eval))
	var out KZGCommitment
	copy(out[:], kzg_bls.ToCompressedG1(g1))
	return out
}

func PolynomialTrans(ploy *common.BlsPrimaryPolynomial) Polynomial {
	frs := make([]kzg_bls.Fr, len(ploy.Coeff))
	for i, v := range ploy.Coeff {
		(*bls.Scalar)(&frs[i]).Set(&v)
	}
	return frs
}

// GeneratePrams 用于离线生成公共参数
func GeneratePrams(secret string, n uint64) ([]kzg_bls.G1Point, []kzg_bls.G2Point) {
	var s kzg_bls.Fr
	kzg_bls.SetFr(&s, secret)

	var sPow kzg_bls.Fr
	(*bls.Scalar)(&sPow).SetOne()

	s1Out := make([]kzg_bls.G1Point, n, n)
	s2Out := make([]kzg_bls.G2Point, n, n)
	for i := uint64(0); i < n; i++ {
		kzg_bls.MulG1(&s1Out[i], (*kzg_bls.G1Point)(bls.G1Generator()), &sPow)
		kzg_bls.MulG2(&s2Out[i], (*kzg_bls.G2Point)(bls.G2Generator()), &sPow)
		var tmp kzg_bls.Fr
		kzg_bls.CopyFr(&tmp, &sPow)
		kzg_bls.MulModFr(&sPow, &tmp, &s)
	}
	return s1Out, s2Out
}

func GetJSONPrams(threshold int) ([]kzg_bls.G1Point, []kzg_bls.G2Point, error) {
	var parsedSetup = JSONKZGTrustedSetup{}
	_ = json.Unmarshal([]byte(kzgSetupStr), &parsedSetup)
	if len(parsedSetup.SetupG1) < threshold || len(parsedSetup.SetupG2) < threshold {
		return nil, nil, errors.New("threshold oversize")
	}

	s1, s2 := make([]kzg_bls.G1Point, threshold), make([]kzg_bls.G2Point, threshold)
	for i := 0; i < threshold; i++ {
		err := (*bls.G1)(&s1[i]).SetBytes(parsedSetup.SetupG1[i])
		if err != nil {
			return nil, nil, err
		}
		err = (*bls.G2)(&s2[i]).SetBytes(parsedSetup.SetupG2[i])
		if err != nil {
			return nil, nil, err
		}
	}
	return s1, s2, nil
}

func TransPloyToKzgPloy(ploy *common.BlsPrimaryPolynomial) []kzg_bls.Fr {
	kzgPloy := make([]kzg_bls.Fr, ploy.Threshold)
	for i, v := range ploy.Coeff {
		kzgPloy[i] = (kzg_bls.Fr)(v)
	}
	return kzgPloy
}
