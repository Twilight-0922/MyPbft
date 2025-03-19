package main

import (
	"IBEUser/pkg/constant"
	"IBEUser/pkg/controller"
	"IBEUser/pkg/handle"
	"IBEUser/src/IBE/user"
	"errors"
	"log"
	"net"
)

var u user.User

func process(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("conn.Close() err: ", err)
		}
	}(conn)
	log.Printf("服务端: %T\n", conn)

	recvByte := make([]byte, 0)
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf[:])
		if err != nil {
			log.Println("read from client failed, err: ", err)
			return
		}
		for i := 0; i < n; i++ {
			recvByte = append(recvByte, buf[i])
		}
		if n < 1024 {
			break
		}
	}

	log.Println("recv length: ", len(recvByte))
	if len(recvByte) < constant.MINLENGTH || recvByte[1] != constant.SEPARATOR {
		// 从客户端读取数据的过程中发生错误
		log.Println("read from client failed, err msg length or operation")
		return
	}

	recv := recvByte[2:]
	log.Println("client operation type: ", recvByte[0]-'0')
	switch recvByte[0] {
	case constant.INIT:
		controller.UserInit(&u, conn, recv)
		break
	case constant.ENCRYPT:
		controller.Encrypt(&u, conn, recv)
		break
	case constant.DECRYPT:
		controller.Decrypt(&u, conn, recv)
		break
	default:
		handle.ErrHandle(conn, errors.New("err operation type"))
	}
}

func main() {
	// 监听当前的tcp连接
	listen, err := net.Listen("tcp", "10.25.2.235:8001")
	log.Printf("服务端: %T=====\n", listen)
	if err != nil {
		log.Println("listen failed, err:", err)
		return
	}
	for {
		conn, err := listen.Accept() // 建立连接
		log.Println("当前建立了tcp连接")
		if err != nil {
			log.Println("accept failed, err:", err)
			continue
		}
		go process(conn)
	}
}
