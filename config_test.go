package drain3

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Drain.GetSimTh() != 0.4 {
		t.Fatalf("expected SimTh 0.4, got %f", cfg.Drain.GetSimTh())
	}
	if cfg.Drain.Depth != 4 {
		t.Fatalf("expected Depth 4, got %d", cfg.Drain.Depth)
	}
	if cfg.Drain.MaxChildren != 100 {
		t.Fatalf("expected MaxChildren 100, got %d", cfg.Drain.MaxChildren)
	}
	if cfg.Drain.MaxClusters != 0 {
		t.Fatalf("expected MaxClusters 0, got %d", cfg.Drain.MaxClusters)
	}
	if !cfg.Drain.GetParametrizeNumericTokens() {
		t.Fatal("expected ParametrizeNumericTokens true")
	}
	if cfg.Snapshot.SnapshotIntervalMinutes != 5 {
		t.Fatalf("expected SnapshotIntervalMinutes 5, got %d", cfg.Snapshot.SnapshotIntervalMinutes)
	}
	if !cfg.Snapshot.CompressState {
		t.Fatal("expected CompressState true")
	}
}

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/drain3_test.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Drain.GetSimTh() != 0.5 {
		t.Fatalf("expected SimTh 0.5, got %f", cfg.Drain.GetSimTh())
	}
	if cfg.Drain.Depth != 5 {
		t.Fatalf("expected Depth 5, got %d", cfg.Drain.Depth)
	}
	if cfg.Drain.MaxChildren != 50 {
		t.Fatalf("expected MaxChildren 50, got %d", cfg.Drain.MaxChildren)
	}
	if cfg.Drain.MaxClusters != 1000 {
		t.Fatalf("expected MaxClusters 1000, got %d", cfg.Drain.MaxClusters)
	}
	if len(cfg.Drain.ExtraDelimiters) != 2 {
		t.Fatalf("expected 2 extra delimiters, got %d", len(cfg.Drain.ExtraDelimiters))
	}
	if cfg.Drain.ExtraDelimiters[0] != "=" || cfg.Drain.ExtraDelimiters[1] != ":" {
		t.Fatalf("unexpected extra delimiters: %v", cfg.Drain.ExtraDelimiters)
	}

	if cfg.Snapshot.SnapshotIntervalMinutes != 10 {
		t.Fatalf("expected SnapshotIntervalMinutes 10, got %d", cfg.Snapshot.SnapshotIntervalMinutes)
	}
	if cfg.Snapshot.CompressState {
		t.Fatal("expected CompressState false")
	}

	if len(cfg.Masking.Instructions) != 2 {
		t.Fatalf("expected 2 masking instructions, got %d", len(cfg.Masking.Instructions))
	}

	if !cfg.Profiling.Enabled {
		t.Fatal("expected Profiling.Enabled true")
	}
	if cfg.Profiling.ReportSec != 60 {
		t.Fatalf("expected ReportSec 60, got %d", cfg.Profiling.ReportSec)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfigSimThZero(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "drain3.yaml")
	yamlData := []byte("drain:\n  sim_th: 0\n")
	if err := os.WriteFile(cfgPath, yamlData, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Drain.GetSimTh() != 0 {
		t.Fatalf("expected SimTh 0.0, got %f", cfg.Drain.GetSimTh())
	}
}
