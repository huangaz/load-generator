package gen

import "time"

type RawReq struct {
	// id
	ID int64

	// data of raw request
	Req []byte
}

type RawResp struct {
	// id
	ID int64

	// data of raw response
	Resp []byte

	Err error

	// elapsed time
	Elapse time.Duration
}

// result code
type RetCode int

const (
	RET_CODE_SUCCESS              RetCode = 0
	RET_CODE_WARNING_CALL_TIMEOUT         = 1001
	RET_CODE_ERROR_CALL                   = 2001
	RET_CODE_ERROR_RESPONSE               = 2002
	RET_CODE_ERROR_CALEE                  = 2003
	RET_CODE_FATAL_CALL                   = 3001
)

type CallResult struct {
	// result id
	ID int64

	// raw request
	Req RawReq

	// raw response
	Resp RawResp

	// response code
	Code RetCode

	// description of response
	Msg string

	// elapsed time
	Elapse time.Duration
}

const (
	STATUS_ORIGINAL uint32 = 0

	STATUS_STARTING uint32 = 1

	STATUS_STARTED uint32 = 2

	STATUS_STOPPING uint32 = 3

	STATUS_STOPPED uint32 = 4
)

type Generator interface {
	// start generator
	Start() bool

	// stop generator
	Stop() bool

	// get status of generator
	Status() uint32

	// get call count of generator
	CallCount() int64
}

func GetRetCodePlain(code RetCode) string {
	var codePlain string
	switch code {
	case RET_CODE_SUCCESS:
		codePlain = "Success"
	case RET_CODE_WARNING_CALL_TIMEOUT:
		codePlain = "Call Timeout Warning"
	case RET_CODE_ERROR_CALL:
		codePlain = "Call Error"
	case RET_CODE_ERROR_RESPONSE:
		codePlain = "Response Error"
	case RET_CODE_ERROR_CALEE:
		codePlain = "Callee Error"
	case RET_CODE_FATAL_CALL:
		codePlain = "Call Fatal Error"
	default:
		codePlain = "Unknown result code"
	}

	return codePlain
}
