package operands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
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
	// In some cases, Previous versions used to have HCO-operator (scope namespace)
	// as the owner of some resources (scope cluster).
	// It's not legal, so remove that.
	removeExistingOwner bool
	// Should the handler add the controller reference
	setControllerReference bool
	// Set of resource handler hooks, to be implement in each handler
	hooks hcoResourceHooks
}

// Set of resource handler hooks, to be implement in each handler
type hcoResourceHooks interface {
	// Generate the required resource, with all the required fields)
	getFullCr(*hcov1beta1.HyperConverged) (client.Object, error)
	// Generate an empty resource, to be used as the input of the client.Get method. After calling this method, it will
	// contains the actual values in K8s.
	getEmptyCr() client.Object
	// an optional hook that is called just after getting the resource from K8s
	postFound(*common.HcoRequest, runtime.Object) error
	// check if there is a change between the required resource and the resource read from K8s, and update K8s accordingly.
	updateCr(*common.HcoRequest, client.Client, runtime.Object, runtime.Object) (bool, bool, error)
	// cast he specific resource to *metav1.ObjectMeta
	getObjectMeta(runtime.Object) *metav1.ObjectMeta
}

// Set of operand handler hooks, to be implement in each handler
type hcoOperandHooks interface {
	hcoResourceHooks
	// get the CR conditions, if exists
	getConditions(runtime.Object) []conditionsv1.Condition
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
		return h.createNewCr(req, err, cr, res)
	}

	return h.handleExistingCr(req, key, found, cr, res)
}

func (h *genericOperand) handleExistingCr(req *common.HcoRequest, key client.ObjectKey, found client.Object, cr client.Object, res *EnsureResult) *EnsureResult {
	req.Logger.Info(h.crType+" already exists", h.crType+".Namespace", key.Namespace, h.crType+".Name", key.Name)

	if err := h.hooks.postFound(req, found); err != nil {
		return res.Error(err)
	}

	h.doRemoveExistingOwners(req, found)

	updated, overwritten, err := h.hooks.updateCr(req, h.Client, found, cr)
	if err != nil {
		return res.Error(err)
	}

	if err = h.addCrToTheRelatedObjectList(req, found); err != nil {
		return res.Error(err)
	}

	if updated {
		return res.SetUpdated().SetOverwritten(overwritten)
	}

	if opr, ok := h.hooks.(hcoOperandHooks); ok { // for operands, perform some more checks
		return h.completeEnsureOperands(req, opr, found, res)
	}
	// For resources that are not CRs, such as priority classes or a config map, there is no new version to upgrade
	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
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
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return err
	}

	err = objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)
	if err != nil {
		return err
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

func (h *genericOperand) doRemoveExistingOwners(req *common.HcoRequest, found client.Object) {
	if h.removeExistingOwner {
		existingOwners := h.hooks.getObjectMeta(found).GetOwnerReferences()

		if len(existingOwners) > 0 {
			req.Logger.Info(h.crType + " has owners, removing...")
			err := h.Client.Update(req.Ctx, found)
			if err != nil {
				req.Logger.Error(err, fmt.Sprintf("Failed to remove %s's previous owners", h.crType))
				return
			}
		}

		// do that only once
		h.removeExistingOwner = false
	}
}

func (h *genericOperand) createNewCr(req *common.HcoRequest, err error, cr client.Object, res *EnsureResult) *EnsureResult {
	if apierrors.IsNotFound(err) {
		req.Logger.Info("Creating " + h.crType)
		err = h.Client.Create(req.Ctx, cr)
		if err != nil {
			req.Logger.Error(err, "Failed to create object for "+h.crType)
			return res.Error(err)
		}
		return res.SetCreated()
	}
	return res.Error(err)
}

func (h *genericOperand) reset() {
	if r, ok := h.hooks.(reseter); ok {
		r.reset()
	}
}

// handleComponentConditions - read and process a sub-component conditions.
// returns true if the the conditions indicates "ready" state and false if not.
func handleComponentConditions(req *common.HcoRequest, component string, componentConds []conditionsv1.Condition) bool {
	if len(componentConds) == 0 {
		getConditionsForNewCr(req, component)
		return false
	}

	return setConditionsByOperandConditions(req, component, componentConds)
}

func setConditionsByOperandConditions(req *common.HcoRequest, component string, componentConds []conditionsv1.Condition) bool {
	isReady := true
	foundAvailableCond := false
	foundProgressingCond := false
	foundDegradedCond := false
	for _, condition := range componentConds {
		switch condition.Type {
		case conditionsv1.ConditionAvailable:
			foundAvailableCond = true
			isReady = handleOperandAvailableCond(req, component, condition) && isReady

		case conditionsv1.ConditionProgressing:
			foundProgressingCond = true
			isReady = handleOperandProgressingCond(req, component, condition) && isReady

		case conditionsv1.ConditionDegraded:
			foundDegradedCond = true
			isReady = handleOperandDegradedCond(req, component, condition) && isReady
		}
	}

	if !foundAvailableCond {
		componentNotAvailable(req, component, `missing "Available" condition`)
	}

	return isReady && foundAvailableCond && foundProgressingCond && foundDegradedCond
}

func handleOperandDegradedCond(req *common.HcoRequest, component string, condition conditionsv1.Condition) bool {
	if condition.Status == corev1.ConditionTrue {
		req.Logger.Info(fmt.Sprintf("%s is 'Degraded'", component))
		req.Conditions.SetStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionDegraded,
			Status:  corev1.ConditionTrue,
			Reason:  fmt.Sprintf("%sDegraded", component),
			Message: fmt.Sprintf("%s is degraded: %v", component, condition.Message),
		})

		return false
	}
	return true
}

func handleOperandProgressingCond(req *common.HcoRequest, component string, condition conditionsv1.Condition) bool {
	if condition.Status == corev1.ConditionTrue {
		req.Logger.Info(fmt.Sprintf("%s is 'Progressing'", component))
		req.Conditions.SetStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  fmt.Sprintf("%sProgressing", component),
			Message: fmt.Sprintf("%s is progressing: %v", component, condition.Message),
		})
		req.Conditions.SetStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  fmt.Sprintf("%sProgressing", component),
			Message: fmt.Sprintf("%s is progressing: %v", component, condition.Message),
		})

		return false
	}
	return true
}

func handleOperandAvailableCond(req *common.HcoRequest, component string, condition conditionsv1.Condition) bool {
	if condition.Status == corev1.ConditionFalse {
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
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionUpgradeable,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

func componentNotAvailable(req *common.HcoRequest, component string, msg string) {
	req.Logger.Info(fmt.Sprintf("%s is not 'Available'", component))
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  fmt.Sprintf("%sNotAvailable", component),
		Message: msg,
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

	return map[string]string{
		hcoutil.AppLabel:          hcoName,
		hcoutil.AppLabelManagedBy: hcoutil.OperatorName,
		hcoutil.AppLabelVersion:   hcoutil.GetHcoKvIoVersion(),
		hcoutil.AppLabelPartOf:    hcoutil.HyperConvergedCluster,
		hcoutil.AppLabelComponent: string(component),
	}
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
