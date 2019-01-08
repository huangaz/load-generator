package testHelper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	gen "github.com/huangaz/load-generator/gen"
)

const (
	DELIM = '\n'
)

var (
	operators = []string{"+", "-", "*", "/"}
)

type TCPComm struct {
	addr string
}

func NewTCPComm(address string) gen.Caller {
	return &TCPComm{
		addr: address,
	}
}

func (t *TCPComm) BuildReq() gen.RawReq {
	id := time.Now().UnixNano()

	sreq := ServerReq{
		ID: id,
		Operands: []int{
			int(rand.Int31n(1000) + 1),
			int(rand.Int31n(1000) + 1),
		},
		Operator: func() string {
			return operators[rand.Int31n(100)%4]
		}(),
	}

	bytes, err := json.Marshal(sreq)
	if err != nil {
		panic(err)
	}

	return gen.RawReq{
		ID:  id,
		Req: bytes,
	}
}

func (t *TCPComm) Call(req []byte, timeoutNS time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", t.addr, timeoutNS)
	if err != nil {
		return nil, err
	}

	_, err = write(conn, req, DELIM)
	if err != nil {
		return nil, err
	}

	return read(conn, DELIM)
}

func (t *TCPComm) CheckResp(rawReq gen.RawReq, rawResp gen.RawResp) *gen.CallResult {

	res := &gen.CallResult{
		ID:   rawResp.ID,
		Req:  rawReq,
		Resp: rawResp,
	}

	var sreq ServerReq
	if err := json.Unmarshal(rawReq.Req, &sreq); err != nil {
		res.Code = gen.RET_CODE_FATAL_CALL
		res.Msg = fmt.Sprintf("Incorrectly formatted Req: %s!\n", string(rawReq.Req))
		return res
	}

	var sresp ServerResp
	if err := json.Unmarshal(rawResp.Resp, &sresp); err != nil {
		res.Code = gen.RET_CODE_ERROR_RESPONSE
		res.Msg = fmt.Sprintf("Incorrectly formatted Resp: %s!\n", string(rawResp.Resp))
		return res
	}

	if sreq.ID != sresp.ID {
		res.Code = gen.RET_CODE_ERROR_RESPONSE
		res.Msg = fmt.Sprintf("Inconsistent raw id! (%d != %d)\n", rawReq.ID, rawResp.ID)
		return res
	}

	if sresp.Err != nil {
		res.Code = gen.RET_CODE_ERROR_CALEE
		res.Msg = fmt.Sprintf("Abnormal server: %s!\n", sresp.Err)
		return res
	}

	if sresp.Result != op(sreq.Operands, sreq.Operator) {
		res.Code = gen.RET_CODE_ERROR_RESPONSE
		res.Msg = fmt.Sprintf("Incorrect result: %s!\n",
			genFormula(sreq.Operands, sreq.Operator, sresp.Result, false))
		return res
	}

	res.Code = gen.RET_CODE_SUCCESS
	res.Msg = fmt.Sprintf("Success. (%s)", sresp.Formula)
	return res
}

func read(conn net.Conn, delim byte) ([]byte, error) {
	readBytes := make([]byte, 1)
	var buffer bytes.Buffer

	for {
		_, err := conn.Read(readBytes)
		if err != nil {
			return nil, err
		}
		readByte := readBytes[0]
		if readByte == delim {
			break
		}
		buffer.WriteByte(readByte)
	}
	return buffer.Bytes(), nil
}

func write(conn net.Conn, content []byte, delim byte) (int, error) {
	writer := bufio.NewWriter(conn)
	n, err := writer.Write(content)
	if err == nil {
		writer.WriteByte(delim)
	}

	if err == nil {
		err = writer.Flush()
	}

	return n, err
}
