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

package dra

import (
	"context"
	"fmt"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/hardware"
)

const (
	CPUDeviceClassName   = "dra.cpu"
	CPUCapacityAttribute = resourcev1.QualifiedName("dra.cpu/cpu")
	CPURequestName       = "req-cpu"
	CPUClaimRefName      = "kubevirt-cpu-claim-ref"
	logVerbosityDebug    = 4
)

func CPUResourceClaimName(vmiName string) string {
	return fmt.Sprintf("%s-cpu-claim", vmiName)
}

// guestVCPUs returns the number of vCPUs the guest sees. GetNumberOfVCPUs reads an unset
// topology field as a factor of one, but an entirely unset topology as no CPUs at all, so fall
// back to the single vCPU such a guest really boots with.
func guestVCPUs(vmi *v1.VirtualMachineInstance) int64 {
	var vCPUs int64
	if cpu := vmi.Spec.Domain.CPU; cpu != nil {
		vCPUs = hardware.GetNumberOfVCPUs(cpu)
	}
	return max(vCPUs, 1)
}

// cpuHostCPUs returns every host CPU a dedicated CPU VM needs: its guest vCPUs, plus the
// supplementalPool IO threads and the isolated emulator thread when it asks for them.
func cpuHostCPUs(vmi *v1.VirtualMachineInstance) int64 {
	vCPUs := guestVCPUs(vmi)
	return vCPUs + hardware.GetSupplementalDedicatedHostCPUs(vmi, vCPUs)
}

func cpuDeviceRequest(name string, hostCPUs int64) resourcev1.DeviceRequest {
	return resourcev1.DeviceRequest{
		Name: name,
		Exactly: &resourcev1.ExactDeviceRequest{
			DeviceClassName: CPUDeviceClassName,
			Count:           1,
			AllocationMode:  resourcev1.DeviceAllocationModeExactCount,
			Capacity: &resourcev1.CapacityRequirements{
				Requests: map[resourcev1.QualifiedName]resource.Quantity{
					CPUCapacityAttribute: *resource.NewQuantity(hostCPUs, resource.DecimalSI),
				},
			},
		},
	}
}

func generateCPUResourceClaim(vmi *v1.VirtualMachineInstance, claimName string) *resourcev1.ResourceClaim {
	return &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: vmi.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
			Labels: map[string]string{
				v1.CreatedByLabel:      string(vmi.UID),
				"kubevirt.io/resource": "cpu-dra",
			},
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{
					cpuDeviceRequest(CPURequestName, cpuHostCPUs(vmi)),
				},
			},
		},
	}
}

func validateCPUResourceClaimOwner(claim *resourcev1.ResourceClaim, vmi *v1.VirtualMachineInstance) error {
	controllerRef := metav1.GetControllerOf(claim)
	if controllerRef == nil || controllerRef.UID != vmi.UID {
		return fmt.Errorf(
			"CPU ResourceClaim %s/%s is not controlled by VMI UID %s",
			claim.Namespace,
			claim.Name,
			vmi.UID,
		)
	}
	return nil
}

func CreateCPUResourceClaim(ctx context.Context, vmi *v1.VirtualMachineInstance, clientset kubecli.KubevirtClient) error {
	claimName := CPUResourceClaimName(vmi.Name)

	existingClaim, err := clientset.ResourceV1().ResourceClaims(vmi.Namespace).Get(ctx, claimName, metav1.GetOptions{})
	if err == nil {
		if ownerErr := validateCPUResourceClaimOwner(existingClaim, vmi); ownerErr != nil {
			return ownerErr
		}
		log.Log.Object(vmi).V(logVerbosityDebug).Infof("CPU ResourceClaim %s/%s already exists", vmi.Namespace, claimName)
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get existing CPU ResourceClaim: %v", err)
	}

	claim := generateCPUResourceClaim(vmi, claimName)
	_, err = clientset.ResourceV1().ResourceClaims(vmi.Namespace).Create(ctx, claim, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create CPU ResourceClaim: %v", err)
	}
	return nil
}
