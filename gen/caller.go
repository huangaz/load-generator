package gen

import "time"

type Caller interface {
	// build request
	BuildReq() RawReq

	// call
	Call(req []byte, timeoutNS time.Duration) ([]byte, error)

	// check response
	CheckResp(rawReq RawReq, rawResp RawResp) *CallResult
}
