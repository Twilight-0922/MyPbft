package handle

import (
	"log"
	"net"
)

func ErrHandle(conn net.Conn, err error) {
	log.Println(err)
	//_, err = conn.Write([]byte("err: " + err.Error() + "\n"))
	//if err != nil {
	//	log.Println(err)
	//}
}
