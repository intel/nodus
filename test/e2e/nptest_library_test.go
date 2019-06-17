package e2e

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/IntelAI/nodus/pkg/config"
	"github.com/IntelAI/nodus/pkg/nptest"
)

func TestNPTestLibrary(t *testing.T) {
	nodeConfig := &config.NodeConfig{NodeClasses: []config.NodeClass{
		config.NodeClass{
			Name:  "large",
			Count: 40,
			Labels: map[string]string{
				"nodus-ponens": "true",
				"np.class":     "large",
			},
			Resources: config.NodeResources{
				Capacity: map[string]string{
					"cpu":    "8",
					"memory": "128Gi",
				},
				Allocatable: map[string]string{
					"cpu":    "8",
					"memory": "128Gi",
				},
			},
		},
		config.NodeClass{
			Name:  "small",
			Count: 2,
			Labels: map[string]string{
				"nodus-ponens": "true",
				"np.class":     "small",
			},
			Resources: config.NodeResources{
				Capacity: map[string]string{
					"cpu":    "8",
					"memory": "8Gi",
				},
				Allocatable: map[string]string{
					"cpu":    "8",
					"memory": "8Gi",
				},
			},
		},
	}}

	podConfig := &config.PodConfig{PodClasses: []config.PodClass{
		config.PodClass{
			Name: "4-cpu",
			Labels: map[string]string{
				"np.class":         "4-cpu",
				"np.runDuration":   "3s",
				"np.terminalPhase": "Succeeded",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{corev1.Container{
					Image:           "busybox",
					ImagePullPolicy: "IfNotPresent",
					Name:            "c1",
					Command:         []string{"sleep", "inf"},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{"cpu": resource.MustParse("4")},
					},
				}},
			},
		},
		config.PodClass{
			Name: "1-cpu",
			Labels: map[string]string{
				"np.class":         "1-cpu",
				"np.runDuration":   "10s",
				"np.terminalPhase": "Succeeded",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{corev1.Container{
					Image:           "busybox",
					ImagePullPolicy: "IfNotPresent",
					Name:            "c1",
					Command:         []string{"sleep", "inf"},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{"cpu": resource.MustParse("1")},
					},
				}},
			},
		},
	}}

	kubeInfo, err := config.KubeInfoFromEnv()
	if err != nil {
		t.Fatal(err.Error())
	}

	np := nptest.New("default", kubeInfo, nodeConfig, podConfig)
	defer np.Shutdown()

	np.Test(t, "assert 0 pods within 10s")
	np.Test(t, "create 1 large node")
	np.Test(t, "assert 1 large node")
	np.Test(t, "create 2 small nodes")
	np.Test(t, "assert 2 small nodes")

	np.Test(t, "create 1 4-cpu pod")
	np.Test(t, "assert 1 4-cpu pod is Running within 4s")
	np.Test(t, "delete 1 4-cpu pod")

	np.Test(t, "create 3 1-cpu pods")
	np.Test(t, "assert 3 1-cpu pods are Running within 4s")

	np.Test(t, "change 1 1-cpu pod from Running to Succeeded")
	np.Test(t, "assert 0 1-cpu pods are Pending")
	np.Test(t, "assert 2 1-cpu pods are Running")
	np.Test(t, "assert 1 1-cpu pod is Succeeded")
	np.Test(t, "assert 0 1-cpu pods are Failed")

	np.Test(t, "change 1 1-cpu pod from Running to Failed")
	np.Test(t, "assert 0 1-cpu pods are Pending")
	np.Test(t, "assert 1 1-cpu pod is Running")
	np.Test(t, "assert 1 1-cpu pod is Succeeded")
	np.Test(t, "assert 1 1-cpu pod is Failed")

	np.Test(t, "assert 2 1-cpu pods are Succeeded within 10s")

	np.Test(t, "assert 2 1-cpu pods are Succeeded within 10s")
}
