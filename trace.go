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
)

// The id 0 have a special semantic. So id should begin from 1.
var idGen uint32 = 1

// Returns incremental uint32 unique ID.
func nextID() uint32 {
	return atomic.AddUint32(&idGen, 1)
}

func StartRootSpan(ctx context.Context, event string, traceID uint64, attachment interface{}) (context.Context, TraceHandle) {
	traceCtx := newTraceContext(traceID, attachment)

	// Root span doesn't have a parent. Its parent span id is set to 0.
	const ParentID = 0

	traceCtx.collectedSpans = append(traceCtx.collectedSpans, Span{})
	span := &traceCtx.collectedSpans[len(traceCtx.collectedSpans)-1]
	span.beginWith(ParentID, event)

	spanCtx := newSpanContext(ctx, traceCtx, span.ID)
	spanHandle := newSpanHandle(spanCtx, len(traceCtx.collectedSpans)-1)

	return spanCtx, TraceHandle{spanHandle}
}

func StartSpanWithContext(ctx context.Context, event string) (context.Context, SpanHandle) {
	handle := StartSpan(ctx, event)
	if !handle.finished {
		return handle.spanContext, handle
	}
	return ctx, handle
}

func StartSpan(ctx context.Context, event string) (handle SpanHandle) {
	var parentCtx context.Context
	var parentSpanCtx *spanContext

	if s, ok := ctx.(*spanContext); ok {
		// Fold spanContext to reduce the depth of context tree.
		parentCtx = s.parent
		parentSpanCtx = s
	} else if s, ok := ctx.Value(activeTraceKey).(*spanContext); ok {
		parentCtx = ctx
		parentSpanCtx = s
	} else {
		handle.finished = true
		return
	}

	traceCtx := parentSpanCtx.traceContext
	traceCtx.mu.Lock()
	defer traceCtx.mu.Unlock()

	traceCtx.collectedSpans = append(traceCtx.collectedSpans, Span{})
	span := &traceCtx.collectedSpans[len(traceCtx.collectedSpans)-1]
	span.beginWith(parentSpanCtx.spanID, event)

	spanCtx := newSpanContext(parentCtx, traceCtx, span.ID)
	return newSpanHandle(spanCtx, len(traceCtx.collectedSpans)-1)
}

func CurrentSpanID(ctx context.Context) (spanID uint32, traceID uint64, ok bool) {
	if s, ok := ctx.Value(activeTraceKey).(*spanContext); ok {
		return s.spanID, s.traceContext.traceID, ok
	}

	return
}

func AccessAttachment(ctx context.Context, fn func(attachment interface{})) (ok bool) {
	spanCtx, ok := ctx.Value(activeTraceKey).(*spanContext)
	if !ok {
		return false
	}
	return spanCtx.traceContext.accessAttachment(fn)
}

type SpanHandle struct {
	spanContext *spanContext
	spanIndex   int
	finished    bool
}

func newSpanHandle(spanCtx *spanContext, index int) SpanHandle {
	return SpanHandle{
		spanContext: spanCtx,
		spanIndex:   index,
		finished:    false,
	}
}

func (sh *SpanHandle) AddProperty(key, value string) {
	if sh.finished {
		return
	}

	traceCtx := sh.spanContext.traceContext
	traceCtx.mu.Lock()
	defer traceCtx.mu.Unlock()

	sh.spanContext.traceContext.collectedSpans[sh.spanIndex].addProperty(key, value)
}

func (sh *SpanHandle) AccessAttachment(fn func(attachment interface{})) {
	if sh.finished {
		return
	}
	sh.spanContext.traceContext.accessAttachment(fn)
}

func (sh *SpanHandle) Finish() {
	if sh.finished {
		return
	}
	sh.finished = true

	traceCtx := sh.spanContext.traceContext
	traceCtx.mu.Lock()
	defer traceCtx.mu.Unlock()

	sh.spanContext.traceContext.collectedSpans[sh.spanIndex].endWith(traceCtx)
}

func (sh *SpanHandle) TraceID() uint64 {
	return sh.spanContext.traceContext.traceID
}

type TraceHandle struct {
	SpanHandle
}

func (th *TraceHandle) Collect() (spans []Span, attachment interface{}) {
	th.SpanHandle.Finish()
	return th.spanContext.traceContext.collect()
}
