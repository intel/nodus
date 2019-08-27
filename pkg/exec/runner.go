package exec

import (
	"fmt"
	"path"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"		
	"github.com/IntelAI/nodus/pkg/config"
	"github.com/IntelAI/nodus/pkg/dynamic"
	"github.com/IntelAI/nodus/pkg/node"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	wait "k8s.io/apimachinery/pkg/util/wait"
)

type ScenarioRunner interface {
	RunScenario(scenario *config.Scenario) error
	RunAssert(step *config.Step) error
	RunCreate(step *config.Step) error
	RunChange(step *config.Step) error
	RunDelete(step *config.Step) error
	RunStep(step *config.Step) error
	Shutdown()
}

func NewScenarioRunner(client *kubernetes.Clientset, heartbeat *kubernetes.Clientset, events v1core.EventsGetter, namespace string, nodeConfig *config.NodeConfig, podConfig *config.PodConfig, dynamicClient *dynamic.DynamicClient) ScenarioRunner {
	return &runner{
		client:        client,
		heartbeat:     heartbeat,
		namespace:     namespace,
	    events:        events,
		nodeConfig:    nodeConfig,
		podConfig:     podConfig,
		gcPods:        map[string]bool{},
		gcNodes:       map[string]bool{},
		dynamicClient: dynamicClient,
		gcObjects:     map[string]bool{},
	}
}

type runner struct {
	client        *kubernetes.Clientset
	heartbeat     *kubernetes.Clientset
	events        v1core.EventsGetter
	dynamicClient *dynamic.DynamicClient
	namespace     string
	podConfig     *config.PodConfig
	nodeConfig    *config.NodeConfig
	gcPods        map[string]bool
	gcNodes       map[string]bool
	gcObjects     map[string]bool
	workingDir    string
}

func (r *runner) Shutdown() {
	log.Info("Cleaning up resources")
	podClient := r.client.CoreV1().Pods(r.namespace)
	deleteOptions := &metav1.DeleteOptions{}
	for pod := range r.gcPods {
		podClient.Delete(pod, deleteOptions)
	}

	nodeClient := r.client.CoreV1().Nodes()
	for node := range r.gcNodes {
		nodeClient.Delete(node, deleteOptions)
	}

	for yaml := range r.gcObjects {
		r.dynamicClient.Delete(yaml)
	}
}

func (r *runner) RunScenario(scenario *config.Scenario) error {
	log.WithFields(log.Fields{"name": scenario.Name}).Info("run scenario")
	numSteps := len(scenario.Steps)
	defer r.Shutdown()
	r.workingDir = scenario.WorkingDir
	for i, step := range scenario.Steps {
		raw := scenario.RawSteps[i]
		log.WithFields(log.Fields{
			"description": raw,
		}).Infof("run step [%d / %d]", i+1, numSteps)
		if err := r.RunStep(step); err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) RunStep(step *config.Step) error {
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

	return err
}

func (r *runner) assertNode(assert *config.AssertStep) error {
	// Supported grammar: "assert" <count> [<class>] <object> [<is> <phase>] [<within> <count> seconds]

	// Get all the nodes with the optional class.

	var labelSelector string
	if assert.Class != "" {
		labelSelector = fmt.Sprintf("np.class=%s", assert.Class)
	}
	nodeList, err := r.client.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}
	if nodeList.Items == nil || uint64(len(nodeList.Items)) != assert.Count {
		if assert.Class != "" {
			return fmt.Errorf("found %d nodes of class %s, but %d expected", len(nodeList.Items), assert.Class, assert.Count)
		}
		return fmt.Errorf("found %d nodes but %d expected", len(nodeList.Items), assert.Count)
	}
	return nil
}

func (r *runner) assertPod(assert *config.AssertStep) error {
	// Supported grammar: "assert" <count> [<class>] <object> [<is> <phase>] [<within> <count> seconds]
	var labelSelector string
	if assert.Class != "" {
		labelSelector = fmt.Sprintf("np.class=%s", assert.Class)
	}
	var fieldSelector string
	if assert.PodPhase != "" {
		fieldSelector = fmt.Sprintf("status.phase=%s", assert.PodPhase)
	}

	podList, err := r.client.CoreV1().Pods(r.namespace).List(metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return err
	}
	if podList.Items == nil || uint64(len(podList.Items)) != assert.Count {
		return fmt.Errorf("found %d pods of class %s and phase: %s, but %d expected", len(podList.Items), assert.Class, assert.PodPhase, assert.Count)
	}

	return nil
}

