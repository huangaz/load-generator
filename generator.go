package load_generator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	gen "github.com/huangaz/load-generator/gen"
)

type myGenerator struct {
	caller gen.Caller

	// timeout(ns)
	timeoutNS time.Duration

	// load per second
	lps uint32

	// duration(ns)
	durationNS time.Duration

	// concurrency
	concurrency uint32

	// pool of goroutines
	tickets gen.GoTickets

	ctx        context.Context
	cancelFunc context.CancelFunc

	// call count
	callCount int64

	// status of generator
	status uint32

	// channel of results
	resultCh chan *gen.CallResult
}

func NewGenerator(pset ParamSet) (gen.Generator, error) {

	fmt.Println("New a load generator.")
	if err := pset.Check(); err != nil {
		return nil, err
	}

	gen := &myGenerator{
		caller:     pset.Caller,
		timeoutNS:  pset.TimeoutNS,
		lps:        pset.LPS,
		durationNS: pset.DurationNS,
		status:     gen.STATUS_ORIGINAL,
		resultCh:   pset.ResultCh,
	}

	if err := gen.init(); err != nil {
		return nil, err
	}

	return gen, nil
}

func (m *myGenerator) init() error {
	fmt.Printf("Initializing the load generator...")

	// init concurrency

	total64 := int64(m.timeoutNS)/int64(1e9/m.lps) + 1
	if total64 > math.MaxInt32 {
		total64 = math.MaxInt32
	}
	m.concurrency = uint32(total64)

	// init tickets
	tickets, err := gen.NewGoTickets(m.concurrency)
	if err != nil {
		return err
	}
	m.tickets = tickets

	fmt.Printf("Done, (concurrency=%d)\n", m.concurrency)
	return nil
}

func (m *myGenerator) genLoad(throttle <-chan time.Time) {
	for {
		select {
		case <-m.ctx.Done():
			m.prepareToStop(m.ctx.Err())
			return
		default:
		}

		m.asyncCall()

		if m.lps > 0 {
			select {
			case <-throttle:
			case <-m.ctx.Done():
				m.prepareToStop(m.ctx.Err())
				return
			}
		}
	}
}

func (m *myGenerator) prepareToStop(ctxError error) {
	fmt.Printf("Prepare to stop load generator (cause: %s)...\n", ctxError.Error())

	atomic.CompareAndSwapUint32(&m.status, gen.STATUS_STARTED, gen.STATUS_STOPPING)

	fmt.Println("Closing result channel...")
	close(m.resultCh)

	atomic.StoreUint32(&m.status, gen.STATUS_STOPPED)
}

func (m *myGenerator) asyncCall() {
	m.tickets.Take()

	go func() {
		defer func() {
			// recover panic
			if p := recover(); p != nil {
				err, ok := interface{}(p).(error)
				var errMsg string
				if ok {
					errMsg = fmt.Sprintf("Async Call Panic! (error: %s)", err)
				} else {
					errMsg = fmt.Sprintf("Async Call Panic! (clue: %#v)", p)
				}

				log.Println(errMsg)

				result := &gen.CallResult{
					ID:   -1,
					Code: gen.RET_CODE_FATAL_CALL,
					Msg:  errMsg,
				}
				m.sendResult(result)
			}

			//return tickets
			m.tickets.Return()
		}()

		rawReq := m.caller.BuildReq()

		// status: 0-calling, 1-call done, 2-call timeout
		var callStatus uint32

		timer := time.AfterFunc(m.timeoutNS, func() {
			if !atomic.CompareAndSwapUint32(&callStatus, 0, 2) {
				return
			}

			result := &gen.CallResult{
				ID:     rawReq.ID,
				Req:    rawReq,
				Code:   gen.RET_CODE_WARNING_CALL_TIMEOUT,
				Msg:    fmt.Sprintf("Timeout! (expected: < %v)", m.timeoutNS),
				Elapse: m.timeoutNS,
			}

			m.sendResult(result)
		})

		rawResp := m.callOne(&rawReq)
		if !atomic.CompareAndSwapUint32(&callStatus, 0, 1) {
			return
		}
		timer.Stop()

		var result *gen.CallResult
		if rawResp.Err != nil {
			result = &gen.CallResult{
				ID:     rawResp.ID,
				Req:    rawReq,
				Code:   gen.RET_CODE_ERROR_CALL,
				Msg:    rawResp.Err.Error(),
				Elapse: rawResp.Elapse,
			}
		} else {
			result = m.caller.CheckResp(rawReq, *rawResp)
			result.Elapse = rawResp.Elapse
		}
		m.sendResult(result)
	}()
}

func (m *myGenerator) sendResult(result *gen.CallResult) bool {
	if atomic.LoadUint32(&m.status) != gen.STATUS_STARTED {
		m.printIgnoreResult(result, "stopped load generator")
		return false
	}

	select {
	case m.resultCh <- result:
		return true
	default:
		m.printIgnoreResult(result, "full result channel")
		return false
	}
}

func (m *myGenerator) callOne(rawReq *gen.RawReq) *gen.RawResp {
	atomic.AddInt64(&m.callCount, 1)

	if rawReq == nil {
		return &gen.RawResp{
			ID:  -1,
			Err: errors.New("Invalid raw request."),
		}
	}

	start := time.Now().UnixNano()
	resp, err := m.caller.Call(rawReq.Req, m.timeoutNS)
	end := time.Now().UnixNano()
	elapsedTime := time.Duration(end - start)

	var rawResp gen.RawResp
	if err != nil {
		errMsg := fmt.Sprintf("Sync Call Error: %s.", err)
		rawResp = gen.RawResp{
			ID:     rawReq.ID,
			Err:    errors.New(errMsg),
			Elapse: elapsedTime,
		}
	} else {
		rawResp = gen.RawResp{
			ID:     rawReq.ID,
			Resp:   resp,
			Elapse: elapsedTime,
		}
	}
	return &rawResp
}

func (m *myGenerator) printIgnoreResult(result *gen.CallResult, cause string) {
	resultMsg := fmt.Sprintf("ID=%d, Code=%d, Msg=%s, Elapse=%v",
		result.ID, result.Code, result.Msg, result.Elapse)

	log.Printf("Ignored result: %s. (cause: %s)\n", resultMsg, cause)
}

func (m *myGenerator) Start() bool {
	fmt.Println("Starting load generator...")

	if !atomic.CompareAndSwapUint32(&m.status, gen.STATUS_ORIGINAL,
		gen.STATUS_STARTING) {
		return false
	}

	var throttle <-chan time.Time
	if m.lps > 0 {
		interval := time.Duration(1e9 / m.lps)
		fmt.Printf("Setting throttle (%v)...\n", interval)
		throttle = time.Tick(interval)
	}

	m.ctx, m.cancelFunc = context.WithTimeout(context.Background(), m.durationNS)

	m.callCount = 0

	atomic.StoreUint32(&m.status, gen.STATUS_STARTED)

	go func() {
		fmt.Println("Generating loads...")
		m.genLoad(throttle)
		fmt.Printf("Stopped. (call count: %d)\n", m.callCount)
	}()

	return true
}

func (m *myGenerator) Stop() bool {
	if !atomic.CompareAndSwapUint32(&m.status, gen.STATUS_STARTED, gen.STATUS_STOPPING) {
		return false
	}

	m.cancelFunc()

	for {
		if atomic.LoadUint32(&m.status) == gen.STATUS_STOPPED {
			break
		}
		time.Sleep(time.Microsecond)
	}
	return true
}

func (m *myGenerator) Status() uint32 {
	return atomic.LoadUint32(&m.status)
}

func (m *myGenerator) CallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}
