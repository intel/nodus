package node

import (
	"time"

	"k8s.io/api/core/v1"
)

const PodPhaseLabel = "np.terminalPhase"
const PodDurationLabel = "np.runDuration"

// Returns the specified terminal phase as declared in a well-known pod label.
// If left unset or the value does not match a known terminal phase, defaults
// to "Succeeded".
func TerminalPhase(pod *v1.Pod) v1.PodPhase {
	phase := v1.PodPhase(pod.ObjectMeta.Labels[PodPhaseLabel])
	if phase == v1.PodFailed {
		return v1.PodFailed
	}
	return v1.PodSucceeded
}

// Returns the specified run duration as declared in a well-known pod label.
// If left unset or the value cannot be parsed as a duration, defaults
// to 1 second.
func RunDuration(pod *v1.Pod) time.Duration {
	raw, ok := pod.ObjectMeta.Labels[PodDurationLabel]
	if !ok {
		return time.Second
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return time.Second
	}
	return d
}
