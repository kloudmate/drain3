package drain3

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAddLogMessageSimple(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	cluster, change := drain.AddLogMessage("hello world")
	if change != ChangeClusterCreated {
		t.Fatalf("expected ChangeClusterCreated, got %v", change)
	}
	if cluster.GetTemplate() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", cluster.GetTemplate())
	}
	if cluster.Size != 1 {
		t.Fatalf("expected size 1, got %d", cluster.Size)
	}

	// Same message again — should match, no change
	cluster2, change2 := drain.AddLogMessage("hello world")
	if change2 != ChangeNone {
		t.Fatalf("expected ChangeNone, got %v", change2)
	}
	if cluster2.ClusterID != cluster.ClusterID {
		t.Fatalf("expected same cluster ID")
	}
	if cluster2.Size != 2 {
		t.Fatalf("expected size 2, got %d", cluster2.Size)
	}
}

func TestAddLogMessageTemplateChange(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	c1, _ := drain.AddLogMessage("user alice logged in")
	if c1.GetTemplate() != "user alice logged in" {
		t.Fatalf("unexpected template: %q", c1.GetTemplate())
	}

	c2, change := drain.AddLogMessage("user bob logged in")
	if c2.ClusterID != c1.ClusterID {
		t.Fatalf("expected same cluster")
	}
	if change != ChangeClusterTemplateChanged {
		t.Fatalf("expected ChangeClusterTemplateChanged, got %v", change)
	}
	if c2.GetTemplate() != "user <*> logged in" {
		t.Fatalf("expected 'user <*> logged in', got %q", c2.GetTemplate())
	}
}

func TestAddLogMessageSSH(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	messages := []string{
		"Failed password for invalid user admin from 192.168.1.1 port 22 ssh2",
		"Failed password for invalid user root from 10.0.0.1 port 22 ssh2",
		"Failed password for invalid user test from 172.16.0.1 port 22 ssh2",
		"Accepted password for admin from 192.168.1.1 port 22 ssh2",
		"Accepted password for admin from 10.0.0.1 port 22 ssh2",
	}

	for _, msg := range messages {
		drain.AddLogMessage(msg)
	}

	clusters := drain.Clusters()
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// Find the "Failed" cluster
	for _, c := range clusters {
		tmpl := c.GetTemplate()
		if strings.HasPrefix(tmpl, "Failed") {
			if c.Size != 3 {
				t.Fatalf("expected Failed cluster size 3, got %d", c.Size)
			}
			if !strings.Contains(tmpl, "<*>") {
				t.Fatalf("expected wildcards in template: %q", tmpl)
			}
		}
	}
}

func TestSimThVariation(t *testing.T) {
	// High sim_th should create separate clusters for different messages
	cfg := DefaultDrainConfig()
	cfg.SimTh = 0.9

	drain := NewDrain(cfg)

	drain.AddLogMessage("request from user alice")
	drain.AddLogMessage("request from user bob")
	drain.AddLogMessage("response to user alice")
	drain.AddLogMessage("response to user bob")

	// With high sim_th, "request" and "response" should be different clusters
	count := drain.ClusterCount()
	if count < 2 {
		t.Fatalf("expected at least 2 clusters with high sim_th, got %d", count)
	}
}

func TestMaxClustersEviction(t *testing.T) {
	cfg := DefaultDrainConfig()
	cfg.MaxClusters = 2

	drain := NewDrain(cfg)

	// Create 3 clusters with distinct templates (different token counts to force new clusters)
	drain.AddLogMessage("a")
	drain.AddLogMessage("b c")
	drain.AddLogMessage("d e f")

	// Should have at most 2 clusters
	if drain.ClusterCount() > 2 {
		t.Fatalf("expected at most 2 clusters, got %d", drain.ClusterCount())
	}
}

func TestMatchOnly(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	drain.AddLogMessage("user alice logged in")
	drain.AddLogMessage("user bob logged in")

	// Match should find the cluster without modifying it
	cluster := drain.Match("user charlie logged in", SearchStrategyNever)
	if cluster == nil {
		t.Fatal("expected a match")
	}
	if cluster.GetTemplate() != "user <*> logged in" {
		t.Fatalf("unexpected template: %q", cluster.GetTemplate())
	}
	// Size should still be 2 (Match doesn't increment)
	if cluster.Size != 2 {
		t.Fatalf("expected size 2 after match, got %d", cluster.Size)
	}
}

func TestMatchFallbackStrategy(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	drain.AddLogMessage("hello world")

	// Fallback: try tree search first, then full search
	cluster := drain.Match("hello world", SearchStrategyFallback)
	if cluster == nil {
		t.Fatal("expected a match with fallback strategy")
	}
}

