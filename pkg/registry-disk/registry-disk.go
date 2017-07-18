/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package registrydisk

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jeevatkm/go-model"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

const registryDiskV1Alpha = "ContainerRegistryDisk:v1alpha"
const defaultIqn = "iqn.2017-01.io.kubevirt:wrapper/1"

func DisksAreReady(pod *kubev1.Pod) bool {
	// Wait for readiness probes on image wrapper containers
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if strings.Contains(containerStatus.Name, "disk") == false {
			// only check readiness of disk containers
			continue
		}
		if containerStatus.Ready == false {
			return false
		}
	}
	return true
}

// The virt-handler converts registry disks to their corresponding iscsi network
// disks when the VM spec is being defined as a domain with libvirt.
// The ports and host of the iscsi disks are already provided here by the controller.
func MapRegistryDisks(vm *v1.VM) (*v1.VM, error) {
	vmCopy := &v1.VM{}
	model.Copy(vmCopy, vm)

	for idx, disk := range vmCopy.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {
			newDisk := v1.Disk{}

			newDisk.Type = "network"
			newDisk.Device = "disk"
			newDisk.Target = disk.Target

			newDisk.Driver = &v1.DiskDriver{
				Type:  "raw",
				Name:  "qemu",
				Cache: "none",
			}

			newDisk.Source.Name = defaultIqn
			newDisk.Source.Protocol = "iscsi"
			newDisk.Source.Host = disk.Source.Host

			vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
		}
	}

	return vmCopy, nil
}

// TODO Introduce logic that dynamically generates iscsi CHAP
// Authentication credentials for a VM spec backed by registry disks.
//
//func ApplyAuth(vm *v1.VM) {
//     INSERT DYNAMIC AUTH LOGIC HERE
//}

// The controller applies ports to registry disks when a VM spec is introduced into the cluster.
func ApplyPorts(vm *v1.VM) {
	wrapperStartingPort := 3261
	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {
			port := fmt.Sprintf("%d", wrapperStartingPort)
			name := "127.0.0.1"
			if disk.Source.Host != nil {
				name = disk.Source.Host.Name
			}
			// We fill in the port here for the iscsi target
			// to coordinate avoiding port collisions in the virt-launcher pod.
			vm.Spec.Domain.Devices.Disks[idx].Source.Host = &v1.DiskSourceHost{
				Port: port,
				Name: name,
			}
			wrapperStartingPort++
		}
	}
}

// The controller uses this function communicate the IP address of the POD
// with the containers hosting the registry disks.
func ApplyHost(vm *v1.VM, pod *kubev1.Pod) {
	ip := pod.Status.PodIP
	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {
			port := "3261"
			name := ip
			if disk.Source.Host != nil {
				port = disk.Source.Host.Port
			}
			vm.Spec.Domain.Devices.Disks[idx].Source.Host = &v1.DiskSourceHost{
				Port: port,
				Name: name,
			}
		}
	}
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func GenerateContainers(vm *v1.VM) ([]kubev1.Container, error) {
	var containers []kubev1.Container

	initialDelaySeconds := 5
	timeoutSeconds := 5
	periodSeconds := 10
	successThreshold := 2
	failureThreshold := 5

	// Make VM Image Wrapper Containers
	diskCount := 0
	for _, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {
			diskContainerName := fmt.Sprintf("disk%d", diskCount)
			// container image is disk.Source.Name
			diskContainerImage := disk.Source.Name
			// disk.Source.Host.Port has the port field expanded before templating.
			port := disk.Source.Host.Port
			portInt, err := strconv.Atoi(port)
			if err != nil {
				return nil, err
			}
			containers = append(containers, kubev1.Container{
				Name:            diskContainerName,
				Image:           diskContainerImage,
				ImagePullPolicy: kubev1.PullIfNotPresent,
				Command:         []string{"/entry-point.sh", port},
				Ports: []kubev1.ContainerPort{
					kubev1.ContainerPort{
						ContainerPort: int32(portInt),
						Protocol:      kubev1.ProtocolTCP,
					},
				},
				Env: []kubev1.EnvVar{
					kubev1.EnvVar{
						Name:  "PORT",
						Value: port,
					},
					// TODO once dynamic auth is implemented, pass creds as
					// PASSWORD and USERNAME env vars. The registry disk base
					// container already knows how to enable authentication
					// when those env vars are present.
				},
				// The readiness probes ensure the ISCSI targets are available
				// before the container is marked as "Ready: True"
				ReadinessProbe: &kubev1.Probe{
					Handler: kubev1.Handler{
						TCPSocket: &kubev1.TCPSocketAction{
							Port: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: int32(portInt),
							},
						},
					},
					InitialDelaySeconds: int32(initialDelaySeconds),
					PeriodSeconds:       int32(periodSeconds),
					TimeoutSeconds:      int32(timeoutSeconds),
					SuccessThreshold:    int32(successThreshold),
					FailureThreshold:    int32(failureThreshold),
				},
			})
		}
	}
	return containers, nil
}
