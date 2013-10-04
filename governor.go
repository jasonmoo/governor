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
	last_run   map[string]time.Time
	interval   time.Duration
	f          func(string)
}

func New(interval time.Duration, f func(string)) *Governor {
	return &Governor{
		ready:      1,
		ready_cond: new(sync.Cond),
		interval:   interval,
		last_run:   make(map[string]time.Time),
		f:          f,
	}
}

func (g *Governor) Reset() {
	g.Lock()
	g.last_run = make(map[string]time.Time)
	g.Unlock()
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
	// if we're turning blocking off release any hangers on
	if !blocking {
		g.ready_cond.Broadcast()
	}
}

func (g *Governor) SetFunc(f func(string)) {
	g.Lock()
	g.f = f
	g.Unlock()
}

func (g *Governor) Do(name string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("governor do panic'd!", r)
		}
	}()

	// swap with 0 ensures only the first ready swap will get a 1
	if ready := atomic.SwapUint32(&g.ready, 0); ready == 0 {
		if g.blocking {
			g.ready_cond.Wait()
		}
		return
	}

	// lock
	g.Lock()
	defer func() {
		// let one in
		atomic.StoreUint32(&g.ready, 1)
		// clear the rest
		if g.blocking {
			g.ready_cond.Broadcast()
		}
		// unlock the fun
		g.Unlock()
	}()

	g.f(name)

	if last_run, exists := g.last_run[name]; !exists {
		g.last_run[name] = time.Now()
		time.Sleep(g.interval)
	} else if remainder := g.interval - time.Since(last_run); remainder > 0 {
		g.last_run[name] = time.Now()
		time.Sleep(remainder)
	}

}
