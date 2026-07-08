package envtest_test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe("Workload Update Controller", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.WithWorkloadUpdateController())
		f.Start()
	})

	AfterEach(func() {
		f.Stop()
	})

	createDeployedKubeVirt := func(methods ...virtv1.WorkloadUpdateMethod) *virtv1.KubeVirt {
		kv := &virtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: virtv1.KubeVirtSpec{
				WorkloadUpdateStrategy: virtv1.KubeVirtWorkloadUpdateStrategy{
					WorkloadUpdateMethods: methods,
				},
			},
			Status: virtv1.KubeVirtStatus{
				Phase:                virtv1.KubeVirtPhaseDeployed,
				TargetDeploymentID:   "abc123",
				ObservedDeploymentID: "abc123",
			},
		}
		kv, err := f.VirtClient().KubeVirt("kubevirt").Create(ctx, kv, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		// The API server may not populate status on create, so update it separately.
		kv.Status.Phase = virtv1.KubeVirtPhaseDeployed
		kv.Status.TargetDeploymentID = "abc123"
		kv.Status.ObservedDeploymentID = "abc123"
		kv, err = f.VirtClient().KubeVirt("kubevirt").UpdateStatus(ctx, kv, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		return kv
	}

	createRunningVM := func() *virtv1.VirtualMachine {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for VMI to reach Scheduled phase")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduled))
		return vm
	}

	makeVMIOutdatedAndMigratable := func(name string) {
		Eventually(func(g Gomega) {
			vmi, err := f.VirtClient().VirtualMachineInstance("default").Get(ctx, name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred(), "VMI Get failed")

			vmi.Status.Phase = virtv1.Running
			vmi.Status.LauncherContainerImageVersion = "old-launcher:outdated"
			vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstanceIsMigratable,
				Status: k8sv1.ConditionTrue,
			})

			patchData, err := json.Marshal(map[string]interface{}{
				"status": vmi.Status,
			})
			g.Expect(err).NotTo(HaveOccurred())

			_, err = f.VirtClient().VirtualMachineInstance("default").Patch(ctx, name, types.MergePatchType, patchData, metav1.PatchOptions{})
			g.Expect(err).NotTo(HaveOccurred(), "VMI Patch failed")

			updated, err := f.VirtClient().VirtualMachineInstance("default").Get(ctx, name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updated.Status.Phase).To(Equal(virtv1.Running))
		}, 10*time.Second, 200*time.Millisecond).Should(Succeed())
	}

	It("should create a migration for an outdated migratable VMI", func() {
		createDeployedKubeVirt(virtv1.WorkloadUpdateMethodLiveMigrate)
		vm := createRunningVM()
		makeVMIOutdatedAndMigratable(vm.Name)

		By("waiting for the workload update controller to create a migration")
		Eventually(func() int {
			migrations, err := f.VirtClient().VirtualMachineInstanceMigration("default").List(ctx, metav1.ListOptions{})
			if err != nil {
				return 0
			}
			return len(migrations.Items)
		}, 10*time.Second, 100*time.Millisecond).Should(BeNumerically(">=", 1))
	})

	It("should not create migrations when KubeVirt is not deployed", func() {
		kv := &virtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: virtv1.KubeVirtSpec{
				WorkloadUpdateStrategy: virtv1.KubeVirtWorkloadUpdateStrategy{
					WorkloadUpdateMethods: []virtv1.WorkloadUpdateMethod{virtv1.WorkloadUpdateMethodLiveMigrate},
				},
			},
			Status: virtv1.KubeVirtStatus{
				Phase: virtv1.KubeVirtPhaseDeploying,
			},
		}
		kv, err := f.VirtClient().KubeVirt("kubevirt").Create(ctx, kv, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		kv.Status.Phase = virtv1.KubeVirtPhaseDeploying
		_, err = f.VirtClient().KubeVirt("kubevirt").UpdateStatus(ctx, kv, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		vm := createRunningVM()
		makeVMIOutdatedAndMigratable(vm.Name)

		By("verifying no migration is created")
		Consistently(func() int {
			migrations, err := f.VirtClient().VirtualMachineInstanceMigration("default").List(ctx, metav1.ListOptions{})
			if err != nil {
				return 0
			}
			return len(migrations.Items)
		}, 3*time.Second, 100*time.Millisecond).Should(Equal(0))
	})

	It("should not create migrations when deployment IDs do not match", func() {
		kv := &virtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: virtv1.KubeVirtSpec{
				WorkloadUpdateStrategy: virtv1.KubeVirtWorkloadUpdateStrategy{
					WorkloadUpdateMethods: []virtv1.WorkloadUpdateMethod{virtv1.WorkloadUpdateMethodLiveMigrate},
				},
			},
			Status: virtv1.KubeVirtStatus{
				Phase:                virtv1.KubeVirtPhaseDeployed,
				TargetDeploymentID:   "new-id",
				ObservedDeploymentID: "old-id",
			},
		}
		kv, err := f.VirtClient().KubeVirt("kubevirt").Create(ctx, kv, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		kv.Status.Phase = virtv1.KubeVirtPhaseDeployed
		kv.Status.TargetDeploymentID = "new-id"
		kv.Status.ObservedDeploymentID = "old-id"
		_, err = f.VirtClient().KubeVirt("kubevirt").UpdateStatus(ctx, kv, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		vm := createRunningVM()
		makeVMIOutdatedAndMigratable(vm.Name)

		By("verifying no migration is created")
		Consistently(func() int {
			migrations, err := f.VirtClient().VirtualMachineInstanceMigration("default").List(ctx, metav1.ListOptions{})
			if err != nil {
				return 0
			}
			return len(migrations.Items)
		}, 3*time.Second, 100*time.Millisecond).Should(Equal(0))
	})
})
