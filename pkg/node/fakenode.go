package node

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	// "k8s.io/kubernetes/pkg/kubelet/events"		
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"	
	"k8s.io/apimachinery/pkg/types"	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"		
	nodeutil "k8s.io/kubernetes/pkg/util/node"
	"github.com/IntelAI/nodus/pkg/config"
	"k8s.io/klog"
    // "encoding/json"	
)

const NodeClassLabel = "np.class"

func NewFakeNode(name string, class string, labels map[string]string, resources config.NodeResources) FakeNode {
	// Add class to node labels
	labels[NodeClassLabel] = class

	return &fakeNode{
		name:      name,
		class:     class,
		labels:    labels,
		resources: resources,
		pods:      NewPodSet(),
		done:      make(chan struct{}),
	}
}

type FakeNode interface {
	Name() string
	Class() string
	Start(kubeClient *kubernetes.Clientset, heartbeatClient *kubernetes.Clientset, eventClient v1core.EventsGetter) error
	Stop() error
}

type fakeNode struct {
	name      string
	class     string
	kubeClient       *kubernetes.Clientset
	heartbeatClient  *kubernetes.Clientset
	node      *v1.Node
	nodeRef   *v1.ObjectReference
	labels    map[string]string
	resources config.NodeResources
	pods      PodSet
	podWatch  watch.Interface
	done      chan struct{}
	recorder  record.EventRecorder
}

func (n *fakeNode) Name() string {
	return n.name
}

func (n *fakeNode) Class() string {
	return n.class
}

func (n *fakeNode) Start(kubeClient *kubernetes.Clientset, heartbeatClient *kubernetes.Clientset, eventClient v1core.EventsGetter) error {
	n.kubeClient = kubeClient
	n.heartbeatClient = heartbeatClient

	n.nodeRef = &v1.ObjectReference{
		Kind:      "Node",
		Name:      string(n.name),
		UID:       types.UID(n.name),
		Namespace: "",
	}
	
	eventBroadcaster := record.NewBroadcaster()
	n.recorder = eventBroadcaster.NewRecorder(legacyscheme.Scheme, v1.EventSource{Component: "kubelet", Host: string(n.name)})
	eventBroadcaster.StartLogging(klog.V(3).Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: eventClient.Events("")})
	
	err := n.startWatchingPods()
	if err != nil {
		return err
	}
	n.startUpdatingPods()
	n.register()

	n.startNodeStatusUpdates()
	// n.setNodeStatus()
	return nil
}

func (n *fakeNode) Stop() error {
	n.podWatch.Stop()
	n.stopUpdatingPods()
	return n.unregister()
}

func (n *fakeNode) register() error {
	node, err := n.k8sNode()
	if err != nil {
		return err
	}
	n.node = node
	node, err = n.kubeClient.CoreV1().Nodes().Create(node)
	if err != nil {
		return err
	}
	n.node = node

	currentTime := metav1.NewTime(time.Now())

	newNodeReadyCondition := v1.NodeCondition{
		Type:              v1.NodeReady,
		Status:            v1.ConditionTrue,
		Reason:            "KubeletReady",
		Message:           "kubelet is posting ready status",
		LastHeartbeatTime: currentTime,
		LastTransitionTime: currentTime,
	}
	n.node.Status.Conditions = append(n.node.Status.Conditions, newNodeReadyCondition)
	
	memoryPressureCondition := v1.NodeCondition{
	    Type:               v1.NodeMemoryPressure,
	    Status:             v1.ConditionFalse,
		Reason:             "KubeletHasInsufficientMemory",
		Message:            "kubelet has insufficient memory available",
		LastTransitionTime:  currentTime,
	}
	n.node.Status.Conditions = append(n.node.Status.Conditions, memoryPressureCondition)

	pidPressureCondition := v1.NodeCondition{
	    Type:               v1.NodePIDPressure,
	    Status:             v1.ConditionFalse,
		Reason:             "KubeletHasSufficientPID",
		Message:            "kubelet has sufficient PID available",
		LastTransitionTime:  currentTime,
	}
	n.node.Status.Conditions = append(n.node.Status.Conditions, pidPressureCondition)

	diskPressureCondition := v1.NodeCondition{
	    Type:               v1.NodeDiskPressure,
	    Status:             v1.ConditionFalse,
		Reason:             "KubeletHasNoDiskPressure",
		Message:            "kubelet has no disk pressure",
		LastTransitionTime:  currentTime,
	}
	n.node.Status.Conditions = append(n.node.Status.Conditions, diskPressureCondition)
	
	return nil
}

