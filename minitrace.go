package minitrace

type SpanSet struct {
    StartTimeNs  uint64
    Spans        []Span
}

type Span struct {
    Id      uint32
    Parent  uint32
    BeginNs uint64
    EndNs   uint64
    Event   uint32
}
