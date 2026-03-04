package drain3

import (
	"testing"
)

func TestTemplateMinerBasic(t *testing.T) {
	tm, err := NewTemplateMiner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	result := tm.AddLogMessage("user alice logged in")
	if result.ChangeType != ChangeClusterCreated {
		t.Fatalf("expected ChangeClusterCreated, got %v", result.ChangeType)
	}

	result = tm.AddLogMessage("user bob logged in")
	if result.ChangeType != ChangeClusterTemplateChanged {
		t.Fatalf("expected ChangeClusterTemplateChanged, got %v", result.ChangeType)
	}
	if result.Cluster.GetTemplate() != "user <*> logged in" {
		t.Fatalf("unexpected template: %q", result.Cluster.GetTemplate())
	}

	result = tm.AddLogMessage("user charlie logged in")
	if result.ChangeType != ChangeNone {
		t.Fatalf("expected ChangeNone, got %v", result.ChangeType)
	}
	if result.Cluster.Size != 3 {
		t.Fatalf("expected size 3, got %d", result.Cluster.Size)
	}
}

func TestTemplateMinerWithMasking(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Masking.Instructions = []MaskingInstructionConfig{
		{Pattern: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, MaskWith: "IP"},
	}

	tm, err := NewTemplateMiner(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}

	result := tm.AddLogMessage("connection from 192.168.1.1")
	if result.Cluster.GetTemplate() != "connection from <IP>" {
		t.Fatalf("expected 'connection from <IP>', got %q", result.Cluster.GetTemplate())
	}

	result = tm.AddLogMessage("connection from 10.0.0.1")
	if result.ChangeType != ChangeNone {
		t.Fatalf("expected ChangeNone (masked to same), got %v", result.ChangeType)
	}
}

