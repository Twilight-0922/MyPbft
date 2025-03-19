package THSecretShare

import (
	"IBEUser/src/THSecretShare/common"
	"crypto/rand"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"log"
	"math/big"

	"github.com/emmansun/gmsm/sm9/bn256"
)

// GenerateRandomPolynomial 初始化需要离线可信第三方
// 输入门限值，生成随机多项式
func GenerateRandomPolynomial(threshold int) *common.PrimaryPolynomial {
	coeff := make([]big.Int, threshold)
	for i := 0; i < threshold; i++ { //randomly choose coeff
		coeff[i] = *RandomBigInt()
	}
	return &common.PrimaryPolynomial{Coeff: coeff, Threshold: threshold}
}

// BlsGenerateRandomPolynomial TODO 修改为 bls 有限域上的的随机多项式生成
func BlsGenerateRandomPolynomial(threshold int) *common.BlsPrimaryPolynomial {
	coeff := make([]bls.Scalar, threshold)
	for i := 0; i < threshold; i++ { //randomly choose coeff
		coeff[i] = *new(bls.Scalar)
		err := coeff[i].Random(rand.Reader)
		if err != nil {
			log.Println("rand coeff err: ", err)
			return nil
		}
	}
	return &common.BlsPrimaryPolynomial{Coeff: coeff, Threshold: threshold}
}

// PolyEval 计算 x 的秘密份额
func PolyEval(polynomial *common.PrimaryPolynomial, x int) *big.Int {
	xi := big.NewInt(int64(x))
	sum := new(big.Int)
	sum.Add(sum, &polynomial.Coeff[0]) //主秘密

	//分发方根据多项式，计算 p(i)
	for i := 1; i < polynomial.Threshold; i++ {
		tmp := new(big.Int).Mul(xi, &polynomial.Coeff[i])
		sum.Add(sum, tmp)
		sum.Mod(sum, bn256.Order)
		xi.Mul(xi, big.NewInt(int64(x)))
		xi.Mod(xi, bn256.Order)
	}
	return sum
}

// BlsPolyEval TODO: 修改为 bls 有限域上的多项式计算
func BlsPolyEval(polynomial *common.BlsPrimaryPolynomial, x int) *bls.Scalar {
	xi, xx := new(bls.Scalar), new(bls.Scalar)
	xi.SetUint64(uint64(x))
	xx.SetUint64(uint64(x))
	sum := new(bls.Scalar)
	sum.Add(sum, &polynomial.Coeff[0]) //主秘密
	//分发方根据多项式，计算 p(i)
	for i := 1; i < polynomial.Threshold; i++ {
		tmp := new(bls.Scalar)
		tmp.Mul(xi, &polynomial.Coeff[i])
		sum.Add(sum, tmp)
		xi.Mul(xi, xx)
	}
	return sum
}

// LagrangeScalar 拉格朗日插值法解主秘密
func LagrangeScalar(shares []common.PrimaryShare, target int) *big.Int {
	secret := new(big.Int)
	for _, share := range shares {
		//when x = 0
		delta := new(big.Int).SetInt64(int64(1))
		upper := new(big.Int).SetInt64(int64(1))
		lower := new(big.Int).SetInt64(int64(1))
		//求拉格朗日系数
		for j := range shares {
			if shares[j].Index != share.Index {
				tempUpper := big.NewInt(int64(target))
				tempUpper.Sub(tempUpper, big.NewInt(int64(shares[j].Index)))
				upper.Mul(upper, tempUpper)
				upper.Mod(upper, bn256.Order)

				tempLower := big.NewInt(int64(share.Index))
				tempLower.Sub(tempLower, big.NewInt(int64(shares[j].Index)))
				tempLower.Mod(tempLower, bn256.Order)

				lower.Mul(lower, tempLower)
				lower.Mod(lower, bn256.Order)
			}
		}
		//elliptic division
		inv := new(big.Int)
		inv.ModInverse(lower, bn256.Order)
		delta.Mul(upper, inv)
		delta.Mod(delta, bn256.Order)

		delta.Mul(&share.Value, delta)
		delta.Mod(delta, bn256.Order)

		secret.Add(secret, delta)
	}
	secret.Mod(secret, bn256.Order)
	return secret
}

