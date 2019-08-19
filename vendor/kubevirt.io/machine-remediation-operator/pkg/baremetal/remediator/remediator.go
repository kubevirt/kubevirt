package remediator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	bmov1 "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/consts"
	"kubevirt.io/machine-remediation-operator/pkg/utils/conditions"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	mapiv1 "sigs.k8s.io/cluster-api/pkg/apis/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	rebootDefaultTimeout = 5
)

// BareMetalRemediator implements Remediator interface for bare metal machines
type BareMetalRemediator struct {
	client client.Client
}

// NewBareMetalRemediator returns new BareMetalRemediator object
func NewBareMetalRemediator(mgr manager.Manager) *BareMetalRemediator {
	return &BareMetalRemediator{
		client: mgr.GetClient(),
	}
}

// Recreate recreates the bare metal machine under the cluster
func (bmr *BareMetalRemediator) Recreate(ctx context.Context, machineRemediation *mrv1.MachineRemediation) error {
	return fmt.Errorf("Not implemented yet")
}

// Reboot reboots the bare metal machine
func (bmr *BareMetalRemediator) Reboot(ctx context.Context, machineRemediation *mrv1.MachineRemediation) error {
	glog.V(4).Infof("MachineRemediation %q has state %q", machineRemediation.Name, machineRemediation.Status.State)

	// Get the machine from the MachineRemediation
	key := types.NamespacedName{
		Namespace: machineRemediation.Namespace,
		Name:      machineRemediation.Spec.MachineName,
	}
	machine := &mapiv1.Machine{}
	if err := bmr.client.Get(context.TODO(), key, machine); err != nil {
		return err
	}

	// Get the bare metal host object
	bmh, err := getBareMetalHostByMachine(bmr.client, machine)
	if err != nil {
		return err
	}

	// Copy the BareMetalHost object to prevent modification of the original one
	bmhCopy := bmh.DeepCopy()

	// Copy the MachineRemediation object to prevent modification of the original one
	mrCopy := machineRemediation.DeepCopy()

	now := time.Now()
	switch machineRemediation.Status.State {
	// initiating the reboot action
	case mrv1.RemediationStateStarted:
		// skip the reboot in case when the machine has power off state before the reboot action
		// it can mean that an user power off the machine by purpose
		if !bmh.Spec.Online {
			glog.V(4).Infof("Skip the remediation, machine %q has power off state before the remediation action", machine.Name)
			mrCopy.Status.State = mrv1.RemediationStateSucceeded
			mrCopy.Status.Reason = "Skip the reboot, the machine power off by an user"
			mrCopy.Status.EndTime = &metav1.Time{Time: now}
		} else {
			// power off the machine
			glog.V(4).Infof("Power off machine %q", machine.Name)
			bmhCopy.Spec.Online = false
			if err := bmr.client.Update(context.TODO(), bmhCopy); err != nil {
				return err
			}

			mrCopy.Status.State = mrv1.RemediationStatePowerOff
			mrCopy.Status.Reason = "Starts the reboot process"
		}
		return bmr.client.Status().Update(context.TODO(), mrCopy)

	case mrv1.RemediationStatePowerOff:
		// failed the remediation on timeout
		if machineRemediation.Status.StartTime.Time.Add(rebootDefaultTimeout * time.Minute).Before(now) {
			glog.Errorf("Remediation of machine %q failed on timeout", machine.Name)
			mrCopy.Status.State = mrv1.RemediationStateFailed
			mrCopy.Status.Reason = "Reboot failed on timeout"
			mrCopy.Status.EndTime = &metav1.Time{Time: now}
			return bmr.client.Status().Update(context.TODO(), mrCopy)
		}

		// host still has state on, we need to reconcile
		if bmh.Status.PoweredOn {
			glog.Warningf("machine %q still has power on state", machine.Name)
			return nil
		}

		// delete the node to release workloads, once we are sure that host has state power off
		if err := deleteMachineNode(bmr.client, machine); err != nil {
			return err
		}

		// power on the machine
		glog.V(4).Infof("Power on machine %q", machine.Name)
		bmhCopy.Spec.Online = true
		if err := bmr.client.Update(context.TODO(), bmhCopy); err != nil {
			return err
		}

		mrCopy.Status.State = mrv1.RemediationStatePowerOn
		mrCopy.Status.Reason = "Reboot in progress"
		return bmr.client.Status().Update(context.TODO(), mrCopy)

	case mrv1.RemediationStatePowerOn:
		// failed the remediation on timeout
		if machineRemediation.Status.StartTime.Time.Add(rebootDefaultTimeout * time.Minute).Before(now) {
			glog.Errorf("Remediation of machine %q failed on timeout", machine.Name)
			mrCopy.Status.State = mrv1.RemediationStateFailed
			mrCopy.Status.Reason = "Reboot failed on timeout"
			mrCopy.Status.EndTime = &metav1.Time{Time: now}
			return bmr.client.Status().Update(context.TODO(), mrCopy)
		}

		node, err := getNodeByMachine(bmr.client, machine)
		if err != nil {
			// we want to reconcile with delay of 10 seconds when the machine does not have node reference
			// or node does not exist
			if errors.IsNotFound(err) {
				glog.Warningf("The machine %q node does not exist", machine.Name)
				return nil
			}
			return err
		}

		// Node back to Ready under the cluster
		if conditions.NodeHasCondition(node, corev1.NodeReady, corev1.ConditionTrue) {
			glog.V(4).Infof("Remediation of machine %q succeeded", machine.Name)
			mrCopy.Status.State = mrv1.RemediationStateSucceeded
			mrCopy.Status.Reason = "Reboot succeeded"
			mrCopy.Status.EndTime = &metav1.Time{Time: now}
			return bmr.client.Status().Update(context.TODO(), mrCopy)
		}
		return nil

	case mrv1.RemediationStateSucceeded:
		// remove machine remediation object
		return bmr.client.Delete(context.TODO(), machineRemediation)
	}
	return nil
}

// getBareMetalHostByMachine returns the bare metal host that linked to the machine
func getBareMetalHostByMachine(c client.Client, machine *mapiv1.Machine) (*bmov1.BareMetalHost, error) {
	bmhKey, ok := machine.Annotations[consts.AnnotationBareMetalHost]
	if !ok {
		return nil, fmt.Errorf("machine does not have bare metal host annotation")
	}

	bmhNamespace, bmhName, err := cache.SplitMetaNamespaceKey(bmhKey)
	bmh := &bmov1.BareMetalHost{}
	key := client.ObjectKey{
		Name:      bmhName,
		Namespace: bmhNamespace,
	}

	err = c.Get(context.TODO(), key, bmh)
	if err != nil {
		return nil, err
	}
	return bmh, nil
}

// getNodeByMachine returns the node object referenced by machine
func getNodeByMachine(c client.Client, machine *mapiv1.Machine) (*corev1.Node, error) {
	if machine.Status.NodeRef == nil {
		return nil, errors.NewNotFound(corev1.Resource("ObjectReference"), machine.Name)
	}

	node := &corev1.Node{}
	key := client.ObjectKey{
		Name:      machine.Status.NodeRef.Name,
		Namespace: machine.Status.NodeRef.Namespace,
	}

	if err := c.Get(context.TODO(), key, node); err != nil {
		return nil, err
	}
	return node, nil
}

// deleteMachineNode deletes the node that mapped to specified machine
func deleteMachineNode(c client.Client, machine *mapiv1.Machine) error {
	node, err := getNodeByMachine(c, machine)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Warningf("The machine %q node does not exist", machine.Name)
			return nil
		}
		return err
	}

	return c.Delete(context.TODO(), node)
}
