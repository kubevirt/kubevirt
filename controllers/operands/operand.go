package operands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

type Operand interface {
	ensure(req *common.HcoRequest) *EnsureResult
	reset()
}

// Handles a specific resource (a CR, a configMap and so on), to be run during reconciliation
type genericOperand struct {
	// K8s client
	Client client.Client
	Scheme *runtime.Scheme
	// printable resource name
	crType string
	// Should the handler add the controller reference
	setControllerReference bool
	// Set of resource handler hooks, to be implemented in each handler
	hooks hcoResourceHooks
}

// Set of resource handler hooks, to be implement in each handler
type hcoResourceHooks interface {
	// Generate the required resource, with all the required fields)
	getFullCr(*hcov1beta1.HyperConverged) (client.Object, error)
	// Generate an empty resource, to be used as the input of the client.Get method. After calling this method, it will
	// contains the actual values in K8s.
	getEmptyCr() client.Object
	// check if there is a change between the required resource and the resource read from K8s, and update K8s accordingly.
	updateCr(*common.HcoRequest, client.Client, runtime.Object, runtime.Object) (bool, bool, error)
	// last hook before completing the operand handling
	justBeforeComplete(req *common.HcoRequest)
}

// Set of operand handler hooks, to be implement in each handler
type hcoOperandHooks interface {
	hcoResourceHooks
	// get the CR conditions, if exists
	getConditions(runtime.Object) []metav1.Condition
	// on upgrade mode, check if the CR is already with the expected version
	checkComponentVersion(runtime.Object) bool
}

type reseter interface {
	// reset handler cached, if exists
	reset()
}

func (h *genericOperand) ensure(req *common.HcoRequest) *EnsureResult {
	cr, err := h.hooks.getFullCr(req.Instance)
	if err != nil {
		return &EnsureResult{
			Err: err,
		}
	}

	res := NewEnsureResult(cr)

	if err := h.doSetControllerReference(req, cr); err != nil {
		return res.Error(err)
	}

	key := client.ObjectKeyFromObject(cr)
	res.SetName(key.Name)
	found := h.hooks.getEmptyCr()
	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			res = h.createNewCr(req, cr, res)
			if apierrors.IsAlreadyExists(res.Err) {
				// we failed trying to create it due to a caching error
				// or we neither tried because we know that the object is already there for sure,
				// but we cannot get it due to a bad cache hit.
				// Let's try updating it bypassing the client cache mechanism
				return h.handleExistingCrSkipCache(req, key, found, cr, res)
			}
		} else {
			return res.Error(err)
		}
	} else {
		res = h.handleExistingCr(req, key, found, cr, res)
	}

	if res.Err == nil {
		h.hooks.justBeforeComplete(req)
	}

	return res
}

func (h *genericOperand) handleExistingCr(req *common.HcoRequest, key client.ObjectKey, found client.Object, cr client.Object, res *EnsureResult) *EnsureResult {
	req.Logger.Info(h.crType+" already exists", h.crType+".Namespace", key.Namespace, h.crType+".Name", key.Name)

	updated, overwritten, err := h.hooks.updateCr(req, h.Client, found, cr)
	if err != nil {
		return res.Error(err)
	}

	if err = h.addCrToTheRelatedObjectList(req, found); err != nil {
		return res.Error(err)
	}

	if updated {
		// update resourceVersions of objects in relatedObjects
		req.StatusDirty = true
		return res.SetUpdated().SetOverwritten(overwritten)
	}

	if opr, ok := h.hooks.(hcoOperandHooks); ok { // for operands, perform some more checks
		return h.completeEnsureOperands(req, opr, found, res)
	}
	// For resources that are not CRs, such as priority classes or a config map, there is no new version to upgrade
	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}

func (h *genericOperand) handleExistingCrSkipCache(req *common.HcoRequest, key client.ObjectKey, found client.Object, cr client.Object, res *EnsureResult) *EnsureResult {
	cfg, configerr := config.GetConfig()
	if configerr != nil {
		req.Logger.Error(configerr, "failed creating a config for a custom client")
		return &EnsureResult{
			Err: configerr,
		}
	}
	apiClient, acerr := client.New(cfg, client.Options{
		Scheme: h.Scheme,
	})
	if acerr != nil {
		req.Logger.Error(acerr, "failed creating a custom client to bypass the cache")
		return &EnsureResult{
			Err: acerr,
		}
	}
	geterr := apiClient.Get(req.Ctx, key, found)
	if geterr != nil {
		req.Logger.Error(geterr, "failed trying to get the object bypassing the cache")
		return &EnsureResult{
			Err: geterr,
		}
	}
	originalClient := h.Client
	// this is not exactly thread safe,
	// but we are not supposed to call twice in parallel
	// the handler for a single CR
	h.Client = apiClient
	existingcrresult := h.handleExistingCr(req, key, found, cr, res)
	h.Client = originalClient
	return existingcrresult
}

