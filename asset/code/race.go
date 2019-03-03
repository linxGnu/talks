package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

type Bar struct {
	Value int
}

type Foo struct {
	Bar *Bar
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var foo Foo

	// Read
	var counter uint64
	for i := 0; i < 4; i++ {
		go func(seed int) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if foo.Bar != nil {
						atomic.AddUint64(&counter, 1)
						if foo.Bar.Value > 100 {
							fmt.Println(foo.Bar)
						}
					}
				}
			}
		}(i)
	}

	// write
	for i := 0; i < 8; i++ {
		go func(seed int) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if seed <= 6 {
						foo.Bar = &Bar{Value: seed}
					} else {
						foo.Bar = nil
					}
				}
			}
		}(i)
	}

	time.Sleep(2 * time.Minute)
	cancel()
}
