package operands

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func newServiceHandler(Client client.Client, Scheme *runtime.Scheme, required *corev1.Service) Operand {
	h := &genericOperand{
		Client:              Client,
		Scheme:              Scheme,
		crType:              "Service",
		removeExistingOwner: false,
		hooks:               &serviceHooks{required: required},
	}

	return h
}

type serviceHooks struct {
	required *corev1.Service
}

func (h serviceHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}

func (h serviceHooks) getEmptyCr() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}

func (h serviceHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*corev1.Service).ObjectMeta
}

func (h serviceHooks) reset() { /* no implementation */ }

func (h serviceHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func (h serviceHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	service, ok1 := required.(*corev1.Service)
	found, ok2 := exists.(*corev1.Service)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Service")
	}
	if !hasServiceRightFields(found, service) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Service Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated Service's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&service.ObjectMeta, &found.ObjectMeta)
		service.Spec.ClusterIP = found.Spec.ClusterIP
		service.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}
