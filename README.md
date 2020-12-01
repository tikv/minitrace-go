# Minitrace-Go
[![Actions Status](https://github.com/tikv/minitrace-go/workflows/CI/badge.svg)](https://github.com/tikv/minitrace-go/actions)
[![LICENSE](https://img.shields.io/github/license/tikv/minitrace-go.svg)](https://github.com/tikv/minitrace-go/blob/master/LICENSE)

A high-performance, ergonomic timeline tracing library for Golang. 

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "strconv"

    "github.com/tikv/minitrace-go"
)

func tracedFunc(ctx context.Context, event string) {
    span := minitrace.StartSpan(ctx, event)
    // code snippet...
    span.Finish()
}

func iterTracedFunc(ctx context.Context) {
    // extend tracing context from parent context
    ctx, span := minitrace.StartSpanWithContext(ctx, "1")

    span.AddProperty("k2", "v2")

    for i := 2; i < 10; i++ {
        tracedFunc(ctx, strconv.Itoa(i))
    }
    
    span.Finish()
}

func main() {
    ctx := context.Background()

    // enable tracing
    ctx, root := minitrace.StartRootSpan(ctx, "root", 0, nil)

    root.AddProperty("k1", "v1")

    // pass the context to traced functions
    iterTracedFunc(ctx)

    // collect tracing results into `spans`
    spans, _ := root.Collect()

    // do something with `spans`
    fmt.Printf("%+v", spans)
}
```