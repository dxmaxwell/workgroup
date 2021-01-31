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

	Work(context.Background(), NewUnlimited(), CancelNeverFirstError(), workers...)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}
}

func TestSimpleWorkFor(t *testing.T) {

	counts := make([]int, 10000)

	WorkFor(context.Background(), len(counts), NewUnlimited(), CancelNeverFirstError(),
		func(ctx Ctx, index int) error {
			time.Sleep(time.Millisecond)
			counts[index]++
			return nil
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
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

	WorkChan(context.Background(), NewUnlimited(), CancelNeverFirstError(), workers)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}
}

func TestLimitedWorkFor(t *testing.T) {

	counts := make([]int, 10000)
	tokens := make(chan struct{}, 8)

	WorkFor(context.Background(), len(counts), NewLimited(8), CancelNeverFirstError(),
		func(ctx Ctx, index int) error {
			select {
			case tokens <- struct{}{}:
				break
			default:
				t.Errorf("Worker %d must wait to send token", index)
			}

			time.Sleep(time.Millisecond)
			counts[index]++

			select {
			case <-tokens:
				break
			default:
				t.Errorf("Worker %d must wait to recieve token", index)
			}

			return nil
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Work %d has not completed", c)
		}
	}
}

func TestPoolWorkFor(t *testing.T) {

	counts := make([]int, 10000)
	tokens := make(chan struct{}, 8)

	ctx, cancel := context.WithCancel(context.Background())

	WorkFor(ctx, len(counts), NewPool(ctx, 8), CancelNeverFirstError(),
		func(ctx Ctx, index int) error {
			select {
			case tokens <- struct{}{}:
				break
			default:
				t.Errorf("Worker %d must wait to send token", index)
			}

			time.Sleep(time.Millisecond)
			counts[index]++

			select {
			case <-tokens:
				break
			default:
				t.Errorf("Worker %d must wait to recieve token", index)
			}

			return nil
		},
	)

	cancel()

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Work %d has not completed", c)
		}
	}
}
