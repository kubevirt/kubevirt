package workloadupdater

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/client-go/testing"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Workload Updater", func() {
	var (
		recorder       *record.FakeRecorder
		fakeVirtClient *kubevirtfake.Clientset
		kubeClient     *fake.Clientset

		controller *WorkloadUpdateController

		expectedImage string
	)

	addKubeVirt := func(kv *v1.KubeVirt) {
		key, err := virtcontroller.KeyFunc(kv)
		Expect(err).To(Not(HaveOccurred()))
		controller.kubeVirtStore.Add(kv)
		controller.queue.Add(key)
	}

	shouldExpectMultiplePodEvictions := func(evictionCount *int) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			if action.GetSubresource() == "eviction" {
				*evictionCount++
				return true, nil, nil
			}
			return false, nil, nil
		})
	}

	sanityExecute := func() {
		controllertesting.SanityExecute(controller, []cache.Store{
			controller.vmiStore, controller.podIndexer, controller.migrationIndexer, controller.kubeVirtStore,
		}, Default)
	}

	BeforeEach(func() {

		expectedImage = "cur-image"

		err := metrics.RegisterLeaderMetrics()
		Expect(err).ToNot(HaveOccurred())
		metrics.SetOutdatedVirtualMachineInstanceWorkloads(0)

		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		fakeVirtClient = kubevirtfake.NewSimpleClientset()
		kubeClient = fake.NewSimpleClientset()

		vmiInformer, _ := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
			"node": func(obj interface{}) (strings []string, e error) {
				return []string{obj.(*v1.VirtualMachineInstance).Status.NodeName}, nil
			},
		})
		migrationInformer, _ := testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstanceMigration{}, virtcontroller.GetVirtualMachineInstanceMigrationInformerIndexers())
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
		recorder = record.NewFakeRecorder(200)
		recorder.IncludeObject = true
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		kubeVirtInformer, _ := testutils.NewFakeInformerFor(&v1.KubeVirt{})

		controller, _ = NewWorkloadUpdateController(expectedImage, vmiInformer, podInformer, migrationInformer, kubeVirtInformer, recorder, virtClient, kubeClient, config)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().KubeVirt(k8sv1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().KubeVirts(k8sv1.NamespaceDefault)).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		// WU tries to create VMIM with empty name, relying on generated name.
		// FakeClient does not provide such server-side behavior, so we need to
		// do it on our own.
		testing.PrependGenerateNameCreateReactor(&fakeVirtClient.Fake, "virtualmachineinstancemigrations")
	})

	Context("workload update in progress", func() {
		It("should migrate the VMI", func() {
			vmi := newVirtualMachineInstance("testvm", true, "madeup")
			pod := newLauncherPodForVMI(vmi)
			kv := newKubeVirt(1)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}

			addKubeVirt(kv)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			waitForNumberOfInstancesOnVMIInformerCache(controller, 1)

			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(1))
			Expect(migrations.Items[0].Spec.VMIName).To(Equal("testvm"))
		})

		It("should do nothing if deployment is updating", func() {
			vmi := newVirtualMachineInstance("testvm", true, "madeup")
			pod := newLauncherPodForVMI(vmi)
			kv := newKubeVirt(1)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			kv.Status.ObservedDeploymentID = "something new"

			addKubeVirt(kv)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			waitForNumberOfInstancesOnVMIInformerCache(controller, 1)

			sanityExecute()
			Expect(recorder.Events).To(BeEmpty())
			Expect(fakeVirtClient.Actions()).To(BeEmpty())
		})

		It("should update out of date value on kv and report prometheus metric", func() {
			By("Checking prometheus metric before sync")
			value, err := metrics.GetOutdatedVirtualMachineInstanceWorkloads()
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(BeZero(), "outdated vmi workload reported should be equal to zero")

			totalVMs := 0
			var reasons []string
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-non-migratable-%d", i), false, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}
			// add vmis that are not outdated to ensure they are not counted as outdated in count
			for i := 0; i < 100; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-up-to-date-%d", i), false, expectedImage)
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, totalVMs)
			kv := newKubeVirt(0)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)
			_, err = fakeVirtClient.KubevirtV1().KubeVirts(k8sv1.NamespaceDefault).Create(context.Background(), kv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)

			By("Checking prometheus metric")
			value, err = metrics.GetOutdatedVirtualMachineInstanceWorkloads()
			Expect(err).ToNot(HaveOccurred())

			Expect(value).To(Equal(100))
			Expect(evictionCount).To(Equal(defaultBatchDeletionCount))

			updatedKV, err := fakeVirtClient.KubevirtV1().KubeVirts(k8sv1.NamespaceDefault).Get(context.Background(), kv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedKV.Status.OutdatedVirtualMachineInstanceWorkloads).To(Equal(pointer.P(100)))

			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(5))
		})

		It("should migrate VMIs up to the global max migration count and delete up to delete batch count", func() {
			totalVMs := 0
			var reasons []string
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-%d", i), false, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, totalVMs)
			kv := newKubeVirt(totalVMs)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)

			Expect(evictionCount).To(Equal(defaultBatchDeletionCount))
			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(5))
		})

		It("should detect in-flight migrations when only migrate VMIs up to the global max migration count", func() {
			const desiredNumberOfVMs = 50
			const vmsPendingMigration = int(virtconfig.ParallelMigrationsPerClusterDefault)
			kv := newKubeVirt(desiredNumberOfVMs)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			By("populating with pending migrations that should be ignored while counting the threshold")
			for i := 0; i < vmsPendingMigration; i++ {
				controller.migrationIndexer.Add(newMigration(fmt.Sprintf("vmim-pending-%d", i), fmt.Sprintf("testvm-migratable-pending-%d", i), v1.MigrationPending))
			}

			var reasons []string
			for i := 0; i < desiredNumberOfVMs; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				// create enough migrations to only allow one more active one to be created
				if i < int(virtconfig.ParallelMigrationsPerClusterDefault)-1 {
					controller.migrationIndexer.Add(newMigration(fmt.Sprintf("vmim-%d", i), vmi.Name, v1.MigrationRunning))
				} else if i < int(virtconfig.ParallelMigrationsPerClusterDefault) {
					controller.migrationIndexer.Add(newMigration(fmt.Sprintf("vmim-%d", i), vmi.Name, v1.MigrationSucceeded))
					// expect only a single migration to occur due to global limit
					reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
				} else {
					controller.migrationIndexer.Add(newMigration(fmt.Sprintf("vmim-%d", i), vmi.Name, v1.MigrationSucceeded))
				}
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, desiredNumberOfVMs)

			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)

			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(1))
		})

		It("should migrate/shutdown outdated VMIs and leave up to date VMIs alone", func() {
			var reasons []string
			vmi := newVirtualMachineInstance("testvm-outdated-migratable", true, "madeup")
			pod := newLauncherPodForVMI(vmi)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)

			vmi = newVirtualMachineInstance("testvm-outdated-non-migratable", false, "madeup")
			pod = newLauncherPodForVMI(vmi)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)

			totalVMs := 2
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-up-to-date-migratable-%d", i), true, expectedImage)
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				vmi = newVirtualMachineInstance(fmt.Sprintf("testvm-up-to-date-non-migratable-%d", i), false, expectedImage)
				pod = newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs += 2
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, totalVMs)
			kv := newKubeVirt(2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(1))

			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(1))
		})

		It("should do nothing if no method is set", func() {
			totalVMs := 0
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-%d", i), false, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
				totalVMs++
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, totalVMs)
			kv := newKubeVirt(totalVMs)
			addKubeVirt(kv)
			sanityExecute()
			Expect(fakeVirtClient.Actions()).To(BeEmpty())
		})

		It("should shutdown VMIs and not migrate when only shutdown method is set", func() {
			const desiredNumberOfVMs = 50
			var reasons []string
			for i := 0; i < desiredNumberOfVMs; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, desiredNumberOfVMs)
			kv := newKubeVirt(desiredNumberOfVMs)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(defaultBatchDeletionCount))

			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(BeEmpty())
		})

		It("should not evict VMIs when an active migration is in flight", func() {
			const desiredNumberOfVMs = 2
			kv := newKubeVirt(desiredNumberOfVMs)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			vmi := newVirtualMachineInstance("testvm-migratable", true, "madeup")
			pod := newLauncherPodForVMI(vmi)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			controller.migrationIndexer.Add(newMigration("vmim-1", vmi.Name, v1.MigrationRunning))
			vmi = newVirtualMachineInstance("testvm-nonmigratable", false, "madeup")
			pod = newLauncherPodForVMI(vmi)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			controller.migrationIndexer.Add(newMigration("vmim-2", vmi.Name, v1.MigrationRunning))

			waitForNumberOfInstancesOnVMIInformerCache(controller, desiredNumberOfVMs)

			sanityExecute()
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should respect custom batch deletion count", func() {
			const desiredNumberOfVMs = 50
			batchDeletions := 30
			var reasons []string
			for i := 0; i < desiredNumberOfVMs; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
			}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			waitForNumberOfInstancesOnVMIInformerCache(controller, desiredNumberOfVMs)
			kv := newKubeVirt(desiredNumberOfVMs)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize = &batchDeletions
			addKubeVirt(kv)

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(batchDeletions))
		})

		It("should respect custom batch interval", func() {
			batchDeletions := 5
			batchInterval := time.Duration(2) * time.Second
			var reasons []string
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			for i := 0; i < batchDeletions*2; i++ {
				vmi := newVirtualMachineInstance(fmt.Sprintf("testvm-migratable-1-%d", i), true, "madeup")
				pod := newLauncherPodForVMI(vmi)
				controller.vmiStore.Add(vmi)
				controller.podIndexer.Add(pod)
			}
			waitForNumberOfInstancesOnVMIInformerCache(controller, batchDeletions*2)
			kv := newKubeVirt(batchDeletions * 2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize = &batchDeletions
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionInterval = &metav1.Duration{
				Duration: batchInterval,
			}

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			addKubeVirt(kv)
			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)

			// Should do nothing this second execute due to interval
			addKubeVirt(kv)
			sanityExecute()
			Expect(recorder.Events).To(BeEmpty())

			// Shift time for batch interval
			controller.lastDeletionBatch = time.Now().Add(-3 * time.Second)

			// Should execute another batch of deletions after sleep
			addKubeVirt(kv)
			sanityExecute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(batchDeletions * 2))
		})

	})

	Context("LiveUpdate features", func() {
		It("VMI needs to be migrated when memory hotplug is requested", func() {
			condition := v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceMemoryChange,
				Status: k8sv1.ConditionTrue,
			}
			vmi := libvmi.New(
				libvmi.WithName("testvm"),
				libvmistatus.WithStatus(
					libvmistatus.New(libvmistatus.WithCondition(condition)),
				),
			)

			virtcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &condition)

			Expect(controller.doesRequireMigration(vmi)).To(BeTrue())
		})
	})

	Context("Abort changes due to an automated live update", func() {
		const (
			withAnnotation               = true
			withoutAnnotation            = false
			withMemoryChangeCondition    = true
			withoutMemoryChangeCondition = false
		)
		createVM := func(hasAbortionAnnotation, hasChangeCondition bool) *v1.VirtualMachineInstance {
			statusOpts := []libvmistatus.Option{
				libvmistatus.WithPhase(v1.Running),
				libvmistatus.WithCondition(v1.VirtualMachineInstanceCondition{
					Type: v1.VirtualMachineInstanceIsMigratable, Status: k8sv1.ConditionTrue},
				),
			}
			if hasChangeCondition {
				statusOpts = append(statusOpts,
					libvmistatus.WithCondition(
						v1.VirtualMachineInstanceCondition{
							Type:   v1.VirtualMachineInstanceMemoryChange,
							Status: k8sv1.ConditionTrue,
						},
					),
				)
			}
			opts := []libvmi.Option{
				libvmi.WithName("testvm"),
				libvmi.WithNamespace(k8sv1.NamespaceDefault),
				libvmistatus.WithStatus(libvmistatus.New(statusOpts...)),
			}
			if hasAbortionAnnotation {
				opts = append(opts, libvmi.WithAnnotation(v1.WorkloadUpdateMigrationAbortionAnnotation, ""))
			}
			vmi := libvmi.New(opts...)
			controller.vmiStore.Add(vmi)
			return vmi
		}
		createMig := func(vmiName string, phase v1.VirtualMachineInstanceMigrationPhase) *v1.VirtualMachineInstanceMigration {
			mig := newMigration("test", vmiName, phase)
			mig.Annotations = map[string]string{v1.WorkloadUpdateMigrationAnnotation: ""}
			controller.migrationIndexer.Add(mig)
			_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(mig.Namespace).Create(context.Background(), mig, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			return mig
		}

		BeforeEach(func() {
			kv := newKubeVirt(0)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate}
			addKubeVirt(kv)
			_, err := fakeVirtClient.KubevirtV1().KubeVirts(k8sv1.NamespaceDefault).Create(context.Background(), kv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("should delete the migration", func(phase v1.VirtualMachineInstanceMigrationPhase) {
			vmi := createVM(withoutAnnotation, withoutMemoryChangeCondition)
			mig := createMig(vmi.Name, phase)

			sanityExecute()
			if mig.IsFinal() {
				Expect(recorder.Events).To(BeEmpty())
			} else {
				testutils.ExpectEvent(recorder, SuccessfulChangeAbortionReason)
				_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(mig.Namespace).Get(context.Background(), mig.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			}
		},
			Entry("in running phase", v1.MigrationRunning),
			Entry("in failed phase", v1.MigrationFailed),
			Entry("in succeeded phase", v1.MigrationSucceeded),
		)

		DescribeTable("should handle", func(hasCond, hasMig bool) {
			vmi := createVM(withoutAnnotation, hasCond)
			var mig *v1.VirtualMachineInstanceMigration
			if hasMig {
				mig = createMig(vmi.Name, v1.MigrationRunning)
			}
			changeAborted := hasMig && !hasCond
			sanityExecute()
			if changeAborted {
				testutils.ExpectEvent(recorder, SuccessfulChangeAbortionReason)
				_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(mig.Namespace).Get(context.Background(), mig.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			} else {
				Expect(recorder.Events).To(BeEmpty())
			}
		},
			Entry("a in progress change update", withMemoryChangeCondition, true),
			Entry("a change abortion", withoutMemoryChangeCondition, true),
			Entry("no change in progress", withoutMemoryChangeCondition, false),
		)

		DescribeTable("should always cancel the migration when the testWorkloadUpdateMigrationAbortion annotation is present", func(hasCond bool) {
			vmi := createVM(withAnnotation, hasCond)
			mig := createMig(vmi.Name, v1.MigrationRunning)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulChangeAbortionReason)
			_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(mig.Namespace).Get(context.Background(), mig.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
		},
			Entry("with the change condition", withMemoryChangeCondition),
			Entry("without the change condition", withoutMemoryChangeCondition),
		)

		It("should return an error if the migration hasn't been deleted", func() {
			vmi := createVM(withAnnotation, withoutMemoryChangeCondition)
			mig := createMig(vmi.Name, v1.MigrationRunning)
			fakeVirtClient.Fake.PrependReactor("delete", "virtualmachineinstancemigrations", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, fmt.Errorf("some error")
			})

			sanityExecute()
			testutils.ExpectEvent(recorder, FailedChangeAbortionReason)
			_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(mig.Namespace).Get(context.Background(), mig.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("shouldn't cancel the migration if the migration object is still in running phase but the domain is ready on the target", func() {
			vmi := libvmi.New(
				libvmi.WithName("testvm"),
				libvmi.WithNamespace(k8sv1.NamespaceDefault),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithCondition(v1.VirtualMachineInstanceCondition{
							Type: v1.VirtualMachineInstanceIsMigratable, Status: k8sv1.ConditionTrue}),
						libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
							StartTimestamp:                 pointer.P(metav1.Now()),
							Completed:                      false,
							Failed:                         false,
							TargetNodeDomainReadyTimestamp: pointer.P(metav1.Now()),
						}),
					),
				),
			)
			controller.vmiStore.Add(vmi)
			createMig(vmi.Name, v1.MigrationRunning)
			sanityExecute()
			Expect(recorder.Events).To(BeEmpty())
			// Ensure that the deletion operation isn't called. The test reproduces the race when the migration operation
			// was completed, but the migration object was still in running phase and it was accidentally deleted.
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstancemigrations")).To(BeEmpty())
		})
	})

	Context("workload volumes update", func() {
		DescribeTable("should use correct label value for filtering", func(vmName, expectedLabelValue string) {
			vmi := newVirtualMachineInstance(vmName, true, "madeup")
			condition := v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceVolumesChange,
				Status: k8sv1.ConditionTrue,
			}
			virtcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &condition)
			pod := newLauncherPodForVMI(vmi)
			kv := newKubeVirt(1)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}

			addKubeVirt(kv)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			waitForNumberOfInstancesOnVMIInformerCache(controller, 1)

			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(1))
			Expect(migrations.Items[0].Spec.VMIName).To(Equal(vmName))
			Expect(migrations.Items[0].Labels[v1.VolumesUpdateMigration]).To(BeEquivalentTo(expectedLabelValue))
		},
			Entry("with regular name", "testvm", "testvm"),
			Entry("with long name", strings.Repeat("a", k8svalidation.DNS1035LabelMaxLength+1), "1234"),
		)
	})

	Context("when MigrationPriorityQueue feature gate is enabled", func() {
		BeforeEach(func() {
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates: []string{featuregate.MigrationPriorityQueue},
				},
			})
			controller.clusterConfig = config
		})

		DescribeTable("should set correct priority to the migration object", func(condition *v1.VirtualMachineInstanceCondition, expectedPriority string) {
			vmi := newVirtualMachineInstance("testvm", true, "madeup")
			if condition != nil {
				virtcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, condition)
			}
			pod := newLauncherPodForVMI(vmi)
			kv := newKubeVirt(1)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}

			addKubeVirt(kv)
			controller.vmiStore.Add(vmi)
			controller.podIndexer.Add(pod)
			waitForNumberOfInstancesOnVMIInformerCache(controller, 1)
			sanityExecute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			migrations, err := fakeVirtClient.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(migrations.Items).To(HaveLen(1))
			Expect(migrations.Items[0].Spec.Priority).To(gstruct.PointTo(BeEquivalentTo(expectedPriority)))
		},
			Entry("system-critical in case of upgrade", nil, "system-critical"),
			Entry("user-triggered in case of hotplug", &v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceMemoryChange,
				Status: k8sv1.ConditionTrue,
			}, "user-triggered"),
			Entry("user-triggered in case of volume update", &v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceVolumesChange,
				Status: k8sv1.ConditionTrue,
			}, "user-triggered"),
		)
	})

	AfterEach(func() {
		Expect(recorder.Events).To(BeEmpty())
	})
})

