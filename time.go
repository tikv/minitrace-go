package minitrace

import (
    _ "unsafe"
)

//go:linkname nanotime runtime.nanotime
func nanotime() int64

//go:linkname walltime runtime.walltime
func walltime() (sec int64, nsec int32)

func monotimeNs() uint64 {
    return uint64(nanotime())
}

func realtimeNs() uint64 {
    sec, nsec := walltime()
    return uint64(sec*1_000_000_000 + int64(nsec))
}
