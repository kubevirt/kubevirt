package operands

import (
	"errors"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aaqv1alpha1 "kubevirt.io/applications-aware-quota/staging/src/kubevirt.io/applications-aware-quota-api/pkg/apis/core/v1alpha1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func newAAQHandler(Client client.Client, Scheme *runtime.Scheme) Operand {
	return &conditionalHandler{
		operand: &genericOperand{
			Client: Client,
			Scheme: Scheme,
			crType: "AAQ",
			hooks:  &aaqHooks{},
		},
		shouldDeploy: func(hc *hcov1beta1.HyperConverged) bool {
			return hc.Spec.ApplicationAwareConfig != nil
		},
		getCRWithName: func(hc *hcov1beta1.HyperConverged) client.Object {
			return NewAAQWithNameOnly(hc)
		},
	}
}

type aaqHooks struct {
	cache *aaqv1alpha1.AAQ
}

func (h *aaqHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		h.cache = NewAAQ(hc)
	}
	return h.cache, nil
}

func (*aaqHooks) getEmptyCr() client.Object { return &aaqv1alpha1.AAQ{} }

func (*aaqHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*aaqv1alpha1.AAQ).Status.Conditions)
}

func (*aaqHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*aaqv1alpha1.AAQ)
	return checkComponentVersion(hcoutil.AaqVersionEnvV, found.Status.ObservedVersion)
}

func (h *aaqHooks) reset() {
	h.cache = nil
}

func (*aaqHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	aaq, ok1 := required.(*aaqv1alpha1.AAQ)
	found, ok2 := exists.(*aaqv1alpha1.AAQ)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to AAQ")
	}

	if !reflect.DeepEqual(found.Spec, aaq.Spec) ||
		!reflect.DeepEqual(found.Labels, aaq.Labels) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing AAQ's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated AAQ's Spec to its opinionated values")
			overwritten = true
		}
		hcoutil.DeepCopyLabels(&aaq.ObjectMeta, &found.ObjectMeta)
		aaq.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, overwritten, nil
	}
	return false, false, nil
}

func (*aaqHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewAAQ(hc *hcov1beta1.HyperConverged) *aaqv1alpha1.AAQ {
	spec := aaqv1alpha1.AAQSpec{
		PriorityClass:   ptr.To[aaqv1alpha1.AAQPriorityClass](kvPriorityClass),
		ImagePullPolicy: corev1.PullIfNotPresent,
		CertConfig: &aaqv1alpha1.AAQCertConfig{
			CA: &aaqv1alpha1.CertConfig{
				Duration:    hc.Spec.CertConfig.CA.Duration,
				RenewBefore: hc.Spec.CertConfig.CA.RenewBefore,
			},
			Server: &aaqv1alpha1.CertConfig{
				Duration:    hc.Spec.CertConfig.Server.Duration,
				RenewBefore: hc.Spec.CertConfig.Server.RenewBefore,
			},
		},
	}

	if hc.Spec.Infra.NodePlacement != nil {
		hc.Spec.Infra.NodePlacement.DeepCopyInto(&spec.Infra)
	}

	if hc.Spec.Workloads.NodePlacement != nil {
		hc.Spec.Workloads.NodePlacement.DeepCopyInto(&spec.Workloads)
	}

	spec.Configuration.VmiCalculatorConfiguration.ConfigName = aaqv1alpha1.DedicatedVirtualResources
	if config := hc.Spec.ApplicationAwareConfig; config != nil {
		if config.VmiCalcConfigName != nil {
			spec.Configuration.VmiCalculatorConfiguration.ConfigName = *config.VmiCalcConfigName
		}

		if config.NamespaceSelector != nil {
			spec.NamespaceSelector = config.NamespaceSelector.DeepCopy()
		}

		spec.Configuration.EnableClusterAppsResourceQuota = config.EnableClusterAppsResourceQuota
	}

	aaq := NewAAQWithNameOnly(hc)
	aaq.Spec = spec
	return aaq
}

func NewAAQWithNameOnly(hc *hcov1beta1.HyperConverged) *aaqv1alpha1.AAQ {
	return &aaqv1alpha1.AAQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "aaq-" + hc.Name,
			Labels: getLabels(hc, hcoutil.AppComponentQuotaMngt),
		},
	}
}