// BlsLagrangeScalar 拉格朗日插值法解主秘密
func BlsLagrangeScalar(shares []common.BlsPrimaryShare, target int) *bls.Scalar {
	secret := new(bls.Scalar)
	for _, share := range shares {
		//when x = 0
		delta, upper, lower := new(bls.Scalar), new(bls.Scalar), new(bls.Scalar)
		delta.SetOne()
		upper.SetOne()
		lower.SetOne()
		//求拉格朗日系数
		for j := range shares {
			if shares[j].Index != share.Index {
				tempUpper, idxJ := new(bls.Scalar), new(bls.Scalar)
				tempUpper.SetUint64(uint64(target))
				idxJ.SetUint64(uint64(shares[j].Index))

				tempUpper.Sub(tempUpper, idxJ)
				upper.Mul(upper, tempUpper)

				tempLower := new(bls.Scalar)
				tempLower.SetUint64(uint64(share.Index))

				tempLower.Sub(tempLower, idxJ)

				lower.Mul(lower, tempLower)
			}
		}
		//elliptic division
		lower.Inv(lower)
		delta.Mul(upper, lower)

		delta.Mul(&share.Value, delta)

		secret.Add(secret, delta)
	}
	return secret
}

// BlsG2LagrangeScalar bls椭圆曲线上 G2 基点标量乘的插值法重建秘密
func BlsG2LagrangeScalar(shares []common.BlsG2PrimaryShare, target int) *bls.G2 {
	secret := new(bls.G2)
	secret.SetIdentity()
	for _, share := range shares {
		//when x = 0
		delta, upper, lower := new(bls.Scalar), new(bls.Scalar), new(bls.Scalar)
		delta.SetOne()
		upper.SetOne()
		lower.SetOne()
		//求拉格朗日系数
		for j := range shares {
			if shares[j].Index != share.Index {
				tempUpper, idxJ := new(bls.Scalar), new(bls.Scalar)
				tempUpper.SetUint64(uint64(target))
				idxJ.SetUint64(uint64(shares[j].Index))

				tempUpper.Sub(tempUpper, idxJ)
				upper.Mul(upper, tempUpper)

				tempLower := new(bls.Scalar)
				tempLower.SetUint64(uint64(share.Index))

				tempLower.Sub(tempLower, idxJ)

				lower.Mul(lower, tempLower)
			}
		}
		//elliptic division
		lower.Inv(lower)
		delta.Mul(upper, lower)

		v := &(share.Value)
		v.ScalarMult(delta, v)

		secret.Add(secret, v)
	}
	return secret
}

// BlsG1LagrangeScalar bls椭圆曲线上 G1 基点标量乘的插值法重建秘密
func BlsG1LagrangeScalar(shares []common.BlsG1PrimaryShare, target int) *bls.G1 {
	secret := new(bls.G1)
	secret.SetIdentity()
	for _, share := range shares {
		//when x = 0
		delta, upper, lower := new(bls.Scalar), new(bls.Scalar), new(bls.Scalar)
		delta.SetOne()
		upper.SetOne()
		lower.SetOne()
		//求拉格朗日系数
		for j := range shares {
			if shares[j].Index != share.Index {
				tempUpper, idxJ := new(bls.Scalar), new(bls.Scalar)
				tempUpper.SetUint64(uint64(target))
				idxJ.SetUint64(uint64(shares[j].Index))

				tempUpper.Sub(tempUpper, idxJ)
				upper.Mul(upper, tempUpper)

				tempLower := new(bls.Scalar)
				tempLower.SetUint64(uint64(share.Index))

				tempLower.Sub(tempLower, idxJ)

				lower.Mul(lower, tempLower)
			}
		}
		//elliptic division
		lower.Inv(lower)
		delta.Mul(upper, lower)

		v := &(share.Value)
		v.ScalarMult(delta, v)

		secret.Add(secret, v)
	}
	return secret
}

// BlsLagrangeCoeff 拉格朗日系数计算
func BlsLagrangeCoeff(idx, target int, idxList []int) *bls.Scalar {
	delta, upper, lower := new(bls.Scalar), new(bls.Scalar), new(bls.Scalar)
	delta.SetOne()
	upper.SetOne()
	lower.SetOne()
	for _, j := range idxList {
		if j != idx {
			tempUpper, J := new(bls.Scalar), new(bls.Scalar)
			tempUpper.SetUint64(uint64(target))
			J.SetUint64(uint64(j))
			tempUpper.Sub(tempUpper, J)
			upper.Mul(upper, tempUpper)

			tempLower := new(bls.Scalar)
			tempLower.SetUint64(uint64(idx))
			tempLower.Sub(tempLower, J)

			lower.Mul(lower, tempLower)
		}
	}
	lower.Inv(lower)
	delta.Mul(upper, lower)

	return delta
}

func RandomBigInt() *big.Int {
	randomInt, _ := rand.Int(rand.Reader, bn256.Order)
	return randomInt
}