func (r *runner) checkIfAPIAvailable(gvk *schema.GroupVersionKind) error {
	resource, err := r.dynamicClient.GetResourceFromObject(*gvk)
	if err != nil {
		return err
	}
	// Try a simple list
	_, err = resource.List(metav1.ListOptions{})

	return err
}

func (r *runner) RunAssert(step *config.Step) error {
	if step.Assert == nil {
		return fmt.Errorf("there is no assert in this step.")
	}

	backoffWait := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1,
		Steps:    int(step.Assert.Delay.Seconds()),
	}

	if step.Assert.GVK != nil {

		err := r.checkIfAPIAvailable(step.Assert.GVK)
		for backoffWait.Steps > 0 {
			if err == nil {
				break
			}
			time.Sleep(backoffWait.Step())
			err = r.checkIfAPIAvailable(step.Assert.GVK)
		}
		return err
	}

	switch step.Assert.Object {
	case config.Node:
		err := r.assertNode(step.Assert)
		for backoffWait.Steps > 0 {
			err = r.assertNode(step.Assert)
			if err == nil {
				break
			}
			time.Sleep(backoffWait.Step())
		}
		return err
	case config.Pod:
		err := r.assertPod(step.Assert)
		for backoffWait.Steps > 0 {
			err = r.assertPod(step.Assert)
			if err == nil {
				break
			}
			time.Sleep(backoffWait.Step())
		}
		return err
	}
	return fmt.Errorf("assert object: %s not supported", step.Assert.Object)
}

func (r *runner) createNode(create *config.CreateStep) error {
	// Supported grammar: "create" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
	// Check if nodeConfig has the specified class
	if r.nodeConfig == nil {
		return fmt.Errorf("no node found for class: %s, please specify a nodes.yml file", create.Class)
	}
	for _, class := range r.nodeConfig.NodeClasses {
		if config.Class(class.Name) == create.Class {
			for i := uint64(0); i < create.Count; i++ {
				nodeName := fmt.Sprintf("%s-%d", class.Name, i)
				n := node.NewFakeNode(nodeName, class.Name, class.Labels, class.Resources)
				err := n.Start(r.client, r.heartbeat, r.events)
				if err != nil {
					return fmt.Errorf("could not create node of class: %s, err: %s", create.Class, err.Error())
				}
				r.gcNodes[nodeName] = true
			}
			return nil
		}
	}
	return fmt.Errorf("class: %s not found in the node config", create.Class)
}

func (r *runner) createPod(create *config.CreateStep) error {
	// Supported grammar: "create" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
	if r.podConfig == nil {
		return fmt.Errorf("no pod found for class: %s, please specify a pods.yml file", create.Class)
	}
	podClient := r.client.CoreV1().Pods(r.namespace)
	// Check if podConfig has the specified class
	for _, class := range r.podConfig.PodClasses {
		if config.Class(class.Name) == create.Class {
			for i := uint64(0); i < create.Count; i++ {
				// Create the pod
				podName := fmt.Sprintf("%s-%d", class.Name, i)
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:   podName,
						Labels: class.Labels,
					},
					Spec: class.Spec,
				}
				if _, err := podClient.Create(pod); err != nil {
					return err
				}
				r.gcPods[podName] = true
			}
			return nil
		}
	}
	return fmt.Errorf("class: %s not found in the pod config", create.Class)
}

func (r *runner) createObject(create *config.CreateStep) error {
	// Supported grammar: "create" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
	create.YamlPath = path.Join(r.workingDir, create.YamlPath)
	r.gcObjects[create.YamlPath] = true
	return r.dynamicClient.Create(create.YamlPath)
}

func (r *runner) RunCreate(step *config.Step) error {
	if step.Create == nil {
		return fmt.Errorf("there is no create in this step.")
	}
	if step.Create.YamlPath != "" {
		return r.createObject(step.Create)
	}

	switch step.Create.Object {
	case config.Node:
		return r.createNode(step.Create)
	case config.Pod:
		return r.createPod(step.Create)
	}

	return fmt.Errorf("create object: %s not supported", step.Create.Object)
}

