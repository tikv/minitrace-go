// Copyright 2020 PingCAP, Inc.
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

package minitrace

import (
    "context"
    "fmt"
    "github.com/opentracing/opentracing-go"
    "sourcegraph.com/sourcegraph/appdash"
    traceImpl "sourcegraph.com/sourcegraph/appdash/opentracing"
    "sync"
    "testing"
)

func BenchmarkMiniTrace(b *testing.B) {
    for i := 10; i < 100001; i *= 10 {
        b.Run(fmt.Sprintf("   %d", i), func(b *testing.B) {
            for j := 0; j < b.N; j++ {
                ctx, handle := TraceEnable(context.Background(), 0)

                for k := 1; k < i; k++ {
                    _, handle := NewSpanWithContext(ctx, uint32(k))
                    handle.Finish()
                }

                spanSets := handle.Collect()
                if i != len(spanSets[0].Spans) {
                    b.Fatalf("expected length %d, got %d", i, len(spanSets[0].Spans))
                }
            }
        })
    }
}

func BenchmarkAppdashTrace(b *testing.B) {
    for i := 10; i < 10_001; i *= 10 {
        b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
            for j := 0; j < b.N; j++ {
                store := appdash.NewMemoryStore()
                tracer := traceImpl.NewTracer(store)
                span, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "trace")

                for k := 1; k < i; k++ {
                    if span := opentracing.SpanFromContext(ctx); span != nil && span.Tracer() != nil {
                        span, _ := opentracing.StartSpanFromContextWithTracer(ctx, span.Tracer(), "child", opentracing.ChildOf(span.Context()))
                        span.Finish()
                    }
                }

                span.Finish()

                traces, err := store.Traces(appdash.TracesOpts{})
                if err != nil {
                    b.Fatal(err)
                }

                if i != len(traces[0].Sub)+1 {
                    b.Fatalf("expected length %d, got %d", i, len(traces[0].Sub)+1)
                }
            }
        })
    }
}

func TestMiniTrace(t *testing.T) {
    ctx, handle := TraceEnable(context.Background(), 0)
    var wg sync.WaitGroup

    for i := 1; i < 5; i++ {
        ctx, handle := NewSpanWithContext(ctx, uint32(i))
        wg.Add(1)
        go func(prefix int) {
            ctx, handle := NewSpanWithContext(ctx, uint32(prefix))
            for i := 0; i < 5; i++ {
                wg.Add(1)
                go func(prefix int) {
                    handle := NewSpan(ctx, uint32(prefix))
                    handle.Finish()
                    wg.Done()
                }((prefix + i) * 10)
            }
            handle.Finish()
            wg.Done()
        }(i * 10)
        handle.Finish()
    }

    wg.Wait()
    spanSets := handle.Collect()
    if len(spanSets) != 25 {
        t.Fatalf("length of spanSets expected %d, but got %d", 25, len(spanSets))
    }

    sum := 0
    for _, spanSet := range spanSets {
        sum += len(spanSet.Spans)
    }
    if sum != 29 {
        t.Fatalf("count of spans expected %d, but got %d", 29, sum)
    }
}
