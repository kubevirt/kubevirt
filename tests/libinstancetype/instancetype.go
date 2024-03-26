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
 * Copyright 2022 Red Hat, Inc.
 *
 */
//nolint:lll
package libinstancetype

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func CheckForVMInstancetypeRevisionNames(vmName string, virtClient kubecli.KubevirtClient) func() error {
	return func() error {
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if vm.Spec.Instancetype.RevisionName == "" {
			return fmt.Errorf("instancetype revision name is expected to not be empty")
		}

		if vm.Spec.Preference.RevisionName == "" {
			return fmt.Errorf("preference revision name is expected to not be empty")
		}
		return nil
	}
}

func WaitForVMInstanceTypeRevisionNames(vmName string, virtClient kubecli.KubevirtClient) {
	Eventually(CheckForVMInstancetypeRevisionNames(vmName, virtClient), 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func EnsureControllerRevisionObjectsEqual(crNameA, crNameB string, virtClient kubecli.KubevirtClient) bool {
	crA, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(nil)).Get(context.Background(), crNameA, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	crB, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(nil)).Get(context.Background(), crNameB, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return equality.Semantic.DeepEqual(crA.Data.Object, crB.Data.Object)
}

func NewInstancetypeFromVMI(vmi *v1.VirtualMachineInstance) *instancetypev1beta1.VirtualMachineInstancetype {
	return &instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-instancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
		Spec: newInstancetypeSpecFromVMI(vmi),
	}
}

func NewClusterInstancetypeFromVMI(vmi *v1.VirtualMachineInstance) *instancetypev1beta1.VirtualMachineClusterInstancetype {
	return &instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-instancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		},
		Spec: newInstancetypeSpecFromVMI(vmi),
	}
}

func newInstancetypeSpecFromVMI(vmi *v1.VirtualMachineInstance) instancetypev1beta1.VirtualMachineInstancetypeSpec {
	// Copy the amount of memory set within the VMI so our tests don't randomly start using more resources
	guestMemory := resource.MustParse("128M")
	if vmi != nil {
		if _, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
			guestMemory = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory].DeepCopy()
		}
	}
	return instancetypev1beta1.VirtualMachineInstancetypeSpec{
		CPU: instancetypev1beta1.CPUInstancetype{
			Guest: uint32(1),
		},
		Memory: instancetypev1beta1.MemoryInstancetype{
			Guest: guestMemory,
		},
	}
}

func NewPreference() *instancetypev1beta1.VirtualMachinePreference {
	return &instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-preference-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
	}
}

func NewClusterPreference() *instancetypev1beta1.VirtualMachineClusterPreference {
	return &instancetypev1beta1.VirtualMachineClusterPreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-preference-",
			Namespace:    testsuite.GetTestNamespace(nil),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		},
	}
}
