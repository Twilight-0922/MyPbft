package controller

import (
	"IBEUser/pkg/constant"
	"IBEUser/pkg/handle"
	"IBEUser/src/IBE/ibecommon"
	"IBEUser/src/IBE/user"
	"encoding/json"
	"errors"
	bls "github.com/cloudflare/circl/ecc/bls12381"
	"log"
	"net"
)

func UserInit(u *user.User, conn net.Conn, recv []byte) {
	err := u.Init(string(recv), "userconfig.yaml")

	if err != nil {
		handle.ErrHandle(conn, err)
		return
	}

	_, err = conn.Write(u.GetSk())
	if err != nil {
		handle.ErrHandle(conn, err)
		return
	}

	log.Println("init success")
}

func Encrypt(u *user.User, conn net.Conn, recv []byte) {
	id, buf, n, m := [1024]byte{0}, make([]byte, constant.ENCRYPTLENGTH), 0, 0
	for ; n < len(recv) && recv[n] != constant.SEPARATOR; n++ {
		id[n] = recv[n]
	}
	if n == len(recv) || recv[n] != constant.SEPARATOR {
		handle.ErrHandle(conn, errors.New("err msg context"))
		return
	}
	idStr := string(id[:n])
	n++
	for ; n < len(recv) && m < constant.ENCRYPTLENGTH; n++ {
		buf[m] = recv[n]
		m++
	}

	encrypt, err := u.Encrypt(idStr, buf)
	if err != nil {
		handle.ErrHandle(conn, err)
		return
	}
	transEncrypt := ibecommon.TransCiphertext{U: encrypt.U.Bytes(), V: encrypt.V}
	marshal, err := json.Marshal(transEncrypt)
	if err != nil {
		log.Println("json.Marshal(encrypt) err: ", err)
		return
	}
	_, err = conn.Write(marshal)
	if err != nil {
		log.Println("conn.Write(marshal) err: ", err)
		return
	}
	log.Println("encrypt: ", encrypt)
}

func Decrypt(u *user.User, conn net.Conn, recv []byte) {
	transCiphertext := new(ibecommon.TransCiphertext)
	err := json.Unmarshal(recv, &transCiphertext)
	if err != nil {
		log.Println("json.Unmarshal([]byte(recvStr), &ciphertext) err: ", err)
		return
	}
	uu := new(bls.G1)
	err = uu.SetBytes(transCiphertext.U)
	if err != nil {
		log.Println("uu.SetBytes(transCiphertext.U) err: ", err)
		return
	}
	ciphertext := &ibecommon.Ciphertext{U: uu, V: transCiphertext.V}
	log.Println("ciphertext: ", ciphertext)
	decrypt, n := u.Decrypt(ciphertext), 0
	for ; n < len(decrypt); n++ {
		if decrypt[n] == 0 {
			break
		}
	}
	_, err = conn.Write(decrypt[:n])
	if err != nil {
		log.Println("conn.Write(marshal) err: ", err)
		return
	}
	log.Println("decrypt", decrypt)
}
