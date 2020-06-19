# Minitrace-go
[![Actions Status](https://github.com/pingcap-incubator/minitrace-go/workflows/CI/badge.svg)](https://github.com/pingcap-incubator/minitrace-go/actions)
[![LICENSE](https://img.shields.io/github/license/pingcap-incubator/minitrace-go.svg)](https://github.com/pingcap-incubator/minitrace-go/blob/master/LICENSE)

A high-performance, ergonomic timeline tracing library for Golang. 

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/pingcap-incubator/minitrace-go"
)

func tracedFunc(ctx context.Context, event uint32) {
    span := minitrace.NewSpan(ctx, event)
    // code snippet...
    span.Finish()
}

func iterTracedFunc(ctx context.Context) {
    // extend tracing context from parent context
    ctx, span := minitrace.NewSpanWithContext(ctx, 1)

    for i := 2; i < 10; i++ {
        tracedFunc(ctx, uint32(i))
    }
    
    span.Finish()
}

func main() {
    ctx := context.Background()

    // enable tracing
    ctx, root := minitrace.TraceEnable(ctx, 0)

    // pass the context to traced functions
    iterTracedFunc(ctx)

    // collect tracing results into `spanSets`
    spanSets := root.Collect()

    // do something with `spanSets`
    fmt.Printf("%+v", spanSets)
}
```