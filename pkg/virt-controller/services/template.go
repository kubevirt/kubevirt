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

	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/networking"
	"kubevirt.io/kubevirt/pkg/precond"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VirtualMachine) (*kubev1.Pod, error)
	RenderMigrationJob(*v1.VirtualMachine, *kubev1.Node, *kubev1.Node, *kubev1.Pod, *v1.MigrationHostInfo) (*kubev1.Pod, error)
}

type templateService struct {
	launcherImage string
	migratorImage string
	virtShareDir  string
}

func (t *templateService) RenderLaunchManifest(vm *v1.VirtualMachine) (*kubev1.Pod, error) {
	precond.MustNotBeNil(vm)
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	uid := precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

	initialDelaySeconds := 2
	timeoutSeconds := 5
	periodSeconds := 2
	successThreshold := 1
	failureThreshold := 5

	// VM target container
	container := kubev1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: kubev1.PullIfNotPresent,
		Command: []string{"/virt-launcher",
			"--qemu-timeout", "5m",
			"--name", domain,
			"--namespace", namespace,
			"--kubevirt-share-dir", t.virtShareDir,
			"--readiness-file", "/tmp/healthy",
		},
		VolumeMounts: []kubev1.VolumeMount{
			{
				Name:      "virt-share-dir",
				MountPath: t.virtShareDir,
			},
		},
		ReadinessProbe: &kubev1.Probe{
			Handler: kubev1.Handler{
				Exec: &kubev1.ExecAction{
					Command: []string{
						"cat",
						"/tmp/healthy",
					},
				},
			},
			InitialDelaySeconds: int32(initialDelaySeconds),
			PeriodSeconds:       int32(periodSeconds),
			TimeoutSeconds:      int32(timeoutSeconds),
			SuccessThreshold:    int32(successThreshold),
			FailureThreshold:    int32(failureThreshold),
		},
	}

	containers, volumes, err := registrydisk.GenerateContainers(vm)
	if err != nil {
		return nil, err
	}

	volumes = append(volumes, kubev1.Volume{
		Name: "virt-share-dir",
		VolumeSource: kubev1.VolumeSource{
			HostPath: &kubev1.HostPathVolumeSource{
				Path: t.virtShareDir,
			},
		},
	})
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
			HostPID:       true,
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers:    containers,
			NodeSelector:  vm.Spec.NodeSelector,
			Volumes:       volumes,
		},
	}

	if vm.Spec.Affinity != nil {
		pod.Spec.Affinity = &kubev1.Affinity{}

		if vm.Spec.Affinity.NodeAffinity != nil {
			pod.Spec.Affinity.NodeAffinity = vm.Spec.Affinity.NodeAffinity
		}
	}

	return &pod, nil
}

func (t *templateService) RenderMigrationJob(vm *v1.VirtualMachine, sourceNode *kubev1.Node, targetNode *kubev1.Node, targetPod *kubev1.Pod, targetHostInfo *v1.MigrationHostInfo) (*kubev1.Pod, error) {

	srcAddr := networking.GetNodeInternalIP(sourceNode)
	if srcAddr == "" {
		err := fmt.Errorf("migration source node is unreachable")
		log.Log.Error("migration target node is unreachable")
		return nil, err
	}
	srcUri := fmt.Sprintf("qemu+tcp://%s/system", srcAddr)

	dstAddr := networking.GetNodeInternalIP(targetNode)
	if dstAddr == "" {
		err := fmt.Errorf("migration target node is unreachable")
		log.Log.Error("migration target node is unreachable")
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
						"--slice", targetHostInfo.Slice,
						"--controller", strings.Join(targetHostInfo.Controller, ","),
					},
				},
			},
		},
	}

	return &job, nil
}

func NewTemplateService(launcherImage string, migratorImage string, virtShareDir string) (TemplateService, error) {
	precond.MustNotBeEmpty(launcherImage)
	precond.MustNotBeEmpty(migratorImage)
	svc := templateService{
		launcherImage: launcherImage,
		migratorImage: migratorImage,
		virtShareDir:  virtShareDir,
	}
	return &svc, nil
}
