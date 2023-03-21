package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute]CPU Hotplug", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("A VM with cpu.maxSockets set higher than cpu.sockets", func() {
		type cpuCount struct {
			enabled  int
			disabled int
		}
		countDomCPUs := func(vmi *v1.VirtualMachineInstance) (count cpuCount) {
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).NotTo(HaveOccurred())
			Expect(domSpec.VCPUs).NotTo(BeNil())
			for _, cpu := range domSpec.VCPUs.VCPU {
				if cpu.Enabled == "yes" {
					count.enabled++
				} else {
					ExpectWithOffset(1, cpu.Enabled).To(Equal("no"))
					ExpectWithOffset(1, cpu.Hotpluggable).To(Equal("yes"))
					count.disabled++
				}
			}
			return
		}
		It("should have spare CPUs that be enabled and more pod CPU resource after a migration", func() {
			By("Creating a running VM with 1 socket and 2 max sockets")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:      2,
				Sockets:    1,
				MaxSockets: 2,
				Threads:    1,
			}
			vm := tests.NewRandomVirtualMachine(vmi, true)
			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				return err
			}).ShouldNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the computee container has 200m CPU")
			pod := tests.GetVmiPod(virtClient, vmi)
			var compute *k8sv1.Container
			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					compute = &container
					break
				}
			}
			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			Expect(*compute.Resources.Requests.Cpu()).To(Equal(resource.MustParse("200m")))

			By("Ensuring the libvirt domain has 2 enabled cores and 2 hotpuggable cores")
			Expect(countDomCPUs(vmi)).To(Equal(cpuCount{
				enabled:  2,
				disabled: 2,
			}))

			By("Enabling the second socket")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the libvirt domain has 4 enabled cores")
			Eventually(func() cpuCount {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return countDomCPUs(vmi)
			}, 30*time.Second, time.Second).Should(Equal(cpuCount{
				enabled:  4,
				disabled: 0,
			}))

			By("Migrating the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Migrate(context.Background(), vm.Name, &v1.MigrateOptions{})
			Expect(err).ToNot(HaveOccurred())
			migrations, err := virtClient.VirtualMachineInstanceMigration(vm.Namespace).List(&k8smetav1.ListOptions{})
			var migration *v1.VirtualMachineInstanceMigration
			for _, mig := range migrations.Items {
				if mig.Spec.VMIName == vmi.Name {
					migration = &mig
					break
				}
			}
			Expect(migration).NotTo(BeNil())
			tests.ExpectMigrationSuccess(virtClient, migration, tests.MigrationWaitTime)

			By("Ensuring the virt-launcher pod now has 400m CPU")
			pod = tests.GetVmiPod(virtClient, vmi)
			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					compute = &container
					break
				}
			}
			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			Expect(*compute.Resources.Requests.Cpu()).To(Equal(resource.MustParse("400m")))

			By("Ensuring the libvirt domain still has 4 enabled cores")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(countDomCPUs(vmi)).To(Equal(cpuCount{
				enabled:  4,
				disabled: 0,
			}))
		})
	})
})
