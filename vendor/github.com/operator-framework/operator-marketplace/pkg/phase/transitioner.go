package phase

import (
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

// NewTransitionerWithClock returns a new Transitioner with the given clock.
// This function can be used for unit testing Transitioner.
func NewTransitionerWithClock(clock clock.Clock) Transitioner {
	return &transitioner{
		clock: clock,
	}
}

// NewTransitioner returns a new PhaseTransitioner with the default RealClock.
func NewTransitioner() Transitioner {
	clock := &clock.RealClock{}
	return NewTransitionerWithClock(clock)
}

// Transitioner is an interface that wraps the TransitionInto method
//
// TransitionInto transitions the OperatorSource object into the specified
// next phase. If the currentPhase is nil, the function returns false to
// indicate no transition took place. If the currentPhase has the same phase and
// message specified in next phase, then the function returns false to indicate
// no transition took place. If a new phase is being set then LastTransitionTime
// is set appropriately, otherwise it is left untouched.
type Transitioner interface {
	TransitionInto(currentPhase *marketplace.ObjectPhase, nextPhase *marketplace.Phase) (changed bool)
}

// transitioner implements Transitioner interface.
type transitioner struct {
	clock clock.Clock
}

func (t *transitioner) TransitionInto(currentPhase *marketplace.ObjectPhase, nextPhase *marketplace.Phase) (changed bool) {
	if currentPhase == nil || nextPhase == nil {
		return false
	}

	if !hasPhaseChanged(currentPhase, nextPhase) {
		return false
	}

	now := metav1.NewTime(t.clock.Now())
	currentPhase.LastUpdateTime = now
	currentPhase.Message = nextPhase.Message

	if currentPhase.Name != nextPhase.Name {
		currentPhase.LastTransitionTime = now
		currentPhase.Name = nextPhase.Name
	}

	return true
}

// hasPhaseChanged returns true if the current phase specified in nextPhase
// has changed from that of the currentPhase.
//
// If both Phase and Message are equal, the function will return false
// indicating no change. Otherwise, the function will return true.
func hasPhaseChanged(currentPhase *marketplace.ObjectPhase, nextPhase *marketplace.Phase) bool {
	if currentPhase.Name == nextPhase.Name && currentPhase.Message == nextPhase.Message {
		return false
	}

	return true
}
