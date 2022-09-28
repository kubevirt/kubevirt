package operands

import (
	"errors"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ttov1alpha1 "github.com/kubevirt/tekton-tasks-operator/api/v1alpha1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

type ttoHandler genericOperand

func newTtoHandler(Client client.Client, Scheme *runtime.Scheme) *ttoHandler {
	ttoHandler := &ttoHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "TektonTasks",
		setControllerReference: true,
		hooks:                  &ttoHooks{},
	}
	return ttoHandler
}

type ttoHooks struct {
	cache *ttov1alpha1.TektonTasks
}

func (h *ttoHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		h.cache = NewTTO(hc)
	}
	return h.cache, nil
}

func (*ttoHooks) getEmptyCr() client.Object { return &ttov1alpha1.TektonTasks{} }

func (*ttoHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*ttov1alpha1.TektonTasks).Status.Conditions)
}
func (*ttoHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*ttov1alpha1.TektonTasks)
	return checkComponentVersion(hcoutil.TtoVersionEnvV, found.Status.ObservedVersion)
}
func (h *ttoHooks) reset() {
	h.cache = nil
}

func (*ttoHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func (*ttoHooks) updateCr(req *common.HcoRequest, client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	tto, ok1 := required.(*ttov1alpha1.TektonTasks)
	found, ok2 := exists.(*ttov1alpha1.TektonTasks)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to TTO")
	}
	if !reflect.DeepEqual(found.Spec, tto.Spec) ||
		!reflect.DeepEqual(found.Labels, tto.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing TTO's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated TTO's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&tto.ObjectMeta, &found.ObjectMeta)
		tto.Spec.DeepCopyInto(&found.Spec)
		err := client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewTTO(hc *hcov1beta1.HyperConverged, opts ...string) *ttov1alpha1.TektonTasks {
	pipelinesNamespace := getNamespace(hc.Namespace, opts)
	if hc.Spec.TektonPipelinesNamespace != nil {
		pipelinesNamespace = *hc.Spec.TektonPipelinesNamespace
	}

	spec := ttov1alpha1.TektonTasksSpec{
		FeatureGates: ttov1alpha1.FeatureGates{
			DeployTektonTaskResources: hc.Spec.FeatureGates.DeployTektonTaskResources,
		},
		Pipelines: ttov1alpha1.Pipelines{
			Namespace: pipelinesNamespace,
		},
	}

	tto := NewTTOWithNameOnly(hc)
	tto.Spec = spec

	return tto
}

func NewTTOWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *ttov1alpha1.TektonTasks {
	return &ttov1alpha1.TektonTasks{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tto-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentTekton),
			Namespace: getNamespace(hc.Namespace, opts),
		},
	}
}
