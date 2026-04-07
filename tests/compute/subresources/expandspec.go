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
 * Copyright The KubeVirt Authors
 *
 */

package subresources

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	instancetypebuilder "kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(compute.SIG("ExpandSpec subresource", decorators.SigComputeInstancetype, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("should expand an instancetype and preference via the expand-spec endpoint", func() {
		vmi := libvmifact.NewGuestless()

		clusterInstancetype := instancetypebuilder.NewClusterInstancetypeFromVMI(vmi)
		clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().
			Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithClusterInstancetype(clusterInstancetype.Name))

		expandedVM, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
		Expect(err).ToNot(HaveOccurred())
		Expect(expandedVM.Spec.Instancetype).To(BeNil())
		Expect(expandedVM.Spec.Template.Spec.Domain.CPU.Cores).To(Equal(clusterInstancetype.Spec.CPU.Guest))
	})
}))
