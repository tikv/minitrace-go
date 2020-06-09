package minitrace

import (
    "context"
    "github.com/silentred/gid"
    "sync/atomic"
)

func TraceEnable(ctx context.Context, event uint32) (context.Context, TraceHandle) {
    tCtx := &tracingContext{
        maxId: 1,
    }
    spanCtx := spanContext{
        parent:         ctx,
        tracingContext: tCtx,
        tracedSpans: &localSpans{
            spans: []Span{{
                Id:      1,
                Parent:  0,
                BeginNs: monotimeNs(),
                EndNs:   0,
                Event:   event,
            }},
            createTimeNs: realtimeNs(),
            refCount:     1,
        },
        currentId:  1,
        currentGid: gid.Get(),
    }

    return spanCtx, TraceHandle{SpanHandle{spanCtx, 0}}
}

func NewSpanWithContext(ctx context.Context, event uint32) (context.Context, SpanHandle) {
    handle := NewSpan(ctx, event)
    if handle.index >= 0 {
        return handle.spanContext, handle
    }
    return ctx, handle
}

func NewSpan(ctx context.Context, event uint32) SpanHandle {
    var spanCtx spanContext
    if s, ok := ctx.(spanContext); ok {
        spanCtx = s
    } else if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
        spanCtx = s
        spanCtx.parent = ctx
    } else {
        return SpanHandle{index: -1}
    }

    id := atomic.AddUint32(&spanCtx.tracingContext.maxId, 1)
    goid := gid.Get()
    span := Span{
        Id:      id,
        Parent:  spanCtx.currentId,
        BeginNs: monotimeNs(),
        EndNs:   0,
        Event:   event,
    }

    if goid == spanCtx.currentGid {
        index := len(spanCtx.tracedSpans.spans)
        spanCtx.tracedSpans.refCount += 1
        spanCtx.tracedSpans.spans = append(spanCtx.tracedSpans.spans, span)
        spanCtx.currentId = id
        spanCtx.currentGid = goid
        return SpanHandle{spanCtx, index}
    } else {
        tracedSpans := &localSpans{
            spans:        []Span{span},
            createTimeNs: realtimeNs(),
            refCount:     1,
        }
        spanCtx.tracedSpans = tracedSpans
        spanCtx.currentId = id
        spanCtx.currentGid = goid
        return SpanHandle{spanCtx, 0}
    }
}

type SpanHandle struct {
    spanContext spanContext
    index       int
}

func (hd SpanHandle) Finish() {
    if hd.index >= 0 {
        spanCtx := hd.spanContext
        spanCtx.tracedSpans.spans[hd.index].EndNs = monotimeNs()
        spanCtx.tracedSpans.refCount -= 1
        if spanCtx.tracedSpans.refCount == 0 {
            spanCtx.tracingContext.mu.Lock()
            spanCtx.tracingContext.collectedSpans = append(spanCtx.tracingContext.collectedSpans, SpanSet{
                StartTimeNs: spanCtx.tracedSpans.createTimeNs,
                Spans:       spanCtx.tracedSpans.spans,
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
