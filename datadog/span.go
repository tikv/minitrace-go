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

//go:generate msgp -unexported -marshal=false -o=span_msgp.go -tests=false

package datadog

import (
    "github.com/tikv/minitrace-go"
    "github.com/tinylib/msgp/msgp"
)

var (
    _ msgp.Encodable = (*SpanList)(nil)
)

type (
    SpanList []*Span
)

type Span struct {
    Name     string            `msg:"name"`
    Service  string            `msg:"service"`
    Start    int64             `msg:"start"`
    Duration int64             `msg:"duration"`
    Meta     map[string]string `msg:"meta,omitempty"`
    SpanID   uint64            `msg:"span_id"`
    TraceID  uint64            `msg:"trace_id"`
    ParentID uint64            `msg:"parent_id"`
}

func miniSpansToDdSpanList(
    serviceName string,
    traceId uint64,
    spanIdPrefix uint32,
    spans []minitrace.Span,
) SpanList {
    ddSpans := make([]*Span, 0, len(spans))

    for _, span := range spans {
        meta := make(map[string]string)
        for _, property := range span.Properties {
            meta[property.Key] = property.Value
        }
        ddSpan := &Span{
            Name:     span.Event,
            Service:  serviceName,
            Start:    int64(span.BeginUnixTimeNs),
            Duration: int64(span.DurationNs),
            Meta:     meta,
            SpanID:   uint64(spanIdPrefix)<<32 | uint64(span.Id),
            TraceID:  traceId,
            ParentID: uint64(spanIdPrefix)<<32 | uint64(span.Parent),
        }
        ddSpans = append(ddSpans, ddSpan)
    }

    return ddSpans
}
