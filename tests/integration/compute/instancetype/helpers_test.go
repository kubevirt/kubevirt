/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

func newInstancetype() *instancetypev1beta1.VirtualMachineInstancetype {
	return &instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "instancetype-"},
		Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU:    instancetypev1beta1.CPUInstancetype{Guest: 2},
			Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
		},
	}
}

func newPreference() *instancetypev1beta1.VirtualMachinePreference {
	return &instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "preference-"},
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets)},
		},
	}
}

func createTestClusterInstancetype(ctx context.Context, f *framework.Framework,
	it *instancetypev1beta1.VirtualMachineClusterInstancetype,
) *instancetypev1beta1.VirtualMachineClusterInstancetype {
	GinkgoHelper()
	it, err := f.VirtClient().VirtualMachineClusterInstancetype().Create(ctx, it, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func(ctx SpecContext) error {
		return f.VirtClient().VirtualMachineClusterInstancetype().Delete(ctx, it.Name, metav1.DeleteOptions{})
	})
	return it
}

func createTestClusterPreference(ctx context.Context, f *framework.Framework,
	pref *instancetypev1beta1.VirtualMachineClusterPreference,
) *instancetypev1beta1.VirtualMachineClusterPreference {
	GinkgoHelper()
	pref, err := f.VirtClient().VirtualMachineClusterPreference().Create(ctx, pref, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func(ctx SpecContext) error {
		return f.VirtClient().VirtualMachineClusterPreference().Delete(ctx, pref.Name, metav1.DeleteOptions{})
	})
	return pref
}

func waitForScheduledVMI(ctx context.Context, f *framework.Framework, vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {
	GinkgoHelper()
	Eventually(ctx, matcher.ThisVMIWith(vm.Namespace, vm.Name), 30*time.Second, 100*time.Millisecond).Should(
		matcher.BeInPhase(virtv1.Scheduled),
	)
	vmi, err := f.VirtClient().VirtualMachineInstance(vm.Namespace).Get(ctx, vm.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	return vmi
}
