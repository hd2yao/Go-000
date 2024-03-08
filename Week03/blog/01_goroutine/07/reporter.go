package main

import (
    "fmt"
    "sync"
    "time"
)

// Reporter 埋点服务上报
type Reporter struct {
    worker  int
    message chan string
    wg      sync.WaitGroup
    closed  bool
    //closed  chan struct{}
    //once    sync.Once
}

func NewReporter(worker, buffer int) *Reporter {
    return &Reporter{
        worker:  worker,
        message: make(chan string, buffer),
        //closed:  make(chan struct{}),
    }
}

func (r *Reporter) run(stop <-chan struct{}) {
    go func() {
        <-stop
        r.shutdown()
    }()

    for i := 0; i < r.worker; i++ {
        r.wg.Add(1)
        go func() {
            for msg := range r.message {
                time.Sleep(5 * time.Second)
                fmt.Printf("report: %s\n", msg)
            }
            r.wg.Done()
        }()
    }
    r.wg.Wait()
}

func (r *Reporter) shutdown() {
    r.closed = true
    // 注意，这个一定要在主服务结束之后再执行，避免关闭 channel 还有其他地方在写入
    close(r.message)
}

func (r *Reporter) report(data string) {
    if r.closed {
        return
    }
    r.message <- data
}
