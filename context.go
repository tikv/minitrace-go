package minitrace

import "sync"

type spanContext struct {
    tracingContext *tracingContext
    tracedSpans    *localSpans
    currentId      uint64
    currentGid     int64
}

type localSpans struct {
    spans        []Span
    createTimeNs uint64
    refCount     int
}

type tracingContext struct {
    maxId uint64

    mu             sync.Mutex
    collectedSpans []SpanSet
}
