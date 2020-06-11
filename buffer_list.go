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
    var cur *buffer
    if idx == 0 {
        cur = bufferPool.Get().(*buffer)
        cur.next = bl.head
        bl.head = cur
    } else {
        cur = bl.head
    }

    bl.len += 1
    return &cur.array[idx]
}

func (bl *bufferList) collect() []Span {
    if bl.len == 0 {
        return nil
    }

    h := bl.head
    next := h.next
    res := make([]Span, bl.len, bl.len)

    size := 1 << POW
    f := bl.len & (size - 1)
    if f == 0 {
        f = size
    }

    copy(res[:f], h.array[:f])
    bufferPool.Put(h)
    h = h.next

    for h != nil {
        copy(res[:size], h.array[:size])
        next = h.next
        bufferPool.Put(h)
        h = next
    }

    return res
}
