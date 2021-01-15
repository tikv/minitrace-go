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

package datadog

import (
	"fmt"
	"io"
	"net/http"

	"github.com/tikv/minitrace-go"
	"github.com/tinylib/msgp/msgp"
)

func Send(buf io.Reader, agent string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/v0.4/traces", agent), buf)
	if err != nil {
		return fmt.Errorf("cannot create http request: %v", err)
	}
	req.Header.Set("Datadog-Meta-Tracer-Version", "v1.27.0")
	req.Header.Set("Content-Type", "application/msgpack")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if code := response.StatusCode; code >= 400 {
		msg := make([]byte, 1000)
		n, _ := response.Body.Read(msg)
		txt := http.StatusText(code)
		if n > 0 {
			return fmt.Errorf("%s (Status: %s)", msg[:n], txt)
		}
		return fmt.Errorf("%s", txt)
	}
	return nil
}

func MessagePackEncode(buf io.Writer, spanList SpanList) error {
	if _, err := buf.Write([]byte{145}); err != nil {
		return err
	}

	if err := msgp.Encode(buf, spanList); err != nil {
		return err
	}

	return nil
}

func MiniSpansToDatadogSpanList(
	serviceName string,
	traceID uint64,
	spanIDPrefix uint32,
	rootParentSpanID uint64,
	spans []minitrace.Span,
) SpanList {
	ddSpans := make([]*Span, 0, len(spans))

	for _, span := range spans {
		parentID := uint64(spanIDPrefix)<<32 | uint64(span.ParentID)
		if span.ParentID == 0 {
			parentID = rootParentSpanID
		}

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
			SpanID:   uint64(spanIDPrefix)<<32 | uint64(span.ID),
			TraceID:  traceID,
			ParentID: parentID,
		}
		ddSpans = append(ddSpans, ddSpan)
	}

	return ddSpans
}
