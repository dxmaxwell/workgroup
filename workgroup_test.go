package workgroup

import (
    "time"
    "strconv"
    "testing"
    //"context"
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
  //cancelled := false
  results := make([]bool, n)
  for {
      select {
      case idx := <- done:
          if results[idx] {
              t.Fatal("go routine sent done, but already done:", idx)
          }
          results[idx] = true

    //   case <- ctxDone:
    //     ctxDone = nil
    //     cancelled = true

    //   case err := <- wg.Errors():
    //       if err == nil {
    //         t.Fatal("errors channel received nil")
    //       }
    //       var idx int
    //       if cancelled {
    //         if i, ok := err.(cancelint); ok {
    //           idx = int(i)
    //         } else {
    //           t.Fatal("go routine sent error, expecting canelint")
    //         }
    //       } else if i, ok := err.(errint); ok {
    //         idx = int(i)
    //       } else {
    //         t.Fatal("go routine send error, expecting errorint")
    //       }
    //       if results[idx] {
    //           t.Fatal("go routine sent error, but already done:", idx)
    //       }
    //       results[idx] = true

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


func TestWorkWithLimit0(t *testing.T) {
    doWorkWithLimit(New(), 1, 10000, t)
    doWorkWithLimit(New(), 10, 10000, t)
    doWorkWithLimit(New(), 100, 10000, t)
}


func TestWorkWithLimit1(t *testing.T) {
    doWorkWithLimit(New(), 1, 1, t)
    doWorkWithLimit(New(), 10, 1, t)
    doWorkWithLimit(New(), 100, 1, t)
}


func TestWorkWithLimit10(t *testing.T) {
    doWorkWithLimit(New(), 1, 10, t)
    doWorkWithLimit(New(), 10, 10, t)
    doWorkWithLimit(New(), 100, 10, t)
}

func doWorkWithLimit(g *Group, n int, limit int, t *testing.T) {

    done := make(chan int)

    g.GoForLimit(n, limit, func(idx int) error {
        RandomSleep()
        done <- idx
        return nil
    })

    WaitForDone(g, nil, done, n, t)
}


func TestLoopWorkWithErrors0(t *testing.T) {
    doWorkWithErrors(New(), 1, 10000, t)
    doWorkWithErrors(New(), 10, 10000, t)
    doWorkWithErrors(New(), 100, 10000, t)
}
// 
// func TestLoopWorkWithErrors1(t *testing.T) {
//     doWorkWithErrors(WithErrors(), 1, 1, t)
//     doWorkWithErrors(WithErrors(), 10, 1, t)
//     doWorkWithErrors(WithErrors(), 100, 1, t)
// }
// 
// func TestLoopWorkWithErrors10(t *testing.T) {
//     doWorkWithErrors(WithErrors(), 1, 10, t)
//     doWorkWithErrors(WithErrors(), 10, 10, t)
//     doWorkWithErrors(WithErrors(), 100, 10, t)
// }
// 
func doWorkWithErrors(g *Group, n int, limit int, t *testing.T) {

    done := make(chan int)

    g.GoForLimit(n, limit, func(idx int) error {
        RandomSleep()
        if rand.Intn(4) == 0 {
            done <- idx
            return errint(idx)
        }
        done <- idx
        return nil
    })

    WaitForDone(g, nil, done, n, t)
}


// func TestReuseWithErrors0(t *testing.T) {
//     g := WithErrors()
//     doWorkWithErrors(g, 1, 10000, t)
//     doWorkWithErrors(g, 10, 10000, t)
//     doWorkWithErrors(g, 100, 10000, t)
// }
// 
// func TestReuseWithErrors1(t *testing.T) {
//     g := WithErrors()
//     doWorkWithErrors(g, 1, 1, t)
//     doWorkWithErrors(g, 10, 1, t)
//     doWorkWithErrors(g, 100, 1, t)
// }
// 
// func TestReuseWithErrors10(t *testing.T) {
//     g := WithErrors()
//     doWorkWithErrors(g, 1, 10, t)
//     doWorkWithErrors(g, 10, 10, t)
//     doWorkWithErrors(g, 100, 10, t)
// }
// 
// 
// func TestContextWithErrors0(t *testing.T) {
//     b := context.Background()
//     g, ctx := WithCtxErrors(b)
//     doWorkWithContext(g, ctx, 1, t)
//     g, ctx = WithCtxErrors(b)
//     doWorkWithContext(g, ctx, 10, t)
//     g, ctx = WithCtxErrors(b)
//     doWorkWithContext(g, ctx, 100, t)
// }
// 
// 
// func doWorkWithContext(wg *Group, ctx context.Context, n int, t *testing.T) {
// 
//     done := make(chan int)
// 
//     for i := 0; i < n; i += 1 {
//         idx := i
//         wg.Go(func() error {
//             select {
//             case <- RandomAfter():
//                 break
//             case <- ctx.Done():
//                 return cancelint(idx)
//             }
//             if idx == 0 {
//                 return errint(idx)
//             }
//             done <- idx
//             return nil
//         })
//     }
// 
//     WaitForDone(wg, ctx.Done(), done, n, t)
// }
