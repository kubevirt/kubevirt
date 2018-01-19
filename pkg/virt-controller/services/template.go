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
	"strconv"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
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

	var volumes []kubev1.Volume
	var userId int64 = 0
	var privileged bool = true

	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vm.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vm.Spec.TerminationGracePeriodSeconds
	}

	// Pad the virt-launcher grace period.
	// Ideally we want virt-handler to handle tearing down
	// the vm without virt-launcher's termination forcing
	// the vm down.
	gracePeriodSeconds = gracePeriodSeconds + int64(15)
	gracePeriodKillAfter := gracePeriodSeconds + int64(15)

	// TODO remove mounting the privateDir once
	// the qemu pid is in the pods mount namespace
	privateDir := fmt.Sprintf("%s-private/%s/%s", t.virtShareDir, namespace, domain)

	// VM target container
	container := kubev1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: kubev1.PullIfNotPresent,
		// Privileged mode is required for /dev/kvm and the
		// ability to create macvtap devices
		SecurityContext: &kubev1.SecurityContext{
			RunAsUser:  &userId,
			Privileged: &privileged,
		},
		Command: []string{"/entrypoint.sh",
			"--qemu-timeout", "5m",
			"--name", domain,
			"--namespace", namespace,
			"--kubevirt-share-dir", t.virtShareDir,
			"--readiness-file", "/tmp/healthy",
			"--grace-period-seconds", strconv.Itoa(int(gracePeriodSeconds)),
		},
		VolumeMounts: []kubev1.VolumeMount{
			{
				Name:      "virt-share-dir",
				MountPath: t.virtShareDir,
			},
			{
				Name:      "virt-private-dir",
				MountPath: privateDir,
			},
			{
				// shared with registry disks on Pod shared volume
				Name:      "libvirt-runtime",
				MountPath: "/var/run/libvirt",
			},
			{
				Name:      "host-dev",
				MountPath: "/host-dev",
			},
			{
				Name:      "host-sys",
				MountPath: "/host-sys",
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

	containers, err := registrydisk.GenerateContainers(vm, "libvirt-runtime", "/var/run/libvirt")
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
	volumes = append(volumes, kubev1.Volume{
		Name: "virt-private-dir",
		VolumeSource: kubev1.VolumeSource{
			EmptyDir: &kubev1.EmptyDirVolumeSource{},
		},
	})

	volumes = append(volumes, kubev1.Volume{
		Name: "libvirt-runtime",
		VolumeSource: kubev1.VolumeSource{
			EmptyDir: &kubev1.EmptyDirVolumeSource{},
		},
	})

	volumes = append(volumes, kubev1.Volume{
		Name: "host-dev",
		VolumeSource: kubev1.VolumeSource{
			HostPath: &kubev1.HostPathVolumeSource{
				Path: "/dev",
			},
		},
	})
	volumes = append(volumes, kubev1.Volume{
		Name: "host-sys",
		VolumeSource: kubev1.VolumeSource{
			HostPath: &kubev1.HostPathVolumeSource{
				Path: "/sys",
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
			HostNetwork: true,
			SecurityContext: &kubev1.PodSecurityContext{
				RunAsUser: &userId,
			},
			TerminationGracePeriodSeconds: &gracePeriodKillAfter,
			RestartPolicy:                 kubev1.RestartPolicyNever,
			Containers:                    containers,
			NodeSelector:                  vm.Spec.NodeSelector,
			Volumes:                       volumes,
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
		log.Log.Error("migration target node is unreachable")
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
