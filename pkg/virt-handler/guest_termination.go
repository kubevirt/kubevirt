/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virthandler

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	vhmetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func guestTerminatedCondition(domain *api.Domain) *v1.VirtualMachineInstanceCondition {
	if domain == nil {
		return nil
	}

	if domain.Status.TerminationEvent != nil && domain.Status.TerminationEvent.Reason != "" {
		terminationEvent := domain.Status.TerminationEvent
		eventTime := terminationConditionTime(terminationEvent.Timestamp)
		return &v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstanceGuestTerminated,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      eventTime,
			LastTransitionTime: eventTime,
			Reason:             string(terminationEvent.Reason),
			Message:            terminationEvent.Reason.Message(),
		}
	}

	if domainIsActive(domain) {
		now := metav1.Now()
		return &v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstanceGuestTerminated,
			Status:             k8sv1.ConditionFalse,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             v1.GuestNotTerminatedReason,
			Message:            "Guest has not terminated",
		}
	}

	return nil
}

func shouldIncGuestOSTermination(oldCondition, condition *v1.VirtualMachineInstanceCondition) bool {
	if condition == nil || condition.Status != k8sv1.ConditionTrue {
		return false
	}

	if oldCondition == nil || oldCondition.Status != k8sv1.ConditionTrue {
		return true
	}

	return oldCondition.Reason != condition.Reason
}

func domainIsActive(domain *api.Domain) bool {
	switch domain.Status.Status {
	// Shutdown is a transitional lifecycle, not a final state. Keep it active so
	// GuestTerminated remains False until a normalized termination event is observed.
	case api.Running, api.Blocked, api.Paused, api.Shutdown, api.PMSuspended:
		return true
	default:
		return false
	}
}

func terminationConditionTime(timestamp metav1.Time) metav1.Time {
	if timestamp.IsZero() {
		return metav1.Now()
	}
	return timestamp
}

func (c *VirtualMachineController) updateGuestTerminatedCondition(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {
	if !c.clusterConfig.GuestTerminationEnabled() {
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceGuestTerminated)
		return
	}

	// A nil condition means no new normalized termination signal was observed.
	// Keep any existing GuestTerminated condition so later low-signal terminal
	// domain updates do not erase the last recorded reason.
	condition := guestTerminatedCondition(domain)
	if condition == nil {
		return
	}

	oldCondition := condManager.GetCondition(vmi, v1.VirtualMachineInstanceGuestTerminated)
	if oldCondition != nil {
		oldConditionCopy := *oldCondition
		oldCondition = &oldConditionCopy
	}

	if shouldIncGuestOSTermination(oldCondition, condition) {
		reason := api.TerminationReason(condition.Reason)
		if api.IsSupportedTerminationReason(reason) {
			vhmetrics.IncGuestOSTermination(vmi.Namespace, vmi.Name, reason)
			c.recordGuestTerminationEvent(vmi, reason)
		}
	}
	condManager.UpdateCondition(vmi, condition)
}

func (c *VirtualMachineController) recordGuestTerminationEvent(vmi *v1.VirtualMachineInstance, reason api.TerminationReason) {
	c.recorder.Event(vmi, guestTerminationEventType(reason), string(reason), reason.Message())
}

func guestTerminationEventType(reason api.TerminationReason) string {
	switch reason {
	case api.TerminationReasonGuestCrashed, api.TerminationReasonHostStoppedFailed:
		return k8sv1.EventTypeWarning
	default:
		return k8sv1.EventTypeNormal
	}
}
