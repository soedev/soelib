package ants

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
	GiB // 1073741824
	TiB // 1099511627776             (超过了int32的范围)
	PiB // 1125899906842624
	EiB // 1152921504606846976
	ZiB // 1180591620717411303424    (超过了int64的范围)
	YiB // 1208925819614629174706176
)
const (
	Param    = 100
	AntsSize = 1000
	TestSize = 10000
	n        = 100000
)

var curMem uint64

func demoPoolFunc(args interface{}) {
	n := args.(int)
	time.Sleep(time.Duration(n) * time.Millisecond)
}

func TestAntsPoolWaitToGetWorker(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := NewPool(AntsSize)
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		p.Submit(func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()
	t.Logf("pool, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestAnts(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	stopCh := make(chan struct{})
	p, _ := NewPool(DEFAULT_ANTS_POOL_SIZE)
	p.Submit(func() {
		worker(stopCh)
		wg.Done()
	})

	wg.Add(1)
	p.Submit(func() {
		worker1(stopCh)
		wg.Done()
	})

	defer p.Release()
	time.Sleep(time.Second * 2)
	close(stopCh)
	stopCh = make(chan struct{})

	wg.Add(1)
	p.Submit(func() {
		worker1(stopCh)
		wg.Done()
	})

	time.Sleep(time.Second * 2)
	close(stopCh)

	wg.Wait()
	fmt.Printf("running goroutines: %d\n", p.Running())

}
func worker(stopCh <-chan struct{}) {
	defer fmt.Println("worker exit")
	t := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-stopCh:
			fmt.Println("Recv stop signal")
			return
		case <-t.C:
			fmt.Println("Working .")
		}
	}
}

func worker1(stopCh <-chan struct{}) {
	defer fmt.Println("worker1 exit")
	t := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-stopCh:
			fmt.Println("Recv1 stop signal")
			return
		case <-t.C:
			fmt.Println("Working1 .")
		}
	}
}

func TestPanicHandler(t *testing.T) {
	defer Release()

	runTimes := 1000

	// Use the common pool.
	var wg sync.WaitGroup
	//syncCalculateSum := func() {
	//	demoFunc()
	//	wg.Done()
	//}
	//for i := 0; i < runTimes; i++ {
	//	wg.Add(1)
	//	_ = Submit(syncCalculateSum)
	//}
	//wg.Wait()
	//fmt.Printf("running goroutines: %d\n", Running())
	//fmt.Printf("finish all tasks.\n")
	//
	// Use the pool with a function,
	// set 10 to the capacity of goroutine pool and 1 second for expired duration.
	p, _ := NewPoolWithFunc(10, func(i interface{}) {
		myFunc(i)
		wg.Done()
	})
	defer p.Release()
	// Submit tasks one by one.
	for i := 0; i < runTimes; i++ {
		wg.Add(1)
		_ = p.Invoke(int32(i))
	}
	wg.Wait()
	fmt.Printf("running goroutines: %d\n", p.Running())
	fmt.Printf("finish all tasks, result is %d\n", sum)
}

var sum int32

func myFunc(i interface{}) {
	n := i.(int32)
	atomic.AddInt32(&sum, n)
	fmt.Printf("run with %d\n", n)
}

func demoFunc() {
	time.Sleep(10 * time.Millisecond)
	fmt.Println("Hello World!")
}
