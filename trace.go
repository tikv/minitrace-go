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
        // We'd like to modify the context on "stack" directly to eliminate heap memory allocation
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

    id := atomic.AddUint64(&res.spanContext.tracingContext.maxId, 1)
    goid := gid.Get()

    // Use per goroutine buffer to reduce synchronization overhead.
    if goid != res.spanContext.currentGid {
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
    } else {
        slot := res.spanContext.tracedSpans.spans.slot()
        slot.Id = id
        slot.Parent = res.spanContext.currentId
        slot.BeginNs = monotimeNs()
        slot.Event = event

        res.spanContext.tracedSpans.refCount += 1
        res.spanContext.currentId = id
        res.spanContext.currentGid = goid
        res.endNs = &slot.EndNs
    }

    return
}

func CurrentSpanId(ctx context.Context) (id uint64, ok bool) {
    if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
        return s.currentId, true
    }

    return
}

type SpanHandle struct {
    spanContext spanContext
    endNs       *uint64
}

// TODO: Prevent users from calling twice
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

func (hd TraceHandle) Collect() (res []SpanSet) {
    hd.SpanHandle.Finish()

    hd.spanContext.tracingContext.mu.Lock()
    res = hd.spanContext.tracingContext.collectedSpans
    hd.spanContext.tracingContext.collectedSpans = nil
    hd.spanContext.tracingContext.mu.Unlock()

    return
}
