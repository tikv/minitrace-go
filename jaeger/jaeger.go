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
	"io"
	"net"

	"github.com/tikv/minitrace-go"
)

func Send(buf []byte, agent string) error {
	conn, err := net.Dial("udp", agent)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(buf)
	return err
}

func MiniSpansToJaegerTrace(
	serviceName string,
	traceID int64,
	spanIDPrefix uint32,
	rootParentSpanID int64,
	spans []minitrace.Span,
) Trace {
	retSpans := make([]Span, 0, len(spans))

	for _, span := range spans {
		parentID := int64(spanIDPrefix)<<32 | int64(span.ParentID)
		if span.ParentID == 0 {
			parentID = rootParentSpanID
		}

		var tags []struct {
			Key   string
			Value string
		}

		for _, property := range span.Properties {
			tags = append(tags, struct {
				Key   string
				Value string
			}{
				Key:   property.Key,
				Value: property.Value,
			})
		}

		retSpans = append(retSpans, Span{
			SpanID:          int64(spanIDPrefix)<<32 | int64(span.ID),
			ParentID:        parentID,
			StartUnixTimeUs: int64(span.BeginUnixTimeNs / 1000),
			DurationUs:      int64(span.DurationNs / 1000),
			OperationName:   span.Event,
			Tags:            tags,
		})
	}

	return Trace{
		TraceID:     traceID,
		ServiceName: serviceName,
		Spans:       retSpans,
	}
}

func ThriftCompactEncode(
	b io.Writer,
	trace Trace,
) error {
	buf := []byte{
		0x82, 0x81, 0x00, 0x09, 0x65, 0x6d, 0x69, 0x74, 0x42, 0x61, 0x74, 0x63, 0x68, 0x1c, 0x1c,
		0x18,
	}

	encodeBytes(&buf, []byte(trace.ServiceName))
	buf = append(buf, 0x00)

	// len of spans
	buf = append(buf, 0x19)
	spanLen := len(trace.Spans)
	if spanLen < 15 {
		buf = append(buf, byte(spanLen<<4)|12)
	} else {
		buf = append(buf, 0b1111_0000|12)
		encodeVarInt(&buf, uint64(spanLen))
	}

	for _, span := range trace.Spans {
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(trace.TraceID))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(trace.TraceID))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(span.SpanID))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(span.ParentID))
		buf = append(buf, 0x18)
		encodeBytes(&buf, []byte(span.OperationName))

		buf = append(buf, []byte{0x19, 0x1c, 0x15}...)
		encodeVarInt(&buf, uint64(zigzagFromI32(int32(1 /* Follow from */))))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(trace.TraceID))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(trace.TraceID))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(span.ParentID))
		buf = append(buf, 0x00)
		buf = append(buf, 0x15)
		buf = append(buf, 0x02)

		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(span.StartUnixTimeUs))
		buf = append(buf, 0x16)
		encodeVarInt(&buf, zigzagFromI64(span.DurationUs))
		tagLen := len(span.Tags)
		if tagLen > 0 {
			buf = append(buf, 0x19)
			if tagLen < 15 {
				buf = append(buf, byte((tagLen<<4)|12))
			} else {
				buf = append(buf, byte(0b1111_0000|12))
				encodeVarInt(&buf, uint64(tagLen))
			}

			for _, t := range span.Tags {
				buf = append(buf, 0x18)
				encodeBytes(&buf, []byte(t.Key))

				buf = append(buf, 0x15)
				buf = append(buf, 0x00)

				buf = append(buf, 0x18)
				encodeBytes(&buf, []byte(t.Value))

				buf = append(buf, 0x00)
			}
		}
		buf = append(buf, 0x00)
	}

	buf = append(buf, 0x00)
	buf = append(buf, 0x00)

	_, err := b.Write(buf)
	return err
}

func encodeBytes(buf *[]byte, bytes []byte) {
	encodeVarInt(buf, uint64(len(bytes)))
	*buf = append(*buf, bytes...)
}

func encodeVarInt(buf *[]byte, n uint64) {
	for {
		b := byte(n & 0b0111_1111)
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
