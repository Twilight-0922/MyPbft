package user

import (
	"log"
	"testing"
)

func TestUser_Init(t *testing.T) {
	user := new(User)
	err := user.Init("IAmUser", "userconfig.yaml")
	if err != nil {
		log.Println("init err: ", err)
		return
	}
}

func TestUser_Encrypt(t *testing.T) {

	user := new(User)
	err := user.Init("IAmUser", "userconfig.yaml")
	if err != nil {
		log.Println("init err: ", err)
		return
	}

	//log.Println(user.PPub.String())
	M := make([]byte, 32)
	copy(M, []byte("HELLO,WORLD!"))

	encrypt, err := user.Encrypt(user.ID, M)
	if err != nil {
		log.Println(err)
		return
	}
	decrypt := user.Decrypt(&encrypt)
	log.Println("plaintext: ", string(M))
	log.Println("ciphertext:\n U:\n", encrypt.U, "\nV: ", encrypt.V)
	log.Println("decrypt: ", string(decrypt))
}

func TestUser_Signature(t *testing.T) {
	user := new(User)
	err := user.Init("IAmUser", "userconfig.yaml")
	if err != nil {
		log.Println("init err: ", err)
		return
	}
	msg := make([]byte, 32)
	for i, v := range "hello,world!" {
		msg[i] = byte(v)
	}
	signature, err := user.Signature(msg)

	if err != nil {
		log.Println("Signature err: ", err)
		return
	}

	log.Println(user.SignatureVerify("IAmUser", &signature, msg))
	signature, _ = user.Signature(msg)
	log.Println(user.SignatureVerify("IAmUser", &signature, msg))
	signature, _ = user.Signature(msg)
	log.Println(user.SignatureVerify("IAmUser", &signature, msg))
}
