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

package libpod

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v13 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func RenderPrivilegedPod(name string, cmd, args []string) *v1.Pod {
	pod := v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Namespace:    testsuite.NamespacePrivileged,
			GenerateName: name,
			Labels: map[string]string{
				v13.AppLabel: "test",
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			HostPID:       true,
			SecurityContext: &v1.PodSecurityContext{
				RunAsUser: pointer.P(int64(0)),
			},
			Containers: []v1.Container{
				renderPrivilegedContainerSpec(
					libregistry.GetUtilityImageFromRegistry("vm-killer"),
					"container",
					cmd,
					args),
			},
		},
	}

	return &pod
}

func RenderPod(name string, cmd, args []string) *v1.Pod {
	pod := v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			GenerateName: name,
			Labels: map[string]string{
				v13.AppLabel: "test",
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				renderContainerSpec(
					libregistry.GetUtilityImageFromRegistry("vm-killer"),
					"container",
					cmd,
					args),
			},
		},
	}

	return &pod
}

func renderContainerSpec(imgPath, name string, cmd, args []string) v1.Container {
	return v1.Container{
		Name:    name,
		Image:   imgPath,
		Command: cmd,
		Args:    args,
		SecurityContext: &v1.SecurityContext{
			Privileged:               pointer.P(false),
			AllowPrivilegeEscalation: pointer.P(false),
			RunAsNonRoot:             pointer.P(true),
			SeccompProfile: &v1.SeccompProfile{
				Type: v1.SeccompProfileTypeRuntimeDefault,
			},
			Capabilities: &v1.Capabilities{
				Drop: []v1.Capability{"ALL"},
			},
		},
	}
}

func renderPrivilegedContainerSpec(imgPath, name string, cmd, args []string) v1.Container {
	return v1.Container{
		Name:    name,
		Image:   imgPath,
		Command: cmd,
		Args:    args,
		SecurityContext: &v1.SecurityContext{
			Privileged: pointer.P(true),
			RunAsUser:  new(int64),
		},
	}
}

func RenderHostPathPod(
	podName, dir string, hostPathType v1.HostPathType, mountPropagation v1.MountPropagationMode, cmd, args []string,
) *v1.Pod {
	pod := RenderPrivilegedPod(podName, cmd, args)
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
		Name:             "hostpath-mount",
		MountPropagation: &mountPropagation,
		MountPath:        dir,
	})
	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: "hostpath-mount",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: dir,
				Type: &hostPathType,
			},
		},
	})

	return pod
}

func RenderTargetcliPod(name, disksPVC string) *v1.Pod {
	const (
		disks   = "disks"
		dbus    = "dbus"
		modules = "modules"
	)
	hostPathDirectory := v1.HostPathDirectory
	targetcliContainer := renderPrivilegedContainerSpec(
		libregistry.GetUtilityImageFromRegistry("vm-killer"),
		"targetcli", []string{"tail", "-f", "/dev/null"}, []string{})
	targetcliContainer.VolumeMounts = []v1.VolumeMount{
		{
			Name:      disks,
			ReadOnly:  false,
			MountPath: "/disks",
		},
		{
			Name:      dbus,
			ReadOnly:  false,
			MountPath: "/var/run/dbus",
		},
		{
			Name:      modules,
			ReadOnly:  false,
			MountPath: "/lib/modules",
		},
	}
	return &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				v13.AppLabel: "test",
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers:    []v1.Container{targetcliContainer},
			Volumes: []v1.Volume{
				// PVC where we store the backend for the SCSI disks
				{
					Name: disks,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: disksPVC,
							ReadOnly:  false,
						},
					},
				},
				{
					Name: dbus,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/var/run/dbus",
							Type: &hostPathDirectory,
						},
					},
				},
				{
					Name: modules,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/lib/modules",
							Type: &hostPathDirectory,
						},
					},
				},
			},
		},
	}
}
