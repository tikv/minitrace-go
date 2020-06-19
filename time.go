// Copyright 2020 TiKV Project Authors. Licensed under Apache-2.0.

package minitrace

import (
    _ "unsafe"
)

//go:linkname nanotime runtime.nanotime
func nanotime() int64

//go:linkname walltime runtime.walltime
func walltime() (sec int64, nsec int32)

// Standard library's `time.Now()` will invoke two syscalls in Linux, one for `CLOCK_REALTIME`,
// another for `CLOCK_MONOTONIC`. In our case, we'd like to separate these two calls to measure
// time for performance purpose.
// `nanotime()` is identical to Linux's `clock_gettime(CLOCK_MONOTONIC, &ts)`
func monotimeNs() uint64 {
    return uint64(nanotime())
}

// Standard library's `time.Now()` will invoke two syscalls in Linux, one for `CLOCK_REALTIME`,
// another for `CLOCK_MONOTONIC`. In our case, we'd like to separate these two calls to measure
// time for performance purpose.
// `nanotime()` is identical to Linux's `clock_gettime(CLOCK_REALTIME, &ts)`
func realtimeNs() uint64 {
    sec, nsec := walltime()
    return uint64(sec*1_000_000_000 + int64(nsec))
}
