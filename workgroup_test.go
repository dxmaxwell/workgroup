package workgroup


import (
    "time"
    "testing"
)


func TestGoFor(t *testing.T) {
    g := &Group{}
    done := make(chan struct{})
    wait := make(chan struct{})

    g.GoFor(100, func(idx int) error {
        time.Sleep(time.Duration(100-idx)*time.Millisecond)
        done <- struct{}{}
        return nil
    })

    go func() {
        g.Wait()
        close(wait)
    }()

    count := 0
    for {
        select {
        case <- done:
            count += 1
        case <- wait:
            if count < 100 {
                t.Fatal("Worker done after group")
            }
            return
        }
    }
}


func TestGoForLimit(t *testing.T) {
    g := &Group{}
    start := make(chan struct{})
    done := make(chan struct{})
    wait := make(chan struct{})

    g.GoForLimit(100, 10, func(idx int) error {
        start <- struct{}{}
        time.Sleep(time.Duration(100-idx)*time.Millisecond)
        done <- struct{}{}
        return nil
    })

    go func() {
        g.Wait()
        close(wait)
    }()

    count := 0
    limit := 0
    for {
        select {
        case <- start:
            limit += 1
            if limit > 10 {
                t.Fatal("Worker limit exceeded")
            }
        case <- done:
            limit -= 1
            count += 1
        case <- wait:
            if count < 10 {
                t.Fatal("Worker done after group")
            }
            return
        }
    }
}
