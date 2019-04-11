package config

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Test_parseAssertStep(t *testing.T) {

	cases := []struct {
		desc           string
		predicate      []string
		apiAssert      bool
		expectedAssert *AssertStep
		err            error
	}{
		{
			desc:      "<object>",
			predicate: []string{"pods"},
			apiAssert: false,
			expectedAssert: &AssertStep{
				Object: Pod,
			},
			err: nil,
		},
		{
			desc:      "<class> <object>",
			predicate: []string{"4-cpu", "pods"},
			apiAssert: false,
			expectedAssert: &AssertStep{
				Class:  Class("4-cpu"),
				Object: Pod,
			},
			err: nil,
		},
		{
			desc:      "<class> <object>",
			predicate: []string{"4-cpu", "nodes"},
			apiAssert: false,
			expectedAssert: &AssertStep{
				Class:  Class("4-cpu"),
				Object: Node,
			},
			err: nil,
		},
		{
			desc:      "<class> <object> <is> <phase>",
			predicate: []string{"4-cpu", "pod", "is", "Running"},
			apiAssert: false,
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
			apiAssert: false,
			expectedAssert: &AssertStep{
				Object:   Pod,
				PodPhase: v1.PodRunning,
			},
			err: nil,
		},
		{
			desc:      "<object> <within> <duration>",
			predicate: []string{"pod", "within", "4s"},
			apiAssert: false,
			expectedAssert: &AssertStep{
				Object: Pod,
				Delay:  4 * time.Second,
			},
			err: nil,
		},
		{
			desc:      "<class> <object> <within> <duration>",
			predicate: []string{"4-cpu", "pod", "within", "4s"},
			apiAssert: false,
			expectedAssert: &AssertStep{
				Class:  Class("4-cpu"),
				Object: Pod,
				Delay:  4 * time.Second,
			},
			err: nil,
		},
		{
			desc:      "<class> <object> <is> <phase> <within> <duration>",
			predicate: []string{"4-cpu", "pod", "is", "Running", "within", "4s"},
			apiAssert: false,
			expectedAssert: &AssertStep{
				Class:    Class("4-cpu"),
				Object:   Pod,
				PodPhase: v1.PodRunning,
				Delay:    4 * time.Second,
			},
			err: nil,
		},
		{
			desc:      "<version> <kind>",
			predicate: []string{"v1", "Pod"},
			apiAssert: true,
			expectedAssert: &AssertStep{
				GVK: &schema.GroupVersionKind{
					Version: "v1",
					Kind:    "Pod",
				},
			},
			err: nil,
		},
		{
			desc:      "<version> <kind> <group>",
			predicate: []string{"v1", "Job", "batch"},
			apiAssert: true,
			expectedAssert: &AssertStep{
				GVK: &schema.GroupVersionKind{
					Version: "v1",
					Kind:    "Job",
					Group:   "batch",
				},
			},
			err: nil,
		},
		{
			desc:      "<version> <kind> <within> <duration>",
			predicate: []string{"v1", "Pod", "within", "4s"},
			apiAssert: true,
			expectedAssert: &AssertStep{
				GVK: &schema.GroupVersionKind{
					Version: "v1",
					Kind:    "Pod",
				},
				Delay: 4 * time.Second,
			},
			err: nil,
		},
		{
			desc:      "<version> <kind> <group> <within> <duration>",
			predicate: []string{"v1", "Job", "batch", "within", "4s"},
			apiAssert: true,
			expectedAssert: &AssertStep{
				GVK: &schema.GroupVersionKind{
					Version: "v1",
					Kind:    "Job",
					Group:   "batch",
				},
				Delay: 4 * time.Second,
			},
			err: nil,
		},

		// Negative tests
		{
			desc:           "<object>, invalid object",
			predicate:      []string{"crd"},
			expectedAssert: nil,
			err:            fmt.Errorf("object must be either `node` or `pod`: (found `crd`)"),
			apiAssert:      false,
		},
		{
			desc:           "<class> <object>, invalid object",
			predicate:      []string{"4-cpu", "crd"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("object must be either `node` or `pod`: (found `crd`)"),
		},
		{
			desc:           "<class> <object> <is> <phase>, Invalid phase",
			predicate:      []string{"4-cpu", "pod", "is", "Foo"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("phase must be one of %s, %s, %s, %s or %s: (found `Foo`)", v1.PodPending, v1.PodRunning, v1.PodSucceeded, v1.PodFailed, v1.PodUnknown),
		},
		{
			desc:           "<object> <is> <phase>, invalid phase",
			predicate:      []string{"pod", "is", "Foo"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("phase must be one of %s, %s, %s, %s or %s: (found `Foo`)", v1.PodPending, v1.PodRunning, v1.PodSucceeded, v1.PodFailed, v1.PodUnknown),
		},
		{
			desc:           "<object> <within> <duration>, invalid duration",
			predicate:      []string{"pod", "within", "foo"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<class> <object> <within> <duration>, invalid count",
			predicate:      []string{"4-cpu", "pod", "within", "foo"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<class> <object> <is> <phase> <within> <duration>, invalid count",
			predicate:      []string{"4-cpu", "pod", "is", "Running", "within", "foo"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<class> <is> <phase> <within> <duration>, no object",
			predicate:      []string{"4-cpu", "is", "Running", "within", "4s"},
			expectedAssert: nil,
			apiAssert:      false,
			err:            fmt.Errorf("object must be either `node` or `pod`: (found `i`)"),
		},
		{
			desc:           "<version> <kind> <group>, two missing",
			predicate:      []string{"v1"},
			expectedAssert: nil,
			apiAssert:      true,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<version> <kind> <group> <within> <duration>, two missing",
			predicate:      []string{"api", "within", "4s"},
			expectedAssert: nil,
			apiAssert:      true,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<version> <kind> <within> <duration>, wrong duration",
			predicate:      []string{"v1", "Pod", "within", "foo"},
			expectedAssert: nil,
			apiAssert:      true,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<version> <kind> <group> <within> <duration>, one missing",
			predicate:      []string{"Job", "v1", "Batch", "4s"},
			expectedAssert: nil,
			apiAssert:      true,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
		{
			desc:           "<version> <kind> <group> <within> <duration>, wrong duration",
			predicate:      []string{"v1", "Job", "batch", "within", "foo"},
			expectedAssert: nil,
			apiAssert:      true,
			err:            fmt.Errorf("syntax: assert ( <count> [<class>] <object> [<is> <phase>] | api <version> <kind> [<group>] ) [<within> <duration>]"),
		},
	}

	for _, c := range cases {
		log.WithFields(log.Fields{"description": c.desc}).Infof("Running test")

		actualAssert, err := parseAssertStep(uint64(0), c.predicate, c.apiAssert)
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
