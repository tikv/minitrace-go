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
