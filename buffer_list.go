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
