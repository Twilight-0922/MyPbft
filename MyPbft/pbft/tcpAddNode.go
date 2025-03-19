package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
)

// 待加入节点使用的tcp监听
func AddJoinNodeTcpListen() {
	defer func() {
		err := recover()
		if err != nil {
			log.Println("tcp连接建立或监听连接出错！")
		}
	}()
	listen, err := net.Listen("tcp", nodeTable[len(nodeTable)-1])
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", nodeTable[len(nodeTable)-1])
	defer listen.Close()

	for {
		conn, err := listen.Accept() // 等待下次请求过来并建立连接
		if err != nil {
			log.Panic(err)
			continue
		}

		go handler(conn) //启动一个协程去处理连接

		if AgreeJoinNum >= 2*f { //达到要求就跳出循环 关闭连接
			break
		}
	}
}

func handler(conn net.Conn) {
	defer conn.Close()
	defer func() {
		err := recover()
		if err != nil {
			log.Println("发来的消息长度不正确！")
		}
	}()
	for {
		buffer := make([]byte, 0)
		buffer2 := make([]byte, 1024)
		buffer3 := make([]byte, 1024)
		length := 0
		L, err := conn.Read(buffer3[:])
		if err != nil {
			//fmt.Println("recv failed1, err:", err)
			return
		}
		size, err := strconv.Atoi(string(buffer3[:L]))
		if err != nil {
			fmt.Println("接收的不是长度：", err)
			return
		}
		conn.Write([]byte("ok"))
		//fmt.Println("回复了ok！")
		for {
			l, err := conn.Read(buffer2[:])
			if err != nil {
				fmt.Println("recv failed3, err:", err)
				return
			}
			//fmt.Println(l)
			buffer = append(buffer, buffer2[:l]...)
			length += l
			//fmt.Println(length)
			if length == size+2 {
				//fmt.Println("跳出了循环读!")
				break
			}
		}
		//截断ok
		buffer = buffer[2:]
		//fmt.Println(string(buffer))
		length -= 2
		//fmt.Println("发来的消息长度为：" + strconv.Itoa(length))
		//fmt.Println(string(buffer))
		_, content := splitMessage(buffer[:length])
		if string(content) == "true" {
			lock.Lock()
			AgreeJoinNum++
			lock.Unlock()
		}
	}
}
