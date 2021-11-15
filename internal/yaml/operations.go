package yaml

import (
	"errors"

	yaml "gopkg.in/yaml.v3"
)

// RemoveKey will remove a given key and value from a MappingNode.
func (n *Node) RemoveKey(key string) bool {
	idx := n.KeyIndex(key)
	if idx == -1 {
		return false
	}

	// Removing the index of the target node twice should drop the key and
	// value.
	n.Content = remove(n.Content, idx)
	n.Content = remove(n.Content, idx)

	return true
}

// EnsureOptions are optional settings when using Node.EnsureKey.
type EnsureOptions struct {
	// Comment lets you specify a comment for this node, if it's added.
	Comment string

	// Force will add the key if it doesn't exist and replace it if it does. If
	// Force is used, the comment will always be overridden.
	Force bool
}

// EnsureKey ensures that a given key exists. If it doesn't, it adds it with the
// value pointing to newNode.
func (n *Node) EnsureKey(key string, newNode *Node, opts *EnsureOptions) (*Node, bool) {
	if opts == nil {
		opts = &EnsureOptions{}
	}

	valNode := n.ValueNode(key)

	if valNode == nil {
		n.Content = append(
			n.Content,
			&yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       key,
				HeadComment: opts.Comment,
			},
			newNode.Node,
		)

		return newNode, true
	}

	if opts.Force {
		// Replace the node and update the comment
		*(valNode.Node) = *(newNode.Node)
		valNode.HeadComment = opts.Comment

		return valNode, true
	}

	return valNode, false
}

// AppendNode will append a value to a SequenceNode.
func (n *Node) AppendNode(newNode *Node) {
	n.Content = append(n.Content, newNode.Node)
}

// AppendNode will append a scalar to a SequenceNode if it does not already
// exist.
func (n *Node) AppendUniqueScalar(newNode *Node) bool {
	for _, iterNode := range n.Content {
		if iterNode.Kind != yaml.ScalarNode {
			continue
		}

		if iterNode.Value == newNode.Value {
			return false
		}
	}

	n.AppendNode(newNode)

	return true
}

// EnsureDocument takes data from a yaml file and ensures a basic document
// structure. It returns the root node, the root content node, or an error if
// the yaml document isn't in a valid format.
func EnsureDocument(data []byte) (*Node, *Node, error) {
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

	return &Node{rootNode}, &Node{targetNode}, nil
}
