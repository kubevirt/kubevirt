package root

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/roundtrip"
)

var groups = []runtime.SchemeBuilder{
	kubevirtv1.SchemeBuilder,
}

func TestCompatibility(t *testing.T) {
	scheme := runtime.NewScheme()

	for _, builder := range groups {
		require.NoError(t, builder.AddToScheme(scheme))
	}

	roundtrip.NewCompatibilityTestOptions(scheme).Complete(t).Run(t)
}
