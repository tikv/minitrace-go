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
    req, err := http.NewRequest("POST",  fmt.Sprintf("http://%s/v0.4/traces", agent), buf)
    if err != nil {
        return fmt.Errorf("cannot create http request: %v", err)
    }

    req.Header.Set("Datadog-Meta-Tracer-Version", "v1.27.0")
    req.Header.Set("Content-Type", "application/msgpack")

    httpClient := &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyFromEnvironment,
        },
    }
    response, err := httpClient.Do(req)
    if err != nil {
        return err
    }
    if code := response.StatusCode; code >= 400 {
        msg := make([]byte, 1000)
        n, _ := response.Body.Read(msg)
        response.Body.Close()
        txt := http.StatusText(code)
        if n > 0 {
            return fmt.Errorf("%s (Status: %s)", msg[:n], txt)
        }
        return fmt.Errorf("%s", txt)
    }
    return nil
}

func MessagePackEncode(
    buf io.Writer,
    serviceName string,
    traceId uint64,
    spanIdPrefix uint32,
    spans []minitrace.Span,
) error {
    spanList := miniSpansToDdSpanList(serviceName, traceId, spanIdPrefix, spans)

    if _, err := buf.Write([]byte{145}); err != nil {
        return err
    }

    if err := msgp.Encode(buf, spanList); err != nil {
        return err
    }

    return nil
}
