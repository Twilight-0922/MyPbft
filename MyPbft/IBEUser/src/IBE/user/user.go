package user

import (
	"IBEUser/pkg/config"
	"IBEUser/src/IBE/ibecommon"
	"IBEUser/src/KZG_ZKSNARK"
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"IBEUser/src/THSecretShare"
	"IBEUser/src/THSecretShare/common"
	"crypto/rand"
	"encoding/json"
	"fmt"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type User struct {
	PPub      *bls.G1           //系统主公钥
	P         *bls.G1           //G1 群的生成元
	Threshold int               //系统门限值
	KGCs      []ibecommon.Party //KGC 节点信息
	IP        string
	Port      int //端口

	ID  string
	Qid *bls.G2
	sk  *bls.G2 //用户私钥

	Kzg *KZG_ZKSNARK.KZGSettings
}

func (user *User) Init(userId, configPath string) error {
	iniC := config.Config{Name: configPath}
	err := iniC.InitConfig()
	if err != nil {
		log.Println(err)
		return err
	}
	err = viper.ReadInConfig()
	if err != nil {
		log.Println(err)
		return err
	}
	conf := ibecommon.Conf{}
	err = viper.Unmarshal(&conf)
	if err != nil {
		log.Println(err)
		return err
	}

	user.P = bls.G1Generator()
	user.KGCs = conf.Parties
	user.IP = conf.IP
	user.Port = conf.Port
	user.Threshold = conf.Threshold
	user.ID = url.QueryEscape(userId) //TODO 调试修改
	user.Qid = ibecommon.H12([]byte(user.ID))

	//TODO KZG
	fftSettings := KZG_ZKSNARK.NewFFTSettings(uint8(conf.FFTMaxScale))
	g1Points, g2Points, err := ibecommon.GetJSONPrams(user.Threshold)
	user.Kzg = KZG_ZKSNARK.NewKZGSettings(fftSettings, g1Points, g2Points)

	//user.getPPub()
	user.getSk()

	return nil
}

func (user *User) getPPub() {

	skShares := make([]common.BlsG1PrimaryShare, user.Threshold)
	vis := make([]bool, len(user.KGCs))
	for shareCnt := 0; shareCnt < user.Threshold; {
		for idx, kgc := range user.KGCs {
			if vis[idx] {
				continue
			}
			reqUrl := "http://" + kgc.IP + ":" + strconv.Itoa(kgc.Port) + "/getPPub"
			req, err := http.NewRequest(http.MethodGet, reqUrl, nil)

			if err != nil {
				log.Println("http.NewRequest err:", err)
				continue
			}

			//5 秒请求超时
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Println("http.Client.Do err:", err)
				continue
			}

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("ioUtil.ReadAll err:", err)
				continue
			}

			var share ibecommon.PointShare
			err = json.Unmarshal(respBody, &share)

			if err != nil {
				log.Println("respBody json.Unmarshal err:", err)
				continue
			}

			b := new(bls.G1)
			err = b.SetBytes(share.Share)
			//log.Println("\n", b)
			if err != nil {
				log.Println("getPPub() err share: ", err)
				continue
			}

			skShares[shareCnt] = common.BlsG1PrimaryShare{Index: share.SendIdx, Value: *b}
			shareCnt++

			err = resp.Body.Close()
			if err != nil {
				log.Println("resp.Body.Close() err: ", err)
				continue
			}

			if shareCnt >= user.Threshold {
				break
			}
		}
	}
	user.PPub = THSecretShare.BlsG1LagrangeScalar(skShares, user.Threshold)
}

// Trust
func (user *User) getSk() {

	skShares := make([]common.BlsG2PrimaryShare, user.Threshold)
	ppubShares := make([]common.BlsG1PrimaryShare, user.Threshold)
	vis := make([]bool, len(user.KGCs))
	for shareCnt := 0; shareCnt < user.Threshold; {
		for idx, kgc := range user.KGCs {
			if vis[idx] {
				continue
			}
			reqUrl := "http://" + kgc.IP + ":" + strconv.Itoa(kgc.Port) + "/privateKey?userId=" + user.ID
			req, err := http.NewRequest(http.MethodGet, reqUrl, nil)

			if err != nil {
				log.Println("http.NewRequest err:", err)
				continue
			}

			//5 秒请求超时
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Println("http.Client.Do err:", err)
				continue
			}

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("ioUtil.ReadAll err:", err)
				continue
			}

			var share ibecommon.TrustPointShare
			err = json.Unmarshal(respBody, &share)

			if err != nil {
				log.Println("respBody json.Unmarshal err:", err)
				continue
			}

			commit, proof, ppubShare, skShare := new(bls.G1), new(bls.G1), new(bls.G1), new(bls.G2)
			err = commit.SetBytes(share.Commit)
			if err != nil {
				continue
			}
			err = proof.SetBytes(share.Proof)
			if err != nil {
				continue
			}
			err = ppubShare.SetBytes(share.PPubShare)
			if err != nil {
				continue
			}
			err = skShare.SetBytes(share.Share)
			if err != nil {
				log.Println("getSk() err share: ", err)
				continue
			}
			//log.Println(share.SendIdx, " kgc generate user.sk share:\n", b)

			x := new(bls.Scalar)
			x.SetUint64(uint64(share.SendIdx))
			//零知识证明验证公钥片
			//TODO 零知识证明
			if !user.Kzg.CheckProofSingleG1((*kzg_bls.G1Point)(commit), (*kzg_bls.G1Point)(proof), (*kzg_bls.Fr)(x), (*kzg_bls.G1Point)(ppubShare)) {
				log.Println(share.SendIdx, "kgc: fail ZK_SNARK, wrong share")
				continue
			}
			//e(PPub, Qid) = e(G1, sk)
			if !bls.Pair(ppubShare, user.Qid).IsEqual(bls.Pair(bls.G1Generator(), skShare)) {
				log.Println(share.SendIdx, "kgc: send err sk")
				continue
			}

			skShares[shareCnt] = common.BlsG2PrimaryShare{Index: share.SendIdx, Value: *skShare}
			ppubShares[shareCnt] = common.BlsG1PrimaryShare{Index: share.SendIdx, Value: *ppubShare}
			shareCnt++
			vis[idx] = true

			err = resp.Body.Close()
			if err != nil {
				log.Println("resp.Body.Close() err: ", err)
				continue
			}

			if shareCnt >= user.Threshold {
				break
			}
		}
	}
	user.sk = THSecretShare.BlsG2LagrangeScalar(skShares, user.Threshold)
	user.PPub = THSecretShare.BlsG1LagrangeScalar(ppubShares, user.Threshold)

	log.Println("PPub : \n", user.PPub)
	log.Println("user.sk:\n", user.sk)
}

