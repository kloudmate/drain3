package drain3_test

import (
	"fmt"

	"github.com/kloudmate/drain3"
)

func ExampleTemplateMiner() {
	// Create a template miner with default config
	tm, err := drain3.NewTemplateMiner(nil, nil)
	if err != nil {
		panic(err)
	}

	// Process sample SSH log messages
	messages := []string{
		"Failed password for invalid user admin from 192.168.1.1 port 22 ssh2",
		"Failed password for invalid user root from 10.0.0.1 port 22 ssh2",
		"Failed password for invalid user test from 172.16.0.1 port 22 ssh2",
		"Accepted password for admin from 192.168.1.1 port 22 ssh2",
		"Accepted password for admin from 10.0.0.1 port 22 ssh2",
	}

	for _, msg := range messages {
		result := tm.AddLogMessage(msg)
		fmt.Printf("change=%s cluster=%s\n", result.ChangeType, result.Cluster.GetTemplate())
	}

	fmt.Printf("\nDiscovered %d templates:\n", tm.ClusterCount())
	for _, c := range tm.Clusters() {
		fmt.Printf("  [size=%d] %s\n", c.Size, c.GetTemplate())
	}
	// Output:
	// change=cluster_created cluster=Failed password for invalid user admin from 192.168.1.1 port 22 ssh2
	// change=cluster_template_changed cluster=Failed password for invalid user <*> from <*> port 22 ssh2
	// change=none cluster=Failed password for invalid user <*> from <*> port 22 ssh2
	// change=cluster_created cluster=Accepted password for admin from 192.168.1.1 port 22 ssh2
	// change=cluster_template_changed cluster=Accepted password for admin from <*> port 22 ssh2
	//
	// Discovered 2 templates:
	//   [size=3] Failed password for invalid user <*> from <*> port 22 ssh2
	//   [size=2] Accepted password for admin from <*> port 22 ssh2
}
