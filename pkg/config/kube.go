package config

import (
	"fmt"
	"os"
)

const (
	NP_MASTER       = "NP_MASTER"
	NP_KCONFIG_PATH = "NP_KCONFIG_PATH"
)

type KubeInfo struct {
	Master      string
	KconfigPath string
}

func KubeInfoFromEnv() (KubeInfo, error) {
	var err error
	k := KubeInfo{
		Master:      os.Getenv(NP_MASTER),
		KconfigPath: os.Getenv(NP_KCONFIG_PATH),
	}
	if k.Master == "" && k.KconfigPath == "" {
		err = fmt.Errorf("must supply one of %s or %s as environment variables", NP_MASTER, NP_KCONFIG_PATH)
	}
	return k, err
}
