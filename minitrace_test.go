package minitrace

import (
	"context"
	"testing"
)

func benchmarkMiniTrace(l int, b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracedFunc(l, b)
	}
}

func BenchmarkMiniTrace10(b *testing.B)    { benchmarkMiniTrace(10, b) }
func BenchmarkMiniTrace100(b *testing.B)   { benchmarkMiniTrace(100, b) }
func BenchmarkMiniTrace1000(b *testing.B)  { benchmarkMiniTrace(1000, b) }
func BenchmarkMiniTrace10000(b *testing.B) { benchmarkMiniTrace(10000, b) }

func tracedFunc(l int, b *testing.B) {
	ctx, handle := TraceEnable(context.Background(), 0)

	for i := 1; i < l; i++ {
		_, handle := NewSpan(ctx, uint32(i))
		handle.Finish()
	}

	spanSets := handle.Finish()
	if l != len(spanSets[0].Spans) {
		b.Fatalf("expected length %d, got %d", l, len(spanSets[0].Spans))
	}
}
