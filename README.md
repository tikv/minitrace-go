# Minitrace-Go
[![Actions Status](https://github.com/tikv/minitrace-go/workflows/CI/badge.svg)](https://github.com/tikv/minitrace-go/actions)
[![LICENSE](https://img.shields.io/github/license/tikv/minitrace-go.svg)](https://github.com/tikv/minitrace-go/blob/master/LICENSE)

A high-performance, ergonomic timeline tracing library for Golang. 

## Basic Usage

```go
package main

import (
    "context"
	"encoding/json"
	"fmt"
    "strconv"

    "github.com/tikv/minitrace-go"
)

func tracedFunc(ctx context.Context, event string) {
    span := minitrace.StartSpan(ctx, event)
    // code snippet...
    span.Finish()
}

func iterTracedFunc(ctx context.Context) {
    // extend tracing context from parent context
    ctx, span := minitrace.StartSpanWithContext(ctx, "1")

    span.AddProperty("k2", "v2")

    for i := 2; i < 10; i++ {
        tracedFunc(ctx, strconv.Itoa(i))
    }
    
    span.Finish()
}

func main() {
    ctx := context.Background()

    // enable tracing
    ctx, root := minitrace.StartRootSpan(ctx, "root", 10010, 0, nil)

    root.AddProperty("k1", "v1")

    // pass the context to traced functions
    iterTracedFunc(ctx)

    // collect tracing results into `spans`
    spans, _ := root.Collect()

    // print `spans` content in JSON format
	spanContent, _ := json.MarshalIndent(spans, "", "  ")
	fmt.Println(string(spanContent))
}
```

The output of spans in JSON format like this:

```json
{
  "TraceID": 10010,
  "Spans": [
    {
      "ID": 15352856648520921629,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306972169,
      "DurationNs": 87,
      "Event": "2",
      "Properties": null
    },
    {
      "ID": 13260572831089785859,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306973148,
      "DurationNs": 61,
      "Event": "3",
      "Properties": null
    },
    {
      "ID": 3916589616287113937,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306979536,
      "DurationNs": 63,
      "Event": "4",
      "Properties": null
    },
    {
      "ID": 6334824724549167320,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306983182,
      "DurationNs": 63,
      "Event": "5",
      "Properties": null
    },
    {
      "ID": 9828766684487745566,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306983484,
      "DurationNs": 59,
      "Event": "6",
      "Properties": null
    },
    {
      "ID": 10667007354186551956,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306987321,
      "DurationNs": 61,
      "Event": "7",
      "Properties": null
    },
    {
      "ID": 894385949183117216,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306987618,
      "DurationNs": 61,
      "Event": "8",
      "Properties": null
    },
    {
      "ID": 11998794077335055257,
      "ParentID": 8674665223082153551,
      "BeginUnixTimeNs": 1662446668306987898,
      "DurationNs": 60,
      "Event": "9",
      "Properties": null
    },
    {
      "ID": 8674665223082153551,
      "ParentID": 5577006791947779410,
      "BeginUnixTimeNs": 1662446668306967492,
      "DurationNs": 20570,
      "Event": "1",
      "Properties": [
        {
          "Key": "k2",
          "Value": "v2"
        }
      ]
    },
    {
      "ID": 5577006791947779410,
      "ParentID": 0,
      "BeginUnixTimeNs": 1662446668306966615,
      "DurationNs": 24319,
      "Event": "root",
      "Properties": [
        {
          "Key": "k1",
          "Value": "v1"
        }
      ]
    }
  ]
}
```