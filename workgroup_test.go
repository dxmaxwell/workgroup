package workgroup

import (
    "time"
    "strconv"
    "testing"
    "context"
    "math/rand"
)

type errint int

func (e errint) Error() string {
    return "Error: " + strconv.Itoa(int(e))
}

type cancelint int

func (e cancelint) Error() string {
    return "Cancelled: " + strconv.Itoa(int(e))
}


func RandomSleep() {
    time.Sleep(time.Duration(rand.Int63n(10000))*time.Microsecond)
}

func RandomAfter() <-chan time.Time {
    return time.After(time.Duration(rand.Int63n(10000))*time.Microsecond)
}

func WaitForDone(wg *Group, ctxDone <-chan struct{}, done chan int, n int, t *testing.T) {
  cancelled := false
  results := make([]bool, n)
  for {
      select {
      case idx := <- done:
          if results[idx] {
              t.Fatal("go routine sent done, but already done:", idx)
          }
          results[idx] = true

      case <- ctxDone:
        ctxDone = nil
        cancelled = true

      case err := <- wg.Errors():
          if err == nil {
            t.Fatal("errors channel received nil")
          }
          var idx int
          if cancelled {
            if i, ok := err.(cancelint); ok {
              idx = int(i)
            } else {
              t.Fatal("go routine sent error, expecting canelint")
            }
          } else if i, ok := err.(errint); ok {
            idx = int(i)
          } else {
            t.Fatal("go routine send error, expecting errorint")
          }
          if results[idx] {
              t.Fatal("go routine sent error, but already done:", idx)
          }
          results[idx] = true

      case <- wg.Done():
          for _, r := range results {
              if !r {
                  t.Fatal("work group done before go routines")
              }
          }
          return
      }
  }
}

func TestLoopWork0(t *testing.T) {
    LoopWork(New(), 1, t)
    LoopWork(New(), 10, t)
    LoopWork(New(), 100, t)
}

func TestLoopWork1(t *testing.T) {
    LoopWork(WithLimit(1), 1, t)
    LoopWork(WithLimit(1), 10, t)
    LoopWork(WithLimit(1), 100, t)
}

func TestLoopWork10(t *testing.T) {
    LoopWork(WithLimit(10), 1, t)
    LoopWork(WithLimit(10), 10, t)
    LoopWork(WithLimit(10), 100, t)
}


func LoopWork(wg *Group, n int, t *testing.T) {

    done := make(chan int)

    wg.GoN(n, func(idx int) error {
        RandomSleep()
        done <- idx
        return nil
    })

    WaitForDone(wg, nil, done, n, t)
}


func TestSliceWork0(t *testing.T) {
    SliceWork(New(), 1, t)
    SliceWork(New(), 10, t)
    SliceWork(New(), 100, t)
}

func TestSliceWork1(t *testing.T) {
    SliceWork(WithLimit(1), 1, t)
    SliceWork(WithLimit(1), 10, t)
    SliceWork(WithLimit(1), 100, t)
}

func TestSliceWork10(t *testing.T) {
    SliceWork(WithLimit(10), 1, t)
    SliceWork(WithLimit(10), 10, t)
    SliceWork(WithLimit(10), 100, t)
}

func SliceWork(wg *Group, n int, t *testing.T) {

    done := make(chan int)

    for i := 0; i < n; i += 1 {
        idx := i
        wg.Go(func() error {
            RandomSleep()
            done <- idx
            return nil
        })
    }

    WaitForDone(wg, nil, done, n, t)
}


func TestLoopWorkWithErrors0(t *testing.T) {
    LoopWorkWithErrors(WithErrors(0), 1, t)
    LoopWorkWithErrors(WithErrors(0), 10, t)
    LoopWorkWithErrors(WithErrors(0), 100, t)
}

func TestLoopWorkWithErrors1(t *testing.T) {
    LoopWorkWithErrors(WithErrors(1), 1, t)
    LoopWorkWithErrors(WithErrors(1), 10, t)
    LoopWorkWithErrors(WithErrors(1), 100, t)
}

func TestLoopWorkWithErrors10(t *testing.T) {
    LoopWorkWithErrors(WithErrors(10), 1, t)
    LoopWorkWithErrors(WithErrors(10), 10, t)
    LoopWorkWithErrors(WithErrors(10), 100, t)
}


func LoopWorkWithErrors(wg *Group, n int, t *testing.T) {

    done := make(chan int)

    wg.GoN(n, func(idx int) error {
        RandomSleep()
        if rand.Intn(4) == 0 {
            return errint(idx)
        }
        done <- idx
        return nil
    })

    WaitForDone(wg, nil, done, n, t)
}


func TestSliceWorkWithErrors0(t *testing.T) {
    SliceWorkWithErrors(WithErrors(0), 1, t)
    SliceWorkWithErrors(WithErrors(0), 10, t)
    SliceWorkWithErrors(WithErrors(0), 100, t)
}

