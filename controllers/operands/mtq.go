package operands

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mtqv1alpha1 "kubevirt.io/managed-tenant-quota/staging/src/kubevirt.io/managed-tenant-quota-api/pkg/apis/core/v1alpha1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func newMtqHandler(Client client.Client, Scheme *runtime.Scheme) Operand {
	return &conditionalHandler{
		operand: &genericOperand{
			Client: Client,
			Scheme: Scheme,
			crType: "MTQ",
			hooks:  &mtqHooks{},
		},
		shouldDeploy: func(hc *hcov1beta1.HyperConverged) bool {
			return false
		},
		getCRWithName: func(hc *hcov1beta1.HyperConverged) client.Object {
			return NewMTQWithNameOnly(hc)
		},
	}
}

type mtqHooks struct{}

func (h *mtqHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewMTQWithNameOnly(hc), nil
}

func (*mtqHooks) getEmptyCr() client.Object { return &mtqv1alpha1.MTQ{} }

func (*mtqHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return nil
}

func (*mtqHooks) checkComponentVersion(cr runtime.Object) bool {
	return true
}

func (*mtqHooks) updateCr(_ *common.HcoRequest, _ client.Client, _ runtime.Object, _ runtime.Object) (bool, bool, error) {
	return false, false, nil
}

func (*mtqHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewMTQWithNameOnly(hc *hcov1beta1.HyperConverged) *mtqv1alpha1.MTQ {
	return &mtqv1alpha1.MTQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "mtq-" + hc.Name,
			Labels: getLabels(hc, hcoutil.AppComponentMultiTenant),
		},
	}
}
