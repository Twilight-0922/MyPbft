package main

import (
	"IBEUser/src/IBE/ibecommon"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

//type Messages struct {
//	UserID     string            `json:"userid"`
//	FileName   string            `json:"filename"`
//	DataSize   int               `json:"datasize"`
//	DataHash   string            `json:"datahash"`
//	BlockIndex map[string]string `json:"blockindex"`
//}
//
//// <REQUEST,o,t,c>
//type Request struct {
//	ClientAddr string                       `json:"uid"`       //相当于client IP
//	Timestamp  string                       `json:"timestamp"` //请求时客户端追加的时间戳
//	Digest     string                       `json:"digest"`    //消息摘要
//	Sign       ibecommon.TransSignatureText `json:"signature"` //消息签名
//	Message    Messages                     `json:"message"`   //消息内容
//}
//

// 元数据结构体
type Metadata struct {
	NodeID     string                       `json:"nodeid"`
	FileName   string                       `json:"filename"`
	FileType   string                       `json:"filetype"`
	Senter     string                       `json:"senter"`
	Reciver    string                       `json:"reciver"`
	DataSize   int                          `json:"datasize"`
	DataHash   string                       `json:"datahash"`
	StoreNode  map[string][]string          `json:"store node"`
	BlockIndex map[string]map[string]string `json:"blockindex"`
}

// <REQUEST,o,t,c>
type Request struct {
	ClientAddr string                       `json:"agent"`
	DataType   string                       `json:"datatype"`
	Timestamp  string                       `json:"timestamp"`
	Digest     string                       `json:"digest"`
	Signature  ibecommon.TransSignatureText `json:"signature"`
	Data       Metadata                     `json:"data"`
}

type GRequest struct {
	Requests []Request
}

// <<PRE-PREPARE,v,n,d>,m>
type PrePrepare struct {
	RequestMessage []Request
	Digest         string                       //摘要
	SequenceID     int                          //队列ID
	Sign           ibecommon.TransSignatureText //签名
	ViewId         int                          //视图ID
}

// <PREPARE,v,n,d,i>
type Prepare struct {
	Digest     string                       // d 消息内容摘要
	SequenceID int                          // n 当前请求编号
	NodeID     int                          //i 节点编号
	Sign       ibecommon.TransSignatureText //签名
	ViewId     int                          // v当前视图编号
}

// <COMMIT,v,n,D(m),i>
type Commit struct {
	Digest     string                       //消息摘要
	SequenceID int                          //当前请求编号
	NodeID     int                          //节点编号
	Sign       ibecommon.TransSignatureText //签名
	ViewId     int                          //当前视图编号
}

// <REPLYM>
type Reply struct {
	NodeID     int
	Result     bool
	ViewId     int    //视图ID
	Digest     string //摘要
	SequenceID int    //当前请求编号
	Requests   []Request
}

// <CHECKPOINT>
type CheckPoint struct {
	SequenceID int    //消息序号
	Digest     string //摘要
	NodeID     int    //节点id
}

// <UPDATA>
type UpData struct {
	SequenceID int //消息序号
	NodeID     int //节点id
	SCP        int //稳定检查点
}

// <UPDATACOUNT>
type UpDataCount struct {
	SequenceID int        //消息序号
	SCP        int        //稳定检查点
	GRequests  []GRequest //消息切片
}

// <REPLY,v,t,c,i,r>
type Reply2 struct {
	NodeID     int
	Result     bool
	ViewId     int    //视图ID
	Timestamp  string //客户端追加的时间戳
	ClientAddr string //客户端标识
}

type Config struct {
	NodesTable []nodes `mapstructure:"nodes"`
}
type nodes struct {
	Nodeid int    `mapstructure:"nodeid"`
	Ip     string `mapstructure:"ip"`
	Port   int    `mapstructure:"port"`
}

const prefixCMDLength = 16

type command string

const (
	cRequest        command = "request"
	cRequests       command = "requests"
	cPrePrepare     command = "preprepare"
	cPrepare        command = "prepare"
	cCommit         command = "commit"
	cReply          command = "reply"
	cCheckPoint     command = "checkpoint"
	cUpdata         command = "updata"
	cNewdata        command = "newdata"
	cViewChange     command = "ViewChange"
	cNewView        command = "NewView"
	cRepair         command = "repair"
	cRecovery       command = "recovery"
	cRecToMeta      command = "rectometa"
	cRequestAddNode command = "requestAddNode"
	cAddNode        command = "AddNode"
	cReplyToAddNode command = "replyToAddNode"
)

// 默认前十六位为命令名称
func jointMessage(cmd command, content []byte) []byte {
	b := make([]byte, prefixCMDLength)
	for i, v := range []byte(cmd) {
		b[i] = v
	}
	joint := make([]byte, 0)
	joint = append(b, content...)
	return joint
}

// 默认前十六位为命令名称
func splitMessage(message []byte) (cmd string, content []byte) {
	cmdBytes := message[:prefixCMDLength]
	newCMDBytes := make([]byte, 0)
	for _, v := range cmdBytes {
		if v != byte(0) && v != []byte("0")[0] {
			newCMDBytes = append(newCMDBytes, v)
		}
	}
	cmd = string(newCMDBytes)
	content = message[prefixCMDLength:]
	return
}

// 获得data的哈希值
func GetDigest(data any) string {
	req, err := json.Marshal(data)
	if err != nil {
		fmt.Println("data marshal err is: ", err)
	}
	//hash := sha256.Sum256(req)
	hash := md5.Sum(req)
	//进行十六进制字符串编码
	res := hex.EncodeToString(hash[:])
	return res
}

// 对消息详情进行摘要
// func getDigest(request string) string {
// 	b, err := json.Marshal(request)
// 	if err != nil {
// 		log.Panic(err)
// 	}
// 	hash := sha256.Sum256(b)
// 	//进行十六进制字符串编码
// 	return hex.EncodeToString(hash[:])
// }
//// 消息hash
//func getDigest(data Messages) string {
//	req, err := json.Marshal(data)
//	if err != nil {
//		fmt.Println("data marshal err is: ", err)
//	}
//	//hash := sha256.Sum256(req)
//	hash := md5.Sum(req)
//	//进行十六进制字符串编码
//	res := hex.EncodeToString(hash[:])
//	return res
//}
//
//// checkpoint消息摘要
//func getDigestcheckpoint(data CheckPoint) string {
//	req, err := json.Marshal(data)
//	if err != nil {
//		fmt.Println("data marshal err is: ", err)
//	}
//	//hash := sha256.Sum256(req)
//	hash := md5.Sum(req)
//	//进行十六进制字符串编码
//	res := hex.EncodeToString(hash[:])
//	return res
//}
//
//// 消息hash
//func getDigestGR(data []Request) string {
//	req, err := json.Marshal(data)
//	if err != nil {
//		fmt.Println("data marshal err is: ", err)
//	}
//	//hash := sha256.Sum256(req)
//	hash := md5.Sum(req)
//	//进行十六进制字符串编码
//	res := hex.EncodeToString(hash[:])
//	return res
//}
