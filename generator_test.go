package load_generator

import (
	"testing"
	"time"

	gen "github.com/huangaz/load-generator/gen"
	helper "github.com/huangaz/load-generator/testHelper"
)

var printDetail = false

func TestStart(t *testing.T) {
	server := helper.NewTCPServer()
	defer server.Close()
	serverAddr := "127.0.0.1:8080"
	t.Logf("Startup TCP server(%s)...\n", serverAddr)

	if err := server.Listen(serverAddr); err != nil {
		t.Fatalf("TCP Server startup failing! (addr=%s)!\n", serverAddr)
		t.FailNow()
	}

	pset := ParamSet{
		Caller:     helper.NewTCPComm(serverAddr),
		TimeoutNS:  50 * time.Millisecond,
		LPS:        uint32(1000),
		DurationNS: 10 * time.Second,
		ResultCh:   make(chan *gen.CallResult, 50),
	}
	t.Logf("Initialize load generator (timeoutNS=%v, lps=%d, durationNS=%v)...",
		pset.TimeoutNS, pset.LPS, pset.DurationNS)

	g, err := NewGenerator(pset)
	if err != nil {
		t.Fatalf("Load generator Initialization failing: %s\n", err)
		t.FailNow()
	}

	t.Log("Start load generator...")
	g.Start()

	countMap := make(map[gen.RetCode]int)
	for r := range pset.ResultCh {
		countMap[r.Code] = countMap[r.Code] + 1
		if printDetail {
			t.Logf("Result: ID=%d, Code=%d, Msg=%s, Elapse=%v.\n",
				r.ID, r.Code, r.Msg, r.Elapse)
		}
	}

	var total int
	t.Log("RetCode Count:")

	for k, v := range countMap {
		codePlain := gen.GetRetCodePlain(k)
		t.Logf("  Code plain: %s (%d), Count: %d.\n", codePlain, k, v)
		total += v
	}

	t.Logf("Total: %d.\n", total)
	successCount := countMap[gen.RET_CODE_SUCCESS]
	tps := float64(successCount) / float64(pset.DurationNS/1e9)
	t.Logf("Loads per second: %d; Treatments per second: %f.\n", pset.LPS, tps)
}

func TestStop(t *testing.T) {
	server := helper.NewTCPServer()
	defer server.Close()
	serverAddr := "127.0.0.1:8080"
	t.Logf("Startup TCP server(%s)...\n", serverAddr)

	if err := server.Listen(serverAddr); err != nil {
		t.Fatalf("TCP Server startup failing! (addr=%s)!\n", serverAddr)
		t.FailNow()
	}

	pset := ParamSet{
		Caller:     helper.NewTCPComm(serverAddr),
		TimeoutNS:  50 * time.Millisecond,
		LPS:        uint32(1000),
		DurationNS: 10 * time.Second,
		ResultCh:   make(chan *gen.CallResult, 50),
	}
	t.Logf("Initialize load generator (timeoutNS=%v, lps=%d, durationNS=%v)...",
		pset.TimeoutNS, pset.LPS, pset.DurationNS)

	g, err := NewGenerator(pset)
	if err != nil {
		t.Fatalf("Load generator Initialization failing: %s\n", err)
		t.FailNow()
	}

	// start
	t.Log("Start load generator...")
	g.Start()
	timeoutNS := 2 * time.Second
	time.AfterFunc(timeoutNS, func() {
		g.Stop()
	})

	countMap := make(map[gen.RetCode]int)
	count := 0
	for r := range pset.ResultCh {
		countMap[r.Code] = countMap[r.Code] + 1
		if printDetail {
			t.Logf("Result: ID=%d, Code=%d, Msg=%s, Elapse=%v.\n",
				r.ID, r.Code, r.Msg, r.Elapse)
		}
		count++
	}

	var total int
	t.Log("RetCode Count:")
	for k, v := range countMap {
		codePlain := gen.GetRetCodePlain(k)
		t.Logf("  Code plain: %s (%d), Count: %d.\n", codePlain, k, v)
		total += v
	}

	t.Logf("Total: %d.\n", total)
	successCount := countMap[gen.RET_CODE_SUCCESS]
	tps := float64(successCount) / float64(timeoutNS/1e9)
	t.Logf("Loads per second: %d; Treatments per second: %f.\n", pset.LPS, tps)
}
