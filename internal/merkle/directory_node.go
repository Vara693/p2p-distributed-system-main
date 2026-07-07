package merkle

import (
	"sort"
)

// DirEntry is used instead of map[string]string for deterministic JSON.
type DirEntry struct {
	Name string `json:"name"`
	CID  string `json:"cid"`
}

// DirectoryNode represents a directory as sorted named entries.
type DirectoryNode struct {
	Type    NodeType   `json:"type"`
	Entries []DirEntry `json:"entries"`
}

func NewDirectoryNode(entries []DirEntry) DirectoryNode {
	cp := append([]DirEntry(nil), entries...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Name < cp[j].Name })
	return DirectoryNode{Type: NodeTypeDirectory, Entries: cp}
}
