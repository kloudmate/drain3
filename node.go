package drain3

// Node represents a node in the Drain prefix tree.
// Internal nodes hold child pointers keyed by token string.
// Leaf nodes hold a list of cluster IDs for fast matching.
type Node struct {
	KeyToChildNode map[string]*Node `json:"key_to_child_node"`
	ClusterIDs     []int            `json:"cluster_ids"`
}

// NewNode creates a new empty prefix tree node.
func NewNode() *Node {
	return &Node{
		KeyToChildNode: make(map[string]*Node),
	}
}
