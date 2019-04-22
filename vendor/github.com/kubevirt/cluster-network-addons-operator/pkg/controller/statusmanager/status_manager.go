package statusmanager

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

// StatusLevel is used to sort priority of reported failure conditions. When operator is failing
// on two levels (e.g. handling configuration and deploying pods), only the higher level will be
// reported. This is needed, so failing pods from previous run won't silence failing render of
// the current run.
type StatusLevel int

const (
	OperatorConfig StatusLevel = iota
	PodDeployment  StatusLevel = iota
	maxStatusLevel StatusLevel = iota
)

// StatusManager coordinates changes to NetworkAddonsConfig.Status
type StatusManager struct {
	client client.Client
	name   string

	failing [maxStatusLevel]*opv1alpha1.NetworkAddonsCondition

	daemonSets  []types.NamespacedName
	deployments []types.NamespacedName
}

func New(client client.Client, name string) *StatusManager {
	return &StatusManager{client: client, name: name}
}

// Set updates the NetworkAddonsConfig.Status with the provided conditions
func (status *StatusManager) Set(conditions ...opv1alpha1.NetworkAddonsCondition) {
	// Read the current NetworkAddonsConfig
	config := &opv1alpha1.NetworkAddonsConfig{ObjectMeta: metav1.ObjectMeta{Name: status.name}}
	err := status.client.Get(context.TODO(), types.NamespacedName{Name: status.name}, config)
	if err != nil {
		log.Printf("Failed to get NetworkAddonsOperator %q in order to update its State: %v", status.name, err)
		return
	}

	oldStatus := config.Status.DeepCopy()

	// Update Status field with given conditions
	for _, condition := range conditions {
		updateCondition(&config.Status, condition)
	}

	// In case that the status field has been updated with "Progressing" condition, make sure that
	// "Ready" condition is set to False, even when not explicitly set.
	progressingCondition := getCondition(&config.Status, opv1alpha1.NetworkAddonsConditionProgressing)
	availableCondition := getCondition(&config.Status, opv1alpha1.NetworkAddonsConditionAvailable)
	if availableCondition == nil && progressingCondition != nil && progressingCondition.Status == corev1.ConditionTrue {
		updateCondition(&config.Status,
			opv1alpha1.NetworkAddonsCondition{
				Type:    opv1alpha1.NetworkAddonsConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "Startup",
				Message: "Configuration is in process",
			},
		)
	}

	if reflect.DeepEqual(oldStatus, config.Status) {
		return
	}

	// Update NetworkAddonsConfig with updated Status field
	err = status.client.Status().Update(context.TODO(), config)
	if err != nil {
		log.Printf("Failed to update NetworkAddonsConfig %q Status: %v", config.Name, err)
	}
}

// syncFailing syncs the current Failing status
func (status *StatusManager) syncFailing() {
	for _, c := range status.failing {
		if c != nil {
			status.Set(*c)
			return
		}
	}
	status.Set(
		opv1alpha1.NetworkAddonsCondition{
			Type:   opv1alpha1.NetworkAddonsConditionFailing,
			Status: corev1.ConditionFalse,
		},
	)
}

