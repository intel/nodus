package client

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewK8sClient(master string, kubeconfigPath string) (*kubernetes.Clientset, error) {
	kconfig, err := clientcmd.BuildConfigFromFlags(master, kubeconfigPath)
	if err != nil {
		log.WithFields(log.Fields{
			"master":         master,
			"kubeconfigPath": kubeconfigPath,
			"error":          err.Error(),
		}).Error("failed to build kubeconfig")
		return nil, err
	}
	return kubernetes.NewForConfig(kconfig)
}
