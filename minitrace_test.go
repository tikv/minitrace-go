package minitrace

import (
    "context"
    "fmt"
    "sync"
    "testing"
)

func BenchmarkMiniTrace(b *testing.B) {
    for i := 10; i < 100_001; i *= 10 {
        b.Run(fmt.Sprintf("MiniTrace%d", i), func(b *testing.B) {
            for j := 0; j < b.N; j++ {
                tracedFunc(i, b)
            }
        })
    }
}

func tracedFunc(l int, b *testing.B) {
    ctx, handle := TraceEnable(context.Background(), 0)

    for i := 1; i < l; i++ {
        _, handle := NewSpanWithContext(ctx, uint32(i))
        handle.Finish()
    }

    spanSets := handle.Finish()
    if l != len(spanSets[0].Spans) {
        b.Fatalf("expected length %d, got %d", l, len(spanSets[0].Spans))
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
    spanSets := handle.Finish()
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
