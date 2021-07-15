package operands

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// This is initially set to 2 replicas, to maintain the behavior of the previous SSP operator.
	// After SSP implements its defaulting webhook, we can change this to 0 replicas,
	// and let the webhook set the default.
	defaultTemplateValidatorReplicas = 2

	defaultCommonTemplatesNamespace = hcoutil.OpenshiftNamespace

	// These group are no longer supported. Use these constants to remove unused resources
	prevSspGroup = "ssp.kubevirt.io"
	origSspGroup = "kubevirt.io"
)

type sspHandler struct {
	genericOperand

	// Old SSP CRDs that need to be removed from the cluster.
	// Removal of the CRDs also leads to removal of its CRs.
	crdsToRemove []schema.GroupKind

	// Old SSP CRs that need to be removed from the list of related objects.
	// This list is maintained separately from the list of CRDs to remove,
	// so we can retry to delete them upon HCO status update failures, even
	// when the CRD itself has been successfully removed.
	relatedObjectsToRemove []schema.GroupKind
}

func newSspHandler(Client client.Client, Scheme *runtime.Scheme) *sspHandler {
	handler := &sspHandler{
		genericOperand: genericOperand{
			Client:                 Client,
			Scheme:                 Scheme,
			crType:                 "SSP",
			removeExistingOwner:    false,
			setControllerReference: false,
			hooks:                  &sspHooks{},
		},

		crdsToRemove: []schema.GroupKind{
			// These are the 2nd generation SSP CRDs,
			// where the group name has been changed to "ssp.kubevirt.io"
			{Group: prevSspGroup, Kind: "KubevirtCommonTemplatesBundle"},
			{Group: prevSspGroup, Kind: "KubevirtNodeLabellerBundle"},
			{Group: prevSspGroup, Kind: "KubevirtTemplateValidator"},
			{Group: prevSspGroup, Kind: "KubevirtMetricsAggregation"},

			// These are the original SSP CRDs, with the group name "kubevirt.io".
			// We attempt to remove these too, for upgrades from even older version.
			{Group: origSspGroup, Kind: "KubevirtCommonTemplatesBundle"},
			{Group: origSspGroup, Kind: "KubevirtNodeLabellerBundle"},
			{Group: origSspGroup, Kind: "KubevirtTemplateValidator"},
			{Group: origSspGroup, Kind: "KubevirtMetricsAggregation"},
		},
	}

	// The list of related objects to remove is initialized empty;
	// Once the corresponding CRD (and hence CR) is removed successfully,
	// the CR can be removed from the list of related objects.
	handler.relatedObjectsToRemove = make([]schema.GroupKind, 0, len(handler.crdsToRemove))

	return handler

}

func (handler *sspHandler) ensure(req *common.HcoRequest) *EnsureResult {
	res := handler.genericOperand.ensure(req)

	// Attempt to remove old CRDs
	if len(handler.crdsToRemove) > 0 && (!req.UpgradeMode || res.UpgradeDone) {
		removed, unremoved := removeCRDs(handler.Client, req, handler.crdsToRemove)

		// For CRDs that failed to remove, we'll retry in the next reconciliation loop.
		handler.crdsToRemove = unremoved

		// For CRDs that were successfully removed, we can proceed to remove
		// their corresponding entries from HCO.Status.RelatedObjects.
		handler.relatedObjectsToRemove = append(handler.relatedObjectsToRemove, removed...)
	}

	if len(handler.relatedObjectsToRemove) > 0 {
		removed := removeRelatedObjects(req, handler.relatedObjectsToRemove)

		// CRDs that were removed from the related objects list are kept around,
		// so that the next reconciliation loop can validate that they were actually removed,
		// and not lost in some failing status update.
		handler.relatedObjectsToRemove = removed
	}

	return res
}

type sspHooks struct {
	cache *sspv1beta1.SSP
}

func (h *sspHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		h.cache = NewSSP(hc)
	}
	return h.cache, nil
}
func (h sspHooks) getEmptyCr() client.Object { return &sspv1beta1.SSP{} }
func (h sspHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*sspv1beta1.SSP).Status.Conditions)
}
func (h sspHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1beta1.SSP)
	return checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h sspHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*sspv1beta1.SSP).ObjectMeta
}
func (h *sspHooks) reset() {
	h.cache = nil
}

