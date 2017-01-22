package workgroup


import (
    "sync"
    "container/list"
)

type workTask struct {
    f func(int) error
    total int
    next int

}

type deferTask struct {
    f func(error)
}


type WorkGroup struct {
    limit int

    dead chan struct{}
    workTasks chan *workTask
    deferTasks chan *deferTask

    wg sync.WaitGroup
    err error
}


func NewWorkGroup(limit int) *WorkGroup {
    g := &WorkGroup{
      limit: limit,
      dead: make(chan struct{}, 1),
      workTasks: make(chan *workTask),
      deferTasks: make(chan *deferTask),
    }
    g.dead <- struct{}{}
    return g
}


func (g *WorkGroup) Go(f func() error...) {
    g.GoN(1, func(int) error {
        return f()
    })
}


func (g *WorkGroup) GoN(n int, f func(int) error) {
    if n < 1 {
      return
    }

    wt := &workTask{ f:f, total:n }
    for {
        select {
        case g.workTasks <- wt:
            return
        case <- g.dead:
            go g.process()
        }
    }
}


func (g *WorkGroup) Defer(f func(error)) {
    dt := &deferTask{ f:f }
    for {
        select {
        case g.deferTasks <- dt:
            return
        case <- g.dead:
            go g.process()
        }
    }
}


func (g *WorkGroup) Wait() error {
    g.wg.Wait()
    return g.err
}


func (g *WorkGroup) process() {
    g.wg.Add(1)
    defer g.wg.Done()
    defer func() { g.dead <- struct{}{} }()

    count := 0

    done := make(chan error)
    workList := list.New()
    deferList := list.New()

    doWorkTasks := func() {
        for (g.limit <= 0 || g.limit > count) && workList.Len() > 0 {
            e := workList.Front()
            wt := e.Value.(*workTask)
            go func(f func(int) error, n int) {
              done <- f(n)
            }(wt.f, wt.next)
            count += 1
            wt.next += 1
            if wt.next == wt.total {
                workList.Remove(e)
            }
        }
    }

    doDeferTasks := func() {
      for deferList.Len() > 0 {
          e := deferList.Front()
          dt := e.Value.(*deferTask)
          dt.f(g.err)
          deferList.Remove(e)
      }
    }

    for {
        select {
        case wt := <- g.workTasks:
            workList.PushBack(wt)
            doWorkTasks()

        case dt := <- g.deferTasks:
            deferList.PushFront(dt)
            if count == 0 {
                doDeferTasks()
                return
            }

        case err := <- done:
            if g.err == nil && err != nil {
                g.err = err
                //if g.cancel != nill {
                //    g.cancel()
                //}
            }
            count -= 1
            doWorkTasks()
            if count == 0 {
              doDeferTasks()
              return
            }
        }
    }
}
