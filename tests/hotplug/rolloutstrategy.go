package hotplug

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	util2 "kubevirt.io/kubevirt/tests/util"

	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[Serial][sig-compute]VM Rollout Strategy", decorators.SigCompute, Serial, func() {
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

			kv := util2.GetCurrentKv(virtClient)
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(kv.Name, types.JSONPatchType, []byte(data), &metav1.PatchOptions{})
				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		})

		It("should set RestartRequired when changing any spec field", func() {
			By("Creating a VM with CPU topology")
			vmi := libvmi.NewCirros()
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Sockets:    1,
				Cores:      2,
				Threads:    1,
				MaxSockets: 2,
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())
			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).NotTo(HaveOccurred())

			By("Updating CPU sockets to a value that would be valid in LiveUpdate")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Expecting RestartRequired")
			Eventually(ThisVM(vm), 4*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineRestartRequired))
		})
	})

})
