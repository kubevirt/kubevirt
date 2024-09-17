package expose_test

import (
	"context"
	"errors"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Expose", func() {

	const vmName = "my-vm"
	const vmNoLabelName = "vm-no-label"
	const unknownVM = "unknown-vm"
	var vmi *v1.VirtualMachineInstance
	var vmNoLabel *v1.VirtualMachineInstance
	var vm *v1.VirtualMachine
	var vmrs *v1.VirtualMachineInstanceReplicaSet
	var kubeclient *fake.Clientset
	var obtainedService *k8sv1.Service

	BeforeEach(func() {
		vmi = libvmi.New(libvmi.WithLabel("key", "value"))
		vmi.Name = vmName
		vmNoLabel = libvmi.New()
		vmNoLabel.Name = vmNoLabelName
		vm = libvmi.NewVirtualMachine(vmi)
		vmrs = kubecli.NewMinimalVirtualMachineInstanceReplicaSet(vmName)

		// create the wrapping environment that would retur the mock virt client
		// to the code being unit tested
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		// create mock interfaces
		vmiInterface := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface := kubecli.NewMockVirtualMachineInterface(ctrl)
		vmrsInterface := kubecli.NewMockReplicaSetInterface(ctrl)
		kubeclient = fake.NewSimpleClientset()

		// set up mock client behavior
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().ReplicaSet(k8smetav1.NamespaceDefault).Return(vmrsInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeclient.CoreV1()).AnyTimes()
		// set vmrs
		vmrs.Spec = v1.VirtualMachineInstanceReplicaSetSpec{Selector: &k8smetav1.LabelSelector{MatchLabels: vmi.ObjectMeta.Labels}, Template: &v1.VirtualMachineInstanceTemplateSpec{}}
		// set up mock interface behavior
		vmiInterface.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil).AnyTimes()
		vmiInterface.EXPECT().Get(context.Background(), vmNoLabel.Name, gomock.Any()).Return(vmNoLabel, nil).AnyTimes()
		vmiInterface.EXPECT().Get(context.Background(), unknownVM, gomock.Any()).Return(nil, errors.New("unknown VM")).AnyTimes()
		vmInterface.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vm, nil).AnyTimes()
		vmInterface.EXPECT().Get(context.Background(), unknownVM, gomock.Any()).Return(nil, errors.New("unknown VM")).AnyTimes()
		vmrsInterface.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmrs, nil).AnyTimes()
		vmrsInterface.EXPECT().Get(context.Background(), unknownVM, gomock.Any()).Return(nil, errors.New("unknonw VMRS")).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		// this should be called first
		kubeclient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, errors.New("kubeclient command not mocked")
		})
		// Mock handler for service creation
		kubeclient.Fake.PrependReactor("create", "services", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			obtainedService = update.GetObject().(*k8sv1.Service)
			return true, obtainedService, nil
		})
	})

	Describe("Create an 'expose' command", func() {
		Context("With empty set of flags", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewVirtctlCommand(expose.COMMAND_EXPOSE)
				Expect(cmd).ToNot(BeNil())
			})
		})
	})

	Describe("Run an 'expose' command", func() {
		Context("When client has an error", func() {
			BeforeEach(func() {
				// only in this test the client call should fail
				kubecli.GetKubevirtClientFromClientConfig = kubecli.GetInvalidKubevirtClientFromClientConfig
			})
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999")
				// not executing the command
				Expect(cmd).NotTo(BeNil())
			})
			AfterEach(func() {
				// set back the value so that other tests won't fail on client
				kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
			})
		})
		Context("With missing input parameters", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE)
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With missing resource", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With missing port and missing pod network ports", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With missing port but existing pod network ports ", func() {
			It("should succeed on vmis", func() {
				addPodNetworkWithPorts(&vmi.Spec)
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service")
				prependServicePortReactor(kubeclient)
				Expect(cmd()).To(Succeed())
			})
			It("should succeed on vms", func() {
				addPodNetworkWithPorts(&vm.Spec.Template.Spec)
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vm", vmName, "--name", "my-service")
				prependServicePortReactor(kubeclient)
				Expect(cmd()).To(Succeed())
			})
			It("should succeed on vmirs", func() {
				addPodNetworkWithPorts(&vmrs.Spec.Template.Spec)
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmirs", vmName, "--name", "my-service")
				prependServicePortReactor(kubeclient)
				Expect(cmd()).To(Succeed())
			})
		})
		Context("With missing service name", func() {
			It("should fail", func() {
				err := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--port", "9999")
				Expect(err()).NotTo(Succeed())
			})
		})
		Context("With invalid type", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--type", "kaboom")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With a invalid protocol", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--protocol", "http")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With unknown resource type", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "kaboom", vmName, "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With unknown flag", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--kaboom")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With cluster-ip on a vm", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).To(Succeed())
			})
		})
		Context("With cluster-ip on a vm that has no label", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmNoLabelName, "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With cluster-ip on an unknown vm", func() {
			It("should fail on a vmi", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", unknownVM, "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).NotTo(Succeed())
			})
			It("should fail on a vm", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vm", unknownVM, "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With node-port service on a vm", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--type", "NodePort")
				Expect(cmd()).To(Succeed())
			})
		})
		Context("With cluster-ip on an vm", func() {
			It("should succeed", func() {
				err := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vm", vmName, "--name", "my-service",
					"--port", "9999")
				Expect(err()).To(Succeed())
			})
		})
		Context("With cluster-ip on an vm replica set", func() {
			It("should succeed", func() {
				err := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmirs", vmName, "--name", "my-service",
					"--port", "9999")
				Expect(err()).To(Succeed())
			})
		})
		Context("With cluster-ip on an unknown vm replica set", func() {
			It("should fail", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmirs", unknownVM, "--name", "my-service",
					"--port", "9999")
				Expect(cmd()).NotTo(Succeed())
			})
		})
		Context("With string target-port", func() {
			It("should succeed", func() {
				err := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http")
				Expect(err()).To(Succeed())
			})
		})
		Context("With parametrized IPFamily", func() {
			var dualStack = k8sv1.IPFamilyPolicyPreferDualStack

			It("should succeed with IPv4", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family", "ipv4")
				Expect(cmd()).To(Succeed(), "should succeed on an valid IP family - ipv4 is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(obtainedService.Spec.IPFamilies).To(ConsistOf(k8sv1.IPv4Protocol))

			})

			It("should succeed with IPv6", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family", "ipv6")
				Expect(cmd()).To(Succeed(), "should succeed on an valid IP family - ipv6 is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(obtainedService.Spec.IPFamilies).To(ConsistOf(k8sv1.IPv6Protocol))
			})

			It("should succeed with IPv4, IPv6", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family", "ipv4,ipv6")
				Expect(cmd()).To(Succeed(), "should succeed on an valid IP family - ipv4,ipv6 is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(obtainedService.Spec.IPFamilies).Should(HaveLen(2))
				Expect(obtainedService.Spec.IPFamilies[0]).To(Equal(k8sv1.IPv4Protocol))
				Expect(obtainedService.Spec.IPFamilies[1]).To(Equal(k8sv1.IPv6Protocol))
			})

			It("should succeed with IPv6, IPv4", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family", "ipv6,ipv4")
				Expect(cmd()).To(Succeed(), "should succeed on an valid IP family - ipv6,ipv4 is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(obtainedService.Spec.IPFamilies).Should(HaveLen(2))
				Expect(obtainedService.Spec.IPFamilies[0]).To(Equal(k8sv1.IPv6Protocol))
				Expect(obtainedService.Spec.IPFamilies[1]).To(Equal(k8sv1.IPv4Protocol))
			})

			It("should succeed with no IPFamily", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http")
				Expect(cmd()).To(Succeed(), "should succeed when no IP family is provided")
				Expect(obtainedService).ToNot(BeNil())
				Expect(obtainedService.Spec.IPFamilies).To(BeEmpty())
			})

			It("should fail with an invalid IPFamily", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family", "ipv14")
				Expect(cmd()).To(HaveOccurred(), "should fail on an invalid IP family")

			})

			DescribeTable("should select a valid default for the IPFamilyPolicy attribute, based on the provided IPFamilies", func(expectedIPFamilyPolicy *k8sv1.IPFamilyPolicyType, ipFamilies ...string) {
				ipFamilyStr := strings.Join(ipFamilies, ",")
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family", ipFamilyStr)
				Expect(cmd()).To(Succeed(), "should succeed with the default IP family policy")
				Expect(obtainedService).NotTo(BeNil())

				Expect(obtainedService.Spec.IPFamilyPolicy).To(Equal(expectedIPFamilyPolicy))
			},
				Entry("a single IPv4 IPFamily", nil, "ipv4"),
				Entry("a single IPv6 IPFamily", nil, "ipv6"),
				Entry("a list of IPv4, IPv6 IPFamilies", &dualStack, "ipv4", "ipv6"),
				Entry("a list of IPv6, IPv4 IPFamilies", &dualStack, "ipv6", "ipv4"))
		})
		Context("With parametrized IPFamilyPolicy", func() {
			It("should succeed with singlestack", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family-policy", "singlestack")
				Expect(cmd()).To(Succeed(), "should succeed on a valid IP family policy - singlestack is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(*obtainedService.Spec.IPFamilyPolicy).To(Equal(k8sv1.IPFamilyPolicySingleStack))

			})

			It("should succeed with PreferDualStack", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family-policy", "PreferDualStack")
				Expect(cmd()).To(Succeed(), "should succeed on a valid IP family policy - PreferDualStack is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(*obtainedService.Spec.IPFamilyPolicy).To(Equal(k8sv1.IPFamilyPolicyPreferDualStack))
			})

			It("should succeed with RequiredualStack", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family-policy", "RequiredualStack")
				Expect(cmd()).To(Succeed(), "should succeed on a valid IP family policy - RequiredualStack is valid")
				Expect(obtainedService).ToNot(BeNil())
				Expect(*obtainedService.Spec.IPFamilyPolicy).To(Equal(k8sv1.IPFamilyPolicyRequireDualStack))
			})

			It("should fail with an invalid IPFamilyPolicy", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmi", vmName, "--name", "my-service",
					"--port", "9999", "--target-port", "http", "--ip-family-policy", "non-valid-policy")
				Expect(cmd()).To(HaveOccurred(), "should fail on an invalid IP family policy")

			})
		})
	})
})

func addPodNetworkWithPorts(spec *v1.VirtualMachineInstanceSpec) {
	ports := []v1.Port{{Name: "a", Protocol: "TCP", Port: 80}, {Name: "b", Protocol: "UDP", Port: 81}}
	spec.Networks = append(spec.Networks, v1.Network{Name: "pod", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}})
	spec.Domain.Devices.Interfaces = append(spec.Domain.Devices.Interfaces, v1.Interface{Name: "pod", Ports: ports})
}

func prependServicePortReactor(kubeclient *fake.Clientset) {
	kubeclient.Fake.PrependReactor("create", "services", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		Expect(update.GetObject().(*k8sv1.Service).Spec.Ports[0]).To(Equal(k8sv1.ServicePort{Name: "port-1", Protocol: "TCP", Port: 80}))
		Expect(update.GetObject().(*k8sv1.Service).Spec.Ports[1]).To(Equal(k8sv1.ServicePort{Name: "port-2", Protocol: "UDP", Port: 81}))
		return false, nil, nil
	})
}