func TestTemplateMinerPersistence(t *testing.T) {
	persistence := NewMemoryPersistence()
	cfg := DefaultConfig()
	cfg.Snapshot.CompressState = false

	tm, err := NewTemplateMiner(persistence, cfg)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("user bob logged in")
	tm.AddLogMessage("server started on port 8080")

	// Save state
	if err := tm.SaveState(); err != nil {
		t.Fatal(err)
	}

	// Create new miner and load state
	tm2, err := NewTemplateMiner(persistence, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if tm2.ClusterCount() != tm.ClusterCount() {
		t.Fatalf("cluster count mismatch: %d != %d", tm2.ClusterCount(), tm.ClusterCount())
	}

	cluster := tm2.Match("user charlie logged in", SearchStrategyNever)
	if cluster == nil {
		t.Fatal("expected match after restore")
	}
	if cluster.GetTemplate() != "user <*> logged in" {
		t.Fatalf("unexpected template after restore: %q", cluster.GetTemplate())
	}
}

func TestTemplateMinerPersistenceCompressed(t *testing.T) {
	persistence := NewMemoryPersistence()
	cfg := DefaultConfig()
	cfg.Snapshot.CompressState = true

	tm, err := NewTemplateMiner(persistence, cfg)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("user bob logged in")

	if err := tm.SaveState(); err != nil {
		t.Fatal(err)
	}

	tm2, err := NewTemplateMiner(persistence, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if tm2.ClusterCount() != tm.ClusterCount() {
		t.Fatalf("cluster count mismatch: %d != %d", tm2.ClusterCount(), tm.ClusterCount())
	}
}

func TestTemplateMinerMatch(t *testing.T) {
	tm, err := NewTemplateMiner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("user bob logged in")

	cluster := tm.Match("user charlie logged in", SearchStrategyNever)
	if cluster == nil {
		t.Fatal("expected match")
	}
	if cluster.GetTemplate() != "user <*> logged in" {
		t.Fatalf("unexpected template: %q", cluster.GetTemplate())
	}

	// No match for very different message
	cluster = tm.Match("something completely different here today", SearchStrategyNever)
	if cluster != nil {
		t.Fatalf("expected no match, got %q", cluster.GetTemplate())
	}
}

func TestTemplateMinerMatchFallback(t *testing.T) {
	tm, err := NewTemplateMiner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("hello world")

	cluster := tm.Match("hello world", SearchStrategyFallback)
	if cluster == nil {
		t.Fatal("expected match with fallback")
	}
}

func TestTemplateMinerFromConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/drain3_test.yaml")
	if err != nil {
		t.Fatal(err)
	}

	tm, err := NewTemplateMiner(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// ExtraDelimiters should be applied
	result := tm.AddLogMessage("key=value logged")
	if result.Cluster.GetTemplate() != "key value logged" {
		t.Fatalf("expected 'key value logged', got %q", result.Cluster.GetTemplate())
	}
}

func TestExtractParameters(t *testing.T) {
	tm, err := NewTemplateMiner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("user bob logged in")

	pe := NewParameterExtractor(tm.Masker, tm.Config.Drain.ExtraDelimiters)

	params := pe.ExtractParameters("user <*> logged in", "user charlie logged in", false)
	if params == nil {
		t.Fatal("expected params")
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].Value != "charlie" {
		t.Fatalf("expected 'charlie', got %q", params[0].Value)
	}
	if params[0].MaskName != "*" {
		t.Fatalf("expected mask name '*', got %q", params[0].MaskName)
	}
}

func TestExtractParametersNoMatch(t *testing.T) {
	pe := NewParameterExtractor(nil, nil)

	params := pe.ExtractParameters("user <*> logged in", "something completely different", false)
	if params != nil {
		t.Fatalf("expected nil for no match, got %v", params)
	}
}

func TestExtractParametersMultiple(t *testing.T) {
	pe := NewParameterExtractor(nil, nil)

	params := pe.ExtractParameters("<*> connected to <*>", "alice connected to server1", false)
	if params == nil {
		t.Fatal("expected params")
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0].Value != "alice" {
		t.Fatalf("expected 'alice', got %q", params[0].Value)
	}
	if params[1].Value != "server1" {
		t.Fatalf("expected 'server1', got %q", params[1].Value)
	}
}

func TestTemplateMinerClusters(t *testing.T) {
	tm, err := NewTemplateMiner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("server started")

	clusters := tm.Clusters()
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
}

func TestExtractParametersExactMatching(t *testing.T) {
	inst, err := NewMaskingInstruction(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP")
	if err != nil {
		t.Fatal(err)
	}
	masker := NewLogMasker([]*MaskingInstruction{inst}, "<", ">")
	pe := NewParameterExtractor(masker, nil)

	params := pe.ExtractParameters("connection from <IP> port <*>", "connection from 192.168.1.1 port 22", true)
	if params == nil {
		t.Fatal("expected params with exact matching")
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	// First param should be the IP
	foundIP := false
	for _, p := range params {
		if p.MaskName == "IP" && p.Value == "192.168.1.1" {
			foundIP = true
		}
	}
	if !foundIP {
		t.Fatalf("expected to find IP=192.168.1.1, got %v", params)
	}
}

func TestExtraDelimitersRegex(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Drain.ExtraDelimiters = []string{`[\[\]]`} // regex pattern for [ and ]

	tm, err := NewTemplateMiner(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}

	result := tm.AddLogMessage("[INFO] server started")
	if result.Cluster.GetTemplate() != "INFO server started" {
		t.Fatalf("expected 'INFO server started', got %q", result.Cluster.GetTemplate())
	}
}

func TestExtractParametersWithRegexExtraDelimiters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Drain.ExtraDelimiters = []string{`[\[\]]`}
	cfg.Drain.SimTh = 0.3

	tm, err := NewTemplateMiner(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("status=[200] path=/a")
	result := tm.AddLogMessage("status=[500] path=/a")
	if result.ChangeType != ChangeClusterTemplateChanged {
		t.Fatalf("expected template change, got %v", result.ChangeType)
	}

	pe := NewParameterExtractor(tm.Masker, tm.Config.Drain.ExtraDelimiters)
	params := pe.ExtractParameters(result.Cluster.GetTemplate(), "status=[404] path=/a", false)
	if params == nil {
		t.Fatal("expected extracted params")
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(params))
	}
	if params[0].Value != "404" {
		t.Fatalf("expected param value '404', got %q", params[0].Value)
	}
}

func TestParamStrDerivedFromMaskPrefixSuffix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Masking.MaskPrefix = "{{"
	cfg.Masking.MaskSuffix = "}}"

	tm, err := NewTemplateMiner(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// param_str should be {{*}} derived from mask prefix/suffix
	if tm.Drain.ParamStr != "{{*}}" {
		t.Fatalf("expected ParamStr '{{*}}', got %q", tm.Drain.ParamStr)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("user bob logged in")

	cluster := tm.Clusters()[0]
	if cluster.GetTemplate() != "user {{*}} logged in" {
		t.Fatalf("expected 'user {{*}} logged in', got %q", cluster.GetTemplate())
	}
}

func TestGetTotalClusterSize(t *testing.T) {
	tm, err := NewTemplateMiner(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("user alice logged in")
	tm.AddLogMessage("user bob logged in")
	tm.AddLogMessage("server started")

	total := tm.Drain.GetTotalClusterSize()
	if total != 3 {
		t.Fatalf("expected total size 3, got %d", total)
	}
}

func TestTemplateMinerProfiler(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Profiling.Enabled = true

	tm, err := NewTemplateMiner(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}

	tm.AddLogMessage("test message")

	report := tm.GetProfilerReport(false)
	if report == "" {
		t.Fatal("expected non-empty profiler report")
	}
}

type brokenPersistence struct{}

func (b *brokenPersistence) SaveState([]byte) error { return nil }
func (b *brokenPersistence) LoadState() ([]byte, error) {
	return nil, errBrokenPersistence
}

var errBrokenPersistence = &brokenPersistenceError{}

type brokenPersistenceError struct{}

func (e *brokenPersistenceError) Error() string { return "load failed" }

func TestTemplateMinerLoadStateErrorIsReturned(t *testing.T) {
	_, err := NewTemplateMiner(&brokenPersistence{}, DefaultConfig())
	if err == nil {
		t.Fatal("expected load-state error from NewTemplateMiner")
	}
}

func TestTemplateMinerLoadStateErrorClearsAndContinues(t *testing.T) {
	persistence := NewMemoryPersistence()
	persistence.data = []byte("invalid-state-data")

	tm, err := NewTemplateMiner(persistence, DefaultConfig())
	if err != nil {
		t.Fatalf("expected recovery and continue, got error: %v", err)
	}
	if tm == nil {
		t.Fatal("expected non-nil miner after recovery")
	}
	if persistence.data != nil {
		t.Fatal("expected corrupted state to be cleared")
	}

	result := tm.AddLogMessage("hello world")
	if result == nil || result.Cluster == nil {
		t.Fatal("expected miner to keep working after recovery")
	}
}
