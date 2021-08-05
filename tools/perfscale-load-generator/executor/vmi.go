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
 * Copyright 2021 IBM, Inc.
 *
 */

package executor

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	kvv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
)

const userData string = "#!/bin/bash\necho 'hello'\n"

// CreateVMISpecWithResources create a new VMI spec with random name and ephemeral disk, and with the given resource and namespace
func CreateVMISpecWithResources(namespace string, spec config.VMISpec) *kvv1.VirtualMachineInstance {
	imgName := fmt.Sprintf("%s/%s-container-disk-demo:%s", flags.VMIImgRepo, spec.VMImage, flags.VMIImgTag)
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(imgName, userData)
	vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse(spec.MEMLimit),
		k8sv1.ResourceCPU:    resource.MustParse(spec.CPULimit),
	}
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse(spec.MEMLimit),
		k8sv1.ResourceCPU:    resource.MustParse(spec.CPULimit),
	}
	vmi.ObjectMeta.Namespace = namespace

	return vmi
}

// CreateVMI create a new VMI with the given resource and namespace
func CreateVMI(virtCli kubecli.KubevirtClient, name string, namespace string, uuid string, spec config.VMISpec) {
	vmi := CreateVMISpecWithResources(namespace, spec)
	vmi.ObjectMeta.Labels = make(map[string]string)
	vmi.ObjectMeta.Labels[scenarioLabel] = uuid

	_, err := virtCli.VirtualMachineInstance(namespace).Create(vmi)
	if err != nil {
		log.Log.V(2).Errorf("Error creating VMI %s: %v", name, err)
	}
}

// WaitForRunningVMIs waits for vmCount VMIs enter in the Running phase
func WaitForRunningVMIs(virtCli kubecli.KubevirtClient, ns string, uuid string, vmCount int, maxWaitTimeout time.Duration) error {
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = fmt.Sprintf("%s=%s", scenarioLabel, uuid)
	return wait.PollImmediate(10*time.Second, maxWaitTimeout, func() (bool, error) {
		vmis, err := virtCli.VirtualMachineInstance(ns).List(&listOptions)
		if err != nil {
			return false, err
		}
		running := 0
		for _, vmi := range vmis.Items {
			if vmi.Status.Phase == kvv1.Running {
				running++
			}
		}
		if running >= vmCount {
			return true, nil
		}
		log.Log.V(4).Infof("Waiting for %d VMIs be Running from %d VMIs", vmCount-running, vmCount)
		return false, nil
	})
}

// DeleteVMIs delete all VMIs with with the given selector
func DeleteVMIs(virtCli kubecli.KubevirtClient, ns string, uuid string, limiter *rate.Limiter) {
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = fmt.Sprintf("%s=%s", scenarioLabel, uuid)
	vmis, err := virtCli.VirtualMachineInstance(ns).List(&listOptions)
	if (err != nil) || (len(vmis.Items) == 0) {
		return
	}
	for _, vmi := range vmis.Items {
		if err := virtCli.VirtualMachineInstance(ns).Delete(vmi.Name, &metav1.DeleteOptions{}); err != nil {
			log.Log.V(2).Errorf("Error creating namespace %s: %v", vmi.Name, err)
		}
		limiter.Wait(context.TODO())
	}
}

// WaitForDeleteVMIs wait all VMIs with the given selector to disapear
func WaitForDeleteVMIs(virtCli kubecli.KubevirtClient, ns string, uuid string) error {
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = fmt.Sprintf("%s=%s", scenarioLabel, uuid)
	return wait.PollImmediateInfinite(10*time.Second, func() (bool, error) {
		vmis, err := virtCli.VirtualMachineInstance(ns).List(&listOptions)
		if err != nil {
			return false, err
		}
		if len(vmis.Items) == 0 {
			return true, nil
		}
		log.Log.V(4).Infof("Waiting for %d VMIs labeled with %s to be removed", len(vmis.Items), fmt.Sprintf("%s=%s", scenarioLabel, uuid))
		return false, nil
	})
}
