package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"github.com/cristim/ec2-instances-info"

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

	ec2, err := Ec2Instances()
	if err != nil {
		return nil, err
	}

	// Validate node class names for uniquenes
	// Fill in standard EC2 instance resource details if not provided
	classNames := map[string]bool{}
	for i, _ := range c.NodeClasses {
		name := strings.ToLower(c.NodeClasses[i].Name)
		if _, exists := classNames[name]; exists {
			return nil, fmt.Errorf("node class name [%s] is not unique", name)
		}
		classNames[name] = true
		if c.NodeClasses[i].Resources.Capacity == nil {
			if _, exists := ec2[name]; exists {
				c.NodeClasses[i].Resources = ec2[name]
				c.NodeClasses[i].Labels = map[string]string {
					"nodus-ponens": "true",
					"np.class": name,
				}
			}
		}
	}

	return c, err
}

func (n *NodeConfig) AsYaml() (string, error) {
	bytes, err := yaml.Marshal(n)
	return string(bytes), err
}

func Ec2Instances() (map[string]NodeResources, error) {
	e := map[string]NodeResources{}
	data, err := ec2instancesinfo.Data()
	if err != nil {
		return nil, err
	}
	for _, d := range *data {
		e[d.InstanceType] = NodeResources {
			Capacity: map[string]string {
				"cpu": fmt.Sprintf("%d", d.VCPU),
				"memory": fmt.Sprintf("%.0fGi", d.Memory),
			},
			Allocatable: map[string]string {
				"cpu": fmt.Sprintf("%d", d.VCPU),
				"memory": fmt.Sprintf("%.0fGi", d.Memory),
			},
		}
	}
	return e, nil
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
