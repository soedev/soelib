package query

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestSyncQuery(t *testing.T) {
	queue := Create()
	var wg sync.WaitGroup
	wg.Add(2)
	go Offer(queue, &wg)
	go Poll(queue, &wg)
	wg.Wait()
	fmt.Println(queue.Size())
}

func TestTestNil(t *testing.T) {
	queue := Create()
	fmt.Println(queue.Size())
	fmt.Println("s")
	fmt.Println(queue.Poll())
	fmt.Println("e")
}
func Offer(q *Query, w *sync.WaitGroup) {
	defer w.Done()
	for i := 10; i > 0; i-- {
		time.Sleep(time.Duration(rand.Intn(6)) * time.Second) //延迟
		q.Offer(i)
	}
}

func Poll(q *Query, w *sync.WaitGroup) {
	defer w.Done()
	for i := 0; i < 10; i++ {
		time.Sleep(time.Duration(rand.Intn(6)) * time.Second) //延迟
		fmt.Println(q.Poll())
	}
}
