package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func PodConfigFromFile(path string) (*PodConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return PodConfigFromBytes(data)
}

func PodConfigFromBytes(data []byte) (*PodConfig, error) {
	c := &PodConfig{}

	reader := bytes.NewReader(data)
	decoder := k8syaml.NewYAMLToJSONDecoder(reader)
	err := decoder.Decode(c)

	if err != nil {
		return nil, err
	}

	// Validate pod class names for uniquenes
	classNames := map[string]bool{}
	for _, class := range c.PodClasses {
		name := strings.ToLower(class.Name)
		if _, exists := classNames[name]; exists {
			return nil, fmt.Errorf("pod class name [%s] is not unique", name)
		}
		classNames[name] = true
	}
	return c, err
}

type PodConfig struct {
	PodClasses []PodClass `yaml:"podClasses"`
}

type PodClass struct {
	Name   string
	Count  uint
	Labels map[string]string
	Spec   corev1.PodSpec
}
