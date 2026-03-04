package drain3

// Option configures a TemplateMiner. Use with New().
type Option func(*Config, *options)

// options holds non-Config settings that options can set.
type options struct {
	persistence PersistenceHandler
}

// WithSimTh sets the similarity threshold (0.0–1.0). Default: 0.4.
func WithSimTh(simTh float64) Option {
	return func(c *Config, _ *options) {
		c.Drain.SimTh = simTh
	}
}

// WithDepth sets the prefix tree depth. Default: 4.
func WithDepth(depth int) Option {
	return func(c *Config, _ *options) {
		c.Drain.Depth = depth
	}
}

// WithMaxChildren sets the maximum children per tree node. Default: 100.
func WithMaxChildren(maxChildren int) Option {
	return func(c *Config, _ *options) {
		c.Drain.MaxChildren = maxChildren
	}
}

// WithMaxClusters sets the maximum number of clusters (0 = unlimited). Default: 0.
func WithMaxClusters(maxClusters int) Option {
	return func(c *Config, _ *options) {
		c.Drain.MaxClusters = maxClusters
	}
}

// WithExtraDelimiters sets additional characters to split tokens on.
func WithExtraDelimiters(delimiters ...string) Option {
	return func(c *Config, _ *options) {
		c.Drain.ExtraDelimiters = delimiters
	}
}

// WithParamStr sets the wildcard placeholder string. Default: "<*>".
func WithParamStr(paramStr string) Option {
	return func(c *Config, _ *options) {
		c.Drain.ParamStr = paramStr
	}
}

// WithParametrizeNumericTokens controls whether numeric tokens are routed to
// the wildcard node in the prefix tree. Default: true.
func WithParametrizeNumericTokens(enabled bool) Option {
	return func(c *Config, _ *options) {
		c.Drain.ParametrizeNumericTokens = boolPtr(enabled)
	}
}

// WithMasking adds a regex masking instruction. The pattern is applied to log
// messages before template mining, and matches are replaced with <maskWith>.
// Can be called multiple times to add multiple masks.
//
// Example:
//
//	drain3.WithMasking(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP")
func WithMasking(regexPattern, maskWith string) Option {
	return func(c *Config, _ *options) {
		c.Masking.Instructions = append(c.Masking.Instructions, MaskingInstructionConfig{
			Pattern:  regexPattern,
			MaskWith: maskWith,
		})
	}
}

// WithMaskPrefix sets the prefix for mask tokens. Default: "<".
func WithMaskPrefix(prefix string) Option {
	return func(c *Config, _ *options) {
		c.Masking.MaskPrefix = prefix
	}
}

// WithMaskSuffix sets the suffix for mask tokens. Default: ">".
func WithMaskSuffix(suffix string) Option {
	return func(c *Config, _ *options) {
		c.Masking.MaskSuffix = suffix
	}
}

// WithPersistence sets a persistence handler for saving/loading state.
func WithPersistence(handler PersistenceHandler) Option {
	return func(_ *Config, o *options) {
		o.persistence = handler
	}
}

// WithFilePersistence sets file-based persistence at the given path.
func WithFilePersistence(filePath string) Option {
	return func(_ *Config, o *options) {
		o.persistence = NewFilePersistence(filePath)
	}
}

// WithSnapshotInterval sets how often state is auto-saved (in minutes). Default: 5.
func WithSnapshotInterval(minutes int) Option {
	return func(c *Config, _ *options) {
		c.Snapshot.SnapshotIntervalMinutes = minutes
	}
}

// WithCompressState enables or disables zlib compression for persisted state. Default: true.
func WithCompressState(enabled bool) Option {
	return func(c *Config, _ *options) {
		c.Snapshot.CompressState = enabled
	}
}

// WithProfiling enables the built-in simple profiler.
func WithProfiling(enabled bool) Option {
	return func(c *Config, _ *options) {
		c.Profiling.Enabled = enabled
	}
}

// WithConfig uses a pre-built Config directly. This overrides all other
// drain/masking/snapshot/profiling options set before it.
func WithConfig(cfg *Config) Option {
	return func(c *Config, _ *options) {
		*c = *cfg
	}
}

// New creates a new TemplateMiner with functional options.
//
// Example:
//
//	tm, err := drain3.New(
//	    drain3.WithSimTh(0.5),
//	    drain3.WithDepth(5),
//	    drain3.WithMaxClusters(1000),
//	    drain3.WithMasking(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP"),
//	    drain3.WithMasking(`\b\d+\b`, "NUM"),
//	    drain3.WithFilePersistence("/tmp/drain3_state.json"),
//	)
func New(opts ...Option) (*TemplateMiner, error) {
	cfg := DefaultConfig()
	o := &options{}

	for _, opt := range opts {
		opt(cfg, o)
	}

	return NewTemplateMiner(o.persistence, cfg)
}
