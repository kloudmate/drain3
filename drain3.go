// Package drain3 implements the Drain log template mining algorithm in Go.
// It is a faithful port of the Python Drain3 library (github.com/logpai/Drain3).
//
// Drain3 processes streaming log messages and automatically extracts log templates
// by replacing variable parts with wildcards (<*>).
package drain3

// DefaultParamStr is the wildcard placeholder used for variable tokens in templates.
const DefaultParamStr = "<*>"

// ChangeType indicates what happened when a log message was processed.
type ChangeType int

const (
	// ChangeNone means the message matched an existing cluster with no template change.
	ChangeNone ChangeType = iota
	// ChangeClusterCreated means a new cluster was created for the message.
	ChangeClusterCreated
	// ChangeClusterTemplateChanged means an existing cluster's template was updated.
	ChangeClusterTemplateChanged
)

func (c ChangeType) String() string {
	switch c {
	case ChangeNone:
		return "none"
	case ChangeClusterCreated:
		return "cluster_created"
	case ChangeClusterTemplateChanged:
		return "cluster_template_changed"
	default:
		return "unknown"
	}
}

// SearchStrategy controls how Drain searches for matching clusters.
type SearchStrategy int

const (
	// SearchStrategyNever disables searching — only new clusters are created.
	SearchStrategyNever SearchStrategy = iota
	// SearchStrategyFallback searches existing clusters only when tree search fails.
	SearchStrategyFallback
	// SearchStrategyAlways always searches all clusters for the best match.
	SearchStrategyAlways
)

func (s SearchStrategy) String() string {
	switch s {
	case SearchStrategyNever:
		return "never"
	case SearchStrategyFallback:
		return "fallback"
	case SearchStrategyAlways:
		return "always"
	default:
		return "unknown"
	}
}
