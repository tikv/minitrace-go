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
    bl := newBufferList()
    s := bl.slot()
    s.Id = 1
    s.Parent = 0
    s.BeginNs = monotimeNs()
    s.Event = event

    spanCtx := spanContext{
        parent:         ctx,
        tracingContext: tCtx,
        tracedSpans: &localSpans{
            spans:        bl,
            createTimeNs: realtimeNs(),
            refCount:     1,
        },
        currentId:  1,
        currentGid: gid.Get(),
    }

    return spanCtx, TraceHandle{SpanHandle{spanCtx, &s.EndNs}}
}

func NewSpanWithContext(ctx context.Context, event uint32) (context.Context, SpanHandle) {
    handle := NewSpan(ctx, event)
    if handle.endNs != nil {
        return handle.spanContext, handle
    }
    return ctx, handle
}

func NewSpan(ctx context.Context, event uint32) (res SpanHandle) {
    if s, ok := ctx.(spanContext); ok {
        res.spanContext = s
    } else if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
        res.spanContext.parent = ctx
        res.spanContext.tracingContext = s.tracingContext
        res.spanContext.tracedSpans = s.tracedSpans
        res.spanContext.currentGid = s.currentGid
        res.spanContext.currentId = s.currentId
    } else {
        return
    }

    id := atomic.AddUint32(&res.spanContext.tracingContext.maxId, 1)
    goid := gid.Get()

    if goid == res.spanContext.currentGid {
        slot := res.spanContext.tracedSpans.spans.slot()
        slot.Id = id
        slot.Parent = res.spanContext.currentId
        slot.BeginNs = monotimeNs()
        slot.Event = event

        res.spanContext.tracedSpans.refCount += 1
        res.spanContext.currentId = id
        res.spanContext.currentGid = goid
        res.endNs = &slot.EndNs
        return
    } else {
        bl := newBufferList()
        slot := bl.slot()
        slot.Id = id
        slot.Parent = res.spanContext.currentId
        slot.BeginNs = monotimeNs()
        slot.Event = event

        res.spanContext.tracedSpans = &localSpans{
            spans:        bl,
            createTimeNs: realtimeNs(),
            refCount:     1,
        }
        res.spanContext.currentId = id
        res.spanContext.currentGid = goid
        res.endNs = &slot.EndNs
        return
    }
}

type SpanHandle struct {
    spanContext spanContext
    endNs       *uint64
}

func (hd *SpanHandle) Finish() {
    if hd.endNs != nil {
        *hd.endNs = monotimeNs()
        hd.spanContext.tracedSpans.refCount -= 1
        if hd.spanContext.tracedSpans.refCount == 0 {
            hd.spanContext.tracingContext.mu.Lock()
            hd.spanContext.tracingContext.collectedSpans = append(hd.spanContext.tracingContext.collectedSpans, SpanSet{
                StartTimeNs: hd.spanContext.tracedSpans.createTimeNs,
                Spans:       hd.spanContext.tracedSpans.spans.collect(),
            })
            hd.spanContext.tracingContext.mu.Unlock()
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
