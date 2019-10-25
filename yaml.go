package main

import (
	"errors"

	yaml "gopkg.in/yaml.v3"
)

func yamlLookupKey(n *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Kind == yaml.ScalarNode && n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}

	return nil
}

func yamlEnsureKey(targetNode *yaml.Node, key string, newNode *yaml.Node) {
	valNode := yamlLookupKey(targetNode, key)

	if valNode == nil {
		targetNode.Content = append(
			targetNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: key,
			},
			newNode,
		)
	} else {
		*valNode = *newNode
	}
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
