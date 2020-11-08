package operands

import (
	"fmt"
	"os"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Operand interface {
	Ensure(req *common.HcoRequest) *EnsureResult
}

type genericOperand struct {
	Client              client.Client
	Scheme              *runtime.Scheme
	crType              string
	removeExistingOwner bool

	// hooks
	getFullCr              func(hc *hcov1beta1.HyperConverged) runtime.Object
	getEmptyCr             func() runtime.Object
	setControllerReference func(hc *hcov1beta1.HyperConverged, controlled metav1.Object, scheme *runtime.Scheme) error
	postFound              func(request *common.HcoRequest, exists runtime.Object) error
	updateCr               func(request *common.HcoRequest, exists runtime.Object, required runtime.Object) (bool, bool, error)
	getConditions          func(cr runtime.Object) []conditionsv1.Condition
	checkComponentVersion  func(cr runtime.Object) bool
	getObjectMeta          func(cr runtime.Object) *metav1.ObjectMeta
}

func (handler *genericOperand) ensure(req *common.HcoRequest) *EnsureResult {
	cr := handler.getFullCr(req.Instance)

	res := NewEnsureResult(cr)
	ref, ok := cr.(metav1.Object)
	if handler.setControllerReference != nil {
		if !ok {
			return res.Error(fmt.Errorf("can't convert %T to k8s.io/apimachinery/pkg/apis/meta/v1.Object", cr))
		}
		if err := handler.setControllerReference(req.Instance, ref, handler.Scheme); err != nil {
			return res.Error(err)
		}
	}

	key, err := client.ObjectKeyFromObject(cr)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for "+handler.crType)
	}

	res.SetName(key.Name)
	found := handler.getEmptyCr()
	err = handler.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating " + handler.crType)
			err = handler.Client.Create(req.Ctx, cr)
			if err == nil {
				return res.SetCreated().SetName(key.Name)
			}
		}
		return res.Error(err)
	}

	key, err = client.ObjectKeyFromObject(found)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for "+handler.crType)
	}
	req.Logger.Info(handler.crType+" already exists", handler.crType+".Namespace", key.Namespace, handler.crType+".Name", key.Name)

	if handler.postFound != nil {
		if err = handler.postFound(req, found); err != nil {
			return res.Error(err)
		}
	}

	if handler.removeExistingOwner && (handler.getObjectMeta != nil) {
		existingOwners := handler.getObjectMeta(found).GetOwnerReferences()

		if len(existingOwners) > 0 {
			req.Logger.Info(handler.crType + " has owners, removing...")
			err = handler.Client.Update(req.Ctx, found)
			if err != nil {
				req.Logger.Error(err, fmt.Sprintf("Failed to remove %s's previous owners", handler.crType))
			}
		}

		if err == nil {
			// do that only once
			handler.removeExistingOwner = false
		}
	}

	if handler.updateCr != nil {
		updated, overwritten, err := handler.updateCr(req, found, cr)
		if err != nil {
			return res.Error(err)
		}
		if updated {
			res.SetUpdated()
			if overwritten {
				res.SetOverwritten()
			}
			return res
		}
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(handler.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	if (handler.getConditions != nil) && (handler.checkComponentVersion) != nil {
		// Handle KubeVirt resource conditions
		isReady := handleComponentConditions(req, handler.crType, handler.getConditions(found))

		upgradeDone := req.ComponentUpgradeInProgress && isReady && handler.checkComponentVersion(found)
		return res.SetUpgradeDone(upgradeDone)
	}

	return res
}

// handleComponentConditions - read and process a sub-component conditions.
// returns true if the the conditions indicates "ready" state and false if not.
func handleComponentConditions(req *common.HcoRequest, component string, componentConds []conditionsv1.Condition) (isReady bool) {
	isReady = true
	if len(componentConds) == 0 {
		isReady = false
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
	} else {
		foundAvailableCond := false
		foundProgressingCond := false
		foundDegradedCond := false
		for _, condition := range componentConds {
			switch condition.Type {
			case conditionsv1.ConditionAvailable:
				foundAvailableCond = true
				if condition.Status == corev1.ConditionFalse {
					isReady = false
					msg := fmt.Sprintf("%s is not available: %v", component, string(condition.Message))
					componentNotAvailable(req, component, msg)
				}
			case conditionsv1.ConditionProgressing:
				foundProgressingCond = true
				if condition.Status == corev1.ConditionTrue {
					isReady = false
					req.Logger.Info(fmt.Sprintf("%s is 'Progressing'", component))
					req.Conditions.SetStatusCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  fmt.Sprintf("%sProgressing", component),
						Message: fmt.Sprintf("%s is progressing: %v", component, string(condition.Message)),
					})
					req.Conditions.SetStatusCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  fmt.Sprintf("%sProgressing", component),
						Message: fmt.Sprintf("%s is progressing: %v", component, string(condition.Message)),
					})
				}
			case conditionsv1.ConditionDegraded:
				foundDegradedCond = true
				if condition.Status == corev1.ConditionTrue {
					isReady = false
					req.Logger.Info(fmt.Sprintf("%s is 'Degraded'", component))
					req.Conditions.SetStatusCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  fmt.Sprintf("%sDegraded", component),
						Message: fmt.Sprintf("%s is degraded: %v", component, string(condition.Message)),
					})
				}
			}
		}

		if !foundAvailableCond {
			componentNotAvailable(req, component, `missing "Available" condition`)
		}

		isReady = isReady && foundAvailableCond && foundProgressingCond && foundDegradedCond
	}

	return isReady
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
