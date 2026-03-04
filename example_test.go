package drain3_test

import (
	"fmt"
	"sort"

	"github.com/kloudmate/drain3"
)

func ExampleNew() {
	// Create with defaults
	tm, err := drain3.New()
	if err != nil {
		panic(err)
	}

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
	clusters := tm.Clusters()
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].ClusterID < clusters[j].ClusterID
	})
	for _, c := range clusters {
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

func ExampleNew_withOptions() {
	// Create with custom options
	tm, err := drain3.New(
		drain3.WithSimTh(0.5),
		drain3.WithMasking(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP"),
		drain3.WithMasking(`\b\d+\b`, "NUM"),
	)
	if err != nil {
		panic(err)
	}

	messages := []string{
		"connection from 192.168.1.1 port 22",
		"connection from 10.0.0.1 port 8080",
		"disconnect from 192.168.1.1 port 443",
	}

	for _, msg := range messages {
		result := tm.AddLogMessage(msg)
		fmt.Printf("template: %s\n", result.Cluster.GetTemplate())
	}
	// Output:
	// template: connection from <IP> port <NUM>
	// template: connection from <IP> port <NUM>
	// template: disconnect from <IP> port <NUM>
}
