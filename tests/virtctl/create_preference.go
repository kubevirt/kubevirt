package virtctl

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/kubecli"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/preference"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute] create preference", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	createPreferenceSpec := func(bytes []byte, namespaced bool) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		switch obj := decodedObj.(type) {
		case *instancetypev1alpha2.VirtualMachinePreference:
			ExpectWithOffset(1, namespaced).To(BeTrue(), "expected VirtualMachinePreference to be created")
			ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachinePreference"))
			preference, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).Create(context.Background(), (*instancetypev1alpha2.VirtualMachinePreference)(obj), metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &preference.Spec, nil
		case *instancetypev1alpha2.VirtualMachineClusterPreference:
			ExpectWithOffset(1, namespaced).To(BeFalse(), "expected VirtualMachineClusterPreference to be created")
			ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineClusterPreference"))
			clusterPreference, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), (*instancetypev1alpha2.VirtualMachineClusterPreference)(obj), metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &clusterPreference.Spec, nil
		default:
			return nil, fmt.Errorf("object must be VirtualMachinePreference or VirtualMachineClusterPreference")
		}
	}

	Context("should create valid preference manifest", func() {
		DescribeTable("[test_id:9836]without arguments", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag)()
			Expect(err).ToNot(HaveOccurred())

			_, err = createPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("VirtualMachinePreference", namespaced, true),
			Entry("VirtualMachineClusterPreference", "", false),
		)

		DescribeTable("[test_id:9837]when machine type defined", func(namespacedFlag, machineType string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag,
				setFlag(MachineTypeFlag, machineType),
			)()
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err := createPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec.Machine.PreferredMachineType).To(Equal(machineType))
		},
			Entry("VirtualMachinePreference", namespaced, "pc-i440fx-2.10", true),
			Entry("VirtualMachineClusterPreference", "", "pc-q35-2.10", false),
		)

		DescribeTable("[test_id:9838]when preferred storageClass defined", func(namespacedFlag, PreferredStorageClass string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag,
				setFlag(VolumeStorageClassFlag, PreferredStorageClass),
			)()
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err := createPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec.Volumes.PreferredStorageClassName).To(Equal(PreferredStorageClass))
		},
			Entry("VirtualMachinePreference", namespaced, "hostpath-provisioner", true),
			Entry("VirtualMachineClusterPreference", "", "local", false),
		)

		DescribeTable("[test_id:9839]when cpu topology defined", func(namespacedFlag, CPUTopology string, namespaced bool, topology instancetypev1alpha2.PreferredCPUTopology) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag,
				setFlag(CPUTopologyFlag, CPUTopology),
			)()
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err := createPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec.CPU.PreferredCPUTopology).To(Equal(topology))
		},
			Entry("VirtualMachinePreference", namespaced, "preferCores", true, instancetypev1alpha2.PreferCores),
			Entry("VirtualMachineClusterPreference", "", "preferThreads", false, instancetypev1alpha2.PreferThreads),
		)
	})
})
