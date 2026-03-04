package drain3

import (
	"strings"
)

// LogCluster represents a discovered log template and the count of messages that match it.
type LogCluster struct {
	ClusterID         int      `json:"cluster_id"`
	LogTemplateTokens []string `json:"log_template_tokens"`
	Size              int      `json:"size"`
}

// NewLogCluster creates a new log cluster with the given template tokens and ID.
func NewLogCluster(logTemplateTokens []string, clusterID int) *LogCluster {
	return &LogCluster{
		ClusterID:         clusterID,
		LogTemplateTokens: logTemplateTokens,
		Size:              1,
	}
}

// GetTemplate returns the template as a single string with tokens joined by spaces.
func (lc *LogCluster) GetTemplate() string {
	return strings.Join(lc.LogTemplateTokens, " ")
}

// String returns a human-readable representation of the cluster.
func (lc *LogCluster) String() string {
	return "ID=" + intToStr(lc.ClusterID) + " : size=" + intToStr(lc.Size) + " : " + lc.GetTemplate()
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + intToStr(-i)
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
