package operands

import (
	"errors"
	"fmt"
	"reflect"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
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
	oldCrdName         string
}

func (handler *sspOperand) ensure(req *common.HcoRequest) *EnsureResult {
	res := handler.genericOperand.ensure(req)
	if handler.shouldRemoveOldCrd && (!req.UpgradeMode || res.Updated) {
		if removeCrd(handler.Client, req, handler.oldCrdName) {
			handler.shouldRemoveOldCrd = false
		}
	}

	return res
}

// *************  KubeVirt Common Template Bundle  *************
type commonTemplateBundleHandler sspOperand

func newCommonTemplateBundleHandler(clt client.Client, scheme *runtime.Scheme) *commonTemplateBundleHandler {
	return &commonTemplateBundleHandler{
		genericOperand: genericOperand{
			Client: clt,
			Scheme: scheme,
			crType: "KubeVirtCommonTemplatesBundle",
			// Previous versions used to have HCO-operator (namespace: kubevirt-hyperconverged)
			// as the owner of kvCTB (namespace: OpenshiftNamespace).
			// It's not legal, so remove that.
			removeExistingOwner:    true,
			isCr:                   true,
			setControllerReference: false,
			hooks:                  &commonTemplateBundleHooks{},
		},
		shouldRemoveOldCrd: true,
		oldCrdName:         commonTemplatesBundleOldCrdName,
	}
}

type commonTemplateBundleHooks struct{}

func (h commonTemplateBundleHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewKubeVirtCommonTemplateBundle(hc)
}
func (h commonTemplateBundleHooks) getEmptyCr() runtime.Object {
	return &sspv1.KubevirtCommonTemplatesBundle{}
}
func (h commonTemplateBundleHooks) validate() error                                    { return nil }
func (h commonTemplateBundleHooks) postFound(*common.HcoRequest, runtime.Object) error { return nil }
func (h commonTemplateBundleHooks) getConditions(cr runtime.Object) []conditionsv1.Condition {
	return cr.(*sspv1.KubevirtCommonTemplatesBundle).Status.Conditions
}
func (h commonTemplateBundleHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1.KubevirtCommonTemplatesBundle)
	return checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h commonTemplateBundleHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*sspv1.KubevirtCommonTemplatesBundle).ObjectMeta
}

func (h *commonTemplateBundleHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	kvCTB, ok1 := required.(*sspv1.KubevirtCommonTemplatesBundle)
	found, ok2 := exists.(*sspv1.KubevirtCommonTemplatesBundle)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Kubevirt Common Templates Bundle")
	}
	if !reflect.DeepEqual(kvCTB.Spec, found.Spec) ||
		!reflect.DeepEqual(kvCTB.Labels, found.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Common Templates Bundle's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Common Templates Bundle's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&kvCTB.ObjectMeta, &found.ObjectMeta)
		kvCTB.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (h commonTemplateBundleHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	handler := sspOperand(h)
	return handler.ensure(req)
}

func NewKubeVirtCommonTemplateBundle(hc *hcov1beta1.HyperConverged, opts ...string) *sspv1.KubevirtCommonTemplatesBundle {
	return &sspv1.KubevirtCommonTemplatesBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "common-templates-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentSchedule),
			Namespace: getNamespace(hcoutil.OpenshiftNamespace, opts),
		},
	}
}

// *************  KubeVirt Node Labeller Bundle  *************
type nodeLabellerBundleHandler sspOperand

func newNodeLabellerBundleHandler(clt client.Client, scheme *runtime.Scheme) *nodeLabellerBundleHandler {
	return &nodeLabellerBundleHandler{
		genericOperand: genericOperand{
			Client:                 clt,
			Scheme:                 scheme,
			crType:                 "KubeVirtNodeLabellerBundle",
			removeExistingOwner:    false,
			isCr:                   true,
			setControllerReference: true,
			hooks:                  &nodeLabellerBundleHooks{},
		},
		shouldRemoveOldCrd: true,
		oldCrdName:         nodeLabellerBundlesOldCrdName,
	}
}

type nodeLabellerBundleHooks struct{}

func (h nodeLabellerBundleHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewKubeVirtNodeLabellerBundleForCR(hc, hc.Namespace)
}
func (h nodeLabellerBundleHooks) getEmptyCr() runtime.Object {
	return &sspv1.KubevirtNodeLabellerBundle{}
}
func (h nodeLabellerBundleHooks) validate() error                                        { return nil }
func (h nodeLabellerBundleHooks) postFound(_ *common.HcoRequest, _ runtime.Object) error { return nil }
func (h nodeLabellerBundleHooks) getConditions(cr runtime.Object) []conditionsv1.Condition {
	return cr.(*sspv1.KubevirtNodeLabellerBundle).Status.Conditions
}
func (h nodeLabellerBundleHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1.KubevirtNodeLabellerBundle)
	return checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h nodeLabellerBundleHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*sspv1.KubevirtNodeLabellerBundle).ObjectMeta
}

func (h *nodeLabellerBundleHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	kvNLB, ok1 := required.(*sspv1.KubevirtNodeLabellerBundle)
	found, ok2 := exists.(*sspv1.KubevirtNodeLabellerBundle)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to KubeVirt Node Labeller Bundle")
	}
	if !reflect.DeepEqual(kvNLB.Spec, found.Spec) ||
		!reflect.DeepEqual(kvNLB.Labels, found.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Node Labeller Bundle's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Node Labeller Bundle's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&kvNLB.ObjectMeta, &found.ObjectMeta)
		kvNLB.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}

		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (h nodeLabellerBundleHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	handler := sspOperand(h)
	return handler.ensure(req)
}

