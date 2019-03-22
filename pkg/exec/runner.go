package exec

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/IntelAI/nodus/pkg/config"
)

type ScenarioRunner interface {
	RunScenario(scenario *config.Scenario) error
	RunAssert(step *config.Step) error
	RunCreate(step *config.Step) error
	RunChange(step *config.Step) error
	RunDelete(step *config.Step) error
}

func NewScenarioRunner(client *kubernetes.Clientset, namespace string) ScenarioRunner {
	return &runner{
		client:    client,
		namespace: namespace,
	}
}

type runner struct {
	client    *kubernetes.Clientset
	namespace string
}

func (r *runner) RunScenario(scenario *config.Scenario) error {
	log.WithFields(log.Fields{"name": scenario.Name}).Info("run scenario")
	numSteps := len(scenario.Steps)
	for i, step := range scenario.Steps {
		raw := scenario.RawSteps[i]
		log.WithFields(log.Fields{
			"description": raw,
		}).Infof("run step [%d / %d]", i+1, numSteps)

		var err error
		switch step.Verb {
		case config.Assert:
			err = r.RunAssert(step)
		case config.Create:
			err = r.RunCreate(step)
		case config.Change:
			err = r.RunChange(step)
		case config.Delete:
			err = r.RunDelete(step)
		default:
			err = fmt.Errorf("unknown verb `%s`", step.Verb)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) RunAssert(scenario *config.Step) error {
	return nil
}

func (r *runner) RunCreate(scenario *config.Step) error {
	return nil
}

func (r *runner) RunChange(scenario *config.Step) error {
	return nil
}

func (r *runner) RunDelete(scenario *config.Step) error {
	return nil
}