func (n *fakeNode) startNodeStatusUpdates() error {
	go wait.Until(n.setNodeStatus, 30 * time.Second, wait.NeverStop)
	return nil
}

func (n *fakeNode) setNodeStatus() {
	// Patch the current status on the API server


	for i := range n.node.Status.Conditions {
		err := nodeutil.SetNodeCondition(n.heartbeatClient, types.NodeName(n.name), n.node.Status.Conditions[i])
		if err != nil {
			log.Error("Unable to update node condition: %v", err)
		}
	}	
}

func (n *fakeNode) startWatchingPods() error {
	// Only list/watch pods bound to this node
	lOpts := metav1.ListOptions{
		Watch:         true,
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", n.name),
	}
	namespace := ""
	podWatch, err := n.kubeClient.CoreV1().Pods(namespace).Watch(lOpts)
	if err != nil {
		return err
	}
	n.podWatch = podWatch

	// Asynchronously consume all watch events
	go n.consumeWatchEvents()
	return nil
}

// Consumes all events from the pod watch channel, updating
// the local pod cache incrementally.
func (n *fakeNode) consumeWatchEvents() {
	for ev := range n.podWatch.ResultChan() {
		switch ev.Type {
		case watch.Added:
			pod := ev.Object.(*v1.Pod)
			log.WithFields(log.Fields{"node": n.name, "pod": pod.Name, "phase": pod.Status.Phase}).Debug("pod added")
			n.pods.Add(pod)
		case watch.Deleted:
			pod := ev.Object.(*v1.Pod)
			log.WithFields(log.Fields{"node": n.name, "pod": pod.Name, "phase": pod.Status.Phase}).Debug("pod deleted")
			n.pods.Remove(pod)
		case watch.Modified:
			pod := ev.Object.(*v1.Pod)
			log.WithFields(log.Fields{"node": n.name, "pod": pod.Name, "phase": pod.Status.Phase}).Debug("pod modified")
			// If pod was marked "deleted" in the API, mimic Kubelet finalization
			// and unblock deleting the pod resource.
			if pod.ObjectMeta.DeletionTimestamp != nil && *pod.ObjectMeta.DeletionGracePeriodSeconds > 0 {
				n.finalizeDeletedPod(pod)
			}
			n.pods.Update(pod)
		}
	}
}

func (n *fakeNode) startUpdatingPods() {
	go n.updatePods()
}

func (n *fakeNode) stopUpdatingPods() {
	close(n.done)
}

// Periodically inspects the local pod cache for pods that need
// to have their phase updated: pending => running or running => terminal.
func (n *fakeNode) updatePods() {
	updateInterval := 2 * time.Second
	t := time.NewTimer(updateInterval)
	for {
		select {
		case <-n.done:
			break
		case <-t.C:
			// Move all bound pending pods to phase running
			pendingPods := n.pods.OfPhase(v1.PodPending)
			n.tryUpdatePodPhase(v1.PodRunning, pendingPods...)
			// Move all expired pods to the specified terminal state.
			for _, pod := range n.pods.Expired() {
				n.tryUpdatePodPhase(TerminalPhase(pod), pod)
			}
			// Reset timer
			t.Reset(updateInterval)
		}
	}
}

// Completes pod deletion by deleting again with no grace period. This mimics
// the behavior of the real Kubelet after any pre-stop hooks, as well as
// local signal escalation: TERM followed by KILL to all of the pod's
// container pids.
func (n *fakeNode) finalizeDeletedPod(pod *v1.Pod) {
	log.WithFields(log.Fields{"node": n.name, "pod": pod.Name}).Debug("finalizing pod")
	gracePeriod := int64(0)
	opts := &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}
	n.kubeClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, opts)
}

