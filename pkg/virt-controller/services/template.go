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

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
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

	// VM target container
	container := kubev1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: kubev1.PullIfNotPresent,
		Command:         []string{"/virt-launcher", "--qemu-timeout", "60s"},
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
