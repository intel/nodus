package nptest

import (
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/IntelAI/nodus/pkg/config"
)

func New(namespace string, nodeConfig *config.NodeConfig, podConfig *config.PodConfig) NPTest {
	// construct clients
	var client *kubernetes.Clientset
	var dynamicClient *dynamic.DynamicClient

	runner := NewScenarioRunner(client, namespace, nodeConfig, podConfig, dynamicClient)

	return &nptest{
		client: client,
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
	log.WithFields("[nptest] run step", log.Fields{"raw": step})
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
