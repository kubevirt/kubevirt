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
	"fmt"
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
		pod := &k8scorev1.Pod{}
		Expect(migration.NewEvaluator(stubClusterConfigurer{}).Evaluate(vmi, pod)).To(Equal(k8scorev1.ConditionUnknown))
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
		pod := &k8scorev1.Pod{}
		Expect(migration.NewEvaluator(stubClusterConfigurer{}).Evaluate(vmi, pod)).To(Equal(k8scorev1.ConditionFalse))
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

		Expect(migration.NewEvaluator(stubClusterConfigurer{}).Evaluate(vmi, &k8scorev1.Pod{})).
			To(Equal(k8scorev1.ConditionTrue))
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

			Expect(migration.NewEvaluatorWithTimeProvider(stubTimeProvider, stubClusterConfigurer{}).
				Evaluate(vmi, &k8scorev1.Pod{})).To(Equal(expectedResult))
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

	Context("NAD name change", func() {
		const (
			testNamespace  = "default"
			otherNamespace = "other-namespace"

			secondaryNetworkName1 = "secondary-network"
			secondaryNetworkName2 = "secondary-network2"

			secondaryPodIfaceName1 = "pod56a025b6623"
			secondaryPodIfaceName2 = "podb84f866e97d"

			nadName1 = "nad1"
			nadName2 = "nad2"

			liveUpdateNADRefEnabled = true
		)

		DescribeTable("should trigger",
			func(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod, isLiveUpdateEnabled bool, expectedMigration k8scorev1.ConditionStatus) {
				evaluator := migration.NewEvaluator(stubClusterConfigurer{isLiveUpdateNADRefEnabled: isLiveUpdateEnabled})
				Expect(evaluator.Evaluate(vmi, pod)).To(Equal(expectedMigration))
			},
			Entry("no migration when NAD name in spec matches that in pod annotation and FG is enabled",
				libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName1)),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName1, nadName1)),
					libvmistatus.WithStatus(libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:             secondaryNetworkName1,
							PodInterfaceName: secondaryPodIfaceName1,
							InfoSource:       multusAndDomainInfoSource,
						}),
					)),
				),
				newPod(map[string]string{
					"k8s.v1.cni.cncf.io/network-status": fmt.Sprintf(`[{"interface": %q, "name": "%s/%s"}]`,
						secondaryPodIfaceName1, testNamespace, nadName1),
				}),
				liveUpdateNADRefEnabled,
				k8scorev1.ConditionUnknown,
			),
			Entry("no migration when NAD name in spec differs from that in pod annotation and FG is disabled",
				libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName1)),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName1, nadName1)),
					libvmistatus.WithStatus(libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:             secondaryNetworkName1,
							PodInterfaceName: secondaryPodIfaceName1,
							InfoSource:       multusAndDomainInfoSource,
						}),
					)),
				),
				newPod(map[string]string{
					"k8s.v1.cni.cncf.io/network-status": fmt.Sprintf(`[{"interface": %q, "name": "%s/%s"}]`,
						secondaryPodIfaceName1, testNamespace, nadName2),
				}),
				!liveUpdateNADRefEnabled,
				k8scorev1.ConditionUnknown,
			),
			Entry("immediate migration when NAD name in spec differs from that in pod annotation and FG is enabled",
				libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName1)),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName1, nadName2)),
					libvmistatus.WithStatus(libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:             secondaryNetworkName1,
							PodInterfaceName: secondaryPodIfaceName1,
							InfoSource:       multusAndDomainInfoSource,
						}),
					)),
				),
				newPod(map[string]string{
					"k8s.v1.cni.cncf.io/network-status": fmt.Sprintf(`[{"interface": %q, "name": "%s/%s"}]`,
						secondaryPodIfaceName1, testNamespace, nadName1),
				}),
				liveUpdateNADRefEnabled,
				k8scorev1.ConditionTrue,
			),
			Entry(
				"immediate migration when a VM has NADs in different namespaces and one is out of sync and FG is enabled",
				libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName1)),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName2)),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName1, testNamespace+"/"+nadName2)),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName2, otherNamespace+"/"+nadName2)),
					libvmistatus.WithStatus(libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:             secondaryNetworkName1,
							PodInterfaceName: secondaryPodIfaceName1,
							InfoSource:       multusAndDomainInfoSource,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:             secondaryNetworkName2,
							PodInterfaceName: secondaryPodIfaceName2,
							InfoSource:       multusAndDomainInfoSource,
						}),
					)),
				),
				newPod(map[string]string{
					"k8s.v1.cni.cncf.io/network-status": fmt.Sprintf(
						`[{"interface": %q, "name": "%s/%s"}, {"interface": %q, "name": "%s/%s"}]`,
						secondaryPodIfaceName1, testNamespace, nadName2,
						secondaryPodIfaceName2, otherNamespace, nadName1,
					),
				}),
				liveUpdateNADRefEnabled,
				k8scorev1.ConditionTrue,
			),
		)
	})
})

type stubClusterConfigurer struct {
	isLiveUpdateNADRefEnabled bool
}

func (s stubClusterConfigurer) LiveUpdateNADRefEnabled() bool {
	return s.isLiveUpdateNADRefEnabled
}

func newPod(annotations map[string]string) *k8scorev1.Pod {
	return &k8scorev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
	}
}