func waitForNumberOfInstancesOnVMIInformerCache(wu *WorkloadUpdateController, vmisNo int) {
	EventuallyWithOffset(1, func() []interface{} {
		return wu.vmiStore.List()
	}, 3*time.Second, 200*time.Millisecond).Should(HaveLen(vmisNo))
}

func newKubeVirt(expectedNumOutdated int) *v1.KubeVirt {
	return &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1.KubeVirtSpec{},
		Status: v1.KubeVirtStatus{
			Phase:                                   v1.KubeVirtPhaseDeployed,
			OutdatedVirtualMachineInstanceWorkloads: &expectedNumOutdated,
		},
	}
}

func newVirtualMachineInstance(name string, isMigratable bool, image string) *v1.VirtualMachineInstance {
	statusOpts := []libvmistatus.Option{
		libvmistatus.WithPhase(v1.Running),
		libvmistatus.WithLauncherContainerImageVersion(image),
	}
	if isMigratable {
		statusOpts = append(statusOpts, libvmistatus.WithCondition(v1.VirtualMachineInstanceCondition{Type: v1.VirtualMachineInstanceIsMigratable, Status: k8sv1.ConditionTrue}))
	}

	vmi := libvmi.New(
		libvmi.WithMemoryRequest("8192Ki"),
		libvmi.WithNamespace(k8sv1.NamespaceDefault),
		libvmi.WithName(name),
		libvmistatus.WithStatus(libvmistatus.New(statusOpts...)),
	)
	vmi.UID = "1234"
	return vmi
}

func newLauncherPodForVMI(vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmi.Name,
			Namespace: vmi.Namespace,
			UID:       types.UID(vmi.Name),
			Labels: map[string]string{
				v1.AppLabel:       "virt-launcher",
				v1.CreatedByLabel: string(vmi.UID),
			},
			Annotations: map[string]string{
				v1.DomainAnnotation: vmi.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind)},
		},
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: false, Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}}},
			},
		},
	}

	libvmistatus.Update(&vmi.Status, libvmistatus.WithActivePod(pod.UID, "node01"))
	return pod
}

func newMigration(name string, vmi string, phase v1.VirtualMachineInstanceMigrationPhase) *v1.VirtualMachineInstanceMigration {
	migration := kubecli.NewMinimalMigration(name)
	migration.Status.Phase = phase
	migration.Spec.VMIName = vmi
	migration.Namespace = k8sv1.NamespaceDefault
	return migration
}
