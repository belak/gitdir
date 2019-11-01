package yaml

import (
	"bytes"

	yaml "gopkg.in/yaml.v3"
)

// Encode is a convenience function which will encode the given node to a byte
// slice.
func (n *Node) Encode() ([]byte, error) {
	// All this is really so we can set indentation to 2 spaces.
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)

	enc.SetIndent(2)

	err := enc.Encode(n.Node)

	return buf.Bytes(), err
}
