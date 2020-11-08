package operands

import (
	"fmt"
	"reflect"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	sspv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	commonTemplatesBundleOldCrdName = "kubevirtcommontemplatesbundles.kubevirt.io"
	metricsAggregationOldCrdName    = "kubevirtmetricsaggregations.kubevirt.io"
	nodeLabellerBundlesOldCrdName   = "kubevirtnodelabellerbundles.kubevirt.io"
	templateValidatorsOldCrdName    = "kubevirttemplatevalidators.kubevirt.io"
)

type sspOperand struct {
	genericOperand
	shouldRemoveOldCrd bool
}
type commonTemplateBundleHandler sspOperand

func newCommonTemplateBundleHandler(clt client.Client, scheme *runtime.Scheme) Operand {
	return &commonTemplateBundleHandler{
		genericOperand:     genericOperand{Client: clt, Scheme: scheme},
		shouldRemoveOldCrd: true,
	}
}

func (h *commonTemplateBundleHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	kvCTB := req.Instance.NewKubeVirtCommonTemplateBundle()
	res := NewEnsureResult(kvCTB)
	// todo if !r.clusterInfo.IsOpenshift() { // SSP operators Only supported in OpenShift. Ignore in K8s.
	//    return res.SetUpgradeDone(true)
	//}

	key, err := client.ObjectKeyFromObject(kvCTB)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for KubeVirt Common Templates Bundle")
	}

	res.SetName(key.Name)
	found := &sspv1.KubevirtCommonTemplatesBundle{}

	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating KubeVirt Common Templates Bundle")
			err = h.Client.Create(req.Ctx, kvCTB)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	existingOwners := found.GetOwnerReferences()

	// Previous versions used to have HCO-operator (namespace: kubevirt-hyperconverged)
	// as the owner of kvCTB (namespace: OpenshiftNamespace).
	// It's not legal, so remove that.
	if len(existingOwners) > 0 {
		req.Logger.Info("kvCTB has owners, removing...")
		found.SetOwnerReferences([]metav1.OwnerReference{})
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			req.Logger.Error(err, "Failed to remove kvCTB's previous owners")
		}
	}

	req.Logger.Info("KubeVirt Common Templates Bundle already exists", "bundle.Namespace", found.Namespace, "bundle.Name", found.Name)

	if !reflect.DeepEqual(kvCTB.Spec, found.Spec) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Common Templates Bundle's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Common Templates Bundle's Spec to its opinionated values")
			overwritten = true
		}
		kvCTB.Spec.DeepCopyInto(&found.Spec)
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		if overwritten {
			res.SetOverwritten()
		}
		return res.SetUpdated()
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	isReady := handleComponentConditions(req, "KubevirtCommonTemplatesBundle", found.Status.Conditions)

	upgradeInProgress := false
	if isReady {
		upgradeInProgress = req.UpgradeMode && checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
		if (upgradeInProgress || !req.UpgradeMode) && h.shouldRemoveOldCrd {
			if removeCrd(h.Client, req, commonTemplatesBundleOldCrdName) {
				h.shouldRemoveOldCrd = false
			}
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress && upgradeInProgress)
}

type nodeLabellerBundleHandler sspOperand

func newNodeLabellerBundleHandler(clt client.Client, scheme *runtime.Scheme) Operand {
	return &nodeLabellerBundleHandler{
		genericOperand:     genericOperand{Client: clt, Scheme: scheme},
		shouldRemoveOldCrd: true,
	}
}

