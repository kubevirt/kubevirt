package statusmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

const (
	conditionsUpdateRetries  = 10
	conditionsUpdateCoolDown = 50 * time.Millisecond
)

var operatorVersion string

func init() {
	operatorVersion = os.Getenv("OPERATOR_VERSION")
}

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

	failing [maxStatusLevel]*conditionsv1.Condition

	daemonSets  []types.NamespacedName
	deployments []types.NamespacedName

	containers []opv1alpha1.Container
}

func New(client client.Client, name string) *StatusManager {
	return &StatusManager{client: client, name: name}
}

// Set updates the NetworkAddonsConfig.Status with the provided conditions.
// Since Update call can fail due to a collision with someone else writing into
// the status, calling set is tried several times.
// TODO: Calling of Patch instead may save some problems. We can reiterate later,
// current collision problem is detected by functional tests
func (status *StatusManager) Set(reachedAvailableLevel bool, conditions ...conditionsv1.Condition) {
	for i := 0; i < conditionsUpdateRetries; i++ {
		err := status.set(reachedAvailableLevel, conditions...)
		if err == nil {
			log.Print("Successfully updated status conditions")
			return
		}
		log.Printf("Failed calling status Set %d/%d: %v", i+1, conditionsUpdateRetries, err)
		time.Sleep(conditionsUpdateCoolDown)
	}
	log.Print("Failed to update conditions within given number of retries")
}

// set updates the NetworkAddonsConfig.Status with the provided conditions
func (status *StatusManager) set(reachedAvailableLevel bool, conditions ...conditionsv1.Condition) error {
	// Read the current NetworkAddonsConfig
	config := &opv1alpha1.NetworkAddonsConfig{ObjectMeta: metav1.ObjectMeta{Name: status.name}}
	err := status.client.Get(context.TODO(), types.NamespacedName{Name: status.name}, config)
	if err != nil {
		log.Printf("Failed to get NetworkAddonsOperator %q in order to update its State: %v", status.name, err)
		return nil
	}

	oldStatus := config.Status.DeepCopy()

	// Update Status field with given conditions
	for _, condition := range conditions {
		conditionsv1.SetStatusCondition(&config.Status.Conditions, condition)
	}

	// Glue condition logic together
	if status.failing[OperatorConfig] != nil {
		// In case the operator is failing, we should not report it as being ready. This has
		// to be done even when the operator is running fine based on the previous configuration
		// and the only failing thing is validation of new config.
		conditionsv1.SetStatusCondition(&config.Status.Conditions,
			conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "Failing",
				Message: "Unable to apply desired configuration",
			},
		)

		// Implicitly mark as not Progressing, that indicates that human interaction is needed
		conditionsv1.SetStatusCondition(&config.Status.Conditions,
			conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionFalse,
				Reason:  "InvalidConfiguration",
				Message: "Human interaction is needed, please fix the desired configuration",
			},
		)
	} else if status.failing[PodDeployment] != nil {
		// In case pod deployment is in progress, implicitly mark as not Available
		conditionsv1.SetStatusCondition(&config.Status.Conditions,
			conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "Failing",
				Message: "Some problems occurred while deploying components' pods",
			},
		)
	} else if conditionsv1.IsStatusConditionTrue(config.Status.Conditions, conditionsv1.ConditionProgressing) {
		// In case that the status field has been updated with "Progressing" condition, make sure that
		// "Ready" condition is set to False, even when not explicitly set.
		conditionsv1.SetStatusCondition(&config.Status.Conditions,
			conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "Startup",
				Message: "Configuration is in process",
			},
		)
	} else if reachedAvailableLevel {
		// If successfully deployed all components and is not failing on anything, mark as Available
		conditionsv1.SetStatusCondition(&config.Status.Conditions,
			conditionsv1.Condition{
				Type:   conditionsv1.ConditionAvailable,
				Status: corev1.ConditionTrue,
			},
		)
		config.Status.ObservedVersion = operatorVersion
	}

	// Make sure to expose deployed containers
	config.Status.Containers = status.containers

	// Expose currently handled version
	config.Status.OperatorVersion = operatorVersion
	config.Status.TargetVersion = operatorVersion

	// Failing condition had been replaced by Degraded in 0.12.0, drop it from CR if needed
	conditionsv1.RemoveStatusCondition(&config.Status.Conditions, conditionsv1.ConditionType("Failing"))

	if reflect.DeepEqual(oldStatus, config.Status) {
		return nil
	}

	// Update NetworkAddonsConfig with updated Status field
	err = status.client.Status().Update(context.TODO(), config)
	if err != nil {
		return fmt.Errorf("Failed to update NetworkAddonsConfig %q Status: %v", config.Name, err)
	}

	return nil
}

// syncFailing syncs the current Failing status
func (status *StatusManager) syncFailing() {
	for _, c := range status.failing {
		if c != nil {
			status.Set(false, *c)
			return
		}
	}
	status.Set(
		false,
		conditionsv1.Condition{
			Type:   conditionsv1.ConditionDegraded,
			Status: corev1.ConditionFalse,
		},
	)
}

// SetFailing marks the operator as Failing with the given reason and message. If it
// is not already failing for a lower-level reason, the operator's status will be updated.
func (status *StatusManager) SetFailing(level StatusLevel, reason, message string) {
	status.failing[level] = &conditionsv1.Condition{
		Type:    conditionsv1.ConditionDegraded,
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

	// If there are any progressing Pods, list them in the condition with their state. Otherwise,
	// mark Progressing condition as False.
	if len(progressing) > 0 {
		status.Set(
			false,
			conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "Deploying",
				Message: strings.Join(progressing, "\n"),
			},
		)
	} else {
		status.Set(
			false,
			conditionsv1.Condition{
				Type:   conditionsv1.ConditionProgressing,
				Status: corev1.ConditionFalse,
			},
		)
	}

	// If all pods are being created, mark deployment as not failing
	status.SetNotFailing(PodDeployment)

	// Finally, if all containers are deployed, mark as Available
	if len(progressing) == 0 {
		status.Set(true)
	}
}

func (status *StatusManager) SetContainers(containers []opv1alpha1.Container) {
	status.containers = containers
}
