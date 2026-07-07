package merkle

import (
	"bytes"
	"encoding/json"
	"errors"
)

// EncodeCanonicalJSON encodes v into canonical JSON bytes for hashing.
func EncodeCanonicalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var tmp any
	if err := json.Unmarshal(b, &tmp); err != nil {
		return nil, err
	}
	return json.Marshal(tmp)
}

// DetectType attempts to detect the node type from stored block bytes.
func DetectType(block []byte) (NodeType, error) {
	block = bytes.TrimSpace(block)
	if len(block) == 0 || block[0] != '{' {
		return "", errors.New("not a JSON dag node")
	}
	var hdr struct {
		Type NodeType `json:"type"`
	}
	if err := json.Unmarshal(block, &hdr); err != nil {
		return "", err
	}
	switch hdr.Type {
	case NodeTypeFile, NodeTypeDirectory:
		return hdr.Type, nil
	default:
		return "", errors.New("unknown dag node type")
	}
}