func (user *User) GetSk() []byte {
	return user.sk.Bytes()
}

// Encrypt M must be 32 bytes.
func (user *User) Encrypt(receiverID string, M []byte) (ibecommon.Ciphertext, error) {

	if len(M) != 32 {
		return ibecommon.Ciphertext{}, fmt.Errorf("error length of message\n")
	}

	r := new(bls.Scalar)
	err := r.Random(rand.Reader)
	if err != nil {
		return ibecommon.Ciphertext{}, fmt.Errorf("error while generating random scalar: %v", err)
	}

	//随机数
	rP := new(bls.G1)
	rP.ScalarMult(r, user.P)

	// Second part of ciphertext
	idBytes := []byte(receiverID)
	rcvQid := ibecommon.H12(idBytes)

	pairRcvQidPPub := bls.Pair(user.PPub, rcvQid)
	pairRcvQidPPubExpR := new(bls.Gt)
	pairRcvQidPPubExpR.Exp(pairRcvQidPPub, r)

	h := ibecommon.H2(pairRcvQidPPubExpR)
	log.Println("e(Qid, PPub)^r: ", h)

	return ibecommon.Ciphertext{
		U: rP,
		V: ibecommon.XorBytes(M, h),
	}, nil
}

func (user *User) Decrypt(c *ibecommon.Ciphertext) []byte {
	log.Println("e(rP, sk): ", ibecommon.H2(bls.Pair(c.U, user.sk)))
	return ibecommon.XorBytes(c.V, ibecommon.H2(bls.Pair(c.U, user.sk)))
}

// Signature 签名
func (user *User) Signature(M []byte) (ibecommon.TransSignatureText, error) {

	if len(M) != 32 {
		return ibecommon.TransSignatureText{}, fmt.Errorf("error length of message\n")
	}

	r := new(bls.Scalar)
	err := r.Random(rand.Reader)
	if err != nil {
		return ibecommon.TransSignatureText{}, fmt.Errorf("error while generating random scalar: %v", err)
	}

	rPPub := new(bls.G1)
	err = rPPub.SetBytes(user.PPub.Bytes())
	if err != nil {
		log.Println("rPPub.SetBytes(user.PPub.Bytes()) err", err)
		return ibecommon.TransSignatureText{}, err
	}
	rPPub.ScalarMult(r, rPPub)

	skP := bls.Pair(user.P, user.sk)
	skPExpR := new(bls.Gt)
	skPExpR.Exp(skP, r)

	h := ibecommon.H2(skPExpR)

	sig := ibecommon.SignatureText{
		U: rPPub,
		V: ibecommon.XorBytes(M, h),
	}

	return ibecommon.TransSignatureText{
		U: sig.U.Bytes(),
		V: sig.V,
	}, nil
}

// SignatureVerify 验签
func (user *User) SignatureVerify(senderID string, sigBytes *ibecommon.TransSignatureText, M []byte) bool {

	uu := new(bls.G1)
	err := uu.SetBytes(sigBytes.U)
	if err != nil {
		log.Println("uu.SetBytes(transCiphertext.U) err: ", err)
		return false
	}
	sig := &ibecommon.SignatureText{U: uu, V: sigBytes.V}
	senderQId := ibecommon.H12([]byte(senderID))
	verify := ibecommon.XorBytes(sig.V, ibecommon.H2(bls.Pair(sig.U, senderQId)))
	if len(verify) != len(M) {
		return false
	}
	log.Println("verify: ", verify, "\nM: ", M)
	for i, v := range verify {
		if v != M[i] {
			return false
		}
	}
	return true
}
