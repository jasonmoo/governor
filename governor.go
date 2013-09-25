package governor

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Governor struct {
	sync.Mutex

	blocking   bool
	ready      uint32
	ready_cond *sync.Cond

	interval time.Duration
	f        func()
}

func NewGovernor(interval time.Duration, f func()) *Governor {
	return &Governor{
		ready:      1,
		ready_cond: new(sync.Cond),
		interval:   interval,
		f:          f,
	}
}

func (g *Governor) SetInterval(interval time.Duration) {
	g.Lock()
	g.interval = interval
	g.Unlock()
}
func (g *Governor) SetBlocking(blocking bool) {
	g.Lock()
	g.blocking = blocking
	g.Unlock()
}

func (g *Governor) SetFunc(f func()) {
	g.Lock()
	g.f = f
	g.Unlock()
}

func (g *Governor) Do() {

	ready := atomic.SwapInt64(&g.ready, 0)

	if ready == 0 {
		if g.blocking {
			g.ready_cond.Wait()
		}
	} else {
		defer func() {
			if r := recover(); r != nil {
				log.Println("governor do panic'd!", r)
			}

			if remainder := g.interval - time.Since(start); remainder > 0 {
				time.Sleep(remainder)
			}

			atomic.StoreUint32(&g.ready, 1)
			g.ready_cond.Broadcast()
		}()

		g.Lock()
		defer g.Unlock()

		start := time.Now()
		g.f()
	}
}
