// Copyright 2021 PingCAP, Inc.
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

// Represents a per goroutine buffer
type localSpanBuffer struct {
	bufferList *bufferList
	refCount   int
}

type localSpanHandle struct {
	span *Span
}

func newLocalSpanBuffer() *localSpanBuffer {
	return &localSpanBuffer{
		bufferList: nil,
		refCount:   0,
	}
}

func (lb *localSpanBuffer) pushSpan(parentID uint32, event string) localSpanHandle {
	if lb.refCount == 0 {
		lb.bufferList = newBufferList()
	}
	lb.refCount += 1
	span := lb.bufferList.slot()
	span.beginWith(parentID, event)
	return localSpanHandle{span: span}
}

func (lb *localSpanBuffer) finishSpan(handle localSpanHandle, ctx *traceContext) {
	handle.span.endWith(ctx)
	lb.refCount -= 1
	if lb.refCount > 0 {
		return
	}

	spans := lb.bufferList.collect()
	lb.bufferList = nil
	ctx.extendSpans(spans)
}