func (h *nodeLabellerBundleHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	kvNLB := NewKubeVirtNodeLabellerBundleForCR(req.Instance, req.Namespace)
	res := NewEnsureResult(kvNLB)
	// todo if !r.clusterInfo.IsOpenshift() { // SSP operators Only supported in OpenShift. Ignore in K8s.
	//    return res.SetUpgradeDone(true)
	//}

	if err := controllerutil.SetControllerReference(req.Instance, kvNLB, h.Scheme); err != nil {
		return res.Error(err)
	}

	key, err := client.ObjectKeyFromObject(kvNLB)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for KubeVirt Node Labeller Bundle")
	}

	res.SetName(key.Name)
	found := &sspv1.KubevirtNodeLabellerBundle{}

	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating KubeVirt Node Labeller Bundle")
			err = h.Client.Create(req.Ctx, kvNLB)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	req.Logger.Info("KubeVirt Node Labeller Bundle already exists", "bundle.Namespace", found.Namespace, "bundle.Name", found.Name)

	if !reflect.DeepEqual(kvNLB.Spec, found.Spec) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Node Labeller Bundle's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Node Labeller Bundle's Spec to its opinionated values")
			overwritten = true
		}
		kvNLB.Spec.DeepCopyInto(&found.Spec)
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		if overwritten {
			res.SetOverwritten()
		}
		return res.SetUpdated()
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	isReady := handleComponentConditions(req, "KubevirtNodeLabellerBundle", found.Status.Conditions)

	upgradeInProgress := false
	if isReady {
		upgradeInProgress = req.UpgradeMode && checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
		if (upgradeInProgress || !req.UpgradeMode) && h.shouldRemoveOldCrd {
			if removeCrd(h.Client, req, nodeLabellerBundlesOldCrdName) {
				h.shouldRemoveOldCrd = false
			}
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress && upgradeInProgress)
}

func NewKubeVirtNodeLabellerBundleForCR(cr *hcov1beta1.HyperConverged, namespace string) *sspv1.KubevirtNodeLabellerBundle {
	labels := map[string]string{
		hcoutil.AppLabel: cr.Name,
	}

	spec := sspv1.ComponentSpec{
		// UseKVM: isKVMAvailable(),
	}

	if cr.Spec.Workloads.NodePlacement != nil {
		if cr.Spec.Workloads.NodePlacement.Affinity != nil {
			cr.Spec.Workloads.NodePlacement.Affinity.DeepCopyInto(&spec.Affinity)
		}

		if cr.Spec.Workloads.NodePlacement.NodeSelector != nil {
			spec.NodeSelector = make(map[string]string)
			for k, v := range cr.Spec.Workloads.NodePlacement.NodeSelector {
				spec.NodeSelector[k] = v
			}
		}

		for _, hcoTolr := range cr.Spec.Workloads.NodePlacement.Tolerations {
			nlbTolr := corev1.Toleration{}
			hcoTolr.DeepCopyInto(&nlbTolr)
			spec.Tolerations = append(spec.Tolerations, nlbTolr)
		}
	}

	return &sspv1.KubevirtNodeLabellerBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-labeller-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

type templateValidatorHandler sspOperand

func newTemplateValidatorHandler(clt client.Client, scheme *runtime.Scheme) Operand {
	return &templateValidatorHandler{
		genericOperand:     genericOperand{Client: clt, Scheme: scheme},
		shouldRemoveOldCrd: true,
	}
}

func (h *templateValidatorHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	kvTV := NewKubeVirtTemplateValidatorForCR(req.Instance, req.Namespace)
	res := NewEnsureResult(kvTV)
	// todo if !r.clusterInfo.IsOpenshift() { // SSP operators Only supported in OpenShift. Ignore in K8s.
	//    return res.SetUpgradeDone(true)
	//}

	if err := controllerutil.SetControllerReference(req.Instance, kvTV, h.Scheme); err != nil {
		return res.Error(err)
	}

	key, err := client.ObjectKeyFromObject(kvTV)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for KubeVirt Template Validator")
	}
	res.SetName(key.Name)

	found := &sspv1.KubevirtTemplateValidator{}
	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating KubeVirt Template Validator")
			err = h.Client.Create(req.Ctx, kvTV)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	req.Logger.Info("KubeVirt Template Validator already exists", "validator.Namespace", found.Namespace, "validator.Name", found.Name)

	if !reflect.DeepEqual(kvTV.Spec, found.Spec) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Template Validator's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Template Validator's Spec to its opinionated values")
			overwritten = true
		}
		kvTV.Spec.DeepCopyInto(&found.Spec)
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		if overwritten {
			res.SetOverwritten()
		}
		return res.SetUpdated()
	}
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	isReady := handleComponentConditions(req, "KubevirtTemplateValidator", found.Status.Conditions)

	upgradeInProgress := false
	if isReady {
		upgradeInProgress = req.UpgradeMode && checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
		if (upgradeInProgress || !req.UpgradeMode) && h.shouldRemoveOldCrd {
			if removeCrd(h.Client, req, templateValidatorsOldCrdName) {
				h.shouldRemoveOldCrd = false
			}
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress && upgradeInProgress)
}

