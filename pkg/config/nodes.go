package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	yaml "gopkg.in/yaml.v2"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func NodeConfigFromFile(path string) (*NodeConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NodeConfigFromBytes(data)
}

func NodeConfigFromBytes(data []byte) (*NodeConfig, error) {
	c := &NodeConfig{}

	reader := bytes.NewReader(data)
	decoder := k8syaml.NewYAMLToJSONDecoder(reader)
	err := decoder.Decode(c)

	if err != nil {
		return nil, err
	}

	// Validate node class names for uniquenes
	classNames := map[string]bool{}
	for _, class := range c.NodeClasses {
		name := strings.ToLower(class.Name)
		if _, exists := classNames[name]; exists {
			return nil, fmt.Errorf("node class name [%s] is not unique", name)
		}
		classNames[name] = true
	}

	return c, err
}

func (n *NodeConfig) AsYaml() (string, error) {
	bytes, err := yaml.Marshal(n)
	return string(bytes), err
}

type NodeConfig struct {
	NodeClasses []NodeClass `yaml:"nodeClasses"`
}

type NodeClass struct {
	Name      string
	Count     uint
	Labels    map[string]string
	Resources NodeResources
}

type NodeResources struct {
	Capacity    map[string]string
	Allocatable map[string]string
}
