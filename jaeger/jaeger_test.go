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
    "context"
    "github.com/pingcap-incubator/minitrace-go"
    "math/rand"
    "net"
    "strconv"
    "sync"
    "testing"
    "time"
)

func TestJaeger(t *testing.T) {
    ctx, handle := minitrace.TraceEnable(context.Background(), 0, 10010)
    var wg sync.WaitGroup

    for i := 1; i < 5; i++ {
        ctx, handle := minitrace.NewSpanWithContext(ctx, uint32(i))
        wg.Add(1)
        go func(prefix int) {
            ctx, handle := minitrace.NewSpanWithContext(ctx, uint32(prefix))
            for i := 0; i < 5; i++ {
                wg.Add(1)
                go func(prefix int) {
                    handle := minitrace.NewSpan(ctx, uint32(prefix))
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
    spanSets := handle.Collect()

    buf := make([]uint8, 0, 4096)
    rand.Seed(time.Now().UnixNano())
    ThriftCompactEncode(&buf, "TiDB", rand.Int63(), rand.Int63(), spanSets, func(span *minitrace.Span) SpanInfo {
        return SpanInfo{
            SelfId:        int64(span.Id),
            ParentId:      int64(span.Parent),
            Ref:           FollowFrom,
            OperationName: strconv.Itoa(int(span.Event)),
        }
    })

    if conn, err := net.Dial("udp", "127.0.0.1:6831"); err == nil {
        _, _ = conn.Write(buf)
    }
}