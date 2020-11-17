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

// The id 0 have a special semantic. So id should begin from 1.
var idGen uint32 = 1

// Returns incremental uint32 unique ID.
func nextId() uint32 {
	return atomic.AddUint32(&idGen, 1)
}

func StartRootSpan(ctx context.Context, event string, traceId uint64) (context.Context, TraceHandle) {
	tCtx := &tracingContext{
		traceId: traceId,
	}

	monoNow := monotimeNs()
	epochNow := realtimeNs()

	// Init a buffer list.
	bl := newBufferList()
	// Get a span slot,
	s := bl.slot()

	// Fill fields of the span slot
	s.Id = nextId()
	// Root span doesn't have a parent. Its parent span id is set to 0.
	s.Parent = 0
	// Fill a monotonic time for now. After span is finished, it will replace by an epoch time.
	s.BeginEpochNs = monoNow
	s.Event = event

	spanCtx := spanContext{
		parent:         ctx,
		tracingContext: tCtx,
		tracedSpans: &localSpans{
			spans:    bl,
			refCount: 1,
		},
		currentSpanId: s.Id,
		currentGid:    gid.Get(),

		createEpochTimeNs: epochNow,
		createMonoTimeNs:  monoNow,
	}

	return spanCtx, TraceHandle{SpanHandle{
		spanCtx,
		&s.BeginEpochNs,
		&s.DurationNs,
		&s.Properties,
		false,
	}}
}

func StartSpanWithContext(ctx context.Context, event string) (context.Context, SpanHandle) {
	handle := StartSpan(ctx, event)
	if handle.durationNs != nil {
		return handle.spanContext, handle
	}
	return ctx, handle
}

func StartSpan(ctx context.Context, event string) (res SpanHandle) {
	if s, ok := ctx.(spanContext); ok {
		// We'd like to modify the context on "stack" directly to eliminate heap memory allocation
		res.spanContext = s
	} else if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
		res.spanContext.parent = ctx
		res.spanContext.tracingContext = s.tracingContext
		res.spanContext.tracedSpans = s.tracedSpans
		res.spanContext.currentGid = s.currentGid
		res.spanContext.currentSpanId = s.currentSpanId
		res.spanContext.createMonoTimeNs = s.createMonoTimeNs
		res.spanContext.createEpochTimeNs = s.createEpochTimeNs
	} else {
		return
	}

	id := nextId()
	goid := gid.Get()
	var slot *Span

	if goid != res.spanContext.currentGid || res.spanContext.tracedSpans.spans.collected {
		// Use "goroutine-local" collection to reduce synchronization overhead.
		// If a previous processing has collected spans, we should allocate a new collection.

		bl := newBufferList()
		slot = bl.slot()

		// Init a new collection. Reference count begin from 1.
		res.spanContext.tracedSpans = &localSpans{
			spans:    bl,
			refCount: 1,
		}
	} else {
		// Fetch a slot from the local collection
		slot = res.spanContext.tracedSpans.spans.slot()

		// Collection is shared to a new span now so the reference count need to update.
		res.spanContext.tracedSpans.refCount += 1
	}

	slot.Id = id
	slot.Parent = res.spanContext.currentSpanId
	// Fill a monotonic time for now. After the span is finished, and it will be replaced by an epoch time.
	slot.BeginEpochNs = monotimeNs()
	slot.Event = event

	res.spanContext.currentSpanId = id
	res.spanContext.currentGid = goid
	res.beginEpochNs = &slot.BeginEpochNs
	res.durationNs = &slot.DurationNs
	res.properties = &slot.Properties

	return
}

func CurrentSpanId(ctx context.Context) (spanId uint32, traceId uint64, ok bool) {
	if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
		return s.currentSpanId, s.tracingContext.traceId, ok
	}

	return
}

type SpanHandle struct {
	spanContext  spanContext
	beginEpochNs *uint64
	durationNs   *uint64
	properties   *[]Property
	finished     bool
}

func (hd *SpanHandle) AddProperty(key, value string) {
	*hd.properties = append(*hd.properties, Property{Key: key, Value: value})
}

func (hd *SpanHandle) Finish() {
	if hd.finished {
		return
	} else {
		hd.finished = true
	}

	if hd.durationNs != nil {
		// For now, `beginEpochNs` is a monotonic time. Here to correct its value to satisfy the semantic.
		*hd.durationNs = monotimeNs() - *hd.beginEpochNs
		*hd.beginEpochNs = (*hd.beginEpochNs - hd.spanContext.createMonoTimeNs) + hd.spanContext.createEpochTimeNs

		hd.spanContext.tracedSpans.refCount -= 1
		if hd.spanContext.tracedSpans.refCount == 0 {
			hd.spanContext.tracingContext.mu.Lock()
			hd.spanContext.tracingContext.collectedSpans = append(hd.spanContext.tracingContext.collectedSpans, hd.spanContext.tracedSpans.spans.collect()...)
			hd.spanContext.tracingContext.mu.Unlock()
		}
	}
}

func (hd *SpanHandle) TraceId() uint64 {
	return hd.spanContext.tracingContext.traceId
}

type TraceHandle struct {
	SpanHandle
}

func (hd *TraceHandle) Collect() (res []Span) {
	hd.SpanHandle.Finish()

	hd.spanContext.tracingContext.mu.Lock()
	res = hd.spanContext.tracingContext.collectedSpans
	hd.spanContext.tracingContext.collectedSpans = nil
	hd.spanContext.tracingContext.mu.Unlock()

	return
}
