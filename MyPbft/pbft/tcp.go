package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
)

// var mp map[string]int
var mp = make(map[string]int)
var mp2 = make(map[string]bool)

// 客户端使用的tcp监听
func clientTcpListen() {
	listen, err := net.Listen("tcp", clientAddr)
	if err != nil {
		log.Panic(err)
	}
	defer listen.Close()

	r := new(Reply2)
	r2 := new(Request)
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err3 := ioutil.ReadAll(conn)
		cmd, content := splitMessage(b)
		if err3 != nil {
			log.Panic(err3)
		} else {
			switch command(cmd) {
			case cReply: //客户端收到回复时
				isReply(r, content)
				fmt.Println(string(content)) //打印reply消息
			case cRequest: //客户端收到重新发送消息的要求时
				err5 := json.Unmarshal(content, r2)
				if err5 != nil {
					log.Panic(err5)
				}
				*(&M_node) = (M_node + 1) % nodeCount //更换主节点
				fmt.Println("主节点为：" + strconv.Itoa(M_node))
				//fmt.Println(r2.Timestamp)
				fmt.Println("请重新输入要发送的数据:")
				//stdReader := bufio.NewReader(os.Stdin)
				//SendMessage(stdReader, r2.Timestamp)
			}
		}
	}
}

// 是回复时
func isReply(r *Reply2, b []byte) {
	defer handleCecover()

	err2 := json.Unmarshal(b, r)
	if err2 != nil {
		log.Panic(err2)
	}
	mp[r.Timestamp]++
	//fmt.Println(mp[r.Timestamp])
	if mp[r.Timestamp] > 2*f && !mp2[r.Timestamp] {
		fmt.Println("时间戳为：" + r.Timestamp + "的消息已达成共识")
		mp2[r.Timestamp] = true
	}
}

// 节点使用的tcp监听
func (p *pbft) tcpListen() {
	defer func() {
		err := recover()
		if err != nil {
			log.Println("tcp连接建立或监听连接出错！")
		}
	}()

	listen, err := net.Listen("tcp", p.node.addr)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", p.node.addr)
	defer listen.Close()

	for {
		conn, err := listen.Accept() // 等待下次请求过来并建立连接
		if err != nil {
			log.Panic(err)
			continue
		}

		go p.handler(conn) //启动一个协程去处理连接
	}
}

// 处理相应的连接
func (p *pbft) handler(conn net.Conn) {
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
		fmt.Println("发来的消息长度为：" + strconv.Itoa(length))
		//fmt.Println(string(buffer))
		cmd, _ := splitMessage(buffer[:length])
		if cmd == "request" {
			go p.handleRequest(buffer[:length], conn)
		} else {
			p.handleRequest(buffer[:length], conn)
		}
	}
}

// 使用tcp发送消息
func tcpDial(context []byte, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("connect error", err)
		return
	}
	//先发长度
	_, err = conn.Write([]byte(strconv.Itoa(len(context)))) //发送数据长度
	if err != nil {
		log.Println("Write error", err)
		return
	}
	//接受对方发来的ok
	buffer := make([]byte, 1024)
	l, err := conn.Read(buffer[:])
	if err != nil {
		fmt.Println("recv failed, err:", err)
		return
	}
	if string(buffer[:l]) == "ok" {
		_, err = conn.Write([]byte("ok")) //接收到对方发来的ok后也向对方发送一个ok 然后再发送数据
		_, err = conn.Write(context)      //发送数据
		fmt.Println("已经给" + addr + "发送了数据")
		fmt.Println("发送的数据长度为：" + strconv.Itoa(len(context)))
		if err != nil {
			log.Println("Write error", err)
			return
		}
	} else {
		fmt.Println("发送消息失败")
	}
	//_, err = conn.Write(context) //发送数据
	//fmt.Println("已经给" + addr + "发送了数据")
	//fmt.Println("发送的数据长度为：" + strconv.Itoa(len(context)))
	//if err != nil {
	//	log.Println("Write error", err)
	//	return
	//}
	conn.Close()
}
