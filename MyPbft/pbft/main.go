package main

import (
	"IBEUser/src/IBE/user"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var nodeCount int //节点数量
var f int         //最大容错量

var M_node = 0 //主节点编号

const Ctime = time.Second * 55 //允许的最大共识时间
const K = 200                  //请求数量
const L = 400                  //

var FlagTime = true
var Firsttime = true
var FlagDB = true

// 客户端的监听地址
var clientAddr = "127.0.0.1:8888"

// 节点池，主要用来存储监听地址
var nodeTable map[int]string

var ThresholdMissChips = 1

// 同意节点加入的数目
var AgreeJoinNum = 0

// 对上面的同意节点加入的数目变量上锁
var lock sync.Mutex

func main() {
	//首先读取节点ip的配置
	nodeTable = Init()
	nodeCount = len(nodeTable)
	f = (nodeCount - 1) / 3

	testClient() //模拟给主节点发送请求消息

	//判断节点是否是后来要主动加入的节点
	var IsJoinNode int
	fmt.Println("输入1表示节点是否是后来要主动加入的节点；输入其他数字的表示不是后来主动加入的节点")
	fmt.Scan(&IsJoinNode)
	if IsJoinNode == 1 {
		//向其他节点广播加入请求
		for i := 0; i < len(nodeTable)-1; i++ {
			message := jointMessage(cRequestAddNode, []byte(nodeTable[len(nodeTable)-1]))
			go tcpDial(message, nodeTable[i])
		}
		fmt.Println("等待其他节点同意本节点加入...")
		go AddJoinNodeTcpListen() //开启节点监听
		for AgreeJoinNum <= 2*f {
		}
		fmt.Println("节点成功加入系统...")
	}

	var P *pbft

	if len(os.Args) != 2 {
		log.Panic("输入的参数有误！")
	}
	nodeID := os.Args[1]
	nodeid, _ := strconv.Atoi(nodeID)
	if addr, ok := nodeTable[nodeid]; ok {
		use := user.User{}
		use.Init(strconv.Itoa(nodeid), "./conf/userconfig.yaml")
		//fmt.Println("节点" + strconv.Itoa(nodeid) + "的私钥为：")
		//fmt.Println(use.GetSk())

		p := NewPBFT(nodeid, addr, &use)
		P = p
		go p.tcpListen() //启动节点
	} else {
		log.Fatal("无此节点编号！")
	}

	//处理因网络等原因的准备消息和确认消息以及防止主程序退出
	select {
	case data := <-P.preMessageChan:
		P.handlePrepare(data)
	case data := <-P.comMessageChan:
		P.handleCommit(data)
	case data := <-P.addNodeMessageChan:
		P.handleAddNode(data)
	}
}

// 充当客户端--给主节点发送请求消息
func testClient() {
	// 读取JSON文件
	filePath := "recclient.json"
	jsonFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	//解析JSON数据
	var request Request
	err = json.Unmarshal(jsonFile, &request)
	if err != nil {
		log.Fatal(err)
	}

	//生成公私钥生成用户
	use := user.User{}
	clientID := "12D3KooWNJXEVQ4yotd41Z6y3m19yf68jyt1ToorqNQUBecVp2VF"
	use.Init(clientID, "./conf/userconfig.yaml")

	//request.Digest = GetDigest(request)
	sign := request.Data.FileName + request.Timestamp
	digest := GetDigest(sign)
	Signature, err := use.Signature([]byte(digest))
	if err != nil {
		log.Panic(err)
	}
	request.Signature = Signature //将签名重新赋值
	fmt.Println("客户端的签名为：" + hex.EncodeToString(request.Signature.V) + "  " + hex.EncodeToString(request.Signature.U))
	//将request消息序列化
	b, err := json.Marshal(request)
	if err != nil {
		log.Panic(err)
	}

	data := jointMessage(cRequest, b)
	//addr := nodeTable[M_node] //发送消息的地址
	addr := "10.25.9.8:9000"
	tcpDial(data, addr)

}

func Init() map[int]string {
	defer func() {
		err := recover()
		if err != nil {
			log.Println("读取配置文件错误！")
		}
	}()
	var v = viper.New()
	v.SetConfigName("nodeconfig")            // 配置文件名称(无扩展名)
	v.SetConfigType("yaml")                  // 如果配置文件的名称中没有扩展名，则需要配置此项
	v.AddConfigPath("./conf/")               // 查找配置文件所在的路径
	if err := v.ReadInConfig(); err != nil { // 查找并读取配置文件并处理配置文件的错误
		log.Panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var conf Config
	err := v.Unmarshal(&conf)
	if err != nil {
		log.Println(err)
	}
	fmt.Println(conf)
	nodeTable := make(map[int]string)
	for _, v := range conf.NodesTable {
		nodeTable[v.Nodeid] = v.Ip + ":" + strconv.Itoa(v.Port)
	}
	return nodeTable
}

// 查询一个db数据库中有多少bucket(表)
func testBucket() {
	db, err := bolt.Open("RecToHash.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		// 遍历所有的 bucket
		err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			// 获取 bucket 的句柄
			log.Printf("Bucket name: %s", name)
			return nil
		})
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

}
