package matcher

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func HaveRevisionNames() types.GomegaMatcher {
	return And(
		HaveInstancetypeRevisionName(),
		HavePreferenceRevisionName(),
	)
}

func HaveInstancetypeRevisionName() types.GomegaMatcher {
	return And(
		HaveField("Spec.Instancetype", Not(BeNil())),
		HaveField("Spec.Instancetype.RevisionName", Not(BeEmpty())),
	)
}

func HavePreferenceRevisionName() types.GomegaMatcher {
	return And(
		HaveField("Spec.Preference", Not(BeNil())),
		HaveField("Spec.Preference.RevisionName", Not(BeEmpty())),
	)
}
