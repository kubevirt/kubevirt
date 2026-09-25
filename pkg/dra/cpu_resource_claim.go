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

func guestCPUTopology(cpu *v1.CPU) (cores, threads, sockets uint32) {
	cores, threads, sockets = 1, 1, 1
	if cpu == nil {
		return cores, threads, sockets
	}
	if cpu.Cores != 0 {
		cores = cpu.Cores
	}
	if cpu.Threads != 0 {
		threads = cpu.Threads
	}
	if cpu.Sockets != 0 {
		sockets = cpu.Sockets
	}
	return cores, threads, sockets
}

// cpuHostCPUs returns every host CPU a dedicated CPU VM needs: its guest vCPUs, plus the
// supplementalPool IO threads and the isolated emulator thread when it asks for them.
func cpuHostCPUs(vmi *v1.VirtualMachineInstance) int64 {
	cores, threads, sockets := guestCPUTopology(vmi.Spec.Domain.CPU)
	guestVCPUs := hardware.GetNumberOfVCPUs(&v1.CPU{
		Cores:   cores,
		Threads: threads,
		Sockets: sockets,
	})
	return guestVCPUs + hardware.GetSupplementalDedicatedHostCPUs(vmi, guestVCPUs)
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
				v1.AppLabel:            "virt-launcher",
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

func CreateCPUResourceClaim(vmi *v1.VirtualMachineInstance, clientset kubecli.KubevirtClient) error {
	logger := log.Log.Object(vmi)
	claimName := CPUResourceClaimName(vmi.Name)

	_, err := clientset.ResourceV1().ResourceClaims(vmi.Namespace).Get(context.TODO(), claimName, metav1.GetOptions{})
	if err == nil {
		logger.V(logVerbosityDebug).Infof("CPU ResourceClaim %s/%s already exists", vmi.Namespace, claimName)
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get existing CPU ResourceClaim: %v", err)
	}

	claim := generateCPUResourceClaim(vmi, claimName)
	_, err = clientset.ResourceV1().ResourceClaims(vmi.Namespace).Create(context.TODO(), claim, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create CPU ResourceClaim: %v", err)
	}
	return nil
}
