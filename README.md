# drain3

A Go implementation of the [Drain3](https://github.com/logpai/Drain3) log template mining library.

Drain3 processes streaming log messages and automatically extracts log templates by replacing variable parts with wildcards (`<*>`). It is based on the [Drain](https://jiemingzhu.github.io/pub/pjhe_icws2017.pdf) algorithm for online log parsing.

## Installation

```bash
go get github.com/kloudmate/drain3
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/kloudmate/drain3"
)

func main() {
    tm, _ := drain3.NewTemplateMiner(nil, nil)

    messages := []string{
        "Failed password for user admin from 192.168.1.1 port 22",
        "Failed password for user root from 10.0.0.1 port 22",
        "Accepted password for user admin from 192.168.1.1 port 22",
        "Accepted password for user admin from 10.0.0.1 port 22",
    }

    for _, msg := range messages {
        result := tm.AddLogMessage(msg)
        fmt.Printf("[%s] %s\n", result.ChangeType, result.Cluster.GetTemplate())
    }

    fmt.Printf("\nDiscovered %d templates:\n", tm.ClusterCount())
    for _, c := range tm.Clusters() {
        fmt.Printf("  [size=%d] %s\n", c.Size, c.GetTemplate())
    }
}
```

## Features

- **Online log parsing** — processes log messages one at a time in a streaming fashion
- **Template extraction** — automatically discovers log templates by replacing variable tokens with `<*>`
- **Masking** — pre-process log messages with regex-based masking (IP addresses, numbers, etc.) before template mining
- **Persistence** — save and restore state to/from files or custom backends (with optional zlib compression)
- **Thread-safe** — safe for concurrent use from multiple goroutines
- **Configurable** — YAML-based configuration with sensible defaults matching the Python Drain3 library
- **Parameter extraction** — extract variable values from log messages given a discovered template

## Configuration

Create a YAML config file:

```yaml
drain:
  sim_th: 0.4                    # Similarity threshold (0.0-1.0)
  depth: 4                       # Prefix tree depth
  max_children: 100              # Max children per tree node
  max_clusters: 0                # Max clusters (0 = unlimited)
  extra_delimiters:              # Additional token delimiters
    - "="
    - ":"
  parametrize_numeric_tokens: true

snapshot:
  snapshot_interval_minutes: 5
  compress_state: true

masking:
  mask_prefix: "<"
  mask_suffix: ">"
  instructions:
    - pattern: '\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}'
      mask_with: "IP"
    - pattern: '\b\d+\b'
      mask_with: "NUM"

profiling:
  enabled: false
  report_sec: 30
```

Load and use it:

```go
cfg, _ := drain3.LoadConfig("drain3.yaml")
persistence := drain3.NewFilePersistence("drain3_state.json")
tm, _ := drain3.NewTemplateMiner(persistence, cfg)
```

## API

### TemplateMiner

The high-level API that integrates all components.

```go
// Create with defaults (no persistence, no masking)
tm, _ := drain3.NewTemplateMiner(nil, nil)

// Process a log message
result := tm.AddLogMessage("user alice logged in from 10.0.0.1")
// result.Cluster    - the matched/created LogCluster
// result.ChangeType - ChangeNone, ChangeClusterCreated, or ChangeClusterTemplateChanged

// Match without modifying state
cluster := tm.Match("user bob logged in from 10.0.0.2", drain3.SearchStrategyNever)

// Save/restore state
tm.SaveState()
tm.LoadState()
```

### Drain

The core algorithm engine, usable standalone without masking or persistence.

```go
cfg := drain3.DefaultDrainConfig()
cfg.SimTh = 0.5
d := drain3.NewDrain(cfg)

cluster, changeType := d.AddLogMessage("hello world")
matched := d.Match("hello world", drain3.SearchStrategyNever)
```

### Parameter Extraction

Extract variable values from messages using discovered templates.

```go
pe := drain3.NewParameterExtractor(tm.Masker, nil)
params := pe.ExtractParameters("user <*> logged in", "user alice logged in", false)
// params[0].Value == "alice", params[0].MaskName == "*"
```

### Search Strategies

| Strategy | Behavior |
|----------|----------|
| `SearchStrategyNever` | Tree search only; fastest |
| `SearchStrategyFallback` | Tree search first, then full scan if no match |
| `SearchStrategyAlways` | Full scan of all clusters; most thorough |

## Dependencies

- [`gopkg.in/yaml.v3`](https://github.com/go-yaml/yaml) — YAML config parsing
- [`github.com/dlclark/regexp2`](https://github.com/dlclark/regexp2) — Perl-compatible regex for masking patterns (Go's stdlib `regexp` uses RE2 which doesn't support lookbehinds)

## Acknowledgements

This is a Go port of the Python [Drain3](https://github.com/logpai/Drain3) library by IBM Research, which implements the Drain algorithm from:

> Pinjia He, Jieming Zhu, Zibin Zheng, and Michael R. Lyu. [Drain: An Online Log Parsing Approach with Fixed Depth Tree](https://jiemingzhu.github.io/pub/pjhe_icws2017.pdf), Proceedings of the IEEE International Conference on Web Services (ICWS), 2017.

## License

MIT
