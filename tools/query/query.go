package query

/**
  CAN命令队列  2019-04-11 luc
  队列支持线程安全
*/

import "sync"

type Element interface{}

type Query struct {
	element []Element
}

//线程安全锁
var lock sync.Mutex

func Create() *Query {
	return &Query{}
}

//存内容
func (entry *Query) Offer(e Element) {
	defer lock.Unlock()
	lock.Lock()
	entry.element = append(entry.element, e)
}

//取出内容
func (entry *Query) Poll() Element {
	defer lock.Unlock()
	lock.Lock()
	if entry.IsEmpty() {
		return nil
	}
	firstElement := entry.element[0]
	entry.element = entry.element[1:]
	return firstElement
}

func (entry *Query) Clear() bool {
	if entry.IsEmpty() {
		return false
	}

	defer lock.Unlock()
	lock.Lock()
	for i := 0; i < entry.Size(); i++ {
		entry.element[i] = nil
	}
	entry.element = nil
	return true
}

func (entry *Query) Size() int {
	return len(entry.element)
}

func (entry *Query) IsEmpty() bool {
	if len(entry.element) == 0 {
		return true
	}
	return false
}
