package utils

import (
	"errors"
	"time"
)

//ErrGoTimeout ErrGoTimeout
var ErrGoTimeout = errors.New("GoTimeoutFunc")

//GoFunc GoFunc
func GoFunc(f func() error) chan error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return ch
}

//GoTimeoutFunc GoTimeoutFunc
func GoTimeoutFunc(timeout time.Duration, f func() error) chan error {
	ch := make(chan error)
	go func() {
		var err error
		select {
		case err = <-GoFunc(f):
			ch <- err
		case <-time.After(timeout):
			ch <- ErrGoTimeout
		}
	}()
	return ch
}

//GoTimeout GoTimeout
func GoTimeout(f func() error, timeout time.Duration) (err error) {
	done := make(chan bool)
	go func() {
		err = f()
		done <- true
	}()
	select {
	case <-time.After(timeout):
		return ErrGoTimeout
	case <-done:
		return
	}
}
