package drain3

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Profiler is an interface for tracking timing of sections in the mining pipeline.
type Profiler interface {
	StartSection(sectionName string)
	EndSection(sectionName string)
	Report(reset bool) string
}

// NullProfiler is a no-op profiler that does nothing.
type NullProfiler struct{}

func (NullProfiler) StartSection(string) {}
func (NullProfiler) EndSection(string)   {}
func (NullProfiler) Report(bool) string  { return "" }

// SimpleProfiler tracks cumulative time spent in named sections.
type SimpleProfiler struct {
	sections   map[string]*sectionStats
	startTimes map[string]time.Time
}

type sectionStats struct {
	totalTime time.Duration
	count     int
}

// NewSimpleProfiler creates a new SimpleProfiler.
func NewSimpleProfiler() *SimpleProfiler {
	return &SimpleProfiler{
		sections:   make(map[string]*sectionStats),
		startTimes: make(map[string]time.Time),
	}
}

// StartSection begins timing a named section.
func (p *SimpleProfiler) StartSection(sectionName string) {
	p.startTimes[sectionName] = time.Now()
}

// EndSection ends timing a named section and records the elapsed time.
func (p *SimpleProfiler) EndSection(sectionName string) {
	start, ok := p.startTimes[sectionName]
	if !ok {
		return
	}
	elapsed := time.Since(start)
	delete(p.startTimes, sectionName)

	stats, ok := p.sections[sectionName]
	if !ok {
		stats = &sectionStats{}
		p.sections[sectionName] = stats
	}
	stats.totalTime += elapsed
	stats.count++
}

// Report returns a formatted string with profiling results.
// If reset is true, all accumulated stats are cleared.
func (p *SimpleProfiler) Report(reset bool) string {
	if len(p.sections) == 0 {
		return "no profiling data"
	}

	// Sort sections by name for deterministic output
	names := make([]string, 0, len(p.sections))
	for name := range p.sections {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("Profiling Report:\n")
	for _, name := range names {
		stats := p.sections[name]
		avg := time.Duration(0)
		if stats.count > 0 {
			avg = stats.totalTime / time.Duration(stats.count)
		}
		sb.WriteString(fmt.Sprintf("  %-30s total=%-12s count=%-8d avg=%s\n",
			name, stats.totalTime, stats.count, avg))
	}

	if reset {
		p.sections = make(map[string]*sectionStats)
		p.startTimes = make(map[string]time.Time)
	}

	return sb.String()
}
