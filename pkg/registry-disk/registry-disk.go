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

package registrydisk

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/jeevatkm/go-model"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/precond"
)

const registryDiskV1Alpha = "ContainerRegistryDisk:v1alpha"
const defaultIqn = "iqn.2017-01.io.kubevirt:wrapper/1"
const defaultPort = 3261
const defaultPortStr = "3261"
const defaultHost = "127.0.0.1"

func generateRandomString(len int) (string, error) {
	bytes := make([]byte, len)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

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

func k8sSecretName(vm *v1.VM) string {
	return fmt.Sprintf("registrydisk-iscsi-%s-%s", vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
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

			newDisk.Auth = disk.Auth

			vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
		}
	}

	return vmCopy, nil
}

func CleanUp(vm *v1.VM, virtCli kubecli.KubevirtClient) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())

	labelSelector, err := labels.Parse(fmt.Sprintf(v1.DomainLabel+" in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}

	listOptions := metav1.ListOptions{LabelSelector: labelSelector.String()}

	if err := virtCli.CoreV1().Secrets(vm.ObjectMeta.Namespace).DeleteCollection(nil, listOptions); err != nil {
		return err
	}

	return nil
}

// The controller applies ports to registry disks when a VM spec is introduced into the cluster.
func Initialize(vm *v1.VM, virtCli kubecli.KubevirtClient) error {

	var err error
	authUser := ""
	secretID := k8sSecretName(vm)
	wrapperStartingPort := defaultPort
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {

			// Generate Dynamic Auth Credential for Registry Disks the first
			// time one is encountered. This random user will be used to authenticate
			// all ephemeral disks associated with this VM.
			if authUser == "" {
				authUser, err = generateRandomString(32)
				if err != nil {
					return err
				}
			}

			port := fmt.Sprintf("%d", wrapperStartingPort)
			name := defaultHost
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

			// Reference k8s secret in Disk device
			vm.Spec.Domain.Devices.Disks[idx].Auth = &v1.DiskAuth{
				Secret: &v1.DiskSecret{
					Type:  "iscsi",
					Usage: secretID,
				},
			}

		}
	}

	/* add k8s secret if authUser was generated */
	if authUser != "" {
		pass, err := generateRandomString(32)
		if err != nil {
			return err
		}
		pass64 := base64.StdEncoding.EncodeToString([]byte(pass))
		user64 := base64.StdEncoding.EncodeToString([]byte(authUser))

		// Store Auth as k8s secret
		secret := kubev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretID,
				Namespace: vm.GetObjectMeta().GetNamespace(),
				Labels: map[string]string{
					v1.DomainLabel: domain,
				},
			},
			Type: "kubernetes.io/iscsi-chap",
			Data: map[string][]byte{
				"node.session.auth.password": []byte(pass64),
				"node.session.auth.username": []byte(user64),
			},
		}

		_, err = virtCli.CoreV1().Secrets(vm.ObjectMeta.Namespace).Get(secretID, metav1.GetOptions{})
		if err != nil {
			if _, err := virtCli.Core().Secrets(vm.GetObjectMeta().GetNamespace()).Create(&secret); err != nil {
				return err
			}
		}
	}
	return nil
}

// The controller uses this function communicate the IP address of the POD
// with the containers hosting the registry disks.
func ApplyHost(vm *v1.VM, pod *kubev1.Pod) {
	ip := pod.Status.PodIP
	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {
			port := defaultPortStr
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
					kubev1.EnvVar{
						Name: "PASSWORD_BASE64",
						ValueFrom: &kubev1.EnvVarSource{
							SecretKeyRef: &kubev1.SecretKeySelector{
								LocalObjectReference: kubev1.LocalObjectReference{
									Name: k8sSecretName(vm),
								},
								Key: "node.session.auth.password",
							},
						},
					},
					kubev1.EnvVar{
						Name: "USERNAME_BASE64",
						ValueFrom: &kubev1.EnvVarSource{
							SecretKeyRef: &kubev1.SecretKeySelector{
								LocalObjectReference: kubev1.LocalObjectReference{
									Name: k8sSecretName(vm),
								},
								Key: "node.session.auth.username",
							},
						},
					},
				},
				// The readiness probes ensure the ISCSI targets are available
				// before the container is marked as "Ready: True"
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
			})
		}
	}
	return containers, nil
}
