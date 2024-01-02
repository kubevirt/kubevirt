package operands

import (
	"errors"
	"reflect"

	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mtqv1alpha1 "kubevirt.io/managed-tenant-quota/staging/src/kubevirt.io/managed-tenant-quota-api/pkg/apis/core/v1alpha1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

type mtqOperand struct {
	operand *genericOperand
}

func (mtq mtqOperand) ensure(req *common.HcoRequest) *EnsureResult {
	if hcoutil.GetClusterInfo().IsInfrastructureHighlyAvailable() && // MTQ is not supported at a single node cluster
		req.Instance.Spec.FeatureGates.EnableManagedTenantQuota != nil && *req.Instance.Spec.FeatureGates.EnableManagedTenantQuota {
		// if the FG is set, make sure the MTQ CR is in place and up-to-date
		return mtq.operand.ensure(req)
	}

	return mtq.ensureDeleted(req)
}

func (mtq mtqOperand) ensureDeleted(req *common.HcoRequest) *EnsureResult {
	// if the FG is not set, make sure the MTQ CR does not exist
	cr := NewMTQWithNameOnly(req.Instance)
	res := NewEnsureResult(cr)
	res.SetName(cr.GetName())

	// hcoutil.EnsureDeleted does check that the MTQ CR exists before removing it. But it also writes a log message each
	// time it happens, i.e. for every reconcile loop. Assuming the client cache is up-to-date, we can safely get it here
	// with no meaningful performance cost.
	err := mtq.operand.Client.Get(req.Ctx, client.ObjectKeyFromObject(cr), cr)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return res.Error(err)
		}
	} else {
		deleted, err := hcoutil.EnsureDeleted(req.Ctx, mtq.operand.Client, cr, req.Instance.Name, req.Logger, false, false, true)
		if err != nil {
			return res.Error(err)
		}

		if deleted {
			res.SetDeleted()
			objectRef, err := reference.GetReference(mtq.operand.Scheme, cr)
			if err != nil {
				return res.Error(err)
			}

			if err = objectreferencesv1.RemoveObjectReference(&req.Instance.Status.RelatedObjects, *objectRef); err != nil {
				return res.Error(err)
			}
			req.StatusDirty = true
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}

func (mtq mtqOperand) reset() {
	mtq.operand.reset()
}

func newMtqHandler(Client client.Client, Scheme *runtime.Scheme) Operand {
	return &mtqOperand{
		operand: &genericOperand{
			Client: Client,
			Scheme: Scheme,
			crType: "MTQ",
			hooks:  &mtqHooks{},
		},
	}
}

type mtqHooks struct {
	cache *mtqv1alpha1.MTQ
}

func (h *mtqHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		h.cache = NewMTQ(hc)
	}
	return h.cache, nil
}

func (*mtqHooks) getEmptyCr() client.Object { return &mtqv1alpha1.MTQ{} }

func (*mtqHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*mtqv1alpha1.MTQ).Status.Conditions)
}

func (*mtqHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*mtqv1alpha1.MTQ)
	return checkComponentVersion(hcoutil.MtqVersionEnvV, found.Status.ObservedVersion)
}

func (h *mtqHooks) reset() {
	h.cache = nil
}

func (*mtqHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	mtq, ok1 := required.(*mtqv1alpha1.MTQ)
	found, ok2 := exists.(*mtqv1alpha1.MTQ)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to MTQ")
	}

	if !reflect.DeepEqual(found.Spec, mtq.Spec) ||
		!reflect.DeepEqual(found.Labels, mtq.Labels) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing MTQ's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated MTQ's Spec to its opinionated values")
			overwritten = true
		}
		hcoutil.DeepCopyLabels(&mtq.ObjectMeta, &found.ObjectMeta)
		mtq.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, overwritten, nil
	}
	return false, false, nil
}

func (*mtqHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewMTQ(hc *hcov1beta1.HyperConverged, opts ...string) *mtqv1alpha1.MTQ {
	spec := mtqv1alpha1.MTQSpec{
		ImagePullPolicy: corev1.PullIfNotPresent,
		CertConfig: &mtqv1alpha1.MTQCertConfig{
			CA: &mtqv1alpha1.CertConfig{
				Duration:    hc.Spec.CertConfig.CA.Duration,
				RenewBefore: hc.Spec.CertConfig.CA.RenewBefore,
			},
			Server: &mtqv1alpha1.CertConfig{
				Duration:    hc.Spec.CertConfig.Server.Duration,
				RenewBefore: hc.Spec.CertConfig.Server.RenewBefore,
			},
		},
		PriorityClass: ptr.To[mtqv1alpha1.MTQPriorityClass](kvPriorityClass),
	}

	if hc.Spec.Infra.NodePlacement != nil {
		hc.Spec.Infra.NodePlacement.DeepCopyInto(&spec.Infra)
	}

	if hc.Spec.Workloads.NodePlacement != nil {
		hc.Spec.Workloads.NodePlacement.DeepCopyInto(&spec.Workloads)
	}

	mtq := NewMTQWithNameOnly(hc, opts...)
	mtq.Spec = spec

	return mtq
}

func NewMTQWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *mtqv1alpha1.MTQ {
	return &mtqv1alpha1.MTQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mtq-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentMultiTenant),
			Namespace: getNamespace(hcoutil.UndefinedNamespace, opts),
		},
	}
}
