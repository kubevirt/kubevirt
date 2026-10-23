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

	"github.com/openshift/library-go/pkg/build/naming"
	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util/hardware"
)

const (
	// Device class and capacity attribute published by the DRA CPU driver.
	// Both are fixed for alpha. Another driver would need them to become configuration.
	CPUDeviceClassName   = "dra.cpu"
	CPUCapacityAttribute = resourcev1.QualifiedName("dra.cpu/cpu")
	// CPURequestName is the single DeviceRequest on the claim. It asks for one device: an
	// exclusive host cpuset covering the guest vCPUs, supplemental IO threads, and the isolated
	// emulator thread. "generic" means that set is not aligned to a NUMA node or a PCIe device.
	// A later request can replace "generic" with a topology, for example numa or pcie.
	CPURequestName             = "exclusive-generic-cpu-request"
	CPUClaimRefName            = "generated-kubevirt-cpu-pod-claim-ref"
	cpuResourceClaimNameSuffix = "generated-kubevirt-generic-cpu"
)

// CPUResourceClaimName is the ResourceClaim object name for vmiName. The VMI name is the base, so
// naming.GetName truncates that name when the result would pass 253 characters and keeps this suffix.
func CPUResourceClaimName(vmiName string) string {
	return naming.GetName(vmiName, cpuResourceClaimNameSuffix, validation.DNS1123SubdomainMaxLength)
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
				v1.CreatedByLabel: string(vmi.UID),
			},
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{
					cpuDeviceRequest(CPURequestName, hardware.GetDedicatedHostCPUs(vmi)),
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
	claim := generateCPUResourceClaim(vmi, claimName)
	_, err := clientset.ResourceV1().ResourceClaims(vmi.Namespace).Create(ctx, claim, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		existing, getErr := clientset.ResourceV1().ResourceClaims(vmi.Namespace).Get(ctx, claimName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return validateCPUResourceClaimOwner(existing, vmi)
	}

	if err != nil {
		return fmt.Errorf("failed to create CPU ResourceClaim: %v", err)
	}

	return nil
}
