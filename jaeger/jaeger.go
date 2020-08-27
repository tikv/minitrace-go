package jaeger

import "github.com/pingcap-incubator/minitrace-go"

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
    traceResults []minitrace.SpanSet,
    spanRemap func(*minitrace.Span) SpanInfo,

) {
    *buf = append(*buf, []uint8{
        0x82, 0x81, 0x00, 0x09, 0x65, 0x6d, 0x69, 0x74, 0x42, 0x61, 0x74, 0x63, 0x68, 0x1c, 0x1c,
        0x18,
    }...)

    encodeBytes(buf, []uint8(serviceName))
    *buf = append(*buf, 0x00)

    // len of spans
    *buf = append(*buf, 0x19)
    spanLen := 0
    for _, result := range traceResults {
        spanLen += len(result.Spans)
    }
    if spanLen < 15 {
        *buf = append(*buf, uint8(spanLen << 4) | 12)
    } else {
        *buf = append(*buf, 0b1111_0000 | 12)
        encodeVarInt(buf, uint64(spanLen))
    }

    for _, traceResult := range traceResults {
        startNs := traceResult.StartTimeNs
        anchorNs := traceResult.Spans[0].BeginNs
        for _, span := range traceResult.Spans {
            jaegerSpan := spanRemap(&span)
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(traceIdLow))
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(traceIdHigh))
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(jaegerSpan.SelfId))
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(jaegerSpan.ParentId))
            *buf = append(*buf, 0x18)
            encodeBytes(buf, []uint8(jaegerSpan.OperationName))

            *buf = append(*buf, []uint8{0x19, 0x1c, 0x15}...)
            encodeVarInt(buf, uint64(zigzagFromI32(int32(jaegerSpan.Ref))))
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(traceIdLow))
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(traceIdHigh))
            *buf = append(*buf, 0x16)
            encodeVarInt(buf, zigzagFromI64(jaegerSpan.ParentId))
            *buf = append(*buf, 0x00)
            *buf = append(*buf, 0x15)
            *buf = append(*buf, 0x02)

            *buf = append(*buf, 0x16)
            timeStampUs := (startNs + (span.BeginNs - anchorNs)) / 1000
            encodeVarInt(buf, zigzagFromI64(int64(timeStampUs)))
            *buf = append(*buf, 0x16)
            durationUs := (span.EndNs - span.BeginNs) / 1000
            encodeVarInt(buf, zigzagFromI64(int64(durationUs)))
            *buf = append(*buf, 0x00)
        }
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