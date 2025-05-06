package hotplug

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]VM Rollout Strategy", decorators.SigCompute, Serial, decorators.WgS390x, func() {
	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("When using the Stage rollout strategy", func() {
		BeforeEach(func() {
			rolloutStrategy := pointer.P(v1.VMRolloutStrategyStage)
			rolloutData, err := json.Marshal(rolloutStrategy)
			Expect(err).To(Not(HaveOccurred()))

			data := fmt.Sprintf(`[{"op": "replace", "path": "/spec/configuration/vmRolloutStrategy", "value": %s}]`, string(rolloutData))

			kv := libkubevirt.GetCurrentKv(virtClient)
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kv.Name, types.JSONPatchType, []byte(data), metav1.PatchOptions{})
				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		})

		It("[test_id:11207]should set RestartRequired when changing any spec field", func() {
			By("Creating a VM with CPU topology")
			vmi := libvmifact.NewAlpine()
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Sockets:    1,
				Cores:      2,
				Threads:    1,
				MaxSockets: 2,
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				vmi, err = kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("Updating CPU sockets to a value that would be valid in LiveUpdate")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Expecting RestartRequired")
			Eventually(ThisVM(vm), time.Minute, time.Second).Should(HaveConditionTrue(v1.VirtualMachineRestartRequired))
		})
	})

})
