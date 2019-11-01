package yaml

import (
	yaml "gopkg.in/yaml.v3"
)

func remove(slice []*yaml.Node, s int) []*yaml.Node {
	return append(slice[:s], slice[s+1:]...)
}
