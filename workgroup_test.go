package workgroup

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func _GoAdaptor() func(Context, int, func(int) error) {
	return func(ctx Context, n int, f func(int) error) {
		for idx := 0; idx < n; idx++ {
			i := idx
			ctx.Go(func() error {
				return f(i)
			})
		}
	}
}

func _GoForAdaptor() func(Context, int, func(int) error) {
	return func(ctx Context, n int, f func(int) error) {
		ctx.GoFor(n, f)
	}
}

func _GoForLimitAdaptor(t *testing.T, limit int) func(Context, int, func(int) error) {
	return func(ctx Context, n int, f func(int) error) {
		ch := make(chan struct{}, limit)
		ctx.GoForPool(n, limit, func(i int) error {
			select {
			case ch <- struct{}{}:
				defer func() {
					<-ch
				}()
			default:
				t.Fatalf("Too many workers, expected to be limited to %d", cap(ch))
			}
			return f(i)
		})
	}
}

func _SimpleSetup(t *testing.T, gofor func(Context, int, func(int) error)) func(Context) {
	return func(ctx Context) {
		results := make([]int, 2000)

		gofor(ctx, len(results), func(i int) error {
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			results[i] = 1
			return nil
		})

		ctx.Defer(func(err *error, _results []error) {
			if *err != nil {
				t.Fatalf("defer 1: expecting *err == nil")
			}
			for idx, r := range results {
				if r != 2 {
					t.Fatalf("defer 1: expecting results[%d] == 2", idx)
				}
			}
		})

		ctx.Defer(func(err *error, _results []error) {
			if *err != nil {
				t.Fatalf("defer 2: expecting *err == nil")
			}
			for idx, r := range results {
				if r != 1 {
					t.Fatalf("defer 2: expecting results[%d] == 1", idx)
				}
				results[idx]++
			}
		})
	}
}

func TestSimpleAllGo(t *testing.T) {
	All(nil, _SimpleSetup(t, _GoAdaptor()))
}

func TestSimpleFirstGo(t *testing.T) {
	First(nil, _SimpleSetup(t, _GoAdaptor()))
}

func TestSimpleAllGoFor(t *testing.T) {
	All(nil, _SimpleSetup(t, _GoForAdaptor()))
}

func TestSimpleFirstGoFor(t *testing.T) {
	First(nil, _SimpleSetup(t, _GoForAdaptor()))
}

func TestSimpleAllGoForLimit(t *testing.T) {
	All(nil, _SimpleSetup(t, _GoForLimitAdaptor(t, 10)))
}

func TestSimpleFirstGoForLimit(t *testing.T) {
	First(nil, _SimpleSetup(t, _GoForLimitAdaptor(t, 10)))
}

func _ErrorSetup(t *testing.T, gofor func(Context, int, func(int) error)) func(Context) {
	return func(ctx Context) {

		ch := make(chan int)

		we := fmt.Errorf("worker error")

		results := make([]error, 2000)

		gofor(ctx, len(results), func(i int) error {
			select {
			case <-ctx.Done():
				results[i] = ctx.Err()
			case ch <- i:
				results[i] = we
			}
			return results[i]
		})

		var erridx int

		// ctx.Defer(func(err *error) {
		// 	if *err != nil {
		// 		t.Fatalf("defer 1: expecting *err == nil")
		// 	}
		// 	for idx, r := range results {
		// 		if r != 2 {
		// 			t.Fatalf("defer 1: expecting results[%d] == 2", idx)
		// 		}
		// 	}
		// })

		ctx.Defer(func(err *error, _results []error) {
			if *err != we {
				t.Fatalf("defer 2: expecting *err == we")
			}
			for idx, r := range results {
				if idx == erridx {
					if r != we {
						t.Fatalf("defer 2: expecting results[%d] == 1", idx)
					}
				} else {
					if r != context.Canceled {
						t.Fatalf("defer 2: expecting results[%d] == 1", idx)
					}
				}
			}

			// for idx, r := range results {
			// 	if r != 1 {
			// 		t.Fatalf("defer 2: expecting results[%d] == 1", idx)
			// 	}
			// 	results[idx]++
			// }
		})

		time.Sleep(time.Duration(10) * time.Millisecond)

		erridx = <-ch

	}
}

