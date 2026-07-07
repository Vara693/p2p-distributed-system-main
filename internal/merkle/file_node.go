package merkle

// NodeType is a small discriminator stored in the DAG node JSON.
type NodeType string

const (
	NodeTypeFile      NodeType = "file"
	NodeTypeDirectory NodeType = "directory"
)

// FileNode represents a file as an ordered list of chunk CIDs (hex strings).
type FileNode struct {
	Type   NodeType `json:"type"`
	Chunks []string `json:"chunks"`
}

func NewFileNode(chunks []string) FileNode {
	cp := append([]string(nil), chunks...)
	return FileNode{Type: NodeTypeFile, Chunks: cp}
}
