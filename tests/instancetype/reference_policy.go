package instancetype

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/builder"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] InstancetypeReferencePolicy",
	decorators.SigCompute, Serial, func() {
		var virtClient kubecli.KubevirtClient

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			kvconfig.EnableFeatureGate(featuregate.InstancetypeReferencePolicy)
		})

		DescribeTable("should result in running VirtualMachine when set to", func(policy virtv1.InstancetypeReferencePolicy) {
			kv := libkubevirt.GetCurrentKv(virtClient)
			kv.Spec.Configuration.Instancetype = &virtv1.InstancetypeConfiguration{
				ReferencePolicy: pointer.P(policy),
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
			testsuite.EnsureKubevirtReady()

			// Generate the VMI first so we can use it to create an instance type
			vmi := libvmifact.NewAlpine()

			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(vmi)).Create(
				context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := builder.NewPreference(
				builder.WithPreferredDiskBus(virtv1.DiskBusVirtio),
				builder.WithPreferredCPUTopology(v1beta1.Cores),
			)
			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(vmi)).Create(
				context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(
				vmi,
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithPreference(preference.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(
				context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert that the VM has the expected {Instancetype,Preference}Matcher values given the tested policy
			switch policy {
			case virtv1.Reference:
				Expect(vm.Spec.Instancetype).ToNot(BeNil())
				Expect(vm.Spec.Instancetype.Name).To(Equal(instancetype.Name))
				Expect(vm.Spec.Instancetype.Kind).To(Equal(instancetypeapi.SingularResourceName))
				Expect(revision.HasControllerRevisionRef(vm.Status.InstancetypeRef)).To(BeTrue())
				Expect(vm.Spec.Preference).ToNot(BeNil())
				Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
				Expect(vm.Spec.Preference.Kind).To(Equal(instancetypeapi.SingularPreferenceResourceName))
				Expect(revision.HasControllerRevisionRef(vm.Status.PreferenceRef)).To(BeTrue())
			case virtv1.Expand, virtv1.ExpandAll:
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.CPU.Cores).To(Equal(instancetype.Spec.CPU.Guest))
				Expect(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()).To(Equal(instancetype.Spec.Memory.Guest.Value()))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).ToNot(BeEmpty())
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Disk.Bus).To(Equal(virtv1.DiskBusVirtio))
			}
		},
			Entry("reference", virtv1.Reference),
			Entry("expand", virtv1.Expand),
			Entry("expandAll", virtv1.ExpandAll),
		)
	},
)
