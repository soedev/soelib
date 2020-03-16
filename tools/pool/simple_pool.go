package pool

import "sync"

//SimplePool 简易协协程池
type SimplePool struct {
	ch chan struct{}
	wg *sync.WaitGroup
}

//NewSimplePool 实例化
// poolSize 协程池大小
// wgSize WaitGroup大小，为0时不等待
func NewSimplePool(poolSize, wgSize int) *SimplePool {
	p := &SimplePool{
		ch: make(chan struct{}, poolSize),
		wg: &sync.WaitGroup{},
	}
	if wgSize > 0 {
		p.wg.Add(wgSize)
	}
	return p
}

//Submit 提交任务
func (p *SimplePool) Submit(task func()) {
	p.ch <- struct{}{}
	go func() {
		defer func() {
			p.wg.Done()
			<-p.ch
		}()
		task()
	}()
}

//Wait 等待WaitGroup执行完毕
func (p *SimplePool) Wait() {
	p.wg.Wait()
}
