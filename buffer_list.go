package minitrace

import (
    "sync"
)

const POW = 8

type buffer struct {
    array [1 << POW]Span
    next  *buffer
}

type bufferList struct {
    head *buffer
    len  int
}

var bufferPool = &sync.Pool{New: func() interface{} { return &buffer{} }}

func newBufferList() *bufferList {
    return &bufferList{nil, 0}
}

func (bl *bufferList) slot() *Span {
    idx := bl.len & ((1 << POW) - 1)
    if idx == 0 {
        n := bufferPool.Get().(*buffer)
        n.next = bl.head
        bl.head = n
    }

    bl.len += 1
    return &bl.head.array[idx]
}

func (bl *bufferList) collect() []Span {
    if bl.len == 0 {
        return nil
    }

    h := bl.head
    next := h.next
    res := make([]Span, bl.len, bl.len)

    size := 1 << POW
    cursor := bl.len & (size - 1)
    if cursor == 0 {
        cursor = size
    }

    copy(res[:cursor], h.array[:cursor])
    bufferPool.Put(h)
    h = next

    for h != nil {
        copy(res[cursor:cursor+size], h.array[:size])
        cursor += size
        next = h.next
        bufferPool.Put(h)
        h = next
    }

    return res
}
