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
    tm, _ := drain3.New()

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
- **Configurable** — functional options API or YAML config, with sensible defaults
- **Parameter extraction** — extract variable values from log messages given a discovered template

## Configuration

### Functional Options (recommended)

Pass options directly when creating a `TemplateMiner`:

```go
tm, err := drain3.New(
    drain3.WithSimTh(0.5),
    drain3.WithDepth(5),
    drain3.WithMaxClusters(1000),
    drain3.WithExtraDelimiters("=", ":"),
    drain3.WithMasking(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP"),
    drain3.WithMasking(`\b\d+\b`, "NUM"),
    drain3.WithFilePersistence("/tmp/drain3_state.json"),
)
```

Available options:

| Option | Default | Description |
|--------|---------|-------------|
| `WithSimTh(float64)` | `0.4` | Similarity threshold (0.0–1.0) |
| `WithDepth(int)` | `4` | Prefix tree depth |
| `WithMaxChildren(int)` | `100` | Max children per tree node |
| `WithMaxClusters(int)` | `0` | Max clusters (0 = unlimited, uses LRU eviction when set) |
| `WithExtraDelimiters(string...)` | none | Additional characters to split tokens on |
| `WithParamStr(string)` | `<*>` | Wildcard placeholder string |
| `WithParametrizeNumericTokens(bool)` | `true` | Route numeric tokens to the wildcard node |
| `WithMasking(pattern, name)` | none | Add a regex masking rule (can be called multiple times) |
| `WithMaskPrefix(string)` | `<` | Prefix for mask tokens |
| `WithMaskSuffix(string)` | `>` | Suffix for mask tokens |
| `WithPersistence(PersistenceHandler)` | `nil` | Custom persistence backend |
| `WithFilePersistence(path)` | none | File-based persistence |
| `WithSnapshotInterval(minutes)` | `5` | Auto-save interval |
| `WithCompressState(bool)` | `true` | Zlib compression for persisted state |
| `WithProfiling(bool)` | `false` | Enable built-in profiler |

### YAML Config (alternative)

```yaml
drain:
  sim_th: 0.5
  depth: 5
  max_children: 100
  max_clusters: 1000
  extra_delimiters: ["=", ":"]
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
```

```go
cfg, _ := drain3.LoadConfig("drain3.yaml")
tm, _ := drain3.New(drain3.WithConfig(cfg))
```

## API

### TemplateMiner

The high-level API that integrates all components.

```go
// Create with defaults
tm, _ := drain3.New()

// Create with options
tm, _ := drain3.New(
    drain3.WithSimTh(0.5),
    drain3.WithMasking(`\d+`, "NUM"),
)

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
