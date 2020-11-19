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

package jaeger

import (
	"github.com/tikv/minitrace-go"
)

type ReferenceType int32

const (
	ChildOf    ReferenceType = 0
	FollowFrom ReferenceType = 1
)

type SpanInfo struct {
	SelfId        int64
	ParentId      int64
	Ref           ReferenceType
	OperationName string
}

func ThriftCompactEncode(
	buf *[]uint8,
	serviceName string,
	traceIdHigh int64,
	traceIdLow int64,
	spans []minitrace.Span,
) {
	*buf = append(*buf, []uint8{
		0x82, 0x81, 0x00, 0x09, 0x65, 0x6d, 0x69, 0x74, 0x42, 0x61, 0x74, 0x63, 0x68, 0x1c, 0x1c,
		0x18,
	}...)

	encodeBytes(buf, []uint8(serviceName))
	*buf = append(*buf, 0x00)

	// len of spans
	*buf = append(*buf, 0x19)
	spanLen := len(spans)
	if spanLen < 15 {
		*buf = append(*buf, uint8(spanLen<<4)|12)
	} else {
		*buf = append(*buf, 0b1111_0000|12)
		encodeVarInt(buf, uint64(spanLen))
	}

	for _, span := range spans {
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(traceIdLow))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(traceIdHigh))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(int64(span.Id)))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(int64(span.Parent)))
		*buf = append(*buf, 0x18)
		encodeBytes(buf, []uint8(span.Event))

		*buf = append(*buf, []uint8{0x19, 0x1c, 0x15}...)
		encodeVarInt(buf, uint64(zigzagFromI32(int32(FollowFrom))))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(traceIdLow))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(traceIdHigh))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(int64(span.Parent)))
		*buf = append(*buf, 0x00)
		*buf = append(*buf, 0x15)
		*buf = append(*buf, 0x02)

		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(int64(span.BeginUnixTimeNs/1_000)))
		*buf = append(*buf, 0x16)
		encodeVarInt(buf, zigzagFromI64(int64(span.DurationNs/1_000)))
		propertiesLen := len(span.Properties)
		if propertiesLen > 0 {
			*buf = append(*buf, 0x19)
			if propertiesLen < 15 {
				*buf = append(*buf, uint8((propertiesLen<<4)|12))
			} else {
				*buf = append(*buf, uint8(0b1111_0000|12))
				encodeVarInt(buf, uint64(propertiesLen))
			}

			for _, p := range span.Properties {
				*buf = append(*buf, 0x18)
				encodeBytes(buf, []byte(p.Key))

				*buf = append(*buf, 0x15)
				*buf = append(*buf, 0x00)

				*buf = append(*buf, 0x18)
				encodeBytes(buf, []byte(p.Value))

				*buf = append(*buf, 0x00)
			}
		}
		*buf = append(*buf, 0x00)
	}

	*buf = append(*buf, 0x00)
	*buf = append(*buf, 0x00)
}

func encodeBytes(buf *[]uint8, bytes []uint8) {
	encodeVarInt(buf, uint64(len(bytes)))
	*buf = append(*buf, bytes...)
}

func encodeVarInt(buf *[]uint8, n uint64) {
	for {
		b := uint8(n & 0b0111_1111)
		n >>= 7
		if n != 0 {
			b |= 0b1000_0000
		}
		*buf = append(*buf, b)
		if n == 0 {
			break
		}
	}
}

func zigzagFromI32(n int32) uint32 {
	return uint32((n << 1) ^ (n >> 31))
}

func zigzagFromI64(n int64) uint64 {
	return uint64((n << 1) ^ (n >> 63))
}
