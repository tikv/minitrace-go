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
	"sync"
	"time"
)

// A span context embedded into ctx.context
type spanContext struct {
	parent context.Context

	// Shared trace context
	traceContext *traceContext
	spanID       uint32
}

func newSpanContext(ctx context.Context, tracingCtx *traceContext) *spanContext {
	return &spanContext{
		parent:       ctx,
		traceContext: tracingCtx,
	}
}

type tracingKey struct{}

var activeTraceKey = tracingKey{}

func (s *spanContext) Deadline() (deadline time.Time, ok bool) {
	return s.parent.Deadline()
}

func (s *spanContext) Done() <-chan struct{} {
	return s.parent.Done()
}

func (s *spanContext) Err() error {
	return s.parent.Err()
}

func (s *spanContext) Value(key interface{}) interface{} {
	if key == activeTraceKey {
		return s
	} else {
		return s.parent.Value(key)
	}
}

type traceContext struct {
	/// Frozen fields
	traceID          uint64
	createUnixTimeNs uint64
	createMonoTimeNs uint64

	/// Shared mutable fields
	mu             sync.Mutex
	collectedSpans []Span
	attachment     interface{}
	collected      bool
}

func newTraceContext(traceID uint64, attachment interface{}) *traceContext {
	return &traceContext{
		traceID:          traceID,
		createUnixTimeNs: unixtimeNs(),
		createMonoTimeNs: monotimeNs(),
		attachment:       attachment,
		collected:        false,
	}
}

func (tc *traceContext) accessAttachment(fn func(attachment interface{})) (ok bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.collected {
		return false
	}

	fn(tc.attachment)
	return true
}

func (tc *traceContext) pushSpan(span *Span) (ok bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.collected {
		return false
	}

	tc.collectedSpans = append(tc.collectedSpans, *span)
	return true
}

func (tc *traceContext) collect() (spans []Span, attachment interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.collected {
		return
	}
	tc.collected = true

	spans = tc.collectedSpans
	attachment = tc.attachment

	tc.collectedSpans = nil
	tc.attachment = nil

	return
}
