package yaml

import (
	yaml "gopkg.in/yaml.v3"
)

// Node is a simple wrapper around the lower level yaml Node.
type Node struct {
	*yaml.Node
}

// NewMappingNode returns a new node pointing to a yaml map.
func NewMappingNode() *Node {
	return &Node{
		&yaml.Node{Kind: yaml.MappingNode},
	}
}

// NewSequenceNode returns a new node pointing to a yaml list.
func NewSequenceNode() *Node {
	return &Node{
		&yaml.Node{Kind: yaml.SequenceNode},
	}
}

// ScalarTag represents the type hints used by yaml. These come from the yaml
// source.
type ScalarTag string

// Each of these was taken from the yaml source. There are additional type
// hints, but they seem to be for specific non-scalar values.
const (
	ScalarTagString    ScalarTag = "!!str"
	ScalarTagBool      ScalarTag = "!!bool"
	ScalarTagInt       ScalarTag = "!!int"
	ScalarTagFloat     ScalarTag = "!!float"
	ScalarTagTimestamp ScalarTag = "!!timestamp"
	ScalarTagBinary    ScalarTag = "!!binary"
)

// NewScalarNode returns a new node pointing to a yaml value. To use the default
// type hinting, use the empty string as the tag.
func NewScalarNode(value string, tag ScalarTag) *Node {
	return &Node{
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: value,
			Tag:   string(tag),
		},
	}
}

// KeyIndex finds the index of the given key or -1 if not found. Note that it
// will only return an index if index + 1 also exists.
func (n *Node) KeyIndex(key string) int {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Kind == yaml.ScalarNode && n.Content[i].Value == key {
			return i
		}
	}

	return -1
}

// ValueNode returns the given value node for a key, or nil if not found.
func (n *Node) ValueNode(key string) *Node {
	if idx := n.KeyIndex(key); idx != -1 {
		return &Node{n.Content[idx+1]}
	}

	return nil
}
