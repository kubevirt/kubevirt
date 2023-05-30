package virtctl

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/kubecli"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const namespaced = "--namespaced"

var _ = Describe("[sig-compute] create instancetype", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	createInstancetypeSpec := func(bytes []byte, namespaced bool) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		switch obj := decodedObj.(type) {
		case *instancetypev1beta1.VirtualMachineInstancetype:
			ExpectWithOffset(1, namespaced).To(BeTrue(), "expected VirtualMachineInstancetype to be created")
			ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineInstancetype"))
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(obj)).Create(context.Background(), (*instancetypev1beta1.VirtualMachineInstancetype)(obj), metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &instancetype.Spec, nil
		case *instancetypev1beta1.VirtualMachineClusterInstancetype:
			ExpectWithOffset(1, namespaced).To(BeFalse(), "expected VirtualMachineClusterInstancetype to be created")
			ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineClusterInstancetype"))
			obj.Labels = map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(obj)): "",
			}
			clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), (*instancetypev1beta1.VirtualMachineClusterInstancetype)(obj), metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &clusterInstancetype.Spec, nil
		default:
			return nil, fmt.Errorf("object must be VirtualMachineInstance or VirtualMachineClusterInstancetype")
		}
	}

	Context("should create valid instancetype manifest", func() {
		DescribeTable("[test_id:9833]when CPU and Memory defined", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.CPU.Guest).To(Equal(uint32(2)))
			Expect(instancetypeSpec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("[test_id:9834]when GPUs defined", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(GPUFlag, "name:gpu1,devicename:nvidia/gpu1"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(instancetypeSpec.GPUs[0].DeviceName).To(Equal("nvidia/gpu1"))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("[test_id:9899]when hostDevice defined", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(HostDeviceFlag, "name:device1,devicename:hostdevice1"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.HostDevices[0].Name).To(Equal("device1"))
			Expect(instancetypeSpec.HostDevices[0].DeviceName).To(Equal("hostdevice1"))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("[test_id:9835]when IOThreadsPolicy defined", func(namespacedFlag, policyStr string, namespaced bool, policy v1.IOThreadsPolicy) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(IOThreadsPolicyFlag, policyStr),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(*instancetypeSpec.IOThreadsPolicy).To(Equal(policy))
		},
			Entry("VirtualMachineInstancetype", namespaced, "auto", true, v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineClusterInstancetype", "", "shared", false, v1.IOThreadsPolicyShared),
		)
	})
})
