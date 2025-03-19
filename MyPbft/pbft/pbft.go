package main

import (
	"IBEUser/src/IBE/user"
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
var localMessagePool []Request

type node struct {
	//节点ID
	nodeID int
	//节点监听地址
	addr string
	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte
	//节点收到消息的开始时间
	start time.Time
}

type pbft struct {
	node                node                    //节点信息
	sequenceID          int                     //每笔请求自增序号
	use                 user.User               //节点的user信息
	h                   int                     //低水位
	H                   int                     //高水位
	lock                sync.Mutex              //锁
	arrayTodigest       []string                //记录消息切片的摘要信息
	arrayToupdata       []string                //记录UpDataCount消息的摘要
	messagePool         map[string][]Request    //临时消息池，消息摘要对应消息本体--根据消息切片的摘要
	prePareConfirmCount map[string]map[int]bool //存放收到的prepare数量(至少需要收到并确认2f个)，根据消息切片摘要来对应
	commitConfirmCount  map[string]map[int]bool //存放收到的commit数量（至少需要收到并确认2f+1个），根据消息切片摘要来对应
	addNodeCount        map[string]map[int]bool //存放收到的AddNode数量(至少需要收到并确认2*f+个)，根据要增加节点的ip+port来对应
	replyConfireCount   map[string]int          //存放收到的reply数量(至少需要收到并确认2f+1个)，根据消息切片摘要来对应
	viewchangeCount     map[string]int          //存放收到的ViewChange数量 -- 根据消息切片的摘要来对应
	newdataCount        map[string]int          //存放收到的newdata数量 -- 根据UpDataCount消息的摘要来对应
	isnewdata           map[string]bool         //需要更新的消息是否已经更新 -- 根据UpDataCount消息的摘要来对应
	isCommitBordcast    map[string]bool         //该笔消息是否已进行Commit广播 -- 根据消息切片的摘要来对应
	isAddNodeBordcast   map[string]bool         //该笔消息是否已经进行增加节点广播 -- 根据IpAndPort来对应
	isReplyToAddNode    map[string]bool         //该笔消息是否已经对要增加节点进行回复 -- 根据IpAndPort来对应
	isReply             map[string]bool         //该笔消息是否已对主节点进行Reply -- 根据消息切片的摘要来对应
	isMReply            map[string]bool         //主节点是否以对客户端进行reply -- 根据消息切片的摘要来对应
	isViewChange        map[string]bool         //节点是否收集到足够的视图切换消息 -- 根据消息切片的摘要来对应
	checkpointCount     map[int]int             //节点记录收到的checkpoint数量
	conn                map[string]*net.Conn    //主节点记录客户端发来的tcp连接 以便后面返回reply消息
	reqMessageChan      chan Request            //主节点将收到的客户端发来的request请求放到通道里
	preMessageChan      chan []byte             //节点收到其他节点发来的准备消息放到这里
	comMessageChan      chan []byte             //节点收到其他节点发来的确认消息放到这里
	addNodeMessageChan  chan []byte             //节点收到其他节点发来的增加节点消息放到这里
	rm                  []Request               //主节点存储一轮共识的请求消息集合
}

func NewPBFT(nodeID int, addr string, use *user.User) *pbft {
	p := new(pbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	//p.node.rsaPrivKey = p.getPivKey(nodeID) //从生成的私钥文件处读取
	//p.node.rsaPubKey = p.getPubKey(nodeID)  //从生成的公钥文件处读取
	p.node.rsaPrivKey = []byte(use.GetSk())         //从生成的私钥文件处读取
	p.node.rsaPubKey = []byte(strconv.Itoa(nodeID)) //从生成的公钥文件处读取
	p.sequenceID = 0
	p.use = *use
	p.h = 0
	p.H = L
	p.arrayTodigest = make([]string, K)
	p.arrayToupdata = make([]string, K)
	p.messagePool = make(map[string][]Request)
	p.prePareConfirmCount = make(map[string]map[int]bool)
	p.commitConfirmCount = make(map[string]map[int]bool)
	p.addNodeCount = make(map[string]map[int]bool)
	p.replyConfireCount = make(map[string]int)
	p.viewchangeCount = make(map[string]int)
	p.newdataCount = make(map[string]int)
	p.isnewdata = make(map[string]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isAddNodeBordcast = make(map[string]bool)
	p.isReplyToAddNode = make(map[string]bool)
	p.isReply = make(map[string]bool)
	p.isMReply = make(map[string]bool)
	p.isViewChange = make(map[string]bool)
	p.checkpointCount = make(map[int]int)
	p.conn = make(map[string]*net.Conn)
	p.reqMessageChan = make(chan Request, 100) //通道缓冲区大小设置为100
	p.preMessageChan = make(chan []byte, 1000)
	p.comMessageChan = make(chan []byte, 1000)
	p.addNodeMessageChan = make(chan []byte, 1000)
	p.rm = make([]Request, 0) //分配长度为0
	return p
}

func (p *pbft) handleRequest(data []byte, conn net.Conn) {
	//切割消息，根据消息命令调用不同的功能
	cmd, content := splitMessage(data)
	switch command(cmd) {
	case cRequest:
		p.handleClientRequest(content, &conn)
	//case cRequests:
	//	p.handleRequests()
	case cPrePrepare:
		p.handlePrePrepare(content)
	case cPrepare:
		p.handlePrepare(content)
	case cCommit:
		p.handleCommit(content)
	case cReply:
		p.handleReply(content)
	case cCheckPoint:
		p.handleCheckPoint(content)
	case cUpdata:
		p.handleUpdata(content)
	case cNewdata:
		p.handleNewdata(content)
	case cViewChange:
		p.handleViewChange(content)
	case cNewView:
		p.handleNewView(content)
	case cRepair:
		p.handleSearchNodeToHash(content, conn)
	case cRecovery:
		p.handleUpdataNodeToHash(content, conn)
	case cRecToMeta:
		p.handleSearchRecToHash(content, conn)
	case cRequestAddNode:
		p.handleRequestAddNode(content)
	case cAddNode:
		p.handleAddNode(content)
	default:
		p.handleFind(data, conn)
	}
}

// 处理增加节点请求
func (p *pbft) handleRequestAddNode(content []byte) {
	defer handleCecover()

	IpAndPort := string(content)
	//将收到的ip+port消息+1
	p.setaddNodeConfirmMap(IpAndPort, p.node.nodeID, true)

	s := strings.Split(IpAndPort, ":")
	ip, port := s[0], s[1]
	Port, err := strconv.Atoi(port)
	if err != nil {
		fmt.Println("不能将string类型的端口号转化为int类型")
		log.Panic(err)
	}

	message := nodes{p.node.nodeID, ip, Port}
	data, err := json.Marshal(message)
	if err != nil {
		log.Panic(err)
	}
	//向其他节点进行广播增加节点消息
	p.broadcast(cAddNode, data)
	p.isAddNodeBordcast[IpAndPort] = true
	fmt.Println("已经向其他节点广播增加节点消息")
}

// 系统内部处理是否增加节点
func (p *pbft) handleAddNode(content []byte) {
	defer handleCecover()

	node := new(nodes)
	//使用json解析出node结构体
	err := json.Unmarshal(content, node)
	if err != nil {
		log.Panic(err)
	}
	port := strconv.Itoa(node.Port)
	IpAndPort := node.Ip + ":" + port
	if !p.isAddNodeBordcast[IpAndPort] {
		p.addNodeMessageChan <- content
		return
	}

	//根据收到的发送消息节点id作为键+1
	p.lock.Lock()
	p.setaddNodeConfirmMap(IpAndPort, node.Nodeid, true)
	p.lock.Unlock()

	count := 0
	for range p.addNodeCount[IpAndPort] { //计算收到多少个IpAndPort需要增加节点的消息
		count++
	}
	p.lock.Lock()
	if count > 2*f && !p.isReplyToAddNode[IpAndPort] {
		fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的AddNode信息 ...")
		nodeCount++                        //总节点数+1
		f = (nodeCount - 1) / 3            //更新系统可以容忍的恶意节点数量
		nodeTable[nodeCount-1] = IpAndPort //更新节点的配置信息--主要是增加节点序号对应的ip+port
		//对待增加节点进行回复
		message := jointMessage(cReplyToAddNode, []byte("true"))
		tcpDial(message, IpAndPort) //回复true表示该节点认可加入
		p.isReplyToAddNode[IpAndPort] = true
		fmt.Println("系统中的总节点数量为：" + strconv.Itoa(nodeCount))
	}
	p.lock.Unlock()
}

// 更新NodeToHash数据库
func (p *pbft) handleUpdataNodeToHash(content []byte, conn net.Conn) {
	go p.handleClientRequest(content, &conn) //开始共识

	//使用json解析出Request结构体
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	//首先根据datahash找到切片hash
	//GRdig := Search("test1.db", []byte("table1"), []byte(r.Data.DataHash))
	GRdig := Search("ToGRdigest.db", []byte("DataHashToGRdigest"), []byte(r.Data.DataHash))
	if GRdig == nil {
		fmt.Println("查找失败！")
		return
	}
	//再根据切片hash找到对应的切片信息
	//gr := Search("test2.db", []byte("table1"), GRdig)
	gr := Search("ToGR.db", []byte("GRdigestToGR"), GRdig)
	if gr == nil {
		fmt.Println("查找失败！")
		return
	}
	GR := new(GRequest)
	err = json.Unmarshal(gr, GR)
	if err != nil {
		log.Panic(err)
	}
	s := []string{}
	//查找不同的nodeid
	for _, v := range GR.Requests {
		if v.Data.DataHash == r.Data.DataHash { //找到了具体的请求消息
			for k, v2 := range v.Data.StoreNode {
				v3 := r.Data.StoreNode[k]
				for i := 0; i < len(v3); i++ {
					if v3[i] != v2[i] {
						s = append(s, v2[i])
					}
				}
			}
		}
	}
	db, err := bolt.Open("NodeToHash.db", 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//删除对应的bucket
	for _, v := range s {
		fmt.Println("需要删除的桶名为：" + v)
		db.Update(func(tx *bolt.Tx) error {
			if err = tx.DeleteBucket([]byte(v)); err != nil {
				log.Println("删除桶名为："+v+"失败：", err)
				return err
			}
			return nil
		})
	}
	//for _, v := range s {
	//	db.View(func(tx *bolt.Tx) error {
	//		b := tx.Bucket([]byte(v))
	//		if b == nil { //表不存在时
	//			return nil
	//		}
	//
	//		b.ForEach(func(k, v2 []byte) error {
	//			err := b.Delete(k)
	//			if err != nil {
	//				return err
	//			}
	//			return nil
	//		})
	//		return nil
	//	})
	//}
	fmt.Println("删除bucket成功！")
}

// 处理客户端发来的请求
func (p *pbft) handleClientRequest(content []byte, conn *net.Conn) {
	defer handleCecover()
	if FlagTime {
		p.node.start = time.Now() //主节点收到消息的开始时间
		//fmt.Println("主节点收到请求消息的时间：")
		//fmt.Println(p.node.start)
		FlagTime = false
		Firsttime = true
	}

	fmt.Println("主节点" + strconv.Itoa(M_node) + "已接收到客户端发来的request ...")
	//使用json解析出Request结构体
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	//将客户端发来的消息写到文件里
	//ToFile(content, "recclient.json")
	//将客户端发来的request请求放入主节点的通道里
	p.lock.Lock()
	p.reqMessageChan <- *r
	p.lock.Unlock()
	//用来记录客户端发来的tcp连接
	p.conn[r.Data.NodeID] = conn
	//将发来的消息收集起来
	p.lock.Lock()
	p.rm = append(p.rm, <-p.reqMessageChan)
	p.lock.Unlock()

	if Firsttime {
		Firsttime = false
		for time.Now().Sub(p.node.start) < time.Second*55 {
		}
		FlagTime = true
		p.handleRequests()
	}
}

//将数据写到outpath中
//func ToFile(data []byte, outpath string) {
//	os.WriteFile(outpath, data, 0755)
//}

func (p *pbft) handleRequests() {
	defer handleCecover()
	r := new(GRequest)
	r.Requests = p.rm

	// 将记录的请求消息集合置为空
	p.lock.Lock()
	p.rm = []Request{}
	p.lock.Unlock()
	//a, err := json.Marshal(*r)
	//if err != nil {
	//	log.Panic(err)
	//}
	////使用json解析出Request结构体
	//r := new(GRequest)
	//err2 := json.Unmarshal(a, r)
	//if err2 != nil {
	//	log.Panic(err2)
	//}

	//获取消息摘要
	digest := GetDigest(r.Requests)
	fmt.Println("已将request存入临时消息池")
	//存入临时消息池
	p.messagePool[digest] = r.Requests
	p.arrayTodigest[p.sequenceID%K] = digest
	//主节点对消息摘要进行签名
	signInfo, err := p.use.Signature([]byte(digest))
	if err != nil {
		log.Panic(err)
	}

	//视图id
	viewid := M_node
	//消息序号加1
	p.sequenceIDAdd()
	//拼接成PrePrepare，准备发往follower节点
	//digest = digest + "1" //TODO:这里模拟恶意行为
	pp := PrePrepare{r.Requests, digest, p.sequenceID, signInfo, viewid}
	//fmt.Println("主节点在pre-prepare阶段计算的消息摘要为:" + digest)
	b, err := json.Marshal(pp)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("正在向其他节点进行进行PrePrepare广播 ...")
	//进行PrePrepare广播
	p.broadcast(cPrePrepare, b)
	fmt.Println("PrePrepare广播完成")
}

// 处理预准备消息
func (p *pbft) handlePrePrepare(content []byte) {
	defer handleCecover()
	p.node.start = time.Now() //非主节点收到消息的开始时间
	fmt.Println("本节点已接收到主节点发来的PrePrepare ...")
	//使用json解析出PrePrepare结构体
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}

	//ToFile(content, "recMain.json")

	//首先判断发来的消息的消息序号是否与节点自身的消息序号相等
	if p.sequenceID+1 < pp.SequenceID { //如果节点自身的消息序号小于发来的消息序号，说明自身已经落后
		fmt.Println("节点的消息序号为：" + strconv.Itoa(p.sequenceID) + "  发来的消息的序号为：" + strconv.Itoa(pp.SequenceID) + "   节点的消息序号小于发来的消息序号，可能落后于系统，进行数据更新")
		c := UpData{p.sequenceID, p.node.nodeID, pp.SequenceID - 1}
		bc, err := json.Marshal(c)
		if err != nil {
			log.Panic(err)
		}
		p.broadcast(cUpdata, bc)
	}
	//获取主节点的公钥，用于数字签名验证
	//primaryNodePubKey := p.getPubKey(M_node)
	//digestByte, _ := hex.DecodeString(pp.Digest) //主节点发来的消息里的摘要

	pp2, _ := json.Marshal(pp.RequestMessage) //序列化Request
	Digest := GetDigest(pp.RequestMessage)
	if digest := GetDigest(pp.RequestMessage); digest != pp.Digest {
		p.handleException(pp2, Digest, true, -1) //处理错误
		fmt.Println("副本节点根据发来的消息计算出来的消息摘要为:" + digest)
		fmt.Println("副本节点在prepare阶段收到主节点发来的消息摘要为：" + pp.Digest)
		fmt.Println("信息摘要对不上，拒绝进行prepare广播")
	} else if !p.use.SignatureVerify(strconv.Itoa(M_node), &pp.Sign, []byte(pp.Digest)) || !p.SignatureVerifyClient(pp) { //既验证了主节点的签名和摘要，又验证了客户端的签名和摘要
		p.handleException(pp2, Digest, true, -1) //处理错误
		fmt.Println("主节点签名验证失败！,拒绝进行prepare广播")
	} else if pp.ViewId != M_node {
		p.handleException(pp2, Digest, true, -1) //处理错误
		fmt.Println("视图ID对不上，拒绝进行prepare广播")
	} else {
		//将信息存入临时消息池
		p.messagePool[pp.Digest] = pp.RequestMessage
		p.arrayTodigest[p.sequenceID%K] = pp.Digest
		fmt.Println("已将消息存入临时节点池")

		//序号赋值
		p.sequenceID = pp.SequenceID
		//fmt.Println("在准备阶段时，节点的序号为:" + strconv.Itoa(p.sequenceID))

		//节点使用私钥对其签名
		//sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		sign, err := p.use.Signature([]byte(pp.Digest))
		if err != nil {
			log.Panic(err)
		}
		//视图id
		viewid := pp.ViewId
		//拼接成Prepare
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.nodeID, sign, viewid}
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		//进行准备阶段的广播
		fmt.Println("正在进行Prepare广播 ...")
		p.broadcast(cPrepare, bPre)
		fmt.Println("Prepare广播完成")
	}
}

func (p *pbft) handleException(content []byte, digest string, IsHandlepreprepare bool, NodeID int) {
	if IsHandlepreprepare || (!IsHandlepreprepare && NodeID == M_node) { //当是处理预准备消息或不是预准备消息但发消息的id是主节点时
		p.broadcast(cViewChange, content) //广播视图更改消息
	}
	p.lock.Lock()
	p.viewchangeCount[digest]++
	p.lock.Unlock()
}

// 处理准备消息
func (p *pbft) handlePrepare(content []byte) {
	defer handleCecover()
	//使用json解析出Prepare结构体
	pre := new(Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("本节点已接收到%d节点发来的Prepare ... \n", pre.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	//MessageNodePubKey := p.getPubKey(pre.NodeID)
	//digestByte, _ := hex.DecodeString(pre.Digest)

	pp2, _ := json.Marshal(p.messagePool[pre.Digest]) //序列化Request
	Digest := GetDigest(p.messagePool[pre.Digest])    //根据请求消息切片获得摘要
	if _, ok := p.messagePool[pre.Digest]; !ok {
		p.preMessageChan <- content
		//p.handleException(pp2, Digest, false, pre.NodeID) //处理错误
		//fmt.Println("当前临时消息池无此摘要，拒绝执行commit广播")
	} else if !p.use.SignatureVerify(strconv.Itoa(pre.NodeID), &pre.Sign, []byte(pre.Digest)) {
		p.handleException(pp2, Digest, false, pre.NodeID) //处理错误
		fmt.Println("节点签名验证失败！,拒绝执行commit广播")
	} else if pre.ViewId != M_node {
		p.handleException(pp2, Digest, false, pre.NodeID) //处理错误
		fmt.Println("视图ID对不上，拒绝进行commit广播")
	} else {
		p.lock.Lock()
		p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		p.lock.Unlock()
		count := 0
		for range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//因为主节点不会发送Prepare，所以不包含自己
		specifiedCount := 0
		if p.node.nodeID == M_node {
			specifiedCount = nodeCount / 3 * 2
		} else {
			specifiedCount = (nodeCount / 3 * 2) - 1
		}
		//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
		p.lock.Lock()
		//获取消息源节点的公钥，用于数字签名验证
		if count >= specifiedCount && !p.isCommitBordcast[pre.Digest] {
			fmt.Println("本节点已收到至少2f个节点(包括本地节点)发来的Prepare信息 ...")
			//节点使用私钥对其签名
			//sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			sign, err := p.use.Signature([]byte(pre.Digest))
			if err != nil {
				p.lock.Unlock()
				log.Panic(err)
			}
			//视图id
			viewid := pre.ViewId

			////恶意行为模拟
			//if p.node.nodeID == 0 && flag == true {
			//	p.sequenceID--
			//	flag = false
			//}
			//fmt.Println("在准备阶段，节点" + strconv.Itoa(p.node.nodeID) + "的消息序号为:" + strconv.Itoa(p.sequenceID)) //

			c := Commit{pre.Digest, p.sequenceID, p.node.nodeID, sign, viewid}
			bc, err := json.Marshal(c)
			if err != nil {
				p.lock.Unlock()
				log.Panic(err)
			}
			//进行提交信息的广播
			p.broadcast(cCommit, bc)
			p.isCommitBordcast[pre.Digest] = true
			fmt.Println("commit广播完成")

			////恢复节点序号
			//if p.node.nodeID == 0 && flag2 == true {
			//	p.sequenceID++
			//	flag2 = false
			//	fmt.Println("在准备阶段，节点0的消息序号恢复为:" + strconv.Itoa(p.sequenceID))
			//}
		}
		p.lock.Unlock()
	}

	//if time.Now().Sub(p.node.start) > Ctime {
	//	p.broadcast(cViewChange, pp2) //广播视图更改消息
	//}
}

// 处理提交确认消息
func (p *pbft) handleCommit(content []byte) {
	defer handleCecover()

	//超时模拟
	// if p.node.nodeID == 3 || p.node.nodeID == 2 {
	// 	if flag {
	// 		time.Sleep(time.Second * 2)
	// 		*(&flag) = false //只超时一次
	// 	}
	// }

	//使用json解析出Commit结构体
	c := new(Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("本节点已接收到%d节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	//MessageNodePubKey := p.getPubKey(c.NodeID)
	//digestByte, _ := hex.DecodeString(c.Digest)

	pp2, _ := json.Marshal(p.messagePool[c.Digest]) //序列化Request
	Digest := GetDigest(p.messagePool[c.Digest])    //根据请求消息切片获得摘要
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		p.comMessageChan <- content
		//p.handleException(pp2, Digest, false, c.NodeID) //处理错误
		//fmt.Println("当前prepare池无此摘要，拒绝将信息持久化到本地消息池")
	} else if !p.use.SignatureVerify(strconv.Itoa(c.NodeID), &c.Sign, []byte(c.Digest)) {
		p.handleException(pp2, Digest, false, c.NodeID) //处理错误
		fmt.Println("节点签名验证失败！,拒绝将信息持久化到本地消息池")
	} else if c.ViewId != M_node {
		p.handleException(pp2, Digest, false, c.NodeID) //处理错误
		fmt.Println("视图ID对不上，拒绝将信息持久化到本地消息池")
	} else {
		p.lock.Lock()
		p.setCommitConfirmMap(c.Digest, c.NodeID, true)
		p.lock.Unlock()
		count := 0
		for range p.commitConfirmCount[c.Digest] {
			count++
		}
		//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！
		p.lock.Lock()
		if count >= f*2 && !p.isReply[c.Digest] && p.isCommitBordcast[c.Digest] {
			//fmt.Println(time.Now().Sub(p.node.start))
			//fmt.Println(Ctime) //时间限制
			//if time.Now().Sub(p.node.start) > Ctime {
			//	fmt.Println("超时了")
			//	p.broadcast(cViewChange, pp2) //广播视图更改消息
			//	p.lock.Unlock()               //记得解锁，否则该节点p会锁死在该进程 不能对其他节点发来的消息做出回应
			//	return
			//}

			fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
			//将消息信息，提交到本地消息池中！
			localMessagePool = append(localMessagePool, p.messagePool[c.Digest]...)
			//构造reply信息
			r := Reply{p.node.nodeID, true, c.ViewId, c.Digest, c.SequenceID, p.messagePool[c.Digest]}
			br, err := json.Marshal(r)
			if err != nil {
				p.lock.Unlock()
				log.Panic(err)
			}
			//进行reply回复 -- 回复给主节点
			message := jointMessage(cReply, br)
			if p.node.nodeID != M_node {
				fmt.Println("正在进行reply回复 ...")
				tcpDial(message, nodeTable[M_node])
				fmt.Println("reply完毕")
			} else {
				p.replyConfireCount[r.Digest]++
			}
			p.isReply[c.Digest] = true

			//广播检查点消息
			if p.sequenceID%K == 0 {
				p.checkpoint(p.sequenceID, p.node.nodeID)
			}

			fmt.Println("本轮共识的消息有：" + strconv.Itoa(len(p.messagePool[c.Digest])) + "个")

			//如果是主节点 将共识的本地消息写到文件中 -- TODO:最后需要删除
			if p.node.nodeID == M_node {
				filepath := "test.txt"
				file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					fmt.Println("文件打开失败", err)
				}
				defer file.Close()

				write := bufio.NewWriter(file)
				length := len(localMessagePool)
				r, err := json.Marshal(localMessagePool[length-1])
				if err != nil {
					p.lock.Unlock()
					log.Panic(err)
				}
				write.Write(r) //序列化后的r(请求消息)写到文件中
				write.Flush()
			}
		}
		p.lock.Unlock()
	}
}

// 主节点处理其他共识节点发来的reply消息
func (p *pbft) handleReply(content []byte) {
	defer handleCecover()

	r := new(Reply)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	r.NodeID = p.node.nodeID
	p.lock.Lock()
	p.replyConfireCount[r.Digest]++
	if p.replyConfireCount[r.Digest] > f && !p.isMReply[r.Digest] {
		//回复给客户端说明共识成功结束了
		for _, v := range r.Requests {
			//对每个request请求进行序列化
			_, err2 := json.Marshal(v)
			if err2 != nil {
				p.lock.Unlock()
				log.Panic(err2)
			}
			//message2 := jointMessage(cReply, message)
			//(*p.conn[v.Message.UserID]).Write(message2)
			fmt.Println("主节点" + string(p.node.nodeID) + "已经给客户端" + v.ClientAddr + "发送reply消息了")
		}
		p.isMReply[r.Digest] = true

		p.handleBlockStorage(r) //共识成功后进行出块存储，即更新数据库

		fmt.Println("持久化存储成功")
		//fmt.Println("共识成功的时间：")
		//fmt.Println(time.Now())
		fmt.Println(r.Requests[0].Data.DataHash)
	}
	p.lock.Unlock()
}

// 出块存储，更新数据库
func (p *pbft) handleBlockStorage(r *Reply) {
	//创建表 -- 应该放在main函数中
	if FlagDB {
		//CreateDB("test1.db", []byte("table1")) //table1用来存储具体的某一个请求消息的hash到消息切片的hash映射
		//CreateDB("test1.db", []byte("table2")) //table2用来存储消息序号到客户端请求消息切片的的hash映射
		//CreateDB("test2.db", []byte("table1")) //test2中的table1用来存储消息切片的hash到具体消息切片的映射
		CreateDB("ToGRdigest.db", []byte("DataHashToGRdigest")) //DataHashToGRdigest用来存储具体的某一个请求消息的hash到消息切片的hash映射
		CreateDB("ToGRdigest.db", []byte("SeqToGRdigest"))      //SeqToGRdigest用来存储消息序号到客户端请求消息切片的的hash映射
		CreateDB("ToGR.db", []byte("GRdigestToGR"))             //ToGR.db中的GRdigestToGR用来存储消息切片的hash到具体消息切片的映射
		FlagDB = false
	}

	g := GRequest{r.Requests}
	RG, err := json.Marshal(g)
	if err != nil {
		p.lock.Unlock()
		log.Panic(err)
	}

	//更新信息
	digest := []byte(GetDigest(r.Requests)) //获得消息切片的摘要
	//Update("test1.db", []byte("table2"), InttoBytes(r.SequenceID), digest) //数据库1中的表2 -- 记录了消息序号到消息切片的hash映射
	Update("ToGRdigest.db", []byte("SeqToGRdigest"), InttoBytes(r.SequenceID), digest) //数据库ToGRdigest.db中的表SeqToGRdigest -- 记录了消息序号到消息切片的hash映射

	//更新区块信息--出块
	Insert("ToGR.db", "GRToBlock", RG)

	//Update("test2.db", []byte("table1"), digest, RG) //更新数据库2中的表1
	Update("ToGR.db", []byte("GRdigestToGR"), digest, RG) //更新数据库ToGR.db中的表GRdigestToGR

	for _, v := range r.Requests {
		key := []byte(v.Data.DataHash)
		//更新数据表
		//Update("test1.db", []byte("table1"), key, digest) //更新数据库1中的表1
		Update("ToGRdigest.db", []byte("DataHashToGRdigest"), key, digest) //更新数据库1中的表1

		mp := make(map[string][]string)
		for k, v2 := range v.Data.StoreNode { //存储 节点ip对应的datahash+名称(k)
			for _, v3 := range v2 {
				mp[v3] = append(mp[v3], k)
			}
		}
		for S, K := range mp {
			s := v.Data.DataHash
			for _, v4 := range K {
				s += ":" + v4
			}
			Insert("NodeToHash.db", S, []byte(s))
		}
		//fmt.Println("NodeToHash.db数据库处理完成！")
		Insert("RecToHash.db", v.Data.Reciver, []byte(v.Data.DataHash)) //存储 Rec(接收者id)对应的hash值
		fmt.Println("共识成功的元数据为：")
		fmt.Println(v.Data)
	}
}

// 一致性检查点协议
func (p *pbft) handleCheckPoint(content []byte) {
	defer handleCecover()

	fmt.Println("进入一致性检查点协议")
	c := new(CheckPoint)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}

	if c.SequenceID <= p.sequenceID { //如果发来的检查点在本节点的消息序号（最高水位）之下，不进行相关操作
		delete(p.checkpointCount, c.NodeID)
		return
	}
	p.lock.Lock()
	p.checkpointCount[c.NodeID] = c.SequenceID
	//当收到的大于本节点的消息序号（最高水位）的检查点消息大于f个时，需要检查一下我们是否落后了
	if len(p.checkpointCount) > f {
		fmt.Println("开始检查我们是否落后了")
		chkptArray := make([]int, len(p.checkpointCount))
		index := 0
		for replicaID, hChkpt := range p.checkpointCount {
			chkptArray[index] = hChkpt
			index++
			if hChkpt <= p.sequenceID {
				delete(p.checkpointCount, replicaID)
			}
		}
		sort.Ints(chkptArray)
		fmt.Println(chkptArray[len(chkptArray)-f-1])
		if m := chkptArray[len(chkptArray)-f-1]; m > p.sequenceID {
			//向其他节点广播更新消息
			c := UpData{p.sequenceID, p.node.nodeID, m}
			bc, err := json.Marshal(c)
			if err != nil {
				p.lock.Unlock()
				log.Panic(err)
			}
			p.broadcast(cUpdata, bc)
		}
	}
	//清除掉消息序号小于c.SequenceID的缓存消息
	fmt.Println("清除掉消息序号小于c.SequenceID的缓存消息")
	p.deleteCacheMessage()

	p.lock.Unlock()
}

// 清除缓存消息
func (p *pbft) deleteCacheMessage() {
	//删除根据消息切片摘要的缓存数据
	for _, v := range p.arrayTodigest {
		for k, _ := range p.messagePool {
			if v == k {
				delete(p.messagePool, k)
				break
			}
		}
		for k, _ := range p.prePareConfirmCount {
			if v == k {
				delete(p.prePareConfirmCount, k)
				break
			}
		}
		for k, _ := range p.commitConfirmCount {
			if v == k {
				delete(p.commitConfirmCount, k)
				break
			}
		}
		for k, _ := range p.replyConfireCount {
			if v == k {
				delete(p.replyConfireCount, k)
				break
			}
		}
		for k, _ := range p.viewchangeCount {
			if v == k {
				delete(p.viewchangeCount, k)
				break
			}
		}
		for k, _ := range p.isCommitBordcast {
			if v == k {
				delete(p.isCommitBordcast, k)
				break
			}
		}
		for k, _ := range p.isReply {
			if v == k {
				delete(p.isReply, k)
				break
			}
		}
		for k, _ := range p.isMReply {
			if v == k {
				delete(p.isMReply, k)
				break
			}
		}
		for k, _ := range p.isViewChange {
			if v == k {
				delete(p.isViewChange, k)
				break
			}
		}
	}

	//删除根据updatacount消息摘要的缓存数据
	for _, v := range p.arrayToupdata {
		for k, _ := range p.newdataCount {
			if v == k {
				delete(p.newdataCount, k)
				break
			}
		}
		for k, _ := range p.isnewdata {
			if v == k {
				delete(p.isnewdata, k)
				break
			}
		}
	}
}

// 更新节点的高低水位
func (p *pbft) moveWatermarks(m int) {
	p.h = m
	p.H = m + L
	//删除相关的日志消息--缓存消息

}

// 其他节点处理请求的更新数据
func (p *pbft) handleUpdata(content []byte) {
	defer handleCecover()

	c := new(UpData)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	if p.sequenceID < c.SCP { //如果该节点的消息序列也小于稳定检查点，直接忽略此节点
		return
	}
	// 从数据库中读取从p.sequenceID到c.SCP的客户端请求消息切片，发给c.NodeID
	var grs []GRequest
	for i := c.SequenceID + 1; i <= c.SCP; i++ {
		// 从test1的table2表中读取序号i对应的消息切片的摘要
		//data := Search("test1.db", []byte("table2"), InttoBytes(i))
		data := Search("ToGRdigest.db", []byte("SeqToGRdigest"), InttoBytes(i))

		// 从test2的table1表中读取消息切片的摘要对应的消息切片
		//data2 := Search("test2.db", []byte("table1"), data)
		data2 := Search("ToGR.db", []byte("GRdigestToGR"), data)

		gr := new(GRequest)
		err := json.Unmarshal(data2, gr)
		if err != nil {
			log.Panic(err)
		}
		grs = append(grs, *gr)
	}
	UDC := UpDataCount{c.SequenceID, c.SCP, grs}

	contents, err := json.Marshal(UDC)
	if err != nil {
		log.Panic(err)
	}

	contents2 := jointMessage(cNewdata, contents)
	fmt.Println("发送给需要更新的节点" + strconv.Itoa(c.NodeID) + "的消息长度为：" + strconv.Itoa(len(contents2)))
	// 发送给c.NodeID
	tcpDial(contents2, nodeTable[c.NodeID])
}

// 节点更新数据库的消息 -- 节点可能收到多个这种消息，需要判别发来的消息的正确性
func (p *pbft) handleNewdata(content []byte) {
	defer handleCecover()

	fmt.Println("进入到节点更新数据库函数，收到的消息长度为：" + strconv.Itoa(len(content)))
	grs := new(UpDataCount)
	err := json.Unmarshal(content, grs)
	if err != nil {
		log.Panic(err)
	}
	p.lock.Lock()
	p.newdataCount[GetDigest(*grs)]++
	p.arrayToupdata[p.sequenceID%K] = GetDigest(*grs)
	if p.newdataCount[GetDigest(*grs)] > 2*f && !p.isnewdata[GetDigest(*grs)] { //如果收到的消息数量没有达到2f个
		for k, v := range (*grs).GRequests {
			// 1 首先根据消息序号将对应的消息切片摘要写到数据库1的表2中
			digest := []byte(GetDigest(v.Requests))
			//Update("test1.db", []byte("table2"), InttoBytes(p.h-len((*grs).GRequests)+k+1), digest)
			Update("ToGRdigest.db", []byte("SeqToGRdigest"), InttoBytes(p.h-len((*grs).GRequests)+k+1), digest)

			// 2 再将切片中的具体每个消息摘要对应的消息切片摘要写到据库1的表1中
			for _, v2 := range v.Requests {
				//Update("test1.db", []byte("table1"), []byte(v2.Data.DataHash), digest)
				Update("ToGRdigest.db", []byte("DataHashToGRdigest"), []byte(v2.Data.DataHash), digest)
			}
			// 3 最后将消息切片的摘要对应的具体消息切片写到据库2中的表1中
			g := GRequest{v.Requests}
			RG, err := json.Marshal(g)
			if err != nil {
				p.lock.Unlock()
				log.Panic(err)
			}
			//Update("test2.db", []byte("table1"), digest, RG)
			Update("ToGR.db", []byte("GRdigestToGR"), digest, RG)
		}
		p.isnewdata[GetDigest(*grs)] = true
		// 更新完成后将节点的序号更新为最新值
		p.sequenceID = grs.SCP
	}
	p.lock.Unlock()
}

// 寻找消息功能
func (p *pbft) handleFind(data []byte, conn net.Conn) {
	fmt.Println("进入寻找消息函数")
	if p.node.nodeID != M_node {
		return
	}
	find(data, conn)

	//for _, v := range localMessagePool {
	//	if CompareByte(v.Message.DataHash, data) {
	//		content, err := json.Marshal(v)
	//		if err != nil {
	//			log.Panic(err)
	//		}
	//		(*conn).Write(content)
	//		fmt.Println("发送成功")
	//		//tcpDial(content, "10.25.3.205:8000")
	//	}
	//}
}

func find(content []byte, conn net.Conn) {
	defer handleCecover()

	//dg := Search("test1.db", []byte("table1"), content)
	//rg := Search("test2.db", []byte("table1"), dg)
	dg := Search("ToGRdigest.db", []byte("DataHashToGRdigest"), content)
	rg := Search("ToGR.db", []byte("GRdigestToGR"), dg)
	if len(dg) == 0 || len(rg) == 0 {
		log.Println("没有找到hash值为：" + string(content) + "对应的数据！")
		return
	}

	g := new(GRequest)
	err2 := json.Unmarshal(rg, g)
	if err2 != nil {
		log.Panic(err2)
	}
	//寻找消息
	for _, v := range g.Requests {
		if v.Data.DataHash == string(content) {
			fmt.Println("找到了消息")
			ans, err := json.Marshal(v.Data)
			if err != nil {
				log.Panic(err)
			}
			fmt.Println("序列化后的messages消息为:" + string(ans))
			fmt.Println("发送的长度为：" + strconv.Itoa(len(ans)))
			conn.Write(ans)
			buffer := make([]byte, 1024)
			_, err = conn.Read(buffer[:])
			if err != nil {
				fmt.Println("recv failed err:", err)
			}
			return
		}
	}
	fmt.Println("没有找到消息")
}

//func (p *pbft) handleSearchNodeToHash(content []byte, conn net.Conn) {
//	fmt.Println("进入查询节点和hash值的函数！！")
//	defer handleCecover()
//	err, data := Search2("NodeToHash.db", string(content))
//	if err != nil {
//		fmt.Println("Search data fail!")
//		fmt.Println("发送失败，没有找到根据node对应的hash值！")
//		return
//	}
//	//对查询到的data进行切分--以:为分隔符
//	mp := make(map[string]bool)
//	for _, v := range data[0] {
//		S := strings.Split(v, ":")
//		for i := 1; i < len(S); i++ {
//			s := S[0] + ":" + S[i]
//			//首先需要查询缺失的数量
//			value := Search("NodeToHash.db", []byte("MissCodeChip"), []byte(s))
//			if value == nil { //说明还没有创建表MissCodeChip
//				Update("NodeToHash.db", []byte("MissCodeChip"), []byte(s), []byte(strconv.Itoa(0)))
//			} else {
//				//再更新数据
//				num, _ := strconv.Atoi(string(value))
//				num++ //缺失数量+1
//				Update("NodeToHash.db", []byte("MissCodeChip"), []byte(s), []byte(strconv.Itoa(num)))
//				//判断缺失码片数量是否达到某一阈值
//				if num >= ThresholdMissChips { //当缺失码片的数量大于等于某一阈值后
//					//一个一个判断发送还是整体发送呢？--目前是整体发送
//					mp[s] = true
//				}
//			}
//		}
//	}
//	if len(mp) == 0 { //如果没有需要修复的码片就直接返回
//		return
//	}
//	message, err := json.Marshal(mp)
//	if err != nil {
//		log.Panic(err)
//	}
//
//	//是否要在发送具体数据之前加个标识--jointMessage()
//
//	conn.Write(message)
//	buffer := make([]byte, 1024)
//	_, err = conn.Read(buffer[:]) //对方会返回一个ok
//	if err != nil {
//		fmt.Println("recv failed, err:", err)
//		return
//	}
//	//fmt.Println(len(message))
//	fmt.Println("发送的数据为：" + string(message))
//	fmt.Println("发送成功！！")
//}

func (p *pbft) handleSearchNodeToHash(content []byte, conn net.Conn) {
	fmt.Println("进入查询节点和hash值的函数！！")
	defer handleCecover()
	err, data := Search2("NodeToHash.db", string(content))
	if err != nil {
		fmt.Println("Search data fail!")
		fmt.Println("发送失败，没有找到根据node对应的hash值！")
		return
	}
	message, err := json.Marshal(data[0])
	if err != nil {
		log.Panic(err)
	}
	//length := len(message)
	//首先把length发给请求端
	//conn.Write([]byte(strconv.Itoa(length)))
	//然后把message发给请求端的另一个端口
	//tcpDial(message, "10.25.3.142:7761")
	conn.Write(message)
	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer[:]) //对方会返回一个ok
	if err != nil {
		fmt.Println("recv failed, err:", err)
		return
	}
	//fmt.Println(len(message))
	fmt.Println("发送的数据为：" + string(message))
	fmt.Println("发送成功！！")
}

// 根据Reciver查找对应的FileName#FileType#DataHash
func (p *pbft) handleSearchRecToHash(content []byte, conn net.Conn) {
	fmt.Println("进入查询Reciver对应的FileName#FileType#DataHash值的函数！！")
	//fmt.Println("要查询的数据库表名为：" + string(content))
	defer handleCecover()
	err, data := Search2("RecToHash.db", string(content)) //这里有问题
	if err != nil {
		fmt.Println("Search data fail!")
		return
	}
	//fmt.Println("根据" + string(content) + "查到的DataHash数据为:")
	fmt.Println(data)
	fmt.Println(len(data))
	for _, V := range data {
		for _, v := range V {
			//rg := Search("test1.db", []byte("table1"), []byte(v))
			//rg2 := Search("test2.db", []byte("table1"), rg)
			rg := Search("ToGRdigest.db", []byte("DataHashToGRdigest"), []byte(v))
			rg2 := Search("ToGR.db", []byte("GRdigestToGR"), rg)
			if len(rg2) == 0 {
				fmt.Println("没有找到！")
				continue
			}
			g := new(GRequest)
			err2 := json.Unmarshal(rg2, g)
			if err2 != nil {
				log.Panic(err2)
			}
			for _, v2 := range g.Requests {
				if v2.Data.DataHash == v {
					message := v2.Data.FileName + "#" + v2.Data.FileType + "#" + v2.Data.DataHash + "#"
					conn.Write([]byte(message))
					fmt.Println("找到并发送的数据为：" + message)
				}
			}
		}
	}
	conn.Write([]byte("finish"))
	fmt.Println("查询完成！！")
}

// 查找destionation到对应hash值
//func (p *pbft) handleSearchRecToHash(content []byte, conn net.Conn) {
//	fmt.Println("进入查询destionation ip对应的hash值的函数！！")
//	defer handleCecover()
//	err, data := Search2("RecToHash.db", string(content))
//	if err != nil {
//		fmt.Println("Search data fail!")
//		return
//	}
//
//	for k, v := range data {
//		rg := Search("test2.db", []byte("table1"), []byte(v[uint64(k)]))
//		g := new(GRequest)
//		err2 := json.Unmarshal(rg, g)
//		if err2 != nil {
//			log.Panic(err2)
//		}
//		for _, v2 := range g.Requests {
//			if v2.Data.DataHash == v[uint64(k)] {
//				message, err := json.Marshal(v2.Data)
//				if err != nil {
//					log.Panic(err)
//				}
//				conn.Write(message)
//			}
//		}
//	}
//	fmt.Println("全部发送成功！！")
//}

// 根据文件查询--TODO：用不到了，需要删除
func find2(data []byte, conn net.Conn) {
	filePath := "test.txt"
	content, err := os.ReadFile(filePath)

	fmt.Println(string(content))

	if err != nil {
		panic(err)
	}
	r := new(Request)
	err2 := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err2)
	}
	fmt.Println(r.Data.DataHash)
	fmt.Println(data)
	fmt.Println(r)
	if r.Digest == string(data) { //比较hash是否相同
		content, err := json.Marshal(r)
		if err != nil {
			log.Panic(err)
		}
		conn.Write(content)
		fmt.Println("发送成功")
	}
}