func TestSliceWorkWithErrors1(t *testing.T) {
    SliceWorkWithErrors(WithErrors(1), 1, t)
    SliceWorkWithErrors(WithErrors(1), 10, t)
    SliceWorkWithErrors(WithErrors(1), 100, t)
}

func TestSliceWorkWithErrors10(t *testing.T) {
    SliceWorkWithErrors(WithErrors(10), 1, t)
    SliceWorkWithErrors(WithErrors(10), 10, t)
    SliceWorkWithErrors(WithErrors(10), 100, t)
}

func SliceWorkWithErrors(wg *Group, n int, t *testing.T) {

    done := make(chan int)

    for i := 0; i < n; i += 1 {
        idx := i
        wg.Go(func() error {
            RandomSleep()
            if rand.Intn(4) == 0 {
                return errint(idx)
            }
            done <- idx
            return nil
        })
    }

    WaitForDone(wg, nil, done, n, t)
}


func TestReuseWithErrors0(t *testing.T) {
    wg := WithErrors(0)
    LoopWorkWithErrors(wg, 10, t)
    SliceWorkWithErrors(wg, 10, t)
    LoopWorkWithErrors(wg, 100, t)
    SliceWorkWithErrors(wg, 100, t)
}

func TestReuseWithErrors1(t *testing.T) {
    wg := WithErrors(1)
    LoopWorkWithErrors(wg, 10, t)
    SliceWorkWithErrors(wg, 10, t)
    LoopWorkWithErrors(wg, 100, t)
    SliceWorkWithErrors(wg, 100, t)
}

func TestReuseWithErrors10(t *testing.T) {
    wg := WithErrors(10)
    LoopWorkWithErrors(wg, 10, t)
    SliceWorkWithErrors(wg, 10, t)
    LoopWorkWithErrors(wg, 100, t)
    SliceWorkWithErrors(wg, 100, t)
}


func TestMixedWorkWithErrors0(t *testing.T) {
    MixedWorkWithErrors(WithErrors(0), 1, t)
    MixedWorkWithErrors(WithErrors(0), 4, t)
    MixedWorkWithErrors(WithErrors(0), 10, t)
}

func TestMixedWorkWithErrors1(t *testing.T) {
    MixedWorkWithErrors(WithErrors(1), 1, t)
    MixedWorkWithErrors(WithErrors(1), 4, t)
    MixedWorkWithErrors(WithErrors(1), 10, t)
}

func TestMixedWorkWithErrors4(t *testing.T) {
    MixedWorkWithErrors(WithErrors(4), 1, t)
    MixedWorkWithErrors(WithErrors(4), 4, t)
    MixedWorkWithErrors(WithErrors(4), 10, t)
}

func MixedWorkWithErrors(wg *Group, n int, t *testing.T) {

    done := make(chan int)

    for i := 0; i < n; i += 1 {
        idx := i
        wg.Go(func() error {
            RandomSleep()
            if rand.Intn(4) == 0 {
                return errint(idx)
            }
            done <- idx
            return nil
        })
    }

    wg.GoN(n, func(idx int) error {
        RandomSleep()
        if rand.Intn(4) == 0 {
            return errint(n+idx)
        }
        done <- (n+idx)
        return nil
    })

    for i := 0; i < n; i += 1 {
        idx := i
        wg.Go(func() error {
            RandomSleep()
            if rand.Intn(4) == 0 {
                return errint(2*n+idx)
            }
            done <- (2*n+idx)
            return nil
        })
    }

    wg.GoN(n, func(idx int) error {
        RandomSleep()
        if rand.Intn(4) == 0 {
            return errint(3*n+idx)
        }
        done <- (3*n+idx)
        return nil
    })

    WaitForDone(wg, nil, done, 4*n, t)
}



func TestContextWithErrors0(t *testing.T) {
    bg := context.Background()

    wg, ctx := WithContext(bg, 0, true)
    ContextWithErrors(ctx, wg, 1, t)
    wg, ctx = WithContext(bg, 0, true)
    ContextWithErrors(ctx, wg, 10, t)
    //wg, ctx = WithContext(context.TODO(), 0, true)
    //ContextWithErrors(ctx, wg, 100, t)
}


func ContextWithErrors(ctx context.Context, wg *Group, n int, t *testing.T) {

    done := make(chan int)

    for i := 0; i < n; i += 1 {
        idx := i
        wg.Go(func() error {
            select {
            case <- RandomAfter():
                break
            case <- ctx.Done():
                return cancelint(idx)
            }
            if idx == 0 {
                return errint(idx)
            }
            done <- idx
            return nil
        })
    }

    WaitForDone(wg, ctx.Done(), done, n, t)
}
