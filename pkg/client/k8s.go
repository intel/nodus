package client

import (
	"time"
	log "github.com/sirupsen/logrus"
	dynamic "k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClientConfig(master string, kubeconfigPath string) (*restclient.Config, error) {
	kconfig, err := clientcmd.BuildConfigFromFlags(master, kubeconfigPath)
	if err != nil {
		log.WithFields(log.Fields{
			"master":         master,
			"kubeconfigPath": kubeconfigPath,
			"error":          err.Error(),
		}).Error("failed to build kubeconfig")
		return nil, err
	}
	return kconfig, err
}

func NewK8sClient(master string, kubeconfigPath string) (*kubernetes.Clientset, error) {
	kconfig, err := NewClientConfig(master, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(kconfig)
}

func NewK8sHeartbeatClient(master string, kubeconfigPath string) (*kubernetes.Clientset, error) {
	kconfig, err := NewClientConfig(master, kubeconfigPath)
	kconfig.Timeout = 30 * time.Second
	kconfig.QPS = float32(-1)	
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(kconfig)
}

func NewDynamicClient(master string, kubeconfigPath string) (dynamic.Interface, error) {
	kconfig, err := NewClientConfig(master, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(kconfig)
}
