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
	"sync/atomic"

	"github.com/silentred/gid"
)

// The id 0 have a special semantic. So id should begin from 1.
var idGen uint32 = 1

// Returns incremental uint32 unique ID.
func nextId() uint32 {
	return atomic.AddUint32(&idGen, 1)
}

func StartRootSpan(ctx context.Context, event string, traceId uint64, attachment interface{}) (context.Context, TraceHandle) {
	tCtx := &tracingContext{
		traceId:    traceId,
		attachment: attachment,
	}

	monoNow := monotimeNs()
	unixNow := unixtimeNs()

	// Init a buffer list.
	bl := newBufferList()
	// Get a span slot,
	s := bl.slot()

	// Fill fields of the span slot
	s.ID = nextId()
	// Root span doesn't have a parent. Its parent span id is set to 0.
	s.ParentID = 0
	// Fill a monotonic time for now. After span is finished, it will replace by a unix time.
	s.BeginUnixTimeNs = monoNow
	s.Event = event

	spanCtx := spanContext{
		parent:         ctx,
		tracingContext: tCtx,
		tracedSpans: &localSpans{
			spans:    bl,
			refCount: 1,
		},
		currentSpanId: s.ID,
		currentGid:    gid.Get(),

		createUnixTimeNs: unixNow,
		createMonoTimeNs: monoNow,
	}

	return spanCtx, TraceHandle{SpanHandle{
		spanCtx,
		&s.BeginUnixTimeNs,
		&s.DurationNs,
		&s.Properties,
		false,
	}}
}

func StartSpanWithContext(ctx context.Context, event string) (context.Context, SpanHandle) {
	handle := StartSpan(ctx, event)
	if !handle.finished {
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
		res.spanContext.createUnixTimeNs = s.createUnixTimeNs
	} else {
		res.finished = true
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

	slot.ID = id
	slot.ParentID = res.spanContext.currentSpanId
	// Fill a monotonic time for now. After the span is finished, and it will be replaced by a unix time.
	slot.BeginUnixTimeNs = monotimeNs()
	slot.Event = event

	res.spanContext.currentSpanId = id
	res.spanContext.currentGid = goid
	res.beginUnixTimeNs = &slot.BeginUnixTimeNs
	res.durationNs = &slot.DurationNs
	res.properties = &slot.Properties

	return
}

func CurrentSpanId(ctx context.Context) (spanId uint32, traceId uint64, ok bool) {
	if s, ok := ctx.(spanContext); ok {
		return s.currentSpanId, s.tracingContext.traceId, ok
	} else if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
		return s.currentSpanId, s.tracingContext.traceId, ok
	}

	return
}

func AccessAttachment(ctx context.Context, fn func(attachment interface{})) (ok bool) {
	var tracingCtx *tracingContext
	if s, ok := ctx.(spanContext); ok {
		tracingCtx = s.tracingContext
	} else if s, ok := ctx.Value(activeTracingKey).(spanContext); ok {
		tracingCtx = s.tracingContext
	} else {
		return false
	}

	tracingCtx.mu.Lock()
	if !tracingCtx.collected {
		fn(tracingCtx.attachment)
		ok = true
	} else {
		ok = false
	}
	tracingCtx.mu.Unlock()

	return
}

type SpanHandle struct {
	spanContext     spanContext
	beginUnixTimeNs *uint64
	durationNs      *uint64
	properties      *[]Property
	finished        bool
}

func (hd *SpanHandle) AddProperty(key, value string) {
	if hd.finished {
		return
	}

	*hd.properties = append(*hd.properties, Property{Key: key, Value: value})
}

func (hd *SpanHandle) AccessAttachment(fn func(attachment interface{})) {
	if hd.finished {
		return
	}

	hd.spanContext.tracingContext.mu.Lock()
	if !hd.spanContext.tracingContext.collected {
		fn(hd.spanContext.tracingContext.attachment)
	}
	hd.spanContext.tracingContext.mu.Unlock()
}

func (hd *SpanHandle) Finish() {
	if hd.finished {
		return
	}

	hd.finished = true

	// For now, `beginUnixTimeNs` is a monotonic time. Here to correct its value to satisfy the semantic.
	*hd.durationNs = monotimeNs() - *hd.beginUnixTimeNs
	*hd.beginUnixTimeNs = (*hd.beginUnixTimeNs - hd.spanContext.createMonoTimeNs) + hd.spanContext.createUnixTimeNs

	hd.spanContext.tracedSpans.refCount -= 1
	if hd.spanContext.tracedSpans.refCount == 0 {
		hd.spanContext.tracingContext.mu.Lock()
		if !hd.spanContext.tracingContext.collected {
			hd.spanContext.tracingContext.collectedSpans = append(hd.spanContext.tracingContext.collectedSpans, hd.spanContext.tracedSpans.spans.collect()...)
		}
		hd.spanContext.tracingContext.mu.Unlock()
	}
}

func (hd *SpanHandle) TraceId() uint64 {
	return hd.spanContext.tracingContext.traceId
}

type TraceHandle struct {
	SpanHandle
}

func (hd *TraceHandle) Collect() (spans []Span, attachment interface{}) {
	hd.SpanHandle.Finish()

	hd.spanContext.tracingContext.mu.Lock()

	if !hd.spanContext.tracingContext.collected {
		spans = hd.spanContext.tracingContext.collectedSpans
		attachment = hd.spanContext.tracingContext.attachment
	}
	hd.spanContext.tracingContext.collected = true

	hd.spanContext.tracingContext.mu.Unlock()

	return
}