// SetFailing marks the operator as Failing with the given reason and message. If it
// is not already failing for a lower-level reason, the operator's status will be updated.
func (status *StatusManager) SetFailing(level StatusLevel, reason, message string) {
	status.failing[level] = &opv1alpha1.NetworkAddonsCondition{
		Type:    opv1alpha1.NetworkAddonsConditionFailing,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
	status.syncFailing()
}

// SetNotFailing marks the operator as not Failing at the given level. If the operator
// status previously indicated failure at this level, it will updated to show the next
// higher-level failure, or else to show that the operator is no longer failing.
func (status *StatusManager) SetNotFailing(level StatusLevel) {
	if status.failing[level] != nil {
		status.failing[level] = nil
	}
	status.syncFailing()
}

func (status *StatusManager) SetDaemonSets(daemonSets []types.NamespacedName) {
	status.daemonSets = daemonSets
}

func (status *StatusManager) SetDeployments(deployments []types.NamespacedName) {
	status.deployments = deployments
}

// TODO refactoring
// SetFromPods sets the operator status to Failing, Progressing, or Available, based on
// the current status of the manager's DaemonSets and Deployments. However, this is a
// no-op if the StatusManager is currently marked as failing due to a configuration error.
func (status *StatusManager) SetFromPods() {
	progressing := []string{}

	// Iterate all owned DaemonSets and check whether they are progressing smoothly or have been
	// already deployed.
	for _, dsName := range status.daemonSets {
		// First check whether DaemonSet namespace exists
		ns := &corev1.Namespace{}
		if err := status.client.Get(context.TODO(), types.NamespacedName{Name: dsName.Namespace}, ns); err != nil {
			if errors.IsNotFound(err) {
				status.SetFailing(PodDeployment, "NoNamespace",
					fmt.Sprintf("Namespace %q does not exist", dsName.Namespace))
			} else {
				status.SetFailing(PodDeployment, "InternalError",
					fmt.Sprintf("Internal error deploying pods: %v", err))
			}
			return
		}

		// Then check whether is the DaemonSet created on Kubernetes API server
		ds := &appsv1.DaemonSet{}
		if err := status.client.Get(context.TODO(), dsName, ds); err != nil {
			if errors.IsNotFound(err) {
				status.SetFailing(PodDeployment, "NoDaemonSet",
					fmt.Sprintf("Expected DaemonSet %q does not exist", dsName.String()))
			} else {
				status.SetFailing(PodDeployment, "InternalError",
					fmt.Sprintf("Internal error deploying pods: %v", err))
			}
			return
		}

		// Finally check whether Pods belonging to this DaemonSets are being started or they
		// are being scheduled.
		if ds.Status.NumberUnavailable > 0 {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q is not available (awaiting %d nodes)", dsName.String(), ds.Status.NumberUnavailable))
		} else if ds.Status.NumberAvailable == 0 {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q is not yet scheduled on any nodes", dsName.String()))
		} else if ds.Status.UpdatedNumberScheduled < ds.Status.DesiredNumberScheduled {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q update is rolling out (%d out of %d updated)", dsName.String(), ds.Status.UpdatedNumberScheduled, ds.Status.DesiredNumberScheduled))
		} else if ds.Generation > ds.Status.ObservedGeneration {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q update is being processed (generation %d, observed generation %d)", dsName.String(), ds.Generation, ds.Status.ObservedGeneration))
		}
	}

	// Do the same for Deployments. Iterate all owned Deployments and check whether they are
	// progressing smoothly or have been already deployed.
	for _, depName := range status.deployments {
		// First check whether Deployment namespace exists
		ns := &corev1.Namespace{}
		if err := status.client.Get(context.TODO(), types.NamespacedName{Name: depName.Namespace}, ns); err != nil {
			if errors.IsNotFound(err) {
				status.SetFailing(PodDeployment, "NoNamespace",
					fmt.Sprintf("Namespace %q does not exist", depName.Namespace))
			} else {
				status.SetFailing(PodDeployment, "InternalError",
					fmt.Sprintf("Internal error deploying pods: %v", err))
			}
			return
		}

		// Then check whether is the Deployment created on Kubernetes API server
		dep := &appsv1.Deployment{}
		if err := status.client.Get(context.TODO(), depName, dep); err != nil {
			if errors.IsNotFound(err) {
				status.SetFailing(PodDeployment, "NoDeployment",
					fmt.Sprintf("Expected Deployment %q does not exist", depName.String()))
			} else {
				status.SetFailing(PodDeployment, "InternalError",
					fmt.Sprintf("Internal error deploying pods: %v", err))
			}
			return
		}

		// Finally check whether Pods belonging to this Deployments are being started or they
		// are being scheduled.
		if dep.Status.UnavailableReplicas > 0 {
			progressing = append(progressing, fmt.Sprintf("Deployment %q is not available (awaiting %d nodes)", depName.String(), dep.Status.UnavailableReplicas))
		} else if dep.Status.AvailableReplicas == 0 {
			progressing = append(progressing, fmt.Sprintf("Deployment %q is not yet scheduled on any nodes", depName.String()))
		} else if dep.Status.ObservedGeneration < dep.Generation {
			progressing = append(progressing, fmt.Sprintf("Deployment %q update is being processed (generation %d, observed generation %d)", depName.String(), dep.Generation, dep.Status.ObservedGeneration))
		}
	}

	// If all pods are being created, mark deployment as not failing
	status.SetNotFailing(PodDeployment)

	// If there are any progressing Pods, list them in the condition with their state. Otherwise,
	// mark NetworkAddonsConfig as "Ready".
	if len(progressing) > 0 {
		status.Set(
			opv1alpha1.NetworkAddonsCondition{
				Type:    opv1alpha1.NetworkAddonsConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "Deploying",
				Message: strings.Join(progressing, "\n"),
			},
		)
	} else {
		status.Set(
			opv1alpha1.NetworkAddonsCondition{
				Type:   opv1alpha1.NetworkAddonsConditionProgressing,
				Status: corev1.ConditionFalse,
			},
			opv1alpha1.NetworkAddonsCondition{
				Type:   opv1alpha1.NetworkAddonsConditionAvailable,
				Status: corev1.ConditionTrue,
			},
		)
	}
}

// Set condition in the Status field. In case condition with given Type already exists, update it.
// This function also takes care of setting timestamps of LastProbeTime and LastTransitionTime.
func updateCondition(status *opv1alpha1.NetworkAddonsConfigStatus, condition opv1alpha1.NetworkAddonsCondition) {
	originalCondition := getCondition(status, condition.Type)

	// Check whether the condition is to be added or updated
	isNew := originalCondition == nil

	// Check whether there are any changes compared to the original condition
	transition := (!isNew && (condition.Status != originalCondition.Status || condition.Reason != originalCondition.Reason || condition.Message != originalCondition.Message))

	// Update Condition's timestamps
	now := time.Now()

	// LastProbeTime indicates the last time the condition has been checked
	condition.LastProbeTime = metav1.Time{
		Time: now,
	}

	// LastTransitionTime indicates the last time the condition has been changed or it has been added
	if isNew || transition {
		condition.LastTransitionTime = metav1.Time{
			Time: now,
		}
	} else {
		condition.LastTransitionTime = originalCondition.LastTransitionTime
	}

	// Update or add desired condition to the Status field
	if isNew {
		status.Conditions = append(status.Conditions, condition)
	} else {
		for i := range status.Conditions {
			if status.Conditions[i].Type == condition.Type {
				status.Conditions[i] = condition
				break
			}
		}
	}
}

// Get conditition of given type from NetworkAddonsConfig.Status
func getCondition(status *opv1alpha1.NetworkAddonsConfigStatus, conditionType opv1alpha1.NetworkAddonsConditionType) *opv1alpha1.NetworkAddonsCondition {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
