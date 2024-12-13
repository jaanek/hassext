package esphome

import (
	"sync"
	"time"
)

func WaitChanResponse[T any](coverState <-chan *T, timeout time.Duration) *T {
	for {
		select {
		case state := <-coverState:
			return state
		case <-time.After(timeout):
			return nil
		}
	}
}

func WaitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
