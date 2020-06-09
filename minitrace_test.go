package minitrace

import (
    "context"
    "fmt"
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
