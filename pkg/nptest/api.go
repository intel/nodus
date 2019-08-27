package nptest

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/IntelAI/nodus/pkg/client"
	"github.com/IntelAI/nodus/pkg/config"
	"github.com/IntelAI/nodus/pkg/dynamic"
	"github.com/IntelAI/nodus/pkg/exec"
)

func New(namespace string, kubeInfo config.KubeInfo, nodeConfig *config.NodeConfig, podConfig *config.PodConfig) NPTest {
	// construct clients
	k8sclient, err := client.NewK8sClient(kubeInfo.Master, kubeInfo.KconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct kubernetes client")
		os.Exit(1)
	}

	heartbeat, err := client.NewK8sHeartbeatClient(kubeInfo.Master, kubeInfo.KconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct heartbeat client")
		os.Exit(1)
	}

	events, err := client.NewK8sEventClient(kubeInfo.Master, kubeInfo.KconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct event client")
		os.Exit(1)
	}
	
	dynamicClientSet, err := client.NewDynamicClient(kubeInfo.Master, kubeInfo.KconfigPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to construct dynamic client")
		os.Exit(1)
	}
	dynamicClient := dynamic.NewDynamicClient(dynamicClientSet, k8sclient, namespace)

	runner := exec.NewScenarioRunner(k8sclient, heartbeat, events, namespace, nodeConfig, podConfig, dynamicClient)

	return &nptest{
		client: k8sclient,
		runner: runner,
	}
}

type NPTest interface {
	Shutdown()
	Run(step string) error
	Test(t *testing.T, step string)
}

type nptest struct {
	client *kubernetes.Clientset
	runner exec.ScenarioRunner
}

func (np *nptest) Shutdown() {
	np.runner.Shutdown()
}

func (np *nptest) Run(step string) error {
	log.WithFields(log.Fields{"raw": step}).Info("[nptest] run step")
	s, err := config.ParseStep(step)
	if err != nil {
		return err
	}
	err = np.runner.RunStep(s)
	if err != nil {
		return err
	}
	return nil
}

func (np *nptest) Test(t *testing.T, step string) {
	if err := np.Run(step); err != nil {
		t.Fatalf("step failed: %s", err.Error())
	}
}
