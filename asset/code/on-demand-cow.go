/*
 * Copyright (c) 2019 LINE Corporation. All rights reserved.
 * LINE Corporation PROPRIETARY/CONFIDENTIAL. Use is subject to license terms.
 */

package model

import (
	"runtime"
	"sync/atomic"
)

// read lock
func (c *Chunk) rLock() {
	var readSem int32
	for {
		readSem = atomic.LoadInt32(&c.readSem)
		if readSem >= 0 && atomic.CompareAndSwapInt32(&c.readSem, readSem, readSem+1) {
			return
		}

		// yields the processor, allowing other goroutines to run. It does not
		// suspend the current goroutine, so execution resumes automatically.
		runtime.Gosched()
	}
}

// read unlock
func (c *Chunk) rUnlock() {
	atomic.AddInt32(&c.readSem, -1)
}

// write lock if possible
func (c *Chunk) lock() (locked bool, w *chunkencWrapper) {
	locked = atomic.CompareAndSwapInt32(&c.readSem, 0, -1)
	if locked {
		w = c.w.Load().(*chunkencWrapper)
	} else {
		w = c.w.Load().(*chunkencWrapper).copy()
	}
	return
}

// write unlock
func (c *Chunk) unlock() {
	atomic.CompareAndSwapInt32(&c.readSem, -1, 0)
}
