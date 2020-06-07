package minitrace

import (
	"context"
	"github.com/silentred/gid"
	"sync/atomic"
)

type tracingKey struct{}

var key = tracingKey{}

func TraceEnable(ctx context.Context, event uint32) (context.Context, TraceHandle) {
	tCtx := &tracingContext{
		maxId: 0,
	}
	spanCtx := &spanContext{
		tracingContext: tCtx,
		tracedSpans: &localSpans{
			spans: []Span{{
				Id:          0,
				Link:        Root{},
				BeginCycles: monotimeNs(),
				EndCycles:   0,
				Event:       event,
			}},
			createTimeNs: realtimeNs(),
			refCount:     1,
		},
		currentId:  0,
		currentGid: gid.Get(),
	}
	nCtx := context.WithValue(ctx, key, spanCtx)

	return nCtx, TraceHandle{SpanHandle{spanCtx, 0}}
}

func NewSpan(ctx context.Context, event uint32) (context.Context, SpanHandle) {
	switch v := ctx.Value(key).(type) {
	case *spanContext:
		id := atomic.AddUint64(&v.tracingContext.maxId, 1)
		goid := gid.Get()
		span := Span{
			Id:          id,
			Link:        Parent{v.currentId},
			BeginCycles: monotimeNs(),
			EndCycles:   0,
			Event:       event,
		}

		if goid == v.currentGid {
			index := len(v.tracedSpans.spans)
			v.tracedSpans.refCount += 1
			v.tracedSpans.spans = append(v.tracedSpans.spans, span)
			spanCtx := &spanContext{
				tracingContext: v.tracingContext,
				tracedSpans:    v.tracedSpans,
				currentId:      id,
				currentGid:     goid,
			}
			return context.WithValue(ctx, key, spanCtx), SpanHandle{spanCtx, index}
		} else {
			tracedSpans := &localSpans{
				spans:        []Span{span},
				createTimeNs: realtimeNs(),
				refCount:     1,
			}
			spanCtx := &spanContext{
				tracingContext: v.tracingContext,
				tracedSpans:    tracedSpans,
				currentId:      id,
				currentGid:     goid,
			}
			return context.WithValue(ctx, key, spanCtx), SpanHandle{spanCtx, 0}
		}
	}

	return ctx, SpanHandle{}
}

type SpanHandle struct {
	spanContext *spanContext
	index       int
}

func (hd SpanHandle) Finish() {
	if hd.spanContext != nil {
		spanCtx := hd.spanContext
		spanCtx.tracedSpans.spans[hd.index].EndCycles = monotimeNs()
		spanCtx.tracedSpans.refCount -= 1
		if spanCtx.tracedSpans.refCount == 0 {
			spanCtx.tracingContext.mu.Lock()
			spanCtx.tracingContext.collectedSpans = append(spanCtx.tracingContext.collectedSpans, SpanSet{
				CreateTimeNs: spanCtx.tracedSpans.createTimeNs,
				StartTimeNs:  spanCtx.tracedSpans.createTimeNs,
				CyclesPerSec: 1_000_000_000, // nanoseconds per second
				Spans:        spanCtx.tracedSpans.spans,
			})
			spanCtx.tracingContext.mu.Unlock()
		}
	}
}

type TraceHandle struct {
	SpanHandle
}

func (hd TraceHandle) Finish() (res []SpanSet) {
	hd.SpanHandle.Finish()

	hd.spanContext.tracingContext.mu.Lock()
	res = hd.spanContext.tracingContext.collectedSpans
	hd.spanContext.tracingContext.collectedSpans = nil
	hd.spanContext.tracingContext.mu.Unlock()

	return
}
