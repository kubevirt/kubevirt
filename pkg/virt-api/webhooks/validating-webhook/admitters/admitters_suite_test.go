package admitters

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/testutils"
)

func TestValidatingWebhook(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

type validatorStub struct {
	statusCauses []metav1.StatusCause
}

func (n validatorStub) ValidateCreation(_ *k8sfield.Path, _ *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	return n.statusCauses
}