func (h *genericOperand) completeEnsureOperands(req *common.HcoRequest, opr hcoOperandHooks, found client.Object, res *EnsureResult) *EnsureResult {
	// Handle KubeVirt resource conditions
	isReady := handleComponentConditions(req, h.crType, opr.getConditions(found))

	versionUpdated := opr.checkComponentVersion(found)
	if isReady && !versionUpdated {
		req.Logger.Info(fmt.Sprintf("could not complete the upgrade process. %s is not with the expected version. Check %s observed version in the status field of its CR", h.crType, h.crType))
	}

	upgradeDone := req.UpgradeMode && isReady && versionUpdated
	return res.SetUpgradeDone(upgradeDone)
}

func (h *genericOperand) addCrToTheRelatedObjectList(req *common.HcoRequest, found client.Object) error {

	changed, err := hcoutil.AddCrToTheRelatedObjectList(&req.Instance.Status.RelatedObjects, found, h.Scheme)
	if err != nil {
		return err
	}

	if changed {
		req.StatusDirty = true
	}
	return nil
}

func (h *genericOperand) doSetControllerReference(req *common.HcoRequest, cr client.Object) error {
	if h.setControllerReference {
		ref, ok := cr.(metav1.Object)
		if !ok {
			return fmt.Errorf("can't convert %T to k8s.io/apimachinery/pkg/apis/meta/v1.Object", cr)
		}
		if err := controllerutil.SetControllerReference(req.Instance, ref, h.Scheme); err != nil {
			return err
		}
	}
	return nil
}

func (h *genericOperand) createNewCr(req *common.HcoRequest, cr client.Object, res *EnsureResult) *EnsureResult {
	req.Logger.Info("Creating " + h.crType)
	if cr.GetResourceVersion() != "" {
		cr.SetResourceVersion("")
	}
	err := h.Client.Create(req.Ctx, cr)
	if err != nil {
		req.Logger.Error(err, "Failed to create object for "+h.crType)
		return res.Error(err)
	}
	return res.SetCreated()
}

func (h *genericOperand) reset() {
	if r, ok := h.hooks.(reseter); ok {
		r.reset()
	}
}

// handleComponentConditions - read and process a sub-component conditions.
// returns true if the the conditions indicates "ready" state and false if not.
func handleComponentConditions(req *common.HcoRequest, component string, componentConds []metav1.Condition) bool {
	if len(componentConds) == 0 {
		getConditionsForNewCr(req, component)
		return false
	}

	return setConditionsByOperandConditions(req, component, componentConds)
}

func setConditionsByOperandConditions(req *common.HcoRequest, component string, componentConds []metav1.Condition) bool {
	isReady := true
	foundAvailableCond := false
	foundProgressingCond := false
	foundDegradedCond := false
	for _, condition := range componentConds {
		switch condition.Type {
		case hcov1beta1.ConditionAvailable:
			foundAvailableCond = true
			isReady = handleOperandAvailableCond(req, component, condition) && isReady

		case hcov1beta1.ConditionProgressing:
			foundProgressingCond = true
			isReady = handleOperandProgressingCond(req, component, condition) && isReady

		case hcov1beta1.ConditionDegraded:
			foundDegradedCond = true
			isReady = handleOperandDegradedCond(req, component, condition) && isReady

		case hcov1beta1.ConditionUpgradeable:
			handleOperandUpgradeableCond(req, component, condition)
		}
	}

	if !foundAvailableCond {
		componentNotAvailable(req, component, `missing "Available" condition`)
	}

	return isReady && foundAvailableCond && foundProgressingCond && foundDegradedCond
}

func handleOperandDegradedCond(req *common.HcoRequest, component string, condition metav1.Condition) bool {
	if condition.Status == metav1.ConditionTrue {
		req.Logger.Info(fmt.Sprintf("%s is 'Degraded'", component))
		req.Conditions.SetStatusCondition(metav1.Condition{
			Type:               hcov1beta1.ConditionDegraded,
			Status:             metav1.ConditionTrue,
			Reason:             fmt.Sprintf("%sDegraded", component),
			Message:            fmt.Sprintf("%s is degraded: %v", component, condition.Message),
			ObservedGeneration: req.Instance.Generation,
		})

		return false
	}
	return true
}