// 更改视图 -- TODO:需要修改
func (p *pbft) handleViewChange(content []byte) {
	defer handleCecover()

	if p.node.nodeID != (M_node+1)%nodeCount { //如果不是下一个主节点
		return
	}
	fmt.Println("节点" + strconv.Itoa(p.node.nodeID) + "收到了视图切换消息")
	//首先使用json解析出发来的request2消息 使用re2接收
	re2 := new(GRequest) // 这里应该是request切片类型
	err := json.Unmarshal(content, &re2.Requests)
	if err != nil {
		log.Panic(err)
	}
	p.lock.Lock()
	digest := GetDigest(re2.Requests)
	p.viewchangeCount[digest]++
	if p.viewchangeCount[digest] > f && !p.isViewChange[digest] { //满足视图切换的要求
		p.isViewChange[digest] = true
		fmt.Println("进行视图切换")
		//向客户端广播一个消息：要求其重新发送要共识的消息
		//message := jointMessage(cRequest, content)
		//tcpDial(message, re2.ClientAddr)

		//向其他节点广播 令他们更改其ViewId
		p.broadcast(cNewView, content)
		//fmt.Println("客户端的ip为：" + re2.ClientAddr)
		//更新主节点编号
		M_node = (M_node + 1) % nodeCount
		for i, v := range localMessagePool {
			for _, re := range re2.Requests {
				if Compare(v, re) { //若本地消息池已经存在了该消息，将其删除
					localMessagePool = append(localMessagePool[:i], localMessagePool[i+1:]...)
					break
				}
			}
		}
		fmt.Println("主节点变为：" + strconv.Itoa(M_node))
	}
	p.lock.Unlock()
}

