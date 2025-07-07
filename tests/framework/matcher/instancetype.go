package matcher

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func HaveControllerRevisionRefs() types.GomegaMatcher {
	return And(
		HaveInstancetypeControllerRevisionRef(),
		HavePreferenceControllerRevisionRef(),
	)
}

func HaveInstancetypeControllerRevisionRef() types.GomegaMatcher {
	return And(
		HaveField("Status.InstancetypeRef", Not(BeNil())),
		HaveField("Status.InstancetypeRef.ControllerRevisionRef", Not(BeNil())),
		HaveField("Status.InstancetypeRef.ControllerRevisionRef.Name", Not(BeEmpty())),
	)
}

func HavePreferenceControllerRevisionRef() types.GomegaMatcher {
	return And(
		HaveField("Status.PreferenceRef", Not(BeNil())),
		HaveField("Status.PreferenceRef.ControllerRevisionRef", Not(BeNil())),
		HaveField("Status.PreferenceRef.ControllerRevisionRef.Name", Not(BeEmpty())),
	)
}
