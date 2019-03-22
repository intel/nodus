package node

import (
	"sync"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodSet interface {
	Add(pod *v1.Pod)
	Remove(pod *v1.Pod)
	Update(pod *v1.Pod)
	OfPhase(phases ...v1.PodPhase) []*v1.Pod
	Expired() []*v1.Pod
}

func NewPodSet() PodSet {
	return &podset{}
}

type podset struct {
	sync.RWMutex
	pods []*v1.Pod
}

func (s *podset) Add(pod *v1.Pod) {
	s.Lock()
	defer s.Unlock()

	s.pods = append(s.pods, pod)
}

func (s *podset) Remove(pod *v1.Pod) {
	s.Lock()
	defer s.Unlock()

	newPods := []*v1.Pod{}
	for _, p := range s.pods {
		if p.Name != pod.Name {
			newPods = append(newPods, p)
		}
	}
	s.pods = newPods
}

func (s *podset) Update(pod *v1.Pod) {
	s.Lock()
	defer s.Unlock()

	newPods := []*v1.Pod{pod}
	for _, p := range s.pods {
		if p.Name != pod.Name {
			newPods = append(newPods, p)
		}
	}
	s.pods = newPods
}

func (s *podset) OfPhase(phases ...v1.PodPhase) []*v1.Pod {
	s.RLock()
	defer s.RUnlock()

	result := []*v1.Pod{}
	for _, p := range s.pods {
		for _, phase := range phases {
			if p.Status.Phase == phase {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

func (s *podset) Expired() []*v1.Pod {
	running := s.OfPhase(v1.PodRunning)

	s.RLock()
	defer s.RUnlock()

	expired := []*v1.Pod{}
	for _, pod := range running {
		// Compute elapsed wall time since pod started running
		// using the lastTransitionTime of the PodRunning pod condition.
		//
		// Compare running time against the per-pod run duration
		//
		// If the pod has been in running phase longer than the
		// desired duration, emit it in the result
		for _, c := range pod.Status.Conditions {
			if c.Type == v1.PodReady {
				ready := c.LastTransitionTime
				deadline := metav1.NewTime(ready.Add(RunDuration(pod)))
				now := metav1.Now()
				if deadline.Before(&now) {
					expired = append(expired, pod)
				}
			}
		}
	}
	return expired
}
