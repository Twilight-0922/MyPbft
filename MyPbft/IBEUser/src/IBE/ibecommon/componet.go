package ibecommon

import bls "github.com/cloudflare/circl/ecc/bls12381"

//type MasterKeyShare struct {
//	S *bls.Scalar
//}

type Conf struct {
	Threshold   int     `yaml:"threshold"`
	IP          string  `yaml:"ip"`
	Port        int     `yaml:"port"`
	Idx         int     `yaml:"idx"`
	FFTMaxScale int     `yaml:"fftMaxScale"`
	Parties     []Party `yaml:"parties"`
}

type Party struct {
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	PartyIdx int    `yaml:"partyIdx"`
}

type Share struct {
	Share   string `json:"share"`
	SendIdx int    `json:"sendIdx"`
}

type TrustShare struct {
	Commit  []byte `json:"commit"`
	Proof   []byte `json:"proof"`
	Share   string `json:"share"`
	SendIdx int    `json:"sendIdx"`
}

type PointShare struct {
	Share   []byte `json:"share"`
	SendIdx int    `json:"sendIdx"`
}

type TrustPointShare struct {
	Commit    []byte `json:"commit"`
	Proof     []byte `json:"proof"`
	PPubShare []byte `json:"ppub"`
	Share     []byte `json:"share"`
	SendIdx   int    `json:"sendIdx"`
}

type Ciphertext struct {
	U *bls.G1 //rP
	V []byte
}

type SignatureText struct {
	U *bls.G1
	V []byte
}

type TransSignatureText struct {
	U []byte //rP
	V []byte
}

type TransCiphertext struct {
	U []byte //rP
	V []byte
}
