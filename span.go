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

type Span struct {
	ID              uint64
	ParentID        uint64 // 0 means Root
	BeginUnixTimeNs uint64
	DurationNs      uint64
	Event           string
	Properties      []Property
}

func (s *Span) beginWith(parentID uint64, event string) {
	s.ID = nextID()
	s.ParentID = parentID

	// Fill a monotonic time for now. After span is finished, it will replace by a unix time.
	s.BeginUnixTimeNs = monotimeNs()

	s.Event = event
}

func (s *Span) endWith(ctx *traceContext) {
	// For now, `beginUnixTimeNs` is a monotonic time. Here to correct its value to satisfy the semantic.
	beginMonoTimeNs := s.BeginUnixTimeNs
	s.DurationNs = monotimeNs() - beginMonoTimeNs
	s.BeginUnixTimeNs = (beginMonoTimeNs - ctx.createMonoTimeNs) + ctx.createUnixTimeNs
}

func (s *Span) addProperty(key, value string) {
	s.Properties = append(s.Properties, Property{Key: key, Value: value})
}

type Property struct {
	Key   string
	Value string
}