func (n *fakeNode) unregister() error {
	// Set all nonterminal pods to failed
	ntPods := n.pods.OfPhase(v1.PodPending, v1.PodUnknown, v1.PodRunning)
	n.tryUpdatePodPhase(v1.PodFailed, ntPods...)

	// Delete this node immediately
	gracePeriod := int64(0)
	opts := &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}
	return n.kubeClient.CoreV1().Nodes().Delete(n.name, opts)
}

// Updates the list of pods to the desired phase, on a best-effort basis.
//
// Note the pod cache is not updated here; the watcher takes care of that
// when a Modified event is received.
func (n *fakeNode) tryUpdatePodPhase(phase v1.PodPhase, pods ...*v1.Pod) {
	for _, pod := range pods {
		originalPhase := pod.Status.Phase

		podClient := n.kubeClient.CoreV1().Pods(pod.Namespace)

		var copy v1.Pod
		copy = *pod
		copy.Status.Phase = phase

		// Add initialized and ready conditions for newly "running" pods
		if originalPhase == v1.PodPending && phase == v1.PodRunning {
			newConds := readyConds(v1.ConditionTrue)
			copy.Status.Conditions = append(copy.Status.Conditions, newConds...)
		}

		// Unset ready conditions for terminal pods
		if phase == v1.PodSucceeded || phase == v1.PodFailed {
			newConds := readyConds(v1.ConditionFalse)
			copy.Status.Conditions = append(copy.Status.Conditions, newConds...)
		}

		pod, err := podClient.UpdateStatus(&copy)

		if err != nil {
			log.WithFields(log.Fields{
				"node":          n.name,
				"pod":           pod.Name,
				"current_phase": pod.Status.Phase,
				"desired_phase": phase,
				"error":         err.Error(),
			}).Warning("unable to patch pod")
		}

		log.WithFields(log.Fields{
			"node":           n.name,
			"pod":            pod.Name,
			"original_phase": originalPhase,
			"current_phase":  pod.Status.Phase,
			"desired_phase":  phase,
		}).Debug("updated pod phase")
	}
}

func readyConds(status v1.ConditionStatus) []v1.PodCondition {
	return []v1.PodCondition{
		{
			Type:               v1.PodInitialized,
			Status:             status,
			LastTransitionTime: metav1.Now(),
		},
		{
			Type:               v1.PodReady,
			Status:             status,
			LastTransitionTime: metav1.Now(),
		},
	}
}

func (n *fakeNode) k8sNode() (*v1.Node, error) {
	defaultPods := resource.MustParse("110")
	defaultCPUs := resource.MustParse("16")
	defaultMemory := resource.MustParse("128Gi")
	defaultStorage := resource.MustParse("2Ti")

	// First set defaults, then override with configuration
	capacity := v1.ResourceList{
		v1.ResourceName("pods"): defaultPods,
		v1.ResourceCPU:          defaultCPUs,
		v1.ResourceMemory:       defaultMemory,
		v1.ResourceStorage:      defaultStorage,
	}
	for name, num := range n.resources.Capacity {
		quantity, err := resource.ParseQuantity(num)
		if err != nil {
			return nil, err
		}
		capacity[v1.ResourceName(name)] = quantity
	}

	allocatable := v1.ResourceList{
		v1.ResourceName("pods"): defaultPods,
		v1.ResourceCPU:          defaultCPUs,
		v1.ResourceMemory:       defaultMemory,
		v1.ResourceStorage:      defaultStorage,
	}
	for name, num := range n.resources.Allocatable {
		quantity, err := resource.ParseQuantity(num)
		if err != nil {
			return nil, err
		}
		allocatable[v1.ResourceName(name)] = quantity
	}

	node := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   n.name,
			Labels: n.labels,
		},
		Status: v1.NodeStatus{
			Capacity:    capacity,
			Allocatable: allocatable,
			Phase:       v1.NodeRunning,
			Addresses:   []v1.NodeAddress{},
			Conditions: []v1.NodeCondition{},
			// 	v1.NodeCondition{
			// 		Type:   v1.NodeReady,
			// 		Status: v1.ConditionTrue,
			// 	},
			// },
		},
	}

	return &node, nil
}
