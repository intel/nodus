package config

import (
	"fmt"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func Test_parseAssertStep(t *testing.T) {

	cases := []struct {
		desc           string
		predicate      []string
		expectedAssert *AssertStep
		err            error
	}{
		{
			desc:      "<object>",
			predicate: []string{"pods"},
			expectedAssert: &AssertStep{
				Object: Pod,
			},
			err: nil,
		},
		{
			desc:      "<class> <object>",
			predicate: []string{"4-cpu", "pods"},
			expectedAssert: &AssertStep{
				Class:  Class("4-cpu"),
				Object: Pod,
			},
			err: nil,
		},
		{
			desc:      "<class> <object>",
			predicate: []string{"4-cpu", "nodes"},
			expectedAssert: &AssertStep{
				Class:  Class("4-cpu"),
				Object: Node,
			},
			err: nil,
		},
		{
			desc:      "<class> <object> <is> <phase>",
			predicate: []string{"4-cpu", "pod", "is", "Running"},
			expectedAssert: &AssertStep{
				Class:    Class("4-cpu"),
				Object:   Pod,
				PodPhase: v1.PodRunning,
			},
			err: nil,
		},
		{
			desc:      "<object> <is> <phase>",
			predicate: []string{"pod", "is", "Running"},
			expectedAssert: &AssertStep{
				Object:   Pod,
				PodPhase: v1.PodRunning,
			},
			err: nil,
		},
		{
			desc:      "<object> <within> <count> <seconds>",
			predicate: []string{"pod", "within", "4", "seconds"},
			expectedAssert: &AssertStep{
				Object: Pod,
				Delay:  4,
			},
			err: nil,
		},
		{
			desc:      "<class> <object> <within> <count> <seconds>",
			predicate: []string{"4-cpu", "pod", "within", "4", "seconds"},
			expectedAssert: &AssertStep{
				Class:  Class("4-cpu"),
				Object: Pod,
				Delay:  4,
			},
			err: nil,
		},
		{
			desc:      "<class> <object> <is> <phase> <within> <count> <seconds>",
			predicate: []string{"4-cpu", "pod", "is", "Running", "within", "4", "seconds"},
			expectedAssert: &AssertStep{
				Class:    Class("4-cpu"),
				Object:   Pod,
				PodPhase: v1.PodRunning,
				Delay:    4,
			},
			err: nil,
		},

		// Negative tests
		{
			desc:           "<object>, invalid object",
			predicate:      []string{"crd"},
			expectedAssert: nil,
			err:            fmt.Errorf("object must be either `node` or `pod`: (found `crd`)"),
		},
		{
			desc:           "<class> <object>, invalid object",
			predicate:      []string{"4-cpu", "crd"},
			expectedAssert: nil,
			err:            fmt.Errorf("object must be either `node` or `pod`: (found `crd`)"),
		},
		{
			desc:           "<class> <object> <is> <phase>, Invalid phase",
			predicate:      []string{"4-cpu", "pod", "is", "Foo"},
			expectedAssert: nil,
			err:            fmt.Errorf("phase must be one of %s, %s, %s, %s or %s: (found `Foo`)", v1.PodPending, v1.PodRunning, v1.PodSucceeded, v1.PodFailed, v1.PodUnknown),
		},
		{
			desc:           "<object> <is> <phase>, invalid phase",
			predicate:      []string{"pod", "is", "Foo"},
			expectedAssert: nil,
			err:            fmt.Errorf("phase must be one of %s, %s, %s, %s or %s: (found `Foo`)", v1.PodPending, v1.PodRunning, v1.PodSucceeded, v1.PodFailed, v1.PodUnknown),
		},
		{
			desc:           "<object> <within> <count> <seconds>, invalid count",
			predicate:      []string{"pod", "within", "foo", "seconds"},
			expectedAssert: nil,
			err:            fmt.Errorf("syntax: assert <count> [<class>] <object> [<is> <phase>] [<within> <count> seconds]"),
		},
		{
			desc:           "<class> <object> <within> <count> <seconds>, invalid count",
			predicate:      []string{"4-cpu", "pod", "within", "foo", "seconds"},
			expectedAssert: nil,
			err:            fmt.Errorf("syntax: assert <count> [<class>] <object> [<is> <phase>] [<within> <count> seconds]"),
		},
		{
			desc:           "<class> <object> <is> <phase> <within> <count> <seconds>, invalid count",
			predicate:      []string{"4-cpu", "pod", "is", "Running", "within", "foo", "seconds"},
			expectedAssert: nil,
			err:            fmt.Errorf("syntax: assert <count> [<class>] <object> [<is> <phase>] [<within> <count> seconds]"),
		},
		{
			desc:           "<class> <is> <phase> <within> <count> <seconds>, no object",
			predicate:      []string{"4-cpu", "is", "Running", "within", "foo", "seconds"},
			expectedAssert: nil,
			err:            fmt.Errorf("object must be either `node` or `pod`: (found `i`)"),
		},
	}

	for _, c := range cases {
		log.WithFields(log.Fields{"description": c.desc}).Infof("Running test")

		actualAssert, err := parseAssertStep(uint64(0), c.predicate)
		if c.err != nil {
			if err == nil || err.Error() != c.err.Error() {
				t.Fatalf("(case: %s) expected error: %s, but got %s", c.desc, c.err, err)
			}
		} else if err != c.err {
			t.Fatalf("(case: %s) expected err to be nil, but got: %s", c.desc, err)
		}
		if !reflect.DeepEqual(c.expectedAssert, actualAssert) {
			t.Fatalf("(case: %s) expected assert: %v, but got %v", c.desc, c.expectedAssert, actualAssert)
		}
	}
}
