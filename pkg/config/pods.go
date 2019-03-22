package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
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
	err := yaml.Unmarshal(data, c)
	return c, err
}

type PodConfig struct{}