// 更改视图确认消息 -- 更改主节点id以及删除本地消息池里的未共识的消息
func (p *pbft) handleNewView(content []byte) {
	defer handleCecover()

	k := new(GRequest)
	err := json.Unmarshal(content, &k.Requests)
	if err != nil {
		log.Panic(err)
	}
	digest := GetDigest(k.Requests)
	if p.isViewChange[digest] { //说明该节点已经更新过视图了
		return
	}
	fmt.Println("节点" + strconv.Itoa(p.node.nodeID) + "收到了" + "更改视图确认消息")
	if p.node.nodeID != (M_node+1)%nodeCount {
		M_node = (M_node + 1) % nodeCount
		p.isViewChange[digest] = true
		fmt.Println("主节点变为：" + strconv.Itoa(M_node))
	}
	for i, v := range localMessagePool {
		for _, re := range k.Requests {
			if Compare(v, re) { //若本地消息池已经存在了该消息，将其删除
				localMessagePool = append(localMessagePool[:i], localMessagePool[i+1:]...)
				break
			}
		}
	}
}

// 广播检查点消息
func (p *pbft) checkpoint(a, b int) {
	defer handleCecover()

	digest := GetDigest(CheckPoint{
		SequenceID: a,
		Digest:     "",
		NodeID:     b,
	})
	c := CheckPoint{a, digest, b}
	bc, err := json.Marshal(c)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("广播检查点消息")
	p.broadcast(cCheckPoint, bc)
}

