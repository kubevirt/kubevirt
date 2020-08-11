package shared

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Phase describes the phase the object is in
type Phase struct {
	// Name of the phase
	Name string `json:"name,omitempty"`

	// A human readable message indicating why the object is in this phase
	Message string `json:"message,omitempty"`
}

// ObjectPhase describes the phase of a Marketplace object is in along with the
// last time a phase transition occurred and when the object was last updated
type ObjectPhase struct {
	// Current phase of the object
	Phase `json:"phase,omitempty"`

	// Last time the object has transitioned from one phase to another
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Last time the status of the object was updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}