func TestErrorAllGo(t *testing.T) {
	All(nil, _ErrorSetup(t, _GoAdaptor()))
}

func TestAllGoWithError(t *testing.T) {
	nworkers := 1000
	done := make(chan error)
	wait := make(chan error)
	var ctxDone <-chan struct{}

	go func() {
		wait <- All(nil, func(ctx Context) {

			ctxDone = ctx.Done()

			worker := func() error {
				time.Sleep(time.Duration(100) * time.Millisecond)

				var err error
				if rand.Intn(2) == 0 {
					err = fmt.Errorf("worker error")
				}

				select {
				case <-ctx.Done():
					// fmt.Printf("Cancelled\n")
					done <- ctx.Err()
					return ctx.Err()
				case done <- err:
					// fmt.Printf("ERROR\n")
					return err
				}
			}

			for i := 0; i < nworkers; i++ {
				ctx.Go(worker)
			}
		})
	}()

	var count = 0
	var first error
	//var canceled = false
	for {
		select {
		case err := <-done:
			count++
			if err != nil && first == nil {

				first = err

				<-ctxDone

			}
			// fmt.Println(err)
			// if !canceled && err == context.Canceled {
			// 	canceled = true
			// }
			// if canceled && err != context.Canceled {
			// 	t.Fatalf("expecting canceled error\n")
			// }
		case err := <-wait:
			if err != first {
				t.Fatalf("Wrong Error!!")
			}
			if count < nworkers {
				t.Fatalf("Worker done after group %d\n", count)
			}
			return
		}
	}
}

func TestGoFor(t *testing.T) {

	done := make(chan struct{})
	wait := make(chan struct{})

	go func() {
		All(nil, func(ctx Context) {

			ctx.GoFor(1000, func(idx int) error {
				time.Sleep(time.Duration(100) * time.Millisecond)
				done <- struct{}{}
				return nil
			})
		})

		close(wait)
	}()

	count := 0
	for {
		select {
		case <-done:
			count++
		case <-wait:
			if count < 100 {
				t.Fatalf("Worker done after group %d\n", count)
			}
			return
		}
	}
}

func TestGoForLimit(t *testing.T) {

	done := make(chan struct{})
	wait := make(chan struct{})

	go func() {
		All(nil, func(ctx Context) {

			ctx.GoForPool(1000, 100, func(idx int) error {
				time.Sleep(time.Duration(100) * time.Millisecond)
				done <- struct{}{}
				return nil
			})
		})

		close(wait)
	}()

	count := 0
	for {
		select {
		case <-done:
			count++
		case <-wait:
			if count < 100 {
				t.Fatalf("Worker done after group %d\n", count)
			}
			return
		}
	}
}

// func TestGoForLimit(t *testing.T) {
// 	g := &Group{}
// 	start := make(chan struct{})
// 	done := make(chan struct{})
// 	wait := make(chan struct{})

// 	g.GoForLimit(100, 10, func(idx int) error {
// 		start <- struct{}{}
// 		time.Sleep(time.Duration(100-idx) * time.Millisecond)
// 		done <- struct{}{}
// 		return nil
// 	})

// 	go func() {
// 		g.Wait()
// 		close(wait)
// 	}()

// 	count := 0
// 	limit := 0
// 	for {
// 		select {
// 		case <-start:
// 			limit += 1
// 			if limit > 10 {
// 				t.Fatal("Worker limit exceeded")
// 			}
// 		case <-done:
// 			limit -= 1
// 			count += 1
// 		case <-wait:
// 			if count < 10 {
// 				t.Fatal("Worker done after group")
// 			}
// 			return
// 		}
// 	}
// }