// 处理panic错误 防止程序宕机
func handleCecover() {
	err := recover()
	if err != nil {
		log.Println("序列化反序列化错误，或者签名有误，请修正错误！")
	}
}

// sign := data.FileName + strconv.FormatInt(int64(timestamp), 10)
// 验证客户端的签名
func (p *pbft) SignatureVerifyClient(pp *PrePrepare) bool {
	RMS := pp.RequestMessage
	for _, v := range RMS {
		sign := v.Data.FileName + v.Timestamp
		digest := GetDigest(sign)
		//log.Println("待验证的摘要为：" + digest)
		//log.Println(v.Signature)
		//log.Println("节点收到的消息中的签名为：" + hex.EncodeToString(v.Signature.V) + "  " + hex.EncodeToString(v.Signature.U))
		if !p.use.SignatureVerify(v.Data.NodeID, &v.Signature, []byte(digest)) {
			log.Println("客户端签名验证失败！")
			return false
		}
	}
	return true
}

// 消息序号是否在高低水位之间
func (p *pbft) inW(n int) bool {
	return n-p.h > 0 && n-p.H <= 0
}

// int转[]byte
func InttoBytes(i int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

// 比较Request类型是否相同
func Compare(a, b Request) bool {
	if a.ClientAddr == b.ClientAddr && a.Digest == b.Digest && Compare2(a.Data, b.Data) && a.Timestamp == b.Timestamp { //其实只需要比较时间戳或摘要即可
		return true
	}
	return false
}

// 比较Metadata类型是否相同
func Compare2(a, b Metadata) bool {
	return a.DataHash == b.DataHash
}

// 比较[]byte类型是否相同
func CompareByte(a, b []byte) bool {
	return string(a) == string(b)
}

// 序号累加
func (p *pbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

// 向除自己外的其他节点进行广播
func (p *pbft) broadcast(cmd command, content []byte) {
	for i := range nodeTable {
		if i == p.node.nodeID {
			continue
		}
		message := jointMessage(cmd, content)
		go tcpDial(message, nodeTable[i])
	}
}

// 为多重映射开辟赋值--预准备消息
func (p *pbft) setPrePareConfirmMap(val string, val2 int, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[int]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

// 为多重映射开辟赋值--确认消息
func (p *pbft) setCommitConfirmMap(val string, val2 int, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[int]bool)
	}
	p.commitConfirmCount[val][val2] = b
}

// 为多重映射开辟赋值--增加节点消息
func (p *pbft) setaddNodeConfirmMap(val string, val2 int, b bool) {
	if _, ok := p.addNodeCount[val]; !ok {
		p.addNodeCount[val] = make(map[int]bool)
	}
	p.addNodeCount[val][val2] = b
}

//// 传入节点编号， 获取对应的公钥
//func (p *pbft) getPubKey(nodeID int) []byte {
//	key, err := ioutil.ReadFile("Keys/" + strconv.Itoa(nodeID) + "/" + strconv.Itoa(nodeID) + "_RSA_PUB")
//	if err != nil {
//		log.Panic(err)
//	}
//	return key
//}
//
//// 传入节点编号， 获取对应的私钥
//func (p *pbft) getPivKey(nodeID int) []byte {
//	key, err := ioutil.ReadFile("Keys/" + strconv.Itoa(nodeID) + "/" + strconv.Itoa(nodeID) + "_RSA_PIV")
//	if err != nil {
//		log.Panic(err)
//	}
//	return key
//}
