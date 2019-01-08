package load_generator

import (
	"errors"
	"fmt"
	"strings"
	"time"

	gen "github.com/huangaz/load-generator/gen"
)

// parameter og generator
type ParamSet struct {
	Caller     gen.Caller
	TimeoutNS  time.Duration
	LPS        uint32
	DurationNS time.Duration
	ResultCh   chan *gen.CallResult
}

func (p *ParamSet) Check() error {
	errMsgs := make([]string, 0)

	fmt.Printf("Checking the parameter...")

	if p.Caller == nil {
		errMsgs = append(errMsgs, "Invalid caller!")
	}
	if p.TimeoutNS == 0 {
		errMsgs = append(errMsgs, "Invalid timeoutNS!")
	}
	if p.LPS == 0 {
		errMsgs = append(errMsgs, "Invalid lps(load per second)!")
	}
	if p.DurationNS == 0 {
		errMsgs = append(errMsgs, "Invalid durationNS!")
	}
	if p.ResultCh == nil {
		errMsgs = append(errMsgs, "Invalid result channel!")
	}

	if len(errMsgs) != 0 {
		errMsg := strings.Join(errMsgs, " ")
		return errors.New(errMsg)
	}

	fmt.Printf("Passed, (timeoutNS=%s, lps=%d, durationNS=%s)\n",
		p.TimeoutNS, p.LPS, p.DurationNS)

	return nil
}
