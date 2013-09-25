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

func New(interval time.Duration, f func(...interface{})) *Governor {
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

func (g *Governor) SetFunc(f func(...interface{})) {
	g.Lock()
	g.f = f
	g.Unlock()
}

func (g *Governor) Do(args ...interface{}) {

	ready := atomic.SwapUint32(&g.ready, 0)

	if ready == 0 {
		if g.blocking {
			g.ready_cond.Wait()
		}
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println("governor do panic'd!", r)
		}

		if remainder := g.interval - time.Since(start); remainder > 0 {
			time.Sleep(remainder)
		}

		atomic.StoreUint32(&g.ready, 1)
		g.Unlock()
		g.ready_cond.Broadcast()
	}()

	g.Lock()
	g.f(args...)

}
