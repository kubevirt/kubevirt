package operands

import (
	"errors"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func newCmHandler(Client client.Client, Scheme *runtime.Scheme, required *corev1.ConfigMap) Operand {
	h := &genericOperand{
		Client:              Client,
		Scheme:              Scheme,
		crType:              "ConfigMap",
		removeExistingOwner: false,
		hooks:               &cmHooks{required: required},
	}

	return h
}

type cmHooks struct {
	required *corev1.ConfigMap
}

func (h cmHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}

func (h cmHooks) getEmptyCr() client.Object {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}

func (h cmHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*corev1.ConfigMap).ObjectMeta
}

func (h cmHooks) reset() { /* no implementation */ }

func (h cmHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	found, ok := exists.(*corev1.ConfigMap)

	if !ok {
		return false, false, errors.New("can't convert to Configmap")
	}

	if !reflect.DeepEqual(found.Data, h.required.Data) ||
		!reflect.DeepEqual(found.Labels, h.required.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Configmap to new opinionated values", "name", h.required.Name)
		} else {
			req.Logger.Info("Reconciling an externally updated Configmap to its opinionated values", "name", h.required.Name)
		}
		util.DeepCopyLabels(&h.required.ObjectMeta, &found.ObjectMeta)
		h.required.DeepCopyInto(found)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}
