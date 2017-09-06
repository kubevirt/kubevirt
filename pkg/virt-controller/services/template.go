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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package services

import (
	"fmt"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
	"strings"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VM) (*kubev1.Pod, error)
	RenderMigrationJob(*v1.VM, *kubev1.Node, *kubev1.Node, *kubev1.Pod) (*kubev1.Pod, error)
}

type templateService struct {
	launcherImage string
	migratorImage string
}

//Deprecated: remove the service and just use a builder or contextcless helper function
func (t *templateService) RenderLaunchManifest(vm *v1.VM) (*kubev1.Pod, error) {
	precond.MustNotBeNil(vm)
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	uid := precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

	// Compute resource requirements taking the VM overhead into account
	memoryRequested, memoryLimit, e := computeTotalMemoryRequirements(vm)
	if e != nil {
		return nil, e
	}

	resources := kubev1.ResourceRequirements{
		Requests: kubev1.ResourceList{
			kubev1.ResourceMemory: *memoryRequested,
		},
		Limits: kubev1.ResourceList{
			kubev1.ResourceMemory: *memoryLimit,
		},
	}

	// VM target container
	container := kubev1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: kubev1.PullIfNotPresent,
		Command:         []string{"/virt-launcher", "--qemu-timeout", "60s"},
		Resources:       resources,
	}

	containers, err := registrydisk.GenerateContainers(vm)
	if err != nil {
		return nil, err
	}
	containers = append(containers, container)

	// TODO use constants for labels
	pod := kubev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-----",
			Labels: map[string]string{
				v1.AppLabel:    "virt-launcher",
				v1.DomainLabel: domain,
				v1.VMUIDLabel:  uid,
			},
		},
		Spec: kubev1.PodSpec{
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers:    containers,
			NodeSelector:  vm.Spec.NodeSelector,
		},
	}

	return &pod, nil
}

func (t *templateService) RenderMigrationJob(vm *v1.VM, sourceNode *kubev1.Node, targetNode *kubev1.Node, targetPod *kubev1.Pod) (*kubev1.Pod, error) {
	srcAddr := ""
	dstAddr := ""
	for _, addr := range sourceNode.Status.Addresses {
		if (addr.Type == kubev1.NodeInternalIP) && (srcAddr == "") {
			srcAddr = addr.Address
			break
		}
	}
	if srcAddr == "" {
		err := fmt.Errorf("migration source node is unreachable")
		logging.DefaultLogger().Error().Msg("migration target node is unreachable")
		return nil, err
	}
	srcUri := fmt.Sprintf("qemu+tcp://%s/system", srcAddr)

	for _, addr := range targetNode.Status.Addresses {
		if (addr.Type == kubev1.NodeInternalIP) && (dstAddr == "") {
			dstAddr = addr.Address
			break
		}
	}
	if dstAddr == "" {
		err := fmt.Errorf("migration target node is unreachable")
		logging.DefaultLogger().Error().Msg("migration target node is unreachable")
		return nil, err
	}
	destUri := fmt.Sprintf("qemu+tcp://%s/system", dstAddr)

	job := kubev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-migration",
			Labels: map[string]string{
				v1.DomainLabel: vm.GetObjectMeta().GetName(),
				v1.AppLabel:    "migration",
			},
		},
		Spec: kubev1.PodSpec{
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers: []kubev1.Container{
				{
					Name:  "virt-migration",
					Image: t.migratorImage,
					Command: []string{
						"/migrate", vm.ObjectMeta.Name,
						"--source", srcUri,
						"--dest", destUri,
						"--node-ip", dstAddr,
						"--namespace", vm.ObjectMeta.Namespace,
					},
				},
			},
		},
	}

	return &job, nil
}

func NewTemplateService(launcherImage string, migratorImage string) (TemplateService, error) {
	precond.MustNotBeEmpty(launcherImage)
	precond.MustNotBeEmpty(migratorImage)
	svc := templateService{
		launcherImage: launcherImage,
		migratorImage: migratorImage,
	}
	return &svc, nil
}

// ComputeTotalMemoryRequirements computes the estimation of total
// memory needed for the domain to operate properly.
// This includes the memory needed for the guest and memory
// for Qemu and OS overhead.
//
// The return values are requested memory and limit memory quantities
//
// Note: This is the best estimation we were able to come up with
//       and is still not 100% accurate
func computeTotalMemoryRequirements(vm *v1.VM) (*resource.Quantity, *resource.Quantity, error) {
	domain := vm.Spec.Domain

	// TODO this is pretty naive and does not work reliably, the libvirt units are different from quantity suffixes
	memory, err := resource.ParseQuantity(fmt.Sprintf("%d%s", vm.Spec.Domain.Memory.Value, vm.Spec.Domain.Memory.Unit))
	if err != nil {
	  return nil, nil, err
	}

	limit, err := resource.ParseQuantity(fmt.Sprintf("%d%s", vm.Spec.Domain.MaxMemory.Value, vm.Spec.Domain.MaxMemory.Unit))
	if err != nil {
		return nil, nil, err
	}

	overhead := resource.NewScaledQuantity(0, resource.Kilo)

	if strings.HasPrefix(domain.OS.Type.Arch, "ppc") {
		// Add memory needed to support memory hotplug
		// maxRam = limit if hotplug supported else memory
		// int powerOf2 = Integer.highestOneBit(maxRam);
		// pageTable = (Memory.Limit > powerOf2 ? powerOf2 * 2 : powerOf2) / 64;
	} else {
		// Add the memory needed for pagetables (one bit for every 512b of RAM size)
		pagetableMemory := resource.NewScaledQuantity(memory.ScaledValue(resource.Kilo), resource.Kilo)
		pagetableMemory.Set(pagetableMemory.Value() / 512)
		overhead.Add(*pagetableMemory)
	}

	// Add fixed overhead for shared libraries and such
	// TODO account for the overhead of kubevirt components running in the pod
	if strings.HasPrefix(domain.OS.Type.Arch, "ppc") {
		overhead.Add(*resource.NewScaledQuantity(100, resource.Mega))
	} else {
		overhead.Add(*resource.NewScaledQuantity(64, resource.Mega))
	}

	// Add CPU table overhead (8 MiB per vCPU and 8 MiB per IO thread)
	// TODO

	// Add video RAM overhead
	for _, vg := range domain.Devices.Video {
		overhead.Add(*resource.NewScaledQuantity(int64(*vg.Ram), resource.Kilo)) // TODO check units!
		overhead.Add(*resource.NewScaledQuantity(int64(*vg.VRam), resource.Mega))
	}

	memory.Add(*overhead)
	limit.Add(*overhead)

	return &memory, &limit, nil
}
