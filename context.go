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
	"sync"
)

// A span context embedded into ctx.context
type spanContext struct {
	tracingContext *tracingContext

	// A "goroutine-local" span collection
	tracedSpans *localSpans

	// Used to build parent-child relation between spans
	currentSpanId uint32

	// Used to check if the new span is created at another goroutine
	currentGid int64

	createUnixTimeNs uint64
	createMonoTimeNs uint64
}

type tracingKey struct{}

var activeTracingKey = tracingKey{}

// Represents a per goroutine buffer
type localSpans struct {
	spans    *bufferList
	refCount int
}

type tracingContext struct {
	traceId uint64

	mu             sync.Mutex
	collectedSpans []Span
	attachment     interface{}
	collected      bool
}
