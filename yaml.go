package main

import (
	"bytes"
	"errors"

	yaml "gopkg.in/yaml.v3"
)

func remove(slice []*yaml.Node, s int) []*yaml.Node {
	return append(slice[:s], slice[s+1:]...)
}

func yamlRemoveKey(targetNode *yaml.Node, key string) bool {
	idx := yamlLookupKeyValueIndex(targetNode, key)
	if idx == -1 {
		return false
	}

	// Removing the index of the target node twice should drop the key and
	// value.
	targetNode.Content = remove(targetNode.Content, idx)
	targetNode.Content = remove(targetNode.Content, idx)

	return true
}

func yamlEncode(rootNode *yaml.Node) ([]byte, error) {
	// All this nonsense is really so we can set indentation to 2 spaces.
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)

	enc.SetIndent(2)

	err := enc.Encode(rootNode)

	return buf.Bytes(), err
}

func yamlLookupKeyValueIndex(n *yaml.Node, key string) int {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Kind == yaml.ScalarNode && n.Content[i].Value == key {
			return i
		}
	}

	return -1
}

func yamlLookupVal(n *yaml.Node, key string) *yaml.Node {
	idx := yamlLookupKeyValueIndex(n, key)
	if idx == -1 {
		return nil
	}

	return n.Content[idx+1]
}

// TODO: clean up the options here
func yamlEnsureKey(
	targetNode *yaml.Node,
	key string,
	newNode *yaml.Node,
	comment string,
	force bool,
) (*yaml.Node, bool) {
	valNode := yamlLookupVal(targetNode, key)

	if valNode == nil {
		// Set valNode so we can return properly
		valNode = newNode

		// Add the key and value to the mapping node.
		targetNode.Content = append(
			targetNode.Content,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       key,
				HeadComment: comment,
			},
			valNode,
		)

		return valNode, true
	} else if force {
		// Replace the node and update the comment
		*valNode = *newNode
		valNode.HeadComment = comment
	}

	return valNode, false
}

func yamlEnsureDocument(data []byte) (*yaml.Node, *yaml.Node, error) {
	rootNode := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	// We explicitly ignore this error so we can manually make a tree
	_ = yaml.Unmarshal(data, rootNode)

	if len(rootNode.Content) == 0 {
		rootNode.Content = append(rootNode.Content, &yaml.Node{
			Kind: yaml.MappingNode,
		})
	}

	if len(rootNode.Content) != 1 || rootNode.Content[0].Kind != yaml.MappingNode {
		return nil, nil, errors.New("root is not a valid yaml document")
	}

	targetNode := rootNode.Content[0]

	return rootNode, targetNode, nil
}
