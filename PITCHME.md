### @color[#607625](Efficient concurrency in Go)

Tran Tuan Linh @ LINE Corp

linxGnu @ github

---

## @color[#607625](Story)

- Writing high intensive read/write metrics database
- 40-80 Billion data points / day (and more)

---

###  @color[#607625](Decision)

In-memory, In-Go

---

###  @color[#607625](Why Go)

- Simple
- Fast
- Productivity
- Easy to write concurrency
- Awesome third-party libraries
- Very nice **pprof** tooling

<em>And, WE love Go!</em><!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](Processing approach)

- Share nothing on write
- On-demand Copy-on-write

---

###  @color[#607625](Share nothing on write)

- No write contention <!-- .element: class="fragment fade-up" -->
- No lock/mutex, less context switching <!-- .element: class="fragment fade-up" -->
- Millions data points / sec (or more) <!-- .element: class="fragment fade-up" -->
 
<em>"Less lock, more speed"</em> <!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](Copy-on-write)

- We still have  @color[red](Query) (many requests at a time)
- @color[red](COW) is a good way to deal with @color[red](consistency)
- Non-blocking
- No data race, thread-safe

<em>COW + Share-nothing == Awesome</em><!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](Single write - multiple read)

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

###  @color[#607625](COW - Cons)

- Memory allocation - GC stress
- @color[red](Copy) is not @color[red](zero-cost)
- When @color[red](Write >> Read) workload ?

=> Let's improve it! <!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](On-demand COW)

- Special `sync.RWMutex`
- Lockless
- Lightweight
- x3-x10 performance gain!

---

<img src="https://gyazo.linecorp.com/images/190226/a50019ba32e8f2cccb2f9182989cdffd.png" style="height:600px"/>

---

##  @color[#607625](Story - So far)

- Go help us writing systems in very productivity
- Just code -> `go run` -> `sleep well`
- Go is fast and safe `if there is no bug in your code` <!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](Things not in heaven)
#### Story of optimization

---

###  @color[#607625](Channel)

- Awesome
- But slow when pushing @color[red](millions) messages through

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

###  @color[#607625](Make it better)

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

##  @color[#607625](6-7x faster!)
### But why?

---

###  @color[#607625](pprof is awesome!)

<img src="https://gyazo.linecorp.com/images/190228/6c050cfb005128d74d42e2e723d161c5.png" style="height:550px" />

---

###  @color[#607625](sync/atomic)

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

###  @color[#607625](sync/atomic)

Use it carefully or you will be spin-lock forever!

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

###  @color[#607625](Defer)

- Adding overhead to your runtime and stack
- `Nanosecond != zero` <!-- .element: class="fragment highlight-red" -->
- `Millions call == dozen millis` <!-- .element: class="fragment highlight-red" -->
- Someone made a benchmark: [bench](https://medium.com/i0exception/runtime-overhead-of-using-defer-in-go-7140d5c40e32)

=> Use defer with care! <!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](sync.Map)

- Fast for many cases, but not all
- Better trying other lock-free, thread-safe data structures

---

###  @color[#607625](Replacing)

- We create our own `int64` set based on: https://github.com/brentp/intintmap
- We @color[red](manually) shards the map and apply @color[red](share-nothing) approach
- 2-5x faster than `map`
- Tiny alloc
- We also see great improvement when using `int64 set + manual shard` compared to `sync.Map`

---

### @color[#607625](Benchmark)

```go
func benchmarkMapSet(b *testing.B, set []int64) {
	for i := 0; i < b.N; i++ {
		s := make(map[int64]struct{})

		var exist bool
		for _, v := range set {
			if _, exist = s[v]; !exist {
				s[v] = struct{}{}
			}
		}
	}
}

func benchmarkIntSet(b *testing.B, set []int64) {
	for i := 0; i < b.N; i++ {
		s, _ := iset.New(128, 0.8)
		for _, v := range set {
			if !s.Exist(v) {
				s.Put(v)
			}
		}
	}
}
```

---

### @color[#607625](Benchmark)

```elm
go1.12
goos: darwin
goarch: amd64
pkg: git.linecorp.com/LINE-DevOps/imon-flash.git/benchmarks/go/intset
BenchmarkIntsetSmall-8    	   50000	     22666 ns/op	   65616 B/op	       4 allocs/op
BenchmarkIntsetMedium-8   	       5	 204731605 ns/op	201293923 B/op	      26 allocs/op
BenchmarkIntsetLarge-8    	       1	1241928873 ns/op	805273680 B/op	      30 allocs/op
BenchmarkMapsetSmall-8    	   20000	    103981 ns/op	   47813 B/op	      66 allocs/op
BenchmarkMapsetMedium-8   	       3	 411189933 ns/op	99879104 B/op	   76652 allocs/op
BenchmarkMapsetLarge-8    	       1	2220622562 ns/op	403991096 B/op	  306845 allocs/op
```

---

###  @color[#607625](context.Context)

- Powerful
- You know when `Done()` and stop your job
- Binding value

---

###  @color[#607625](But be careful)

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

###  @color[#607625](context.Context)

- Understand what you are doing with @color[red](ctx)
- Control its scope

Then feel the great!<!-- .element: class="fragment fade-up" -->

---

###  @color[#607625](Data race)

- Evil
- Causing unpredictable situation and data inconsistent

---

<img src="https://gyazo.linecorp.com/images/190228/f16c841b9120fef8c5b7258c01c9c910.png" style="height:600px" />

---

```elm
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

####  @color[#607625](Only test help you)

- Run test with @color[red](-race)<!-- .element: class="fragment fade-up" -->
- Stress test with concurrent read-write<!-- .element: class="fragment fade-up" -->
- Avoid enabling @color[red](-race) on production<!-- .element: class="fragment fade-up" -->

---

# .... many others

---

###  @color[#607625](Show case)

<img src="https://gyazo.linecorp.com/images/190228/41b1b8a665a37f162b129810d8b958fa.png" />

---

# Join us!!