func TestMatchAlwaysStrategy(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	drain.AddLogMessage("hello world")

	// Always: always do full search
	cluster := drain.Match("hello world", SearchStrategyAlways)
	if cluster == nil {
		t.Fatal("expected a match with always strategy")
	}
}

func TestEmptyLogMessage(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	cluster, change := drain.AddLogMessage("")
	if change != ChangeClusterCreated {
		t.Fatalf("expected ChangeClusterCreated, got %v", change)
	}
	if cluster.GetTemplate() != "" {
		t.Fatalf("expected empty template, got %q", cluster.GetTemplate())
	}
}

func TestExtraDelimiters(t *testing.T) {
	cfg := DefaultDrainConfig()
	cfg.ExtraDelimiters = []string{"=", ":"}

	drain := NewDrain(cfg)

	cluster, _ := drain.AddLogMessage("key=value:pair")
	// "key=value:pair" becomes "key value pair"
	if cluster.GetTemplate() != "key value pair" {
		t.Fatalf("expected 'key value pair', got %q", cluster.GetTemplate())
	}
}

func TestParametrizeNumericTokens(t *testing.T) {
	cfg := DefaultDrainConfig()
	cfg.ParametrizeNumericTokens = true

	drain := NewDrain(cfg)

	drain.AddLogMessage("error code 123 in module A")
	drain.AddLogMessage("error code 456 in module B")

	clusters := drain.Clusters()
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
}

func TestCreateTemplate(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	tmpl := drain.createTemplate(
		[]string{"hello", "world"},
		[]string{"hello", "earth"},
	)
	if !tokensEqual(tmpl, []string{"hello", "<*>"}) {
		t.Fatalf("expected [hello <*>], got %v", tmpl)
	}
}

func TestGetSeqDistance(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	// Exact match
	sim, paramCount := drain.getSeqDistance(
		[]string{"hello", "world"},
		[]string{"hello", "world"},
		false,
	)
	if sim != 1.0 {
		t.Fatalf("expected sim 1.0, got %f", sim)
	}
	if paramCount != 0 {
		t.Fatalf("expected paramCount 0, got %d", paramCount)
	}

	// Half match
	sim, _ = drain.getSeqDistance(
		[]string{"hello", "world"},
		[]string{"hello", "earth"},
		false,
	)
	if sim != 0.5 {
		t.Fatalf("expected sim 0.5, got %f", sim)
	}

	// Template with param
	sim, paramCount = drain.getSeqDistance(
		[]string{"hello", "<*>"},
		[]string{"hello", "earth"},
		true,
	)
	if sim != 1.0 {
		t.Fatalf("expected sim 1.0 with includeParams, got %f", sim)
	}
	if paramCount != 1 {
		t.Fatalf("expected paramCount 1, got %d", paramCount)
	}

	// Different lengths
	sim, _ = drain.getSeqDistance(
		[]string{"hello"},
		[]string{"hello", "world"},
		false,
	)
	if sim != 0 {
		t.Fatalf("expected sim 0 for different lengths, got %f", sim)
	}
}

func TestDrainJSONRoundTrip(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	drain.AddLogMessage("user alice logged in")
	drain.AddLogMessage("user bob logged in")
	drain.AddLogMessage("server started on port 8080")

	data, err := json.Marshal(drain)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	drain2 := NewDrain(DefaultDrainConfig())
	if err := drain2.UnmarshalState(data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if drain2.ClusterCount() != drain.ClusterCount() {
		t.Fatalf("cluster count mismatch: %d != %d", drain2.ClusterCount(), drain.ClusterCount())
	}

	// Verify a match still works
	cluster := drain2.Match("user charlie logged in", SearchStrategyNever)
	if cluster == nil {
		t.Fatal("expected match after restore")
	}
	if cluster.GetTemplate() != "user <*> logged in" {
		t.Fatalf("unexpected template after restore: %q", cluster.GetTemplate())
	}
}

func TestDrainConcurrency(t *testing.T) {
	drain := NewDrain(DefaultDrainConfig())

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				drain.AddLogMessage("message from goroutine " + intToStr(n) + " iter " + intToStr(j))
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no panic/race
	count := drain.ClusterCount()
	if count == 0 {
		t.Fatal("expected some clusters")
	}
}

func TestHasNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", false},
		{"123", true},
		{"abc123", true},
		{"", false},
		{"192.168.1.1", true},
	}

	for _, tt := range tests {
		got := hasNumbers(tt.input)
		if got != tt.expected {
			t.Errorf("hasNumbers(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestLogClusterString(t *testing.T) {
	c := NewLogCluster([]string{"hello", "world"}, 42)
	s := c.String()
	if !strings.Contains(s, "42") || !strings.Contains(s, "hello world") {
		t.Fatalf("unexpected String(): %q", s)
	}
}