func NewKubeVirtNodeLabellerBundleForCR(cr *hcov1beta1.HyperConverged, namespace string) *sspv1.KubevirtNodeLabellerBundle {
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
			Labels:    getLabels(cr, hcoutil.AppComponentSchedule),
			Namespace: namespace,
		},
		Spec: spec,
	}
}

// *************  KubeVirt Template Validator  *************
type templateValidatorHandler sspOperand

func newTemplateValidatorHandler(clt client.Client, scheme *runtime.Scheme) *templateValidatorHandler {
	return &templateValidatorHandler{
		genericOperand: genericOperand{
			Client:                 clt,
			Scheme:                 scheme,
			crType:                 "KubevirtTemplateValidator",
			removeExistingOwner:    false,
			isCr:                   true,
			setControllerReference: true,
			hooks:                  &templateValidatorHooks{},
		},
		shouldRemoveOldCrd: true,
		oldCrdName:         templateValidatorsOldCrdName,
	}
}

type templateValidatorHooks struct{}

func (h templateValidatorHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewKubeVirtTemplateValidatorForCR(hc, hc.Namespace)
}
func (h templateValidatorHooks) getEmptyCr() runtime.Object {
	return &sspv1.KubevirtTemplateValidator{}
}
func (h templateValidatorHooks) validate() error                                        { return nil }
func (h templateValidatorHooks) postFound(_ *common.HcoRequest, _ runtime.Object) error { return nil }
func (h templateValidatorHooks) getConditions(cr runtime.Object) []conditionsv1.Condition {
	return cr.(*sspv1.KubevirtTemplateValidator).Status.Conditions
}
func (h templateValidatorHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1.KubevirtTemplateValidator)
	return checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h templateValidatorHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*sspv1.KubevirtTemplateValidator).ObjectMeta
}

func (h *templateValidatorHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	kvTV, ok1 := required.(*sspv1.KubevirtTemplateValidator)
	found, ok2 := exists.(*sspv1.KubevirtTemplateValidator)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to KubeVirt Template Validator")
	}

	if !reflect.DeepEqual(kvTV.Spec, found.Spec) ||
		!reflect.DeepEqual(kvTV.Labels, found.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Template Validator's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Template Validator's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&kvTV.ObjectMeta, &found.ObjectMeta)
		kvTV.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (h templateValidatorHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	handler := sspOperand(h)
	return handler.ensure(req)
}

func NewKubeVirtTemplateValidatorForCR(cr *hcov1beta1.HyperConverged, namespace string) *sspv1.KubevirtTemplateValidator {
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
			Labels:    getLabels(cr, hcoutil.AppComponentSchedule),
			Namespace: namespace,
		},
		Spec: spec,
	}
}

// *************  KubeVirt Metrics Aggregation  *************
type metricsAggregationHandler sspOperand

func newMetricsAggregationHandler(clt client.Client, scheme *runtime.Scheme) *metricsAggregationHandler {
	return &metricsAggregationHandler{
		genericOperand: genericOperand{
			Client:                 clt,
			Scheme:                 scheme,
			crType:                 "KubevirtMetricsAggregation",
			removeExistingOwner:    false,
			isCr:                   true,
			setControllerReference: true,
			hooks:                  &metricsAggregationHooks{},
		},
		shouldRemoveOldCrd: true,
		oldCrdName:         metricsAggregationOldCrdName,
	}
}

type metricsAggregationHooks struct{}

func (h metricsAggregationHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewKubeVirtMetricsAggregationForCR(hc, hc.Namespace)
}
func (h metricsAggregationHooks) getEmptyCr() runtime.Object {
	return &sspv1.KubevirtMetricsAggregation{}
}
func (h metricsAggregationHooks) validate() error                                        { return nil }
func (h metricsAggregationHooks) postFound(_ *common.HcoRequest, _ runtime.Object) error { return nil }
func (h metricsAggregationHooks) getConditions(cr runtime.Object) []conditionsv1.Condition {
	return cr.(*sspv1.KubevirtMetricsAggregation).Status.Conditions
}
func (h metricsAggregationHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1.KubevirtMetricsAggregation)
	return checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h metricsAggregationHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*sspv1.KubevirtMetricsAggregation).ObjectMeta
}

func (h *metricsAggregationHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	kubevirtMetricsAggregation, ok1 := required.(*sspv1.KubevirtMetricsAggregation)
	found, ok2 := exists.(*sspv1.KubevirtMetricsAggregation)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to KubeVirt Metrics Aggregation")
	}

	if !reflect.DeepEqual(kubevirtMetricsAggregation.Spec, found.Spec) ||
		!reflect.DeepEqual(kubevirtMetricsAggregation.Labels, found.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt Template Validator's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt Template Validator's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&kubevirtMetricsAggregation.ObjectMeta, &found.ObjectMeta)
		kubevirtMetricsAggregation.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
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
	return &sspv1.KubevirtMetricsAggregation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-aggregation-" + cr.Name,
			Labels:    getLabels(cr, hcoutil.AppComponentSchedule),
			Namespace: namespace,
		},
	}
}

// *************  Common Methods  *************

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