func (h *sspHooks) updateCr(req *common.HcoRequest, client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	ssp, ok1 := required.(*sspv1beta1.SSP)
	found, ok2 := exists.(*sspv1beta1.SSP)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to SSP")
	}
	if !reflect.DeepEqual(found.Spec, ssp.Spec) ||
		!reflect.DeepEqual(found.Labels, ssp.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing SSP's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated SSP's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&ssp.ObjectMeta, &found.ObjectMeta)
		ssp.Spec.DeepCopyInto(&found.Spec)
		err := client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewSSP(hc *hcov1beta1.HyperConverged, opts ...string) *sspv1beta1.SSP {
	replicas := int32(defaultTemplateValidatorReplicas)
	templatesNamespace := defaultCommonTemplatesNamespace

	if hc.Spec.CommonTemplatesNamespace != nil {
		templatesNamespace = *hc.Spec.CommonTemplatesNamespace
	}

	spec := sspv1beta1.SSPSpec{
		TemplateValidator: sspv1beta1.TemplateValidator{
			Replicas: &replicas,
		},
		CommonTemplates: sspv1beta1.CommonTemplates{
			Namespace: templatesNamespace,
		},
		// NodeLabeller field is explicitly initialized to its zero-value,
		// in order to future-proof from bugs if SSP changes it to pointer-type,
		// causing nil pointers dereferences at the DeepCopyInto() below.
		NodeLabeller: sspv1beta1.NodeLabeller{},
	}

	if hc.Spec.Infra.NodePlacement != nil {
		spec.TemplateValidator.Placement = hc.Spec.Infra.NodePlacement.DeepCopy()
	}

	if hc.Spec.Workloads.NodePlacement != nil {
		spec.NodeLabeller.Placement = hc.Spec.Workloads.NodePlacement.DeepCopy()
	}

	return &sspv1beta1.SSP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ssp-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentSchedule),
			Namespace: getNamespace(hc.Namespace, opts),
		},
		Spec: spec,
	}
}

// returns a slice of CRDs that were successfully removed or don't exist, and a slice of CRDs that failed to remove.
func removeCRDs(clt client.Client, req *common.HcoRequest, crds []schema.GroupKind) ([]schema.GroupKind, []schema.GroupKind) {
	removed := make([]schema.GroupKind, 0, len(crds))
	unremoved := make([]schema.GroupKind, 0, len(crds))

	// The deletion is performed concurrently for all CRDs.
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(crds))

	for _, crd := range crds {
		go func(crd schema.GroupKind) {
			isRemoved := removeCRD(clt, req, crd)

			mutex.Lock()
			defer mutex.Unlock()

			if isRemoved {
				removed = append(removed, crd)
			} else {
				unremoved = append(unremoved, crd)
			}

			wg.Done()
		}(crd)
	}

	wg.Wait()

	return removed, unremoved
}

// returns true if not found or if deletion succeeded, and false otherwise.
func removeCRD(clt client.Client, req *common.HcoRequest, crd schema.GroupKind) bool {
	crdName := groupKindToCRDName(crd)
	found := &apiextensionsv1.CustomResourceDefinition{}
	key := client.ObjectKey{Namespace: hcoutil.UndefinedNamespace, Name: crdName}
	err := clt.Get(req.Ctx, key, found)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			req.Logger.Error(err, fmt.Sprintf("failed to read the %s CRD; %s", crdName, err.Error()))
			return false
		}
	} else {
		err = clt.Delete(req.Ctx, found)
		if err != nil {
			req.Logger.Error(err, fmt.Sprintf("failed to remove the %s CRD; %s", crdName, err.Error()))
			return false
		}

		req.Logger.Info("successfully removed CRD", "CRD Name", crdName)
	}

	return true
}

// This function creates a CRD name with the form <plural>.<group>,
// where <plural> is the lowercase plural form of the Kind.
// Note that this doesn't use any generic pluralization, so may not
// work for CRDs other than the old SSP CRDs we're trying to handle here.
func groupKindToCRDName(gk schema.GroupKind) string {
	plural := strings.ToLower(gk.Kind) + "s"
	return fmt.Sprintf("%s.%s", plural, gk.Group)
}

// Removes CRs of the given CRDs from the list of related objects in HCO status.
// Returns a list of CRDs that were removed from the related objects list.
func removeRelatedObjects(req *common.HcoRequest, crds []schema.GroupKind) []schema.GroupKind {
	// Collect CRD GKs into a set
	crdSet := make(map[schema.GroupKind]struct{})
	for _, crd := range crds {
		crdSet[crd] = struct{}{}
	}

	objRefsToRemove := make([]corev1.ObjectReference, 0, len(crds))
	removedCRDs := make([]schema.GroupKind, 0, len(crds))

	// Find related objects to be removed
	for _, objRef := range req.Instance.Status.RelatedObjects {
		objGK := objRef.GroupVersionKind().GroupKind()
		if _, exist := crdSet[objGK]; exist {
			objRefsToRemove = append(objRefsToRemove, objRef)
			removedCRDs = append(removedCRDs, objGK)
		}
	}

	// Remove from related objects
	if len(objRefsToRemove) > 0 {
		req.StatusDirty = true
		for _, objRef := range objRefsToRemove {
			err := objectreferencesv1.RemoveObjectReference(&req.Instance.Status.RelatedObjects, objRef)
			if err != nil {
				// This shouldn't really happen, but...
				req.Logger.Error(err, "Failed removing object reference from HCO.Status.RelatedObjects",
					"ObjectReference.Name", objRef.Name,
					"ObjectReference.Namespace", objRef.Namespace,
					"ObjectReference.Kind", objRef.Kind,
					"ObjectReference.APIVersion", objRef.APIVersion)
			}
		}

	}

	return removedCRDs
}
