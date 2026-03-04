package drain3

import (
	"fmt"
	"testing"
)

func BenchmarkAddLogMessage(b *testing.B) {
	drain := NewDrain(DefaultDrainConfig())

	// Seed with some variety
	for i := 0; i < 100; i++ {
		drain.AddLogMessage(fmt.Sprintf("Failed password for user%d from 192.168.1.%d port 22 ssh2", i, i%256))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drain.AddLogMessage(fmt.Sprintf("Failed password for user%d from 10.0.0.%d port 22 ssh2", i%1000, i%256))
	}
}

func BenchmarkMatch(b *testing.B) {
	drain := NewDrain(DefaultDrainConfig())

	for i := 0; i < 100; i++ {
		drain.AddLogMessage(fmt.Sprintf("Failed password for user%d from 192.168.1.%d port 22 ssh2", i, i%256))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drain.Match(fmt.Sprintf("Failed password for user%d from 10.0.0.%d port 22 ssh2", i%1000, i%256), SearchStrategyNever)
	}
}

func BenchmarkTemplateMiner(b *testing.B) {
	tm, _ := NewTemplateMiner(nil, nil)

	for i := 0; i < 100; i++ {
		tm.AddLogMessage(fmt.Sprintf("Connection from 192.168.1.%d port %d", i%256, 1000+i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.AddLogMessage(fmt.Sprintf("Connection from 10.0.0.%d port %d", i%256, 2000+i))
	}
}
