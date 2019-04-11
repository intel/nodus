package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ScenarioFromFile(path string) (*Scenario, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	scenario, err := ScenarioFromBytes(data)
	if err != nil {
		return nil, err
	}
	scenario.WorkingDir = filepath.Dir(path)
	return scenario, nil
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
	Name       string
	Version    uint64
	RawSteps   []string `yaml:"steps"`
	WorkingDir string
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
// <step>        => <assertStep> | <createStep> | <changeStep> | <deleteStep>
// <assertStep>  => "assert" ( <count> [<class>] <object> [<is> <phase>] | api  <version> <kind> [<group>] ) [<within> <duration>]
// <createStep>  => "create" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
// <changeStep>  => "change" <count> <class> <object> "from" <phase> "to" <phase>
// <deleteStep>  => "delete" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
// <is>         => "is" | "are"
// <count>      => [1-9][0-9]*
// <class>      => [A-Za-z0-9\-]+
// <object>     => "pod[s]" | "node[s]"
// <phase>      => "Pending" | "Running" | "Succeeded" | "Failed" | "Unknown"
// <duration>   => time.Duration

func parseStep(raw string) (*Step, error) {
	raw = strings.ToLower(raw)
	parts := strings.Split(raw, " ")
	if len(parts) < 3 {
		return nil, fmt.Errorf(`not enough words (need at least: "verb count object"), but given: %s`, raw)
	}

	step := &Step{
		Verb: Verb(strings.TrimSpace(parts[0])),
	}

	var count uint64
	apiAssert := false
	count, err := parseCount(parts[1])
	if err != nil {
		if parts[1] == "api" {
			count = 0
			apiAssert = true
		} else {
			return nil, err
		}
	}
	predicate := parts[2:]
	switch step.Verb {
	case Assert:
		a, err := parseAssertStep(count, predicate, apiAssert)
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

func getNext(array []string) (string, []string, error) {
	if len(array) == 0 {
		return "", []string{}, fmt.Errorf("Insufficient elements in array")
	}
	next, rem := array[0], array[1:]
	return next, rem, nil
}

// <assertStep> => "assert" ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]
func parseAssertStep(count uint64, predicate []string, apiAssert bool) (*AssertStep, error) {
	result := &AssertStep{Count: count}

	// Check for the first 2 predicates.
	// Check if first predicate is object
	next, rem, err := getNext(predicate)
	if err != nil {
		return nil, fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]")
	}

	if apiAssert {
		result.GVK = &schema.GroupVersionKind{
			Version: next,
		}
		next, rem, err = getNext(rem)
		if err != nil || next == "within" {
			return nil, fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]")
		}
		result.GVK.Kind = next

		next, rem, err = getNext(rem)
		if err != nil {
			return result, nil
		}
		if next != "within" {
			result.GVK.Group = next

			next, rem, err = getNext(rem)
			if err != nil {
				return result, nil
			}
		}
	} else {

		obj, err := parseObject(next)
		if err != nil {
			// Check if count is provided and check if second predicate is object
			var e error
			next, rem, e = getNext(rem)
			if e != nil {
				return nil, err
			}

			obj, err = parseObject(next)
			if err != nil {
				return nil, err
			}

			// This means first is class
			result.Class = Class(predicate[0])
			result.Object = obj
		} else {
			// No count is provided, use object instead.
			result.Object = obj
		}

		// Now check if either phase is provided or delay is provided.
		next, rem, err = getNext(rem)
		if err != nil {
			return result, nil
		}
		// Check if there is a "is" or "are"
		if next == "is" || next == "are" {
			next, rem, err = getNext(rem)
			if err != nil {
				return nil, fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]")
			}
			ph, err := parsePhase(next)

			if err == nil {

				result.PodPhase = ph
				next, rem, err = getNext(rem)
				if err != nil {
					return result, nil
				}
			} else if next != "within" {
				return nil, err
			}
		}
	}
	// Check if there is within
	if next == "within" {
		next, rem, err = getNext(rem)
		if err != nil {
			return nil, fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]")
		}
		duration, err := time.ParseDuration(next)
		if err != nil {
			return nil, fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]")
		}
		result.Delay = duration
	} else if next != "" {
		return nil, fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]")
	}

	return result, nil
}

// <createStep> => "create" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
func parseCreateStep(count uint64, predicate []string) (*CreateStep, error) {
	if len(predicate) != 2 && len(predicate) != 3 {
		return nil, fmt.Errorf("syntax: create <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )")
	}

	var result *CreateStep
	if len(predicate) == 3 {
		instanceString := strings.TrimRight(strings.TrimSpace(predicate[0]), "s")
		if instanceString != "instance" || predicate[1] != "of" {
			return nil, fmt.Errorf("syntax: create <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )")
		}

		result = &CreateStep{
			Count:    count,
			YamlPath: predicate[2],
		}
	} else {
		obj, err := parseObject(predicate[1])
		if err != nil {
			return nil, err
		}
		result = &CreateStep{
			Count:  count,
			Class:  Class(predicate[0]),
			Object: obj,
		}
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

// <deleteStep> => "delete" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
func parseDeleteStep(count uint64, predicate []string) (*DeleteStep, error) {
	if len(predicate) > 2 && len(predicate) == 0 {
		return nil, fmt.Errorf("syntax: delete <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )")
	}
	var result *DeleteStep
	if len(predicate) == 3 {
		instanceString := strings.TrimRight(strings.TrimSpace(predicate[0]), "s")
		if instanceString != "instance" || predicate[1] != "of" {
			return nil, fmt.Errorf("syntax: delete <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )")
		}

		result = &DeleteStep{
			Count:    count,
			YamlPath: predicate[2],
		}
	} else {
		obj, err := parseObject(predicate[1])
		if err != nil {
			return nil, err
		}
		result = &DeleteStep{
			Count:  count,
			Class:  Class(predicate[0]),
			Object: obj,
		}
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
	Delay    time.Duration
	GVK      *schema.GroupVersionKind
}

type CreateStep struct {
	Count    uint64
	Class    Class
	Object   Object
	YamlPath string
}

type ChangeStep struct {
	Count        uint64
	Class        Class
	Object       Object
	FromPodPhase v1.PodPhase
	ToPodPhase   v1.PodPhase
}

type DeleteStep struct {
	Count    uint64
	Class    Class
	Object   Object
	YamlPath string
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
