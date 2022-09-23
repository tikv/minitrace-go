// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.
//go:build windows || (linux && amd64)
// +build windows linux,amd64

package minitrace

import (
	_ "unsafe"
)

//go:linkname nanotime runtime.nanotime
func nanotime() int64

//go:linkname time_now time.now
func time_now() (sec int64, nsec int32, mono int64)

// Standard library's `time.Now()` will invoke two syscalls in Linux. One is `CLOCK_REALTIME` and
// another is `CLOCK_MONOTONIC`. In our case, we'd like to separate these two calls to measure
// time for performance purpose.
// `nanotime()` is identical to Linux's `clock_gettime(CLOCK_MONOTONIC, &ts)`
func monotimeNs() uint64 {
	return uint64(nanotime())
}

// Standard library's `time.Now()` will invoke two syscalls in Linux. One is `CLOCK_REALTIME` and
// another is `CLOCK_MONOTONIC`. In our case, we'd like to separate these two calls to measure
// time for performance purpose.
// `unixtimeNs()` is identical to Linux's `clock_gettime(CLOCK_REALTIME, &ts)`
func unixtimeNs() uint64 {
	sec, nsec, _ := time_now()
	return uint64(sec*1_000_000_000 + int64(nsec))
}