func NewKubeVirtTemplateValidatorForCR(cr *hcov1beta1.HyperConverged, namespace string) *sspv1.KubevirtTemplateValidator {
	labels := map[string]string{
		hcoutil.AppLabel: cr.Name,
	}

	spec := sspv1.TemplateValidatorSpec{}
	if cr.Spec.Infra.NodePlacement != nil {
		if cr.Spec.Infra.NodePlacement.Affinity != nil {
			cr.Spec.Infra.NodePlacement.Affinity.DeepCopyInto(&spec.Affinity)
		}

		if cr.Spec.Infra.NodePlacement.NodeSelector != nil {
			spec.NodeSelector = make(map[string]string)
			for k, v := range cr.Spec.Infra.NodePlacement.NodeSelector {
				spec.NodeSelector[k] = v
			}
		}

		for _, hcoTolr := range cr.Spec.Infra.NodePlacement.Tolerations {
			tvTolr := corev1.Toleration{}
			hcoTolr.DeepCopyInto(&tvTolr)
			spec.Tolerations = append(spec.Tolerations, tvTolr)
		}
	}

	return &sspv1.KubevirtTemplateValidator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template-validator-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

type metricsAggregationHandler sspOperand

func newMetricsAggregationHandler(clt client.Client, scheme *runtime.Scheme) Operand {
	return &metricsAggregationHandler{
		genericOperand:     genericOperand{Client: clt, Scheme: scheme},
		shouldRemoveOldCrd: true,
	}
}

func (h *metricsAggregationHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	kubevirtMetricsAggregation := NewKubeVirtMetricsAggregationForCR(req.Instance, req.Namespace)
	res := NewEnsureResult(kubevirtMetricsAggregation)
	// todo if !r.clusterInfo.IsOpenshift() { // SSP operators Only supported in OpenShift. Ignore in K8s.
	//    return res.SetUpgradeDone(true)
	//}

	err := controllerutil.SetControllerReference(req.Instance, kubevirtMetricsAggregation, h.Scheme)
	if err != nil {
		return res.Error(err)
	}

	key, err := client.ObjectKeyFromObject(kubevirtMetricsAggregation)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for KubeVirt Metrics Aggregation")
	}

	res.SetName(key.Name)
	found := &sspv1.KubevirtMetricsAggregation{}

	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating KubeVirt Metrics Aggregation")
			err = h.Client.Create(req.Ctx, kubevirtMetricsAggregation)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	req.Logger.Info("KubeVirt Metrics Aggregation already exists", "metrics.Namespace", found.Namespace, "metrics.Name", found.Name)

	if !reflect.DeepEqual(kubevirtMetricsAggregation.Spec, found.Spec) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Metrics Aggregation's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Metrics Aggregation's Spec to its opinionated values")
			overwritten = true
		}
		kubevirtMetricsAggregation.Spec.DeepCopyInto(&found.Spec)
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		if overwritten {
			res.SetOverwritten()
		}
		return res.SetUpdated()
	}
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	isReady := handleComponentConditions(req, "KubeVirtMetricsAggregation", found.Status.Conditions)

	upgradeInProgress := false
	if isReady {
		upgradeInProgress = req.UpgradeMode && checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
		if (upgradeInProgress || !req.UpgradeMode) && h.shouldRemoveOldCrd {
			if removeCrd(h.Client, req, metricsAggregationOldCrdName) {
				h.shouldRemoveOldCrd = false
			}
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress && upgradeInProgress)
}

func NewKubeVirtMetricsAggregationForCR(cr *hcov1beta1.HyperConverged, namespace string) *sspv1.KubevirtMetricsAggregation {
	labels := map[string]string{
		hcoutil.AppLabel: cr.Name,
	}
	return &sspv1.KubevirtMetricsAggregation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-aggregation-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

// return true if not found or if deletion succeeded
func removeCrd(clt client.Client, req *common.HcoRequest, crdName string) bool {
	found := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "CustomResourceDefinition",
			"apiVersion": "apiextensions.k8s.io/v1",
		},
	}
	key := client.ObjectKey{Namespace: req.Namespace, Name: crdName}
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
