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
 */

package expose_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

const (
	labelKey   = "my-key"
	labelValue = "my-value"
)

var _ = Describe("Expose", func() {
	var (
		kubeClient *fake.Clientset
		virtClient *kubevirtfake.Clientset
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().ReplicaSet(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
	})

	Context("should fail", func() {
		DescribeTable("with invalid argument count", func(args ...string) {
			err := runCommand(args...)
			Expect(err).To(MatchError(ContainSubstring("accepts 2 arg(s), received")))
		},
			Entry("no arguments"),
			Entry("single argument", "vmi"),
			Entry("three arguments", "vmi", "test", "invalid"),
		)

		It("with invalid resource type", func() {
			err := runCommand("kaboom", "my-vm", "--name", "my-service")
			Expect(err).To(MatchError("unsupported resource type: kaboom"))
		})

		It("with unknown flag", func() {
			err := runCommand("vmi", "my-vm", "--name", "my-service", "--kaboom")
			Expect(err).To(MatchError("unknown flag: --kaboom"))
		})

		It("missing --name flag", func() {
			err := runCommand("vmi", "my-vm")
			Expect(err).To(MatchError("required flag(s) \"name\" not set"))
		})

		DescribeTable("invalid flag value", func(arg, errMsg string) {
			err := runCommand("vmi", "my-vm", "--name", "my-service", arg)
			Expect(err).To(MatchError(errMsg))
		},
			Entry("invalid protocol", "--protocol=madeup", "unknown protocol: madeup"),
			Entry("invalid service type", "--type=madeup", "unknown service type: madeup"),
			Entry("service type externalname", "--type=externalname", "type: externalname not supported"),
			Entry("invalid ip family", "--ip-family=madeup", "unknown IPFamily/s: madeup"),
			Entry("invalid ip family policy", "--ip-family-policy=madeup", "unknown IPFamilyPolicy/s: madeup"),
		)

		It("when client has an error", func() {
			kubecli.GetKubevirtClientFromClientConfig = kubecli.GetInvalidKubevirtClientFromClientConfig
			err := runCommand("vmi", "my-vm", "--name", "my-service")
			Expect(err).To(MatchError(ContainSubstring("cannot obtain KubeVirt client")))
		})

		DescribeTable("with missing resource", func(resource, errMsg string) {
			err := runCommand(resource, "unknown", "--name", "my-service")
			Expect(err).To(MatchError(ContainSubstring(errMsg)))
		},
			Entry("vmi", "vmi", "virtualmachineinstances.kubevirt.io \"unknown\" not found"),
			Entry("vm", "vm", "virtualmachines.kubevirt.io \"unknown\" not found"),
			Entry("vmirs", "vmirs", "virtualmachineinstancereplicasets.kubevirt.io \"unknown\" not found"),
		)

		It("with missing port and missing pod network ports", func() {
			vmi := libvmi.New(libvmi.WithLabel("key", "value"))
			vmi, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = runCommand("vmi", vmi.Name, "--name", "my-service")
			Expect(err).To(MatchError("couldn't find port via --port flag or introspection"))
		})

		It("when labels are missing with VirtualMachineInstanceReplicaSet", func() {
			vmirs := kubecli.NewMinimalVirtualMachineInstanceReplicaSet("vmirs")
			vmirs, err := virtClient.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Create(context.Background(), vmirs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = runCommand("vmirs", vmirs.Name, "--name", "my-service")
			Expect(err).To(MatchError(ContainSubstring("cannot expose VirtualMachineInstanceReplicaSet without any selector labels")))
		})

		It("when VirtualMachineInstanceReplicaSet has MatchExpressions", func() {
			vmirs := kubecli.NewMinimalVirtualMachineInstanceReplicaSet("vmirs")
			vmirs.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"something": "something",
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "test"},
				},
			}
			vmirs, err := virtClient.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Create(context.Background(), vmirs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = runCommand("vmirs", vmirs.Name, "--name", "my-service")
			Expect(err).To(MatchError(ContainSubstring("cannot expose VirtualMachineInstanceReplicaSet with match expressions")))
		})
	})

	Context("should succeed", func() {
		const (
			serviceName    = "my-service"
			servicePort    = int32(9999)
			servicePortStr = "9999"
		)

		var (
			vmi   *v1.VirtualMachineInstance
			vm    *v1.VirtualMachine
			vmirs *v1.VirtualMachineInstanceReplicaSet
		)

		getResName := func(resType string) string {
			switch resType {
			case "vmi":
				return vmi.Name
			case "vm":
				return vm.Name
			case "vmirs":
				return vmirs.Name
			default:
				Fail("unknown resource type")
				return ""
			}
		}

		BeforeEach(func() {
			var err error
			vmi, err = virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), libvmi.New(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Create(context.Background(), libvmi.NewVirtualMachine(vmi), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmirs = kubecli.NewMinimalVirtualMachineInstanceReplicaSet("vmirs")
			vmirs.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			vmirs.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: labelValue,
				},
			}
			vmirs, err = virtClient.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Create(context.Background(), vmirs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("creating a service with default settings", func(resType string) {
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr)
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(servicePort))
			Expect(service.Spec.Ports[0].Protocol).To(Equal(k8sv1.ProtocolTCP))
			Expect(service.Spec.ClusterIP).To(BeEmpty())
			Expect(service.Spec.Type).To(Equal(k8sv1.ServiceTypeClusterIP))
			Expect(service.Spec.IPFamilies).To(BeEmpty())
			Expect(service.Spec.ExternalIPs).To(BeEmpty())
			Expect(service.Spec.IPFamilyPolicy).To(BeNil())
		},
			Entry("with VirtualMachineInstance", "vmi"),
			Entry("with VirtualMachine", "vm"),
			Entry("with VirtualMachineInstanceReplicaSet", "vmirs"),
		)

		Context("with missing port but existing pod network ports", func() {
			BeforeEach(func() {
				addPodNetworkWithPorts := func(spec *v1.VirtualMachineInstanceSpec) {
					ports := []v1.Port{{Name: "a", Protocol: "TCP", Port: 80}, {Name: "b", Protocol: "UDP", Port: 81}}
					spec.Networks = append(spec.Networks, v1.Network{Name: "pod", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}})
					spec.Domain.Devices.Interfaces = append(spec.Domain.Devices.Interfaces, v1.Interface{Name: "pod", Ports: ports})
				}

				var err error
				addPodNetworkWithPorts(&vmi.Spec)
				vmi, err = virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Update(context.Background(), vmi, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addPodNetworkWithPorts(&vm.Spec.Template.Spec)
				vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Update(context.Background(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addPodNetworkWithPorts(&vmirs.Spec.Template.Spec)
				vmirs, err = virtClient.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Update(context.Background(), vmirs, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("to create a service", func(resType string) {
				resName := getResName(resType)
				err := runCommand(resType, resName, "--name", serviceName)
				Expect(err).ToNot(HaveOccurred())

				service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(service.Spec.Selector).To(HaveLen(1))
				key, value := getSelectorKeyAndValue(resType, resName)
				Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
				Expect(service.Spec.Ports).To(ConsistOf(
					k8sv1.ServicePort{Name: "port-1", Protocol: "TCP", Port: 80},
					k8sv1.ServicePort{Name: "port-2", Protocol: "UDP", Port: 81},
				))
			},
				Entry("with VirtualMachineInstance", "vmi"),
				Entry("with VirtualMachine", "vm"),
				Entry("with VirtualMachineInstanceReplicaSet", "vmirs"),
			)
		})

		DescribeTable("creating a service with cluster-ip", func(resType string) {
			const clusterIP = "1.2.3.4"
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--cluster-ip", clusterIP)
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.ClusterIP).To(Equal(clusterIP))
		},
			Entry("with VirtualMachineInstance", "vmi"),
			Entry("with VirtualMachine", "vm"),
			Entry("with VirtualMachineInstanceReplicaSet", "vmirs"),
		)

		DescribeTable("creating a service with external-ip", func(resType string) {
			const externalIP = "1.2.3.4"
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--external-ip", externalIP)
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.ExternalIPs).To(ConsistOf(externalIP))
		},
			Entry("with VirtualMachineInstance", "vmi"),
			Entry("with VirtualMachine", "vm"),
			Entry("with VirtualMachineInstanceReplicaSet", "vmirs"),
		)

		DescribeTable("creating a service", func(resType string, protocol k8sv1.Protocol) {
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--protocol", string(protocol))
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Protocol).To(Equal(protocol))
		},
			Entry("with VirtualMachineInstance and protocol TCP", "vmi", k8sv1.ProtocolTCP),
			Entry("with VirtualMachineInstance and protocol UDP", "vmi", k8sv1.ProtocolUDP),
			Entry("with VirtualMachine and protocol TCP", "vm", k8sv1.ProtocolTCP),
			Entry("with VirtualMachine and protocol UDP", "vm", k8sv1.ProtocolUDP),
			Entry("with VirtualMachineInstanceReplicaSet and protocol TCP", "vmirs", k8sv1.ProtocolTCP),
			Entry("with VirtualMachineInstanceReplicaSet and protocol UDP", "vmirs", k8sv1.ProtocolUDP),
		)

		DescribeTable("creating a service", func(resType string, targetPort string, expected intstr.IntOrString) {
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--target-port", targetPort)
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(expected))
		},
			Entry("with VirtualMachineInstance and target-port", "vmi", "8000", intstr.IntOrString{Type: intstr.Int, IntVal: 8000}),
			Entry("with VirtualMachineInstance and string target-port", "vmi", "http", intstr.IntOrString{Type: intstr.String, StrVal: "http"}),
			Entry("with VirtualMachine and target-port", "vm", "8000", intstr.IntOrString{Type: intstr.Int, IntVal: 8000}),
			Entry("with VirtualMachine and string target-port", "vm", "http", intstr.IntOrString{Type: intstr.String, StrVal: "http"}),
			Entry("with VirtualMachineInstanceReplicaSet and target-port", "vmirs", "8000", intstr.IntOrString{Type: intstr.Int, IntVal: 8000}),
			Entry("with VirtualMachineInstanceReplicaSet and string target-port", "vmirs", "http", intstr.IntOrString{Type: intstr.String, StrVal: "http"}),
		)

		DescribeTable("creating a service", func(resType string, serviceType k8sv1.ServiceType) {
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--type", string(serviceType))
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(servicePort))
		},
			Entry("with VirtualMachineInstance and type ClusterIP", "vmi", k8sv1.ServiceTypeClusterIP),
			Entry("with VirtualMachineInstance and type NodePort", "vmi", k8sv1.ServiceTypeNodePort),
			Entry("with VirtualMachineInstance and type LoadBalancer", "vmi", k8sv1.ServiceTypeLoadBalancer),
			Entry("with VirtualMachine and type ClusterIP", "vm", k8sv1.ServiceTypeClusterIP),
			Entry("with VirtualMachine and type NodePort", "vm", k8sv1.ServiceTypeNodePort),
			Entry("with VirtualMachine and type LoadBalancer", "vm", k8sv1.ServiceTypeLoadBalancer),
			Entry("with VirtualMachineInstanceReplicaSet and type ClusterIP", "vmirs", k8sv1.ServiceTypeClusterIP),
			Entry("with VirtualMachineInstanceReplicaSet and type NodePort", "vmirs", k8sv1.ServiceTypeNodePort),
			Entry("with VirtualMachineInstanceReplicaSet and type LoadBalancer", "vmirs", k8sv1.ServiceTypeLoadBalancer),
		)

		DescribeTable("creating a service with named port", func(resType string) {
			const portName = "test-port"
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--port-name", portName)
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Name).To(Equal(portName))
		},
			Entry("with VirtualMachineInstance", "vmi"),
			Entry("with VirtualMachine", "vm"),
			Entry("with VirtualMachineInstanceReplicaSet", "vmirs"),
		)

		DescribeTable("creating a service selecting a suitable default IPFamilyPolicy", func(resType, ipFamily string, ipFamilyPolicy *k8sv1.IPFamilyPolicy, expected ...k8sv1.IPFamily) {
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--ip-family", ipFamily)
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(service.Spec.IPFamilies).To(ConsistOf(expected))
			if ipFamilyPolicy != nil {
				Expect(*service.Spec.IPFamilyPolicy).To(Equal(*ipFamilyPolicy))
			} else {
				Expect(service.Spec.IPFamilyPolicy).To(BeNil())
			}
		},
			Entry("with VirtualMachineInstance and IPFamily IPv4", "vmi", "ipv4", nil, k8sv1.IPv4Protocol),
			Entry("with VirtualMachineInstance and IPFamily IPv6", "vmi", "ipv6", nil, k8sv1.IPv6Protocol),
			Entry("with VirtualMachineInstance and IPFamily IPv4,IPv6", "vmi", "ipv4,ipv6", pointer.P(k8sv1.IPFamilyPolicyPreferDualStack), k8sv1.IPv4Protocol, k8sv1.IPv6Protocol),
			Entry("with VirtualMachineInstance and IPFamily IPv6,IPv4", "vmi", "ipv6,ipv4", pointer.P(k8sv1.IPFamilyPolicyPreferDualStack), k8sv1.IPv6Protocol, k8sv1.IPv4Protocol),
			Entry("with VirtualMachine and IPFamily IPv4", "vm", "ipv4", nil, k8sv1.IPv4Protocol),
			Entry("with VirtualMachine and IPFamily IPv6", "vm", "ipv6", nil, k8sv1.IPv6Protocol),
			Entry("with VirtualMachine and IPFamily IPv4,IPv6", "vm", "ipv4,ipv6", pointer.P(k8sv1.IPFamilyPolicyPreferDualStack), k8sv1.IPv4Protocol, k8sv1.IPv6Protocol),
			Entry("with VirtualMachine and IPFamily IPv6,IPv4", "vm", "ipv6,ipv4", pointer.P(k8sv1.IPFamilyPolicyPreferDualStack), k8sv1.IPv6Protocol, k8sv1.IPv4Protocol),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamily IPv4", "vmirs", "ipv4", nil, k8sv1.IPv4Protocol),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamily IPv6", "vmirs", "ipv6", nil, k8sv1.IPv6Protocol),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamily IPv4,IPv6", "vmirs", "ipv4,ipv6", pointer.P(k8sv1.IPFamilyPolicyPreferDualStack), k8sv1.IPv4Protocol, k8sv1.IPv6Protocol),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamily IPv6,IPv4", "vmirs", "ipv6,ipv4", pointer.P(k8sv1.IPFamilyPolicyPreferDualStack), k8sv1.IPv6Protocol, k8sv1.IPv4Protocol),
		)

		DescribeTable("creating a service", func(resType string, ipFamilyPolicy k8sv1.IPFamilyPolicy) {
			resName := getResName(resType)
			err := runCommand(resType, resName, "--name", serviceName, "--port", servicePortStr, "--ip-family-policy", string(ipFamilyPolicy))
			Expect(err).ToNot(HaveOccurred())

			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), serviceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Selector).To(HaveLen(1))
			key, value := getSelectorKeyAndValue(resType, resName)
			Expect(service.Spec.Selector).To(HaveKeyWithValue(key, value))
			Expect(*service.Spec.IPFamilyPolicy).To(Equal(ipFamilyPolicy))

		},
			Entry("with VirtualMachineInstance and IPFamilyPolicy SingleStack", "vmi", k8sv1.IPFamilyPolicySingleStack),
			Entry("with VirtualMachineInstance and IPFamilyPolicy PreferDualStack", "vmi", k8sv1.IPFamilyPolicyPreferDualStack),
			Entry("with VirtualMachineInstance and IPFamilyPolicy RequireDualStack", "vmi", k8sv1.IPFamilyPolicyRequireDualStack),
			Entry("with VirtualMachine and IPFamilyPolicy SingleStack", "vm", k8sv1.IPFamilyPolicySingleStack),
			Entry("with VirtualMachine and IPFamilyPolicy PreferDualStack", "vm", k8sv1.IPFamilyPolicyPreferDualStack),
			Entry("with VirtualMachine and IPFamilyPolicy RequireDualStack", "vm", k8sv1.IPFamilyPolicyRequireDualStack),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamilyPolicy SingleStack", "vmirs", k8sv1.IPFamilyPolicySingleStack),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamilyPolicy PreferDualStack", "vmirs", k8sv1.IPFamilyPolicyPreferDualStack),
			Entry("with VirtualMachineInstanceReplicaSet and IPFamilyPolicy RequireDualStack", "vmirs", k8sv1.IPFamilyPolicyRequireDualStack),
		)
	})
})

func runCommand(args ...string) error {
	return testing.NewRepeatableVirtctlCommand(append([]string{expose.COMMAND_EXPOSE}, args...)...)()
}

func getSelectorKeyAndValue(resType, resName string) (string, string) {
	if resType == "vmirs" {
		return labelKey, labelValue
	} else {
		return v1.VirtualMachineNameLabel, resName
	}
}
