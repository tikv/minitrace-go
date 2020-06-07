package minitrace

type SpanSet struct {
    CreateTimeNs uint64
    StartTimeNs  uint64
    CyclesPerSec uint64
    Spans        []Span
}

type Span struct {
    Id          uint64
    Link        Link
    BeginCycles uint64
    EndCycles   uint64
    Event       uint32
}

type Link interface {
    isLink()
}

func (_ Root) isLink()     {}
func (_ Parent) isLink()   {}
func (_ Continue) isLink() {}

type Root struct{}
type Parent struct {
    Id uint64
}
type Continue struct {
    Id uint64
}
