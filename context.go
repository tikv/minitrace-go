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

type spanContext struct {
	parent context.Context

	tracingContext *tracingContext
	tracedSpans    *localSpans
	currentId      uint32
	currentGid     int64
}

type tracingKey struct{}

var activeTracingKey = tracingKey{}

func (s spanContext) Deadline() (deadline time.Time, ok bool) {
	return s.parent.Deadline()
}

func (s spanContext) Done() <-chan struct{} {
	return s.parent.Done()
}

func (s spanContext) Err() error {
	return s.parent.Err()
}

func (s spanContext) Value(key interface{}) interface{} {
	if key == activeTracingKey {
		return s
	} else {
		return s.parent.Value(key)
	}
}

// Represents a per goroutine buffer
type localSpans struct {
	spans        *bufferList
	createTimeNs uint64
	refCount     int
}

type tracingContext struct {
	traceId uint64

	mu             sync.Mutex
	collectedSpans []SpanSet
}