func handleOperandProgressingCond(req *common.HcoRequest, component string, condition metav1.Condition) bool {
	if condition.Status == metav1.ConditionTrue {
		req.Logger.Info(fmt.Sprintf("%s is 'Progressing'", component))
		req.Conditions.SetStatusCondition(metav1.Condition{
			Type:               hcov1beta1.ConditionProgressing,
			Status:             metav1.ConditionTrue,
			Reason:             fmt.Sprintf("%sProgressing", component),
			Message:            fmt.Sprintf("%s is progressing: %v", component, condition.Message),
			ObservedGeneration: req.Instance.Generation,
		})
		req.Conditions.SetStatusConditionIfUnset(metav1.Condition{
			Type:               hcov1beta1.ConditionUpgradeable,
			Status:             metav1.ConditionFalse,
			Reason:             fmt.Sprintf("%sProgressing", component),
			Message:            fmt.Sprintf("%s is progressing: %v", component, condition.Message),
			ObservedGeneration: req.Instance.Generation,
		})

		return false
	}
	return true
}

func handleOperandUpgradeableCond(req *common.HcoRequest, component string, condition metav1.Condition) {
	if condition.Status == metav1.ConditionFalse {
		req.Upgradeable = false
		req.Logger.Info(fmt.Sprintf("%s is 'Progressing'", component))
		req.Conditions.SetStatusCondition(metav1.Condition{
			Type:               hcov1beta1.ConditionUpgradeable,
			Status:             metav1.ConditionFalse,
			Reason:             fmt.Sprintf("%sNotUpgradeable", component),
			Message:            fmt.Sprintf("%s is not upgradeable: %v", component, condition.Message),
			ObservedGeneration: req.Instance.Generation,
		})
	}
}

func handleOperandAvailableCond(req *common.HcoRequest, component string, condition metav1.Condition) bool {
	if condition.Status == metav1.ConditionFalse {
		msg := fmt.Sprintf("%s is not available: %v", component, condition.Message)
		componentNotAvailable(req, component, msg)
		return false
	}
	return true
}

func getConditionsForNewCr(req *common.HcoRequest, component string) {
	reason := fmt.Sprintf("%sConditions", component)
	message := fmt.Sprintf("%s resource has no conditions", component)
	req.Logger.Info(fmt.Sprintf("%s's resource is not reporting Conditions on it's Status", component))
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionAvailable,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: req.Instance.Generation,
	})
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionProgressing,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: req.Instance.Generation,
	})
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionUpgradeable,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: req.Instance.Generation,
	})
}

func componentNotAvailable(req *common.HcoRequest, component string, msg string) {
	req.Logger.Info(fmt.Sprintf("%s is not 'Available'", component))
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionAvailable,
		Status:             metav1.ConditionFalse,
		Reason:             fmt.Sprintf("%sNotAvailable", component),
		Message:            msg,
		ObservedGeneration: req.Instance.Generation,
	})
}

func checkComponentVersion(versionEnvName, actualVersion string) bool {
	expectedVersion := os.Getenv(versionEnvName)
	return expectedVersion != "" && expectedVersion == actualVersion
}

func getNamespace(defaultNamespace string, opts []string) string {
	if len(opts) > 0 {
		return opts[0]
	}
	return defaultNamespace
}

func getLabels(hc *hcov1beta1.HyperConverged, component hcoutil.AppComponent) map[string]string {
	hcoName := hcov1beta1.HyperConvergedName

	if hc.Name != "" {
		hcoName = hc.Name
	}

	return hcoutil.GetLabels(hcoName, component)
}

func applyAnnotationPatch(obj runtime.Object, annotation string) error {
	patches, err := jsonpatch.DecodePatch([]byte(annotation))
	if err != nil {
		return err
	}

	for _, patch := range patches {
		path, err := patch.Path()
		if err != nil {
			return err
		}

		if !strings.HasPrefix(path, "/spec/") {
			return errors.New("can only modify spec fields")
		}
	}

	specBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	patchedBytes, err := patches.Apply(specBytes)
	if err != nil {
		return err
	}
	return json.Unmarshal(patchedBytes, obj)
}

func applyPatchToSpec(hc *hcov1beta1.HyperConverged, annotationName string, obj runtime.Object) error {
	if jsonpathAnnotation, ok := hc.Annotations[annotationName]; ok {
		if err := applyAnnotationPatch(obj, jsonpathAnnotation); err != nil {
			return fmt.Errorf("invalid jsonPatch in the %s annotation: %v", annotationName, err)
		}
	}

	return nil
}

func osConditionToK8s(condition conditionsv1.Condition) metav1.Condition {
	return metav1.Condition{
		Type:    string(condition.Type),
		Reason:  condition.Reason,
		Status:  metav1.ConditionStatus(condition.Status),
		Message: condition.Message,
	}
}

func osConditionsToK8s(conditions []conditionsv1.Condition) []metav1.Condition {
	if len(conditions) == 0 {
		return nil
	}

	newCond := make([]metav1.Condition, len(conditions))
	for i, c := range conditions {
		newCond[i] = osConditionToK8s(c)
	}

	return newCond
}
