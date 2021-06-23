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
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/tikv/minitrace-go"
)

func TestDatadog(t *testing.T) {
	ctx, handle := minitrace.StartRootSpan(context.Background(), "root", 10010, 0,nil)
	handle.AddProperty("event1", "root")
	handle.AddProperty("event2", "root")
	var wg sync.WaitGroup

	for i := 1; i < 5; i++ {
		ctx, handle := minitrace.StartSpanWithContext(ctx, fmt.Sprintf("span%d", i))
		handle.AddProperty("event1", fmt.Sprintf("span%d", i))
		handle.AddProperty("event2", fmt.Sprintf("span%d", i))
		wg.Add(1)
		go func(prefix int) {
			ctx, handle := minitrace.StartSpanWithContext(ctx, fmt.Sprintf("span%d", prefix))
			handle.AddProperty("event1", fmt.Sprintf("span%d", prefix))
			handle.AddProperty("event2", fmt.Sprintf("span%d", prefix))
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(prefix int) {
					handle := minitrace.StartSpan(ctx, fmt.Sprintf("span%d", prefix))
					handle.AddProperty("event1", fmt.Sprintf("span%d", prefix))
					handle.AddProperty("event2", fmt.Sprintf("span%d", prefix))
					handle.AddProperty("event3", fmt.Sprintf("span%d", prefix))
					handle.Finish()
					wg.Done()
				}((prefix + i) * 10)
			}
			handle.Finish()
			wg.Done()
		}(i * 10)
		handle.Finish()
	}

	wg.Wait()
	trace, _ := handle.Collect()

	rand.Seed(time.Now().UnixNano())

	buf := bytes.NewBuffer([]byte{})
	spanList := MiniSpansToDatadogSpanList("datadog-test", trace)
	if err := MessagePackEncode(buf, spanList); err == nil {
		_ = Send(buf, "127.0.0.1:8126")
	} else {
		t.Fatal(err)
	}
}
