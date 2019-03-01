### Efficient concurrency in Go

Tran Tuan Linh @ LINE Corp

linxGnu @ github

---

## Overview

- Writing high intensive read/write metrics database
- 40-80 Billion data points / day (and more)

---

### Decision

In-memory, In-Go

---

### Why Go

- Simple
- Fast
- Productivity
- Easy to write concurrency
- Awesome third-party libraries
- Very nice **pprof** tooling

<em>And, WE love Go!</em><!-- .element: class="fragment fade-up" -->

---

### Processing approach

- Share nothing on write
- On-demand Copy-on-write

---

### Share nothing on write

- No write contention <!-- .element: class="fragment fade-up" -->
- No lock/mutex, less context switching <!-- .element: class="fragment fade-up" -->
- Millions data points / sec (or more) <!-- .element: class="fragment fade-up" -->
 
<em>"Less lock, more speed"</em> <!-- .element: class="fragment fade-up" -->

---

### Copy-on-write

- We still have `"Query"` (many requests at a time)
- `COW` is a good way to deal with `consistency`

`+ Share-nothing == Awesome`<!-- .element: class="fragment fade-up" -->

---

### Single write - multiple read

```go
// Thread safe multi reading
data = storage.Load().([]byte)

// Thread safe writing - consistent for single write
data = storage.Load().([]byte)             // Load
newData = make([]byte, len(data) + 8)      // Allocation
copy(newData, data)                        // Copy
endian.PutUint64(newData[len(data):], v)   // Write
storage.Store(data)                        // Save
```

---

### Cons

- Memory allocation - GC stress
- `Copy` is not `zero-cost`
- When `Write >> Read` workload ?

=> Let's improve it! <!-- .element: class="fragment fade-up" -->

---

### On-demand COW

- Special `sync.RWMutex`
- Lockless
- Lightweight
- x3-x10 performance gain!

---

<img src="https://gyazo.linecorp.com/images/190226/a50019ba32e8f2cccb2f9182989cdffd.png" style="height:600px"/>

---

## So far

- Go help us writing systems in very productivity
- Just code -> `go run` -> `sleep well`
- Go is fast and safe `if there is no bug in your code` <!-- .element: class="fragment fade-up" -->

---

### Things not in heaven

---

### Channel

- Awesome
- But slow when pushing `million` points through

---

<img src="https://gyazo.linecorp.com/images/190228/77dd180f88867a55ddbd649dee8a1220.png" style="height:600px" />

---

```elm
time go run chan.go

Counter: 475070402

real	2m0.705s
user	5m18.062s
sys	1m24.319s
```

---

### Make it better

- Buffering at each layer of pipeline <!-- .element: class="fragment fade-up" -->
- Send a slice instead of single point <!-- .element: class="fragment fade-up" -->

---

<img src="https://gyazo.linecorp.com/images/190228/213a49a3f229d90bd5a99ca6133af84d.png" style="height:600px" />

---

```elm
time go run batch_chan.go

Counter: 3094277120

real	2m0.706s
user	2m42.327s
sys	0m25.084s
```

---

## 6-7x faster! 
### But why?

---

<img src="https://gyazo.linecorp.com/images/190228/6c050cfb005128d74d42e2e723d161c5.png" style="height:600px" />

---

### `sync/atomic`

- Highly recommend
- Fast
- Safe


```go
// mutex way
lock.RLock()
c = core.Config
lock.RUnlock()

// atomic way
var Config atomic.Value
core.Config.Store(loadConfig())
c = core.Config.Load().(*model.Config)
```

---

### `sync/atomic`

But use it carefully or you will be spin-lock forever!

```go
func acquireLockAndDoSomething() {
  for {
     if atomic.CompareAndSwapInt32(&state, 0, 1) {
        break
     }
     runtime.Gosched()
  }
  
  // do some thing
  // and you forget to reset state: 
  // atomic.CompareAndSwapInt32(&state, 1, 0)
}

```

---

### Defer

- Adding overhead to your runtime and stack
- `Nanosecond != zero` <!-- .element: class="fragment highlight-red" -->
- `Millions call == dozen millis` <!-- .element: class="fragment highlight-red" -->
- Someone made a benchmark: [bench](https://medium.com/i0exception/runtime-overhead-of-using-defer-in-go-7140d5c40e32)

=> Use defer with care! <!-- .element: class="fragment fade-up" -->

---

### `sync.Map`

- Fast for many cases, but not all
- Use it wisely !!
- Better trying other lock-free, thread-safe data structures

---

### Example of replacing `sync.Map`

- We create our own `int64` set based on: https://github.com/brentp/intintmap
- We `manually` shards the map and `share-nothing` approach

---

### `context.Context`

- Powerful
- You know when `Done()` and stop your job
- Binding value

---

### But be careful

```go
ctx, cancel := context.WithTimeout(parentCtx, time.Second)

DoSomething(ctx)

func DoSomething(ctx context.Context) {
    // your code here
    time.Sleep(2 * time.Second)
    myCtx, cancel := ctx.WithCancel(ctx)

    doOtherThing(myCtx) /* `myCtx` is cancelled
                            due to timeout of `ctx` */
}
```

---

### `context.Context`

- Understand what you are doing with `ctx`
- Control its scope

Then you have an awesome friend!<!-- .element: class="fragment fade-up" -->

---

### `Data race`

- Evil
- Causing unpredictable situation and data inconsistent

---

<img src="https://gyazo.linecorp.com/images/190228/f16c841b9120fef8c5b7258c01c9c910.png" style="height:600px" />

---

```bash
go run race.go
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x1098e6a]

goroutine 19 [running]:
main.main.func1(0x10e8da0, 0xc000098000, 0xc00008e010, 0xc000094010, 0x2)
	/Users/JP22782/race.go:34 +0x6a
created by main.main
	/Users/JP22782/race.go:26 +0xd9
exit status 2

real	0m0.260s
user	0m0.255s
sys	0m0.168s
```

---

#### Only test help you

- Run test with "-race"<!-- .element: class="fragment fade-up" -->
- Stress test with concurrent read-write<!-- .element: class="fragment fade-up" -->
- Don't enable "-race" on production<!-- .element: class="fragment fade-up" -->

---

# .... many others

---

### Show case

<img src="https://gyazo.linecorp.com/images/190228/41b1b8a665a37f162b129810d8b958fa.png" />

---

# Join us!!
