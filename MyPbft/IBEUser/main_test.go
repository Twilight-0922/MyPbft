package main

import (
	"IBEUser/src/IBE/user"
	"log"
	"testing"
)

func TestUser_Init(t *testing.T) {

	user := new(user.User)
	err := user.Init("IAmUser", "F:\\golang\\DistributedBFIBE\\DistributedBFIBE\\DistributedBFIBE\\conf\\userconfig.yaml")
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
