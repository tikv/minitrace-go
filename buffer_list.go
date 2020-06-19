// Copyright 2020 TiKV Project Authors. Licensed under Apache-2.0.

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
    tail *buffer
    len  int
}

var bufferPool = &sync.Pool{New: func() interface{} { return &buffer{} }}

func newBufferList() *bufferList {
    n := bufferPool.Get().(*buffer)
    return &bufferList{n, n, 0}
}

func (bl *bufferList) slot() *Span {
    idx := bl.len & ((1 << POW) - 1)
    if idx == 0 {
        n := bufferPool.Get().(*buffer)

        bl.tail.next = n
        bl.tail = n
    }

    bl.len += 1
    return &bl.tail.array[idx]
}

func (bl *bufferList) collect() []Span {
    if bl.len == 0 {
        return nil
    }

    h := bl.head.next
    bufferPool.Put(bl.head)
    bl.head = nil

    res := make([]Span, bl.len, bl.len)

    remainingLen := bl.len
    sizePerBuffer := 1 << POW
    for remainingLen > sizePerBuffer {
        cursor := bl.len - remainingLen
        copy(res[cursor:cursor+sizePerBuffer], h.array[:])
        remainingLen -= sizePerBuffer
        n := h.next
        bufferPool.Put(h)
        h = n
    }

    cursor := bl.len - remainingLen
    copy(res[cursor:], h.array[:remainingLen])
    bufferPool.Put(h)

    return res
}
