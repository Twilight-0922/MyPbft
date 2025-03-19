package THSecretShare

import (
	"DistributedBFIBE/src/internal/IBE/ibecommon"
	"DistributedBFIBE/src/internal/THSecretShare/common"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"log"
	"testing"
)

func TestGenerateRandomPolynomial(t *testing.T) {
	polynomial := GenerateRandomPolynomial(10)
	log.Printf("%v", polynomial)
}

func TestRandomBigInt(t *testing.T) {
	for i := 0; i < 10; i++ {
		bigInt := RandomBigInt()
		log.Print(len(bigInt.Bytes()), bigInt)
	}
}

func TestSetUp(t *testing.T) {
	addX1, addX2, addX3, addX4, addX5 := new(bls.Scalar), new(bls.Scalar), new(bls.Scalar), new(bls.Scalar), new(bls.Scalar)
	_ = addX1.SetString("0x1d3cfe16cec29a481922ef5d429ce33c06a322b3b31abe98524fa2ec1ff52f56")
	_ = addX2.SetString("0x4493d8881622bc0131da23a45656c29316cb81e55ea4a431e937d49f82d47b78")
	_ = addX3.SetString("0x4352667dd8c0bad0e9bf97d7e81ada8ec097aa007a3f75d8e6fc417cc7087eb6")
	_ = addX4.SetString("0x3cdaa2f77de9a865c2174eb24e5ec6b7b385bd21ec1225dd4baaa214342d43e7")
	_ = addX5.SetString("0x5694a475ae76d9e1e992088de07e74d88f40600dcec523c0f100a03836b4dd26")

	a1, a2, a3, a4, a5 := new(bls.G2), new(bls.G2), new(bls.G2), new(bls.G2), new(bls.G2)
	a1.ScalarMult(addX1, bls.G2Generator())
	a2.ScalarMult(addX2, bls.G2Generator())
	a3.ScalarMult(addX3, bls.G2Generator())
	a4.ScalarMult(addX4, bls.G2Generator())
	a5.ScalarMult(addX5, bls.G2Generator())

	a1.Add(a1, a2)
	a1.Add(a1, a3)
	a1.Add(a1, a4)
	a1.Add(a1, a5)

	log.Println("a: \n", a1.String())
	addX1.Add(addX1, addX2)
	addX1.Add(addX1, addX3)
	addX1.Add(addX1, addX4)
	addX1.Add(addX1, addX5)

	g2 := new(bls.G2)
	g2.ScalarMult(addX1, bls.G2Generator())

	//log.Println("KGC节点多项式主秘密加和值:\n", addX1.String())
	log.Println("KGC节点多项式主秘密加和值计算主公钥:\n", g2.String())
	log.Println("a equal g2 : ", a1.IsEqual(g2))

	S1, S2, S3, S4, S5 := new(bls.Scalar), new(bls.Scalar), new(bls.Scalar), new(bls.Scalar), new(bls.Scalar)
	_ = S1.SetString("0x21939d76df1887738affcefdd7f0240402f6a2984229287647f4c9abb69bd3f4")
	_ = S2.SetString("0x4bd508f0518afc746a4aa9d2602148d865963bd5ab83504422e4e4c1f7d3aab7")
	_ = S3.SetString("0x5b8dd0fcc4857a8be4990a7f2b99a25b4d724b7882e985adefff4c9a985bced7")
	_ = S4.SetString("0x50bdf59c380801b9f9eaf1043a59308cba8ad180c85bc8b3af44013598344054")
	_ = S5.SetString("0x2b6576ceac1291feaa405d618c5ff36cacdfcdee7bda195560b30292f75cff2e")

	P1, P2, P3, P4, P5 := new(bls.G2), new(bls.G2), new(bls.G2), new(bls.G2), new(bls.G2)
	P1.ScalarMult(S1, bls.G2Generator())
	P2.ScalarMult(S2, bls.G2Generator())
	P3.ScalarMult(S3, bls.G2Generator())
	P4.ScalarMult(S4, bls.G2Generator())
	P5.ScalarMult(S5, bls.G2Generator())
	log.Println("P1", P1.String())

	_ = P1.SetBytes(P1.Bytes())
	_ = P2.SetBytes(P2.Bytes())
	_ = P3.SetBytes(P3.Bytes())
	_ = P4.SetBytes(P4.Bytes())
	_ = P5.SetBytes(P5.Bytes())
	log.Println("P1", P1.String())

	S := []common.BlsPrimaryShare{
		{1, *S1},
		{2, *S2},
		{3, *S3},
		{4, *S4},
		{5, *S5},
	}
	SG2 := []common.BlsG2PrimaryShare{
		{1, *P1},
		{2, *P2},
		{3, *P3},
		{4, *P4},
		{5, *P5},
	}

	PPub := BlsG2LagrangeScalar(SG2, 0)
	scalar := BlsLagrangeScalar(S, 0)
	//g2Scalar.ScalarMult(scalar, bls.G2Generator())
	log.Println("BlsG2LagrangeScalar: \n", PPub.String())
	log.Println("g2 equal BlsG2LagrangeScalar: ", g2.IsEqual(PPub))
	log.Println("a equal BlsG2LagrangeScalar: ", a1.IsEqual(PPub))

	log.Println("BlsLagrangeScalar: ", scalar.String())
	log.Println("sum of Xi == LagrangeScalar Si : ", addX1.IsEqual(scalar))

	log.Println("---------------------------------------------------------------------------")
	Qid := ibecommon.H1([]byte("hello"))

	H1, H2, H3, H4, H5 := new(bls.G1), new(bls.G1), new(bls.G1), new(bls.G1), new(bls.G1)

	H1.ScalarMult(S1, Qid)
	H2.ScalarMult(S2, Qid)
	H3.ScalarMult(S3, Qid)
	H4.ScalarMult(S4, Qid)
	H5.ScalarMult(S5, Qid)

	_ = H1.SetBytes(H1.Bytes())
	_ = P2.SetBytes(H2.Bytes())
	_ = H3.SetBytes(H3.Bytes())
	_ = H4.SetBytes(H4.Bytes())
	_ = H5.SetBytes(H5.Bytes())

	SG1 := []common.BlsG1PrimaryShare{
		{1, *H1},
		{2, *H2},
		{3, *H3},
		{4, *H4},
		{5, *H5},
	}
	h1Test := new(bls.G1)
	h1Test.ScalarMult(addX1, Qid)
	log.Println("h1 :\n", h1Test.String())

	SQid := BlsG1LagrangeScalar(SG1, 0)
	log.Println("g1LagrangeScalar: \n", SQid.String())
	log.Println("h1 equal g1LagrangeScalar: ", h1Test.IsEqual(SQid))

	PairSQidP := bls.Pair(SQid, bls.G2Generator())
	PairQidSP := bls.Pair(ibecommon.H1([]byte("hello")), PPub)

	test := new(bls.G2)
	_ = test.SetBytes(PPub.Bytes())
	testPair := bls.Pair(ibecommon.H1([]byte("hello")), test)

	log.Println(PPub.String())
	log.Println(testPair.String())
	log.Println("PairSQidP.IsEqual(testPair): ", PairSQidP.IsEqual(testPair))

	log.Println("SQidP.IsEqual(QidSP): ", PairSQidP.IsEqual(PairQidSP))
	log.Println("SQidP h2:  ", ibecommon.H2(PairSQidP))
	log.Println("QidSP h2:  ", ibecommon.H2(PairQidSP))
}

func TestBls(t *testing.T) {
	poly := BlsGenerateRandomPolynomial(10)
	log.Println(poly.Coeff[0].String())

	shares := make([]common.BlsPrimaryShare, 10)
	for i := 1; i <= 10; i++ {
		shares[i-1].Index = i
		shares[i-1].Value = *BlsPolyEval(poly, i)
	}

	scalar := BlsLagrangeScalar(shares, 0)
	log.Println(scalar.String())
	log.Println("poly.Coeff[0] equal scalar: ", poly.Coeff[0].IsEqual(scalar))
}
