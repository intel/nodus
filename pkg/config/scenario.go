package config

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
)

func ScenarioFromFile(path string) (*Scenario, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ScenarioFromBytes(data)
}

func ScenarioFromBytes(data []byte) (*Scenario, error) {
	sy := ScenarioYaml{}
	err := yaml.Unmarshal(data, &sy)
	if err != nil {
		return nil, err
	}

	steps, err := parseSteps(sy.RawSteps)
	if err != nil {
		return nil, fmt.Errorf("unable to parse: %s", err.Error())
	}
	if len(steps) != len(sy.RawSteps) {
		return nil, fmt.Errorf("number of parsed steps does not equal raw input steps")
	}

	return &Scenario{ScenarioYaml: sy, Steps: steps}, nil
}

type Scenario struct {
	ScenarioYaml
	Steps []*Step
}

type ScenarioYaml struct {
	Name     string
	Version  uint64
	RawSteps []string `yaml:"steps"`
}

func parseSteps(rawSteps []string) ([]*Step, error) {
	steps := []*Step{}
	for i, raw := range rawSteps {
		step, err := parseStep(raw)
		if err != nil {
			return nil, fmt.Errorf("step [%d]: %s (input: `%s`", i, err.Error(), raw)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// Step grammar:
//
// <step>       => <assertStep> | <createStep> | <changeStep> | <deleteStep>
// <assertStep> => "assert" <count> [<class>] <object> [<is> <phase>]
// <createStep> => "create" <count> <class> <object>
// <changeStep> => "change" <count> <class> <object> "from" <phase> "to" <phase>
// <deleteStep> => "delete" <count> <class> <object>
// <is>         => "is" | "are"
// <count>      => [1-9][0-9]*
// <class>      => [A-Za-z0-9\-]+
// <object>     => "pod[s]" | "node[s]"
// <phase>      => "Pending" | "Running" | "Succeeded" | "Failed" | "Unknown"

func parseStep(raw string) (*Step, error) {
	raw = strings.ToLower(raw)
	parts := strings.Split(raw, " ")
	if len(parts) < 3 {
		return nil, fmt.Errorf(`not enough words (need at least: "verb count object")`, raw)
	}

	step := &Step{
		Verb: Verb(strings.TrimSpace(parts[0])),
	}
	count, err := parseCount(parts[1])
	if err != nil {
		return nil, err
	}
	predicate := parts[2:]
	switch step.Verb {
	case Assert:
		a, err := parseAssertStep(count, predicate)
		if err != nil {
			return nil, err
		}
		step.Assert = a
		return step, nil
	case Create:
		c, err := parseCreateStep(count, predicate)
		if err != nil {
			return nil, err
		}
		step.Create = c
		return step, nil
	case Change:
		c, err := parseChangeStep(count, predicate)
		if err != nil {
			return nil, err
		}
		step.Change = c
		return step, nil
	case Delete:
		d, err := parseDeleteStep(count, predicate)
		if err != nil {
			return nil, err
		}
		step.Delete = d
		return step, nil
	}
	return nil, fmt.Errorf(`unknown verb: "%s"`, step.Verb)
}

func parseCount(raw string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
}

// <assertStep> => "assert" <count> [<class>] <object> [<is> <phase>]
func parseAssertStep(count uint64, predicate []string) (*AssertStep, error) {
	result := &AssertStep{Count: count}
	// assert <count> <object>
	if len(predicate) == 1 {
		obj, err := parseObject(predicate[0])
		if err != nil {
			return nil, err
		}
		result.Object = obj
		return result, nil
	}
	// assert <count> <class> <object>
	if len(predicate) == 2 {
		result.Class = Class(predicate[0])
		obj, err := parseObject(predicate[1])
		if err != nil {
			return nil, err
		}
		result.Object = obj
		return result, nil
	}
	// assert <count> <object> <is> <phase>
	if len(predicate) == 3 {
		obj, err := parseObject(predicate[0])
		if err != nil {
			return nil, err
		}
		result.Object = obj
		if predicate[1] != "is" || predicate[1] != "are" {
			return nil, fmt.Errorf("syntax: assert <count> [<class>] <object> [<is> <phase>]")
		}
		ph, err := parsePhase(predicate[2])
		if err != nil {
			return nil, err
		}
		result.PodPhase = ph
		return result, nil
	}
	// assert <count> <class> <object> <is> <phase>
	if len(predicate) == 4 {
		result.Class = Class(predicate[0])
		obj, err := parseObject(predicate[1])
		if err != nil {
			return nil, err
		}
		result.Object = obj
		if predicate[2] != "is" && predicate[2] != "are" {
			return nil, fmt.Errorf("syntax: assert <count> [<class>] <object> [<is> <phase>]")
		}
		ph, err := parsePhase(predicate[3])
		if err != nil {
			return nil, err
		}
		result.PodPhase = ph
		return result, nil
	}
	return nil, fmt.Errorf("syntax: assert <count> [<class>] <object> [<is> <phase>]")
}

// <createStep> => "create" <count> <class> <object>
func parseCreateStep(count uint64, predicate []string) (*CreateStep, error) {
	if len(predicate) != 2 {
		return nil, fmt.Errorf("syntax: create <count> <class> <object>")
	}
	obj, err := parseObject(predicate[1])
	if err != nil {
		return nil, err
	}
	result := &CreateStep{
		Count:  count,
		Class:  Class(predicate[0]),
		Object: obj,
	}
	return result, nil
}

// <changeStep> => "change" <count> <class> <object> "from" <phase> "to" <phase>
func parseChangeStep(count uint64, predicate []string) (*ChangeStep, error) {
	if len(predicate) != 6 {
		return nil, fmt.Errorf("syntax: change <count> <class> <object> from <phase> to <phase>")
	}
	class := Class(predicate[0])
	obj, err := parseObject(predicate[1])
	if err != nil {
		return nil, err
	}
	fromPhase, err := parsePhase(predicate[3])
	if err != nil {
		return nil, err
	}
	toPhase, err := parsePhase(predicate[5])
	if err != nil {
		return nil, err
	}
	result := &ChangeStep{
		Count:        count,
		Class:        class,
		Object:       obj,
		FromPodPhase: fromPhase,
		ToPodPhase:   toPhase,
	}
	return result, nil
}

// <deleteStep> => "delete" <count> <class> <object>
func parseDeleteStep(count uint64, predicate []string) (*DeleteStep, error) {
	if len(predicate) != 2 {
		return nil, fmt.Errorf("syntax: delete <count> <class> <object>")
	}
	obj, err := parseObject(predicate[1])
	if err != nil {
		return nil, err
	}
	result := &DeleteStep{
		Count:  count,
		Class:  Class(predicate[0]),
		Object: obj,
	}
	return result, nil
}

func parseObject(o string) (Object, error) {
	canonical := strings.TrimRight(strings.TrimSpace(o), "s")
	obj := Object(canonical)
	switch obj {
	case Pod:
		return Pod, nil
	case Node:
		return Node, nil
	}
	return obj, fmt.Errorf("object must be either `node` or `pod`: (found `%s`)", obj)
}

func parsePhase(p string) (v1.PodPhase, error) {
	ph := v1.PodPhase(strings.Title(strings.TrimSpace(p)))
	switch ph {
	case v1.PodPending:
		return v1.PodPending, nil
	case v1.PodRunning:
		return v1.PodRunning, nil
	case v1.PodSucceeded:
		return v1.PodSucceeded, nil
	case v1.PodFailed:
		return v1.PodFailed, nil
	case v1.PodUnknown:
		return v1.PodUnknown, nil
	}
	return ph, fmt.Errorf("phase must be one of %s, %s, %s, %s or %s: (found `%s`)",
		v1.PodPending, v1.PodRunning, v1.PodSucceeded, v1.PodFailed, v1.PodUnknown, ph)
}

type Step struct {
	Verb   Verb
	Assert *AssertStep
	Create *CreateStep
	Change *ChangeStep
	Delete *DeleteStep
}

func (s *Step) AsYaml() string {
	data, _ := yaml.Marshal(s)
	return string(data)
}

type AssertStep struct {
	Count    uint64
	Class    Class // optional
	Object   Object
	PodPhase v1.PodPhase // optional
}

type CreateStep struct {
	Count  uint64
	Class  Class
	Object Object
}

type ChangeStep struct {
	Count        uint64
	Class        Class
	Object       Object
	FromPodPhase v1.PodPhase
	ToPodPhase   v1.PodPhase
}

type DeleteStep struct {
	Count  uint64
	Class  Class
	Object Object
}

type Verb string

const (
	Assert Verb = "assert"
	Create Verb = "create"
	Change Verb = "change"
	Delete Verb = "delete"
)

type Object string

const (
	Node Object = "node"
	Pod  Object = "pod"
)

type Class string
