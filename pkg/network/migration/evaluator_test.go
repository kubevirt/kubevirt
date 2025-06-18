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

package migration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/migration"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Evaluator", func() {
	const (
		secondaryNetworkName = "secondary-network"
		nadName              = "my-nad"
	)

	multusAndDomainInfoSource := vmispec.NewInfoSource(vmispec.InfoSourceMultusStatus, vmispec.InfoSourceDomain)

	DescribeTable("Should not require migration", func(vmi *v1.VirtualMachineInstance) {
		Expect(migration.NewEvaluator().Evaluate(vmi)).To(Equal(k8scorev1.ConditionUnknown))
	},
		Entry("when no networks are specified",
			libvmi.New(libvmi.WithAutoAttachPodInterface(false)),
		),
		Entry("when status equals to spec",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, nadName)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       secondaryNetworkName,
							InfoSource: multusAndDomainInfoSource,
						}),
					),
				),
			),
		),
		Entry("when a secondary iface using bridge binding was not yet hot-unplugged from the domain",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(v1.Interface{
					Name: secondaryNetworkName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
					State: v1.InterfaceStateAbsent,
				}),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, nadName)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       secondaryNetworkName,
							InfoSource: multusAndDomainInfoSource,
						}),
					),
				),
			),
		),
	)

	DescribeTable("Should require a pending migration", func(vmi *v1.VirtualMachineInstance) {
		Expect(migration.NewEvaluator().Evaluate(vmi)).To(Equal(k8scorev1.ConditionFalse))
	},
		Entry("when a secondary iface using bridge binding is hotplugged",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, nadName)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
					),
				),
			),
		),
		Entry("when a secondary iface using bridge binding was hot-unplugged from the domain",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(v1.Interface{
					Name: secondaryNetworkName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
					State: v1.InterfaceStateAbsent,
				}),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, nadName)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       secondaryNetworkName,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
					),
				),
			),
		),
	)

	It("Should require an immediate migration when a secondary iface using SR-IOV binding is hotplugged", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(secondaryNetworkName)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, nadName)),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
						Name:       "default",
						InfoSource: vmispec.InfoSourceDomain,
					}),
				),
			),
		)

		Expect(migration.NewEvaluator().Evaluate(vmi)).To(Equal(k8scorev1.ConditionTrue))
	})

	Context("Time based scenarios", func() {
		lastTransitionTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

		DescribeTable("When migration is pending", func(stubNow time.Time, expectedResult k8scorev1.ConditionStatus) {
			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, nadName)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithCondition(v1.VirtualMachineInstanceCondition{
							Type:               v1.VirtualMachineInstanceMigrationRequired,
							Status:             k8scorev1.ConditionFalse,
							LastTransitionTime: metav1.Time{Time: lastTransitionTime},
							Reason:             v1.VirtualMachineInstanceReasonAutoMigrationPending,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
					),
				),
			)

			stubTimeProvider := func() metav1.Time {
				return metav1.Time{Time: stubNow}
			}

			Expect(migration.NewEvaluatorWithTimeProvider(stubTimeProvider).Evaluate(vmi)).To(Equal(expectedResult))
		},
			Entry("Should require a pending migration when timeout hasn't expired",
				lastTransitionTime.Add(migration.DynamicNetworkControllerGracePeriod-1*time.Second),
				k8scorev1.ConditionFalse,
			),
			Entry("Should require an immediate migration when timeout has expired",
				lastTransitionTime.Add(migration.DynamicNetworkControllerGracePeriod+1*time.Second),
				k8scorev1.ConditionTrue,
			),
		)
	})
})