func (r *runner) changePod(change *config.ChangeStep) error {
	// Supported grammar: "change" <count> <class> <object> "from" <phase> "to" <phase>

	if change.FromPodPhase == change.ToPodPhase {
		return fmt.Errorf("the change requested is to the same phase. From phase: %s, to phase: %s", change.FromPodPhase, change.ToPodPhase)
	}

	pods, err := r.client.CoreV1().Pods(r.namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("np.class=%s", change.Class),
		FieldSelector: fmt.Sprintf("status.phase=%s", change.FromPodPhase),
	})
	if err != nil {
		return err
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("found 0 pods of class: %s and phase: %s, expected: %d", change.Class, change.FromPodPhase, change.Count)
	}

	if uint64(len(pods.Items)) < change.Count {
		return fmt.Errorf("expected atleast %d pods of class: %s and phase: %s, but found: %d", change.Count, change.Class, change.FromPodPhase, len(pods.Items))
	}

	// Get a slice

	podClient := r.client.CoreV1().Pods(r.namespace)
	for i := uint64(0); i < change.Count; i++ {
		pod := pods.Items[i]
		// Copy pod
		var copy corev1.Pod
		copy = pod
		copy.Status.Phase = change.ToPodPhase
		// Get current conditions
		var cond corev1.ConditionStatus
		if pod.Status.Phase == corev1.PodPending && change.ToPodPhase == corev1.PodRunning {
			cond = corev1.ConditionTrue
		}

		if change.ToPodPhase == corev1.PodSucceeded || change.ToPodPhase == corev1.PodFailed {
			cond = corev1.ConditionFalse
		}

		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:               corev1.PodConditionType(change.ToPodPhase),
			Status:             cond,
			LastTransitionTime: metav1.Now(),
		})
		_, err := podClient.UpdateStatus(&copy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) RunChange(step *config.Step) error {
	if step.Change == nil {
		return fmt.Errorf("there is no change in this step.")
	}
	switch step.Change.Object {
	case config.Pod:
		return r.changePod(step.Change)
	}
	return fmt.Errorf("change object: %s not supported", step.Change.Object)
}

func (r *runner) deleteNode(del *config.DeleteStep) error {
	// Supported grammar: "delete" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
	nodes, err := r.client.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("np.class=%s", del.Class),
	})
	if err != nil {
		return fmt.Errorf("no nodes found for class: %s", del.Class)
	}
	if uint64(len(nodes.Items)) < del.Count {
		return fmt.Errorf("found %d nodes of class: %s, but expected: %d", len(nodes.Items), del.Class, del.Count)
	}

	for i := uint64(0); i < del.Count; i++ {
		err = r.client.CoreV1().Nodes().Delete(nodes.Items[i].Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		delete(r.gcNodes, nodes.Items[i].Name)
	}

	return nil
}

func (r *runner) deletePod(del *config.DeleteStep) error {
	// Supported grammar: "delete" <count> ( <class> <object> | instance[s] of <path/to/yaml/file> )
	pods, err := r.client.CoreV1().Pods(r.namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("np.class=%s", del.Class),
	})
	if err != nil {
		return fmt.Errorf("no pods found for class: %s", del.Class)
	}
	if uint64(len(pods.Items)) < del.Count {
		return fmt.Errorf("found %d pods of class: %s, but expected: %d", len(pods.Items), del.Class, del.Count)
	}

	for i := uint64(0); i < del.Count; i++ {
		err = r.client.CoreV1().Pods(r.namespace).Delete(pods.Items[i].Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		delete(r.gcPods, pods.Items[i].Name)
	}
	return nil
}

func (r *runner) deleteObject(del *config.DeleteStep) error {
	del.YamlPath = path.Join(r.workingDir, del.YamlPath)
	delete(r.gcObjects, del.YamlPath)
	return r.dynamicClient.Delete(del.YamlPath)
}

func (r *runner) RunDelete(step *config.Step) error {
	if step.Delete == nil {
		return fmt.Errorf("there is no delete in this step.")
	}

	if step.Delete.YamlPath != "" {
		return r.deleteObject(step.Delete)
	}

	switch step.Delete.Object {
	case config.Node:
		return r.deleteNode(step.Delete)
	case config.Pod:
		return r.deletePod(step.Delete)
	}

	return fmt.Errorf("delete object: %s not supported", step.Delete.Object)
}
