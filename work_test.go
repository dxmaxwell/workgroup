package workgroup

import (
	"context"
	"fmt"
	"sync"
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

type AccumulateManager struct {
	mutex   sync.Mutex
	manager Manager
	Errors  []error
}

func (m *AccumulateManager) Result() error {
	return m.manager.Result()
}

func (m *AccumulateManager) Manage(ctx Ctx, c Canceller, idx int, err error) int {
	n := m.manager.Manage(ctx, c, idx, err)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for len(m.Errors) < n {
		m.Errors = append(m.Errors, nil)
	}
	m.Errors[n-1] = err

	return n
}

func TestCancelNeverFirstError(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelNeverFirstError(),
	}

	err := WorkFor(context.Background(), len(counts), NewUnlimited(), m,
		func(ctx Ctx, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index >= 500 {
				err = fmt.Errorf("worker %d failed", index)
			}

			select {
			case <-ctx.Done():
				t.Errorf("Work group context cancelled")
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err == nil {
		t.Errorf("Work group error is nil")
	}
	for _, e := range m.Errors {
		if e != nil {
			if e != err {
				t.Fatal("Work group error is not first error")
			}
			break
		}
	}
}

func TestCancelOnFirstError(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelOnFirstError(),
	}

	err := WorkFor(context.Background(), len(counts), NewUnlimited(), m,
		func(ctx Ctx, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index >= 500 {
				err = fmt.Errorf("worker %d failed", index)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err == nil {
		t.Errorf("Work group error is nil")
	}
	errored := false
	for i, e := range m.Errors {
		if errored {
			if e == nil {
				t.Fatalf("Expecting accumulated error (%d) to not be nil", i)
			}
			if e != context.Canceled {
				t.Fatalf("Expecting accumulated error (%d) to be canceled", i)
			}
		} else {
			if e != nil {
				if e != err {
					t.Fatalf("Work group error is not first accumulated error (%d)", i)
				}
				errored = true
			}
		}
	}
}

func TestCancelOnFirstSuccess(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelOnFirstSuccess(),
	}

	err := WorkFor(context.Background(), len(counts), NewUnlimited(), m,
		func(ctx Ctx, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			if index < 500 {
				err = fmt.Errorf("worker %d failed", index)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err != nil {
		t.Errorf("Work group error is not nil: %s", err)
	}
	success := false
	for i, e := range m.Errors {
		if success {
			if e == nil {
				t.Fatalf("Expecting accumulated error (%d) to not be nil", i)
			}
			if e != context.Canceled {
				t.Fatalf("Expecting accumulated error (%d) to be canceled", i)
			}
		} else {
			if e != nil {
				if e == context.Canceled {
					t.Fatalf("Expecting accumulated error (%d) to not be canceled", i)
				}
			} else {
				success = true
			}
		}
	}
}

func TestCancelOnFirstComplete(t *testing.T) {

	counts := make([]int, 10000)

	m := &AccumulateManager{
		manager: CancelOnFirstComplete(),
	}

	err := WorkFor(context.Background(), len(counts), NewUnlimited(), m,
		func(ctx Ctx, index int) (err error) {
			time.Sleep(time.Millisecond)
			counts[index]++

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		},
	)

	for _, c := range counts {
		if c != 1 {
			t.Errorf("Worker %d has not completed", c)
		}
	}

	if err != nil {
		t.Errorf("Work group error is not nil: %s", err)
	}
	for i, e := range m.Errors {
		if i > 0 {
			if e != nil && e != context.Canceled {
				t.Fatalf("Expecting accumulated error (%d) to be canceled: %s", i, e)
			}
		} else {
			if e != nil {
				t.Fatalf("Expecting accumulated error (%d) to be nil: %s", i, e)
			}
		}
	}
}
