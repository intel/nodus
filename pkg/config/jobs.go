package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func JobConfigFromFile(path string) (*JobConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return JobConfigFromBytes(data)
}

func JobConfigFromBytes(data []byte) (*JobConfig, error) {
	c := &JobConfig{}

	reader := bytes.NewReader(data)
	decoder := k8syaml.NewYAMLToJSONDecoder(reader)
	err := decoder.Decode(c)

	if err != nil {
		return nil, err
	}

	// Validate pod class names for uniquenes
	classNames := map[string]bool{}
	for _, class := range c.JobClasses {
		name := strings.ToLower(class.Name)
		if _, exists := classNames[name]; exists {
			return nil, fmt.Errorf("job class name [%s] is not unique", name)
		}
		classNames[name] = true
	}
	return c, err
}

type JobConfig struct {
	JobClasses []JobClass `yaml:"jobClasses"`
}

type JobClass struct {
	Name        string
	Count       uint
	Labels      map[string]string
	Annotations map[string]string
	Spec        batchv1.JobSpec
}
