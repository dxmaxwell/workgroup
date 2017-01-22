package workgroup


import (
    "sync"
    "context"
)

type Group struct {

    cancel func()
    count int
    stop bool
    err error

    start chan struct{}
    stop chan struct{}

    work chan func() error
    done chan struct{}
    err error
}


func New() *Group {
    g := &Group{
      work: make(chan)
      start: make(chan struct{}, 1),
      stop: make(chan struct{}),
      done: make(chan struct{}),
      errDone: make(chan struct {}),
    }
    return g
}


func WithContext(ctx context.Context) (*Group, context.Context) {
    ctx, cancel := context.WithCancel(ctx)
    g := &Group{
      //cancel: cancel,
       shutdown: make(chan struct{}),
      done: make(chan struct{}),
      errDone: make(chan struct {}),
    }
    return g, ctx
}


func (g *Group) Go(f func() error) {
    g.Lock()
    defer g.Unlock()
    if g.stop {
        panic("")
    }

    g.count += 1
    go func() {
        err := f()
        g.Lock()
        defer g.Unlock()
        if g.err == nil && err != nil {
            g.err = err
            if g.cancel != nil {
              g.cancel()
            }
        }
        count -= 1
        if g.stop && count == 0 {
            if g.cancel != nil {
              g.cancel()
            }
            close(g.done)
        }
    }()


    select {
    default:
        g.count += 1
        go func() {
            err := f()
            g.Lock()
            defer g.Unlock()
            if g.err == nil && err != nil {
                g.err = err
            }
            count -= 1
            if g.stop && count == 0 {
                if g.cancel != nil {
                  g.cancel()
                }
                close(done)
            }
        }()
    case <- g.done:
        panic("!!!!!!")
    }


    g.procOnce.Do(g.process)
    for {
        select {
        case g.start <- f:
            return
        case <- g.done:
            panic("!!!!!")
        }
    }
}

func (g *Group) GoFor(n int, f func(int) error) {
    g.Go(func () error {
        for i := 0; i < n; i += 1 {
            idx := i
            g.Go(func() error {
                return f(idx)
            })
        }
        return nil
    })
}


func (g *Group) GoForLimit(n int, limit int, f func(int) error) {
    if n <= limit {
        g.GoFor(n, f)
        return
    }

    limiter := make(chan struct{}, limit)

    g.Go(func() error {
        for i := 0; i < n; i += 1 {
            idx := i
            limiter <- struct{}{}
            g.Go(func() error {
                defer func() {
                    <- limiter
                }()
                return f(idx)
            })
        }
        return nil
    })
}


func (g *Group) Done() <-chan struct{} {
    g.Lock()
    defer g.UnLock()

    if g.stop {
      return g.done
    }

    g.stop = true
    if count == 0 {
        if g.cancel != nil {
          g.cancel()
        }
        close(g.close)
    }
    return g.close


    return g.done
    g.stop = true
    if count == 0 {
      close(g.done)
    }


    g.procOnce.Do(g.process)
    for {
        select {
        case g.shutdown <- struct{}{}:
            return g.done
        case <- g.done:
            return g.done
        }
    }
}


func (g *Group) Wait() error {
    <- g.Done()
    return g.Err()
}


func (g *Group) ErrDone() <-chan struct{} {
    return g.errDone
}

func (g *Group) Err() error {
    return g.err
}

func (g *Group) process() {
    go func () {
        // defer func() {
        //     //if g.cancel != nil {
        //     //    g.cancel()
        //     //}
        //     close(g.done)
        // }()

        count := 0
        shutdown := false
        done := make(chan error)

        for {
            select {
            case f := <- g.start:
                count += 1
                go func() {
                    done <- f()

                    g.Lock()
                    defer g.Unlock()
                    count -= 1
                    if g.err == nil && err != nil {
                      g.err = err
                    }
                    if count == 0 {
                      close(g.done)
                    }
                }()

            case err := <- done:
                count -= 1
                if g.err == nil && err != nil {
                    g.err = err
                    close(g.errDone)
                    //if g.cancel != nil {
                    //    g.cancel()
                    //}
                }
                if shutdown && count == 0 {
                    close(g.done)
                    return
                }

            case <- g.shutdown:
                shutdown = true
                if count == 0 {
                    close(g.done)
                    return
                }
            }
        }
    }()
}
