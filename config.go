package drain3

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the TemplateMiner.
type Config struct {
	Drain     DrainSection     `yaml:"drain"`
	Snapshot  SnapshotSection  `yaml:"snapshot"`
	Masking   MaskingSection   `yaml:"masking"`
	Profiling ProfilingSection `yaml:"profiling"`
}

// DrainSection holds Drain algorithm parameters.
type DrainSection struct {
	SimTh                    float64  `yaml:"sim_th"`
	Depth                    int      `yaml:"depth"`
	MaxChildren              int      `yaml:"max_children"`
	MaxClusters              int      `yaml:"max_clusters"`
	ExtraDelimiters          []string `yaml:"extra_delimiters"`
	ParamStr                 string   `yaml:"param_str"`
	ParametrizeNumericTokens bool     `yaml:"parametrize_numeric_tokens"`
}

// SnapshotSection holds persistence/snapshot configuration.
type SnapshotSection struct {
	SnapshotIntervalMinutes int  `yaml:"snapshot_interval_minutes"`
	CompressState           bool `yaml:"compress_state"`
}

// MaskingSection holds masking configuration.
type MaskingSection struct {
	MaskPrefix   string                     `yaml:"mask_prefix"`
	MaskSuffix   string                     `yaml:"mask_suffix"`
	Instructions []MaskingInstructionConfig `yaml:"instructions"`
}

// MaskingInstructionConfig is the YAML representation of a masking instruction.
type MaskingInstructionConfig struct {
	Pattern  string `yaml:"pattern"`
	MaskWith string `yaml:"mask_with"`
}

// ProfilingSection holds profiling configuration.
type ProfilingSection struct {
	Enabled   bool `yaml:"enabled"`
	ReportSec int  `yaml:"report_sec"`
}

// DefaultConfig returns a Config with Python Drain3 defaults.
func DefaultConfig() *Config {
	return &Config{
		Drain: DrainSection{
			SimTh:                    0.4,
			Depth:                    4,
			MaxChildren:              100,
			MaxClusters:              0,
			ExtraDelimiters:          nil,
			ParamStr:                 DefaultParamStr,
			ParametrizeNumericTokens: true,
		},
		Snapshot: SnapshotSection{
			SnapshotIntervalMinutes: 5,
			CompressState:           true,
		},
		Masking: MaskingSection{
			MaskPrefix: "<",
			MaskSuffix: ">",
		},
		Profiling: ProfilingSection{
			Enabled:   false,
			ReportSec: 30,
		},
	}
}

// LoadConfig loads a Config from a YAML file.
// Missing fields are filled with default values.
func LoadConfig(filename string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for zero values that should have defaults
	if cfg.Drain.SimTh == 0 {
		cfg.Drain.SimTh = 0.4
	}
	if cfg.Drain.Depth == 0 {
		cfg.Drain.Depth = 4
	}
	if cfg.Drain.MaxChildren == 0 {
		cfg.Drain.MaxChildren = 100
	}
	if cfg.Drain.ParamStr == "" {
		cfg.Drain.ParamStr = DefaultParamStr
	}
	if cfg.Masking.MaskPrefix == "" {
		cfg.Masking.MaskPrefix = "<"
	}
	if cfg.Masking.MaskSuffix == "" {
		cfg.Masking.MaskSuffix = ">"
	}

	return cfg, nil
}
