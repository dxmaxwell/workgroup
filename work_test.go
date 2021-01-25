package workgroup

import (
	"context"
	"testing"
	"time"
)

func TestSimpleWork(t *testing.T) {

	counts := make([]int, 10000)
	workers := make([]Worker, len(counts))
	for i := 0; i < len(counts); i++ {
		index := i
		workers[i] = func(ctx Ctx) error {
			time.Sleep(time.Millisecond)
			counts[index]++
			return nil
		}
	}

	Work(nil, nil, nil, workers...)

	for _, c := range counts {
		if c != 1 {
			t.Fail()
		}
	}
}

func TestSimpleWorkFor(t *testing.T) {

	counts := make([]int, 10000)

	WorkFor(nil, len(counts), nil, nil, func(ctx Ctx, index int) error {
		time.Sleep(time.Millisecond)
		counts[index]++
		return nil
	})

	for _, c := range counts {
		if c != 1 {
			t.Fail()
		}
	}
}

func TestSimpleWorkChan(t *testing.T) {

	counts := make([]int, 10000)
	workers := make(chan Worker)
	go func() {
		for i := 0; i < len(counts); i++ {
			index := i
			workers <- func(ctx Ctx) error {
				time.Sleep(time.Millisecond)
				counts[index]++
				return nil
			}
		}
		close(workers)
	}()

	WorkChan(nil, nil, nil, workers)

	for _, c := range counts {
		if c != 1 {
			t.Fail()
		}
	}
}

func TestLimitedWorkFor(t *testing.T) {

	counts := make([]int, 10000)
	tokens := make(chan struct{}, 8)

	WorkFor(nil, len(counts), NewLimited(8), nil, func(ctx Ctx, index int) error {
		select {
		case tokens <- struct{}{}:
			break
		default:
			t.Fail()
		}

		time.Sleep(time.Millisecond)
		counts[index]++

		select {
		case <-tokens:
			break
		default:
			t.Fail()
		}

		return nil
	})

	for _, c := range counts {
		if c != 1 {
			t.Fail()
		}
	}
}

func TestPoolWorkFor(t *testing.T) {

	counts := make([]int, 10000)
	tokens := make(chan struct{}, 8)

	ctx, cancel := context.WithCancel(context.Background())

	WorkFor(nil, len(counts), NewPool(ctx, 8), nil, func(ctx Ctx, index int) error {
		select {
		case tokens <- struct{}{}:
			break
		default:
			t.Fail()
		}

		time.Sleep(time.Millisecond)
		counts[index]++

		select {
		case <-tokens:
			break
		default:
			t.Fail()
		}

		return nil
	})

	cancel()

	for _, c := range counts {
		if c != 1 {
			t.Fail()
		}
	}
}
