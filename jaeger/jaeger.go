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

func ThriftCompactEncode(
    b io.Writer,
    serviceName string,
    traceIdHigh int64,
    traceIdLow int64,
    spanIdPrefix uint32,
    spans []minitrace.Span,
) error {
    buf := []byte{
        0x82, 0x81, 0x00, 0x09, 0x65, 0x6d, 0x69, 0x74, 0x42, 0x61, 0x74, 0x63, 0x68, 0x1c, 0x1c,
        0x18,
    }

    encodeBytes(&buf, []byte(serviceName))
    buf = append(buf, 0x00)

    // len of spans
    buf = append(buf, 0x19)
    spanLen := len(spans)
    if spanLen < 15 {
        buf = append(buf, byte(spanLen<<4)|12)
    } else {
        buf = append(buf, 0b1111_0000|12)
        encodeVarInt(&buf, uint64(spanLen))
    }

    for _, span := range spans {
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(traceIdLow))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(traceIdHigh))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(int64(spanIdPrefix)<<32|int64(span.Id)))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(int64(spanIdPrefix)<<32|int64(span.Parent)))
        buf = append(buf, 0x18)
        encodeBytes(&buf, []byte(span.Event))

        buf = append(buf, []byte{0x19, 0x1c, 0x15}...)
        encodeVarInt(&buf, uint64(zigzagFromI32(int32(1 /* Follow from */))))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(traceIdLow))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(traceIdHigh))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(int64(spanIdPrefix)<<32|int64(span.Parent)))
        buf = append(buf, 0x00)
        buf = append(buf, 0x15)
        buf = append(buf, 0x02)

        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(int64(span.BeginUnixTimeNs/1_000)))
        buf = append(buf, 0x16)
        encodeVarInt(&buf, zigzagFromI64(int64(span.DurationNs/1_000)))
        propertiesLen := len(span.Properties)
        if propertiesLen > 0 {
            buf = append(buf, 0x19)
            if propertiesLen < 15 {
                buf = append(buf, byte((propertiesLen<<4)|12))
            } else {
                buf = append(buf, byte(0b1111_0000|12))
                encodeVarInt(&buf, uint64(propertiesLen))
            }

            for _, p := range span.Properties {
                buf = append(buf, 0x18)
                encodeBytes(&buf, []byte(p.Key))

                buf = append(buf, 0x15)
                buf = append(buf, 0x00)

                buf = append(buf, 0x18)
                encodeBytes(&buf, []byte(p.Value))

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
