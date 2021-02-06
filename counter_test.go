package workgroup

//
// Evaluate the performance difference between a
// Lock-based and Lock-free concurrent counter.
//
// Benchmarks show that the Lock-free version is
// only marginally more performant.
//

import (
	"sync"
	"sync/atomic"
	"testing"
)

type atomicCounter struct {
	count uint64
}

const atomicCounterNErrorShift = 62
const atomicCounterNErrorBits = uint64(0b11) << atomicCounterNErrorShift
const atomicCounterNSuccessShift = 60
const atomicCounterNSuccessBits = uint64(0b11) << atomicCounterNSuccessShift
const atomicCounterNCompleteBits = ^(atomicCounterNSuccessBits | atomicCounterNErrorBits)

func (c *atomicCounter) add(err error) (int, int, int) {
	for {
		count := atomic.LoadUint64(&c.count)
		ncomplete := count & atomicCounterNCompleteBits
		nsuccess := count & atomicCounterNSuccessBits >> atomicCounterNSuccessShift
		nerror := (count & atomicCounterNErrorBits) >> atomicCounterNErrorShift

		if err != nil {
			if nerror < 0b11 {
				nerror++
			}
		} else {
			if nsuccess < 0b11 {
				nsuccess++
			}
		}
		if ncomplete < atomicCounterNCompleteBits {
			ncomplete++
		}

		newCount := ncomplete
		newCount |= nsuccess << atomicCounterNSuccessBits
		newCount |= nerror << atomicCounterNErrorShift

		if atomic.CompareAndSwapUint64(&c.count, count, newCount) {
			return int(ncomplete), int(nsuccess), int(nerror)
		}
	}
}

func BenchmarkAtomicCounter10(b *testing.B) {
	benchmarkAtomicCounter(10, b)
}
func BenchmarkAtomicCounter100(b *testing.B) {
	benchmarkAtomicCounter(100, b)
}
func BenchmarkAtomicCounter1000(b *testing.B) {
	benchmarkAtomicCounter(1000, b)
}
func BenchmarkAtomicCounter10000(b *testing.B) {
	benchmarkAtomicCounter(10000, b)
}
func BenchmarkAtomicCounter100000(b *testing.B) {
	benchmarkAtomicCounter(100000, b)
}

func benchmarkAtomicCounter(t int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		counter := &atomicCounter{}
		// var count uint64
		wg := sync.WaitGroup{}
		wg.Add(t)
		for i := 0; i < t; i++ {
			go func() {
				defer wg.Done()
				counter.add(nil)
				//atomic.AddUint64(&count, 1)
			}()
		}
		wg.Wait()
	}
}

type mutexCounter struct {
	mtx      sync.Mutex
	nerror   int
	nsuccess int
}

func (c *mutexCounter) add(err error) (int, int, int) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if err != nil {
		c.nerror++
	} else {
		c.nsuccess++
	}
	return c.nsuccess + c.nerror, c.nsuccess, c.nerror
}

func BenchmarkMutexCounter10(b *testing.B) {
	benchmarkAtomicCounter(10, b)
}
func BenchmarkMutexCounter100(b *testing.B) {
	benchmarkAtomicCounter(100, b)
}
func BenchmarkMutexCounter1000(b *testing.B) {
	benchmarkAtomicCounter(1000, b)
}
func BenchmarkMutexCounter10000(b *testing.B) {
	benchmarkMutexCounter(10000, b)
}

func BenchmarkMutexCounter100000(b *testing.B) {
	benchmarkMutexCounter(100000, b)
}

func benchmarkMutexCounter(t int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		counter := &mutexCounter{}
		wg := sync.WaitGroup{}
		wg.Add(t)
		for i := 0; i < t; i++ {
			go func() {
				defer wg.Done()
				counter.add(nil)
			}()
		}
		wg.Wait()
	}
}
