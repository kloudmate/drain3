package drain3

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"sync"
	"time"
)

// AddLogMessageResult is the result of processing a log message.
type AddLogMessageResult struct {
	Cluster    *LogCluster
	ChangeType ChangeType
}

// TemplateMiner is the high-level API that integrates Drain, masking, persistence, and profiling.
type TemplateMiner struct {
	mu sync.Mutex

	Drain       *Drain
	Config      *Config
	Persistence PersistenceHandler
	Profiler    Profiler
	Masker      *LogMasker

	lastSaveTime time.Time
}

// NewTemplateMiner creates a new TemplateMiner with the given persistence handler and config.
// If config is nil, DefaultConfig() is used. If persistence is nil, no persistence is performed.
func NewTemplateMiner(persistence PersistenceHandler, config *Config) (*TemplateMiner, error) {
	if config == nil {
		config = DefaultConfig()
	}

	drainCfg := DrainConfig{
		SimTh:                    config.Drain.SimTh,
		Depth:                    config.Drain.Depth,
		MaxChildren:              config.Drain.MaxChildren,
		MaxClusters:              config.Drain.MaxClusters,
		ExtraDelimiters:          config.Drain.ExtraDelimiters,
		ParamStr:                 config.Drain.ParamStr,
		ParametrizeNumericTokens: config.Drain.ParametrizeNumericTokens,
	}

	drain := NewDrain(drainCfg)

	// Build masking instructions
	var instructions []*MaskingInstruction
	for _, ic := range config.Masking.Instructions {
		inst, err := NewMaskingInstruction(ic.Pattern, ic.MaskWith)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, inst)
	}

	masker := NewLogMasker(instructions, config.Masking.MaskPrefix, config.Masking.MaskSuffix)

	var profiler Profiler
	if config.Profiling.Enabled {
		profiler = NewSimpleProfiler()
	} else {
		profiler = NullProfiler{}
	}

	tm := &TemplateMiner{
		Drain:        drain,
		Config:       config,
		Persistence:  persistence,
		Profiler:     profiler,
		Masker:       masker,
		lastSaveTime: time.Now(),
	}

	// Load saved state if available
	if persistence != nil {
		if err := tm.LoadState(); err != nil {
			// Non-fatal: just start fresh
			_ = err
		}
	}

	return tm, nil
}

// AddLogMessage processes a log message through the masking and Drain pipeline.
func (tm *TemplateMiner) AddLogMessage(message string) *AddLogMessageResult {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.Profiler.StartSection("masking")
	maskedContent := tm.Masker.Mask(message)
	tm.Profiler.EndSection("masking")

	tm.Profiler.StartSection("drain")
	cluster, changeType := tm.Drain.AddLogMessage(maskedContent)
	tm.Profiler.EndSection("drain")

	// Auto-save on snapshot interval
	if tm.Persistence != nil && changeType != ChangeNone {
		sinceLastSave := time.Since(tm.lastSaveTime)
		interval := time.Duration(tm.Config.Snapshot.SnapshotIntervalMinutes) * time.Minute
		if sinceLastSave >= interval {
			tm.Profiler.StartSection("save_state")
			_ = tm.saveStateInternal()
			tm.Profiler.EndSection("save_state")
		}
	}

	return &AddLogMessageResult{
		Cluster:    cluster,
		ChangeType: changeType,
	}
}

// Match finds the best matching cluster for a log message without modifying state.
func (tm *TemplateMiner) Match(message string, strategy SearchStrategy) *LogCluster {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	maskedContent := tm.Masker.Mask(message)
	return tm.Drain.Match(maskedContent, strategy)
}

// SaveState persists the current Drain state.
func (tm *TemplateMiner) SaveState() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.saveStateInternal()
}

func (tm *TemplateMiner) saveStateInternal() error {
	if tm.Persistence == nil {
		return nil
	}

	data, err := json.Marshal(tm.Drain)
	if err != nil {
		return err
	}

	if tm.Config.Snapshot.CompressState {
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		data = buf.Bytes()
	}

	if err := tm.Persistence.SaveState(data); err != nil {
		return err
	}
	tm.lastSaveTime = time.Now()
	return nil
}

// LoadState restores Drain state from persistence.
func (tm *TemplateMiner) LoadState() error {
	if tm.Persistence == nil {
		return nil
	}

	data, err := tm.Persistence.LoadState()
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	// Try to decompress (zlib)
	if tm.Config.Snapshot.CompressState {
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err == nil {
			decompressed, err := io.ReadAll(r)
			r.Close()
			if err == nil {
				data = decompressed
			}
		}
	}

	return tm.Drain.UnmarshalState(data)
}

// Clusters returns all current log clusters.
func (tm *TemplateMiner) Clusters() []*LogCluster {
	return tm.Drain.Clusters()
}

// ClusterCount returns the number of active clusters.
func (tm *TemplateMiner) ClusterCount() int {
	return tm.Drain.ClusterCount()
}

// GetProfilerReport returns the profiler report string.
func (tm *TemplateMiner) GetProfilerReport(reset bool) string {
	return tm.Profiler.Report(reset)
}
