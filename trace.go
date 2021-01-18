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
	spanCtx := newSpanContext(ctx, traceCtx)

	// Root span doesn't have a parent. Its parent span id is set to 0.
	const ParentID = 0
	spanHandle := newSpanHandle(spanCtx, ParentID, event)

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
	spanCtx := newSpanContext(parentCtx, traceCtx)
	return newSpanHandle(spanCtx, parentSpanCtx.spanID, event)
}

func CurrentID(ctx context.Context) (spanID uint32, traceID uint64, ok bool) {
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
	span        Span
	finished    bool
}

func newSpanHandle(spanCtx *spanContext, parentSpanID uint32, event string) (sh SpanHandle) {
	sh.spanContext = spanCtx
	sh.span.beginWith(parentSpanID, event)
	sh.finished = false
	spanCtx.spanID = sh.span.ID
	return
}

func (sh *SpanHandle) AddProperty(key, value string) {
	if sh.finished {
		return
	}
	sh.span.addProperty(key, value)
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
	sh.span.endWith(traceCtx)

	traceCtx.mu.Lock()
	traceCtx.collectedSpans = append(traceCtx.collectedSpans, sh.span)
	traceCtx.mu.Unlock()
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
