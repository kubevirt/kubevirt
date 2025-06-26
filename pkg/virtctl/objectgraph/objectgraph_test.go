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

package objectgraph_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Object graph command", func() {
	const (
		objectGraphCommand = "objectgraph"
		vmName             = "testvm"
	)

	var (
		vmInterface  *kubecli.MockVirtualMachineInterface
		vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
	})

	Context("For VM", func() {
		It("should fail with missing input parameters", func() {
			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand)
			Expect(cmd()).To(MatchError("accepts 1 arg(s), received 0"))
		})

		It("should fail with non-existing VM", func() {
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(true),
			}

			vmInterface.EXPECT().ObjectGraph(gomock.Any(), vmName, opts).Return(v1.ObjectGraphNode{}, fmt.Errorf("test-error")).Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName)
			Expect(cmd()).To(MatchError("error listing object graph of VirtualMachine testvm: test-error"))
		})

		It("should return object graph from VM", func() {
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(true),
			}
			objectGraph := v1.ObjectGraphNode{}

			vmInterface.EXPECT().ObjectGraph(gomock.Any(), vmName, opts).Return(objectGraph, nil).Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName)
			Expect(cmd()).To(Succeed())
		})

		It("should apply label selectors and exclude optional flags", func() {
			objectGraph := v1.ObjectGraphNode{}
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(false),
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubevirt.io/dependency-type": "storage",
					},
				},
			}

			vmInterface.EXPECT().ObjectGraph(gomock.Any(), vmName, opts).Return(objectGraph, nil).Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand,
				vmName,
				"--exclude-optional",
				"--selector", "kubevirt.io/dependency-type=storage",
			)
			Expect(cmd()).To(Succeed())
		})

		It("should return object graph in YAML format", func() {
			objectGraph := v1.ObjectGraphNode{}
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(true),
			}

			vmInterface.EXPECT().
				ObjectGraph(gomock.Any(), vmName, opts).
				Return(objectGraph, nil).
				Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName, "--output", "yaml")
			Expect(cmd()).To(Succeed())
		})

		It("should fail with unsupported output format", func() {
			objectGraph := v1.ObjectGraphNode{}
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(true),
			}
			vmInterface.EXPECT().
				ObjectGraph(gomock.Any(), vmName, opts).
				Return(objectGraph, nil).
				Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName, "--output", "unsupported")
			Expect(cmd()).To(MatchError("unsupported output format: unsupported (must be 'json' or 'yaml')"))
		})
	})

	Context("For VMI", func() {
		It("should return object graph from VMI", func() {
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(true),
			}
			objectGraph := v1.ObjectGraphNode{}

			vmiInterface.EXPECT().ObjectGraph(gomock.Any(), vmName, opts).Return(objectGraph, nil).Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName, "--vmi")
			Expect(cmd()).To(Succeed())
		})

		It("should fail with non-existing VMI", func() {
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(true),
			}

			vmiInterface.EXPECT().ObjectGraph(gomock.Any(), vmName, opts).Return(v1.ObjectGraphNode{}, fmt.Errorf("test-error")).Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand, vmName, "--vmi")
			Expect(cmd()).To(MatchError("error listing object graph of VirtualMachineInstance testvm: test-error"))
		})

		It("should apply label selectors and exclude optional flags for VMI", func() {
			objectGraph := v1.ObjectGraphNode{}
			opts := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(false),
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubevirt.io/dependency-type": "storage",
					},
				},
			}

			vmiInterface.EXPECT().ObjectGraph(gomock.Any(), vmName, opts).Return(objectGraph, nil).Times(1)

			cmd := testing.NewRepeatableVirtctlCommand(objectGraphCommand,
				vmName,
				"--vmi",
				"--exclude-optional",
				"--selector", "kubevirt.io/dependency-type=storage",
			)
			Expect(cmd()).To(Succeed())
		})
	})
})
