package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	// enable profiling
	_ "net/http/pprof"
	"sync/atomic"
	"time"
)

type Foo struct {
	Bar uint64
}

func main() {
	var (
		total       uint64
		wg          sync.WaitGroup
		ctx, cancel = context.WithCancel(context.Background())
	)

	wg.Add(5)

	// write
	c := make(chan *Foo, 10000)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				close(c)
				return
			case c <- &Foo{Bar: 1}:
			}
		}
	}()

	// read
	for i := 0; i < 4; i++ {
		go func() {
			for bar := range c {
				atomic.AddUint64(&total, bar.Bar)
			}
			wg.Done()
		}()
	}

	// get profile via http
	// import _ "net/http/pprof"
	go http.ListenAndServe(":6060", nil)

	// print count after 2 minute
	time.Sleep(2 * time.Minute)
	fmt.Printf("\nCounter: %d\n", atomic.LoadUint64(&total))

	// cancel and wait done
	cancel()
	wg.Wait()
}
