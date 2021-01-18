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
	"strconv"
	"sync"
	"testing"

	"github.com/opentracing/opentracing-go"
	"sourcegraph.com/sourcegraph/appdash"
	traceImpl "sourcegraph.com/sourcegraph/appdash/opentracing"
)

func BenchmarkMiniTrace(b *testing.B) {
	for i := 10; i < 100001; i *= 10 {
		b.Run(fmt.Sprintf("   %d", i), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				ctx, handle := StartRootSpan(context.Background(), "root", 10086, nil)

				for k := 1; k < i; k++ {
					_, handle := StartSpanWithContext(ctx, strconv.Itoa(k))
					handle.Finish()
				}

				spans, _ := handle.Collect()
				if i != len(spans) {
					b.Fatalf("expected length %d, got %d", i, len(spans))
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
	var traceID uint64 = 9527
	ctx, handle := StartRootSpan(context.Background(), "root", traceID, nil)
	var wg sync.WaitGroup

	if spanID1, traceID1, ok := CurrentSpanID(ctx); ok {
		spanID := handle.spanContext.span.ID
		if spanID != spanID1 {
			t.Fatalf("unmatched span ID: expected %d got %d", spanID, spanID1)
		}
		if traceID != traceID1 {
			t.Fatalf("unmatched trace ID: expected %d got %d", traceID, traceID1)
		}
	} else {
		t.Fatalf("cannot get current span ID")
	}

	for i := 1; i < 5; i++ {
		ctx, handle := StartSpanWithContext(ctx, strconv.Itoa(i))

		if spanID1, traceID1, ok := CurrentSpanID(ctx); ok {
			spanID := handle.spanContext.span.ID
			if spanID != spanID1 {
				t.Fatalf("unmatched span ID: expected %d got %d", spanID, spanID1)
			}
			if traceID != traceID1 {
				t.Fatalf("unmatched trace ID: expected %d got %d", traceID, traceID1)
			}
		} else {
			t.Fatalf("cannot get current span ID")
		}

		wg.Add(1)
		go func(prefix int) {
			ctx, handle := StartSpanWithContext(ctx, strconv.Itoa(prefix))
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(prefix int) {
					handle := StartSpan(ctx, strconv.Itoa(prefix))
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
	spans, _ := handle.Collect()
	if len(spans) != 29 {
		t.Fatalf("length of spanSets expected %d, but got %d", 25, len(spans))
	}
}
