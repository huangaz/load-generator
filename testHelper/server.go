package testHelper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync/atomic"
)

type ServerReq struct {
	ID       int64  `json:"id"`
	Operands []int  `json:"operands"`
	Operator string `json:"operator"`
}

type ServerResp struct {
	ID      int64  `json:"id"`
	Formula string `json:"formula"`
	Result  int    `json:"result"`
	Err     error  `json:"err"`
}

func op(operands []int, operator string) int {
	var res int
	switch {
	case operator == "+":
		for _, v := range operands {
			if res == 0 {
				res = v
			} else {
				res += v
			}
		}
	case operator == "-":
		for _, v := range operands {
			if res == 0 {
				res = v
			} else {
				res -= v
			}
		}
	case operator == "*":
		for _, v := range operands {
			if res == 0 {
				res = v
			} else {
				res *= v
			}
		}
	case operator == "/":
		for _, v := range operands {
			if res == 0 {
				res = v
			} else {
				res /= v
			}
		}
	}
	return res
}

func genFormula(operands []int, operator string, result int, equal bool) string {
	var buff bytes.Buffer
	n := len(operands)
	for i := 0; i < n; i++ {
		if i > 0 {
			buff.WriteString(" ")
			buff.WriteString(operator)
			buff.WriteString(" ")
		}

		buff.WriteString(strconv.Itoa(operands[i]))
	}

	if equal {
		buff.WriteString(" = ")
	} else {
		buff.WriteString(" != ")
	}

	buff.WriteString(strconv.Itoa(result))
	return buff.String()
}

func reqHandler(conn net.Conn) {
	var errMsg string
	var sresp ServerResp
	req, err := read(conn, DELIM)
	if err != nil {
		errMsg = fmt.Sprintf("Server: Req Read Error: %s", err)
	} else {
		var sreq ServerReq
		if err := json.Unmarshal(req, &sreq); err != nil {
			errMsg = fmt.Sprintf("Server: Req Unmarshal Error: %s", err)
		} else {
			sresp.ID = sreq.ID
			sresp.Result = op(sreq.Operands, sreq.Operator)
			sresp.Formula = genFormula(sreq.Operands, sreq.Operator, sresp.Result, true)
		}
	}

	if errMsg != "" {
		sresp.Err = errors.New(errMsg)
	}

	bytes, err := json.Marshal(sresp)
	if err != nil {
		log.Fatalf("Server: Resp Marshal Error: %s", err)
	}

	_, err = write(conn, bytes, DELIM)
	if err != nil {
		log.Fatalf("Server: Resp Write error: %s", err)
	}
}

type TCPServer struct {
	listener net.Listener
	active   uint32
}

func NewTCPServer() *TCPServer {
	return &TCPServer{}
}

func (t *TCPServer) init(addr string) error {
	if !atomic.CompareAndSwapUint32(&t.active, 0, 1) {
		return nil
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		atomic.StoreUint32(&t.active, 0)
		return err
	}
	t.listener = ln
	return nil
}

func (t *TCPServer) Listen(addr string) error {
	err := t.init(addr)
	if err != nil {
		return err
	}

	go func() {
		for {
			if atomic.LoadUint32(&t.active) != 1 {
				break
			}

			conn, err := t.listener.Accept()
			if err != nil {
				if atomic.LoadUint32(&t.active) == 1 {
					log.Fatalf("Server: Request Acception Error: %s\n", err)
				} else {
					log.Printf("Server: Broken acception because of closed network connection.")
				}

				continue
			}

			go reqHandler(conn)
		}
	}()
	return nil
}

func (t *TCPServer) Close() bool {
	if !atomic.CompareAndSwapUint32(&t.active, 1, 0) {
		return false
	}
	t.listener.Close()
	return true
}
