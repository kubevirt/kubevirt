package workloadupdater

import (
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	io_prometheus_client "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Workload Updater", func() {
	var ctrl *gomock.Controller
	var stop chan struct{}
	var virtClient *kubecli.MockKubevirtClient
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var kubeVirtInterface *kubecli.MockKubeVirtInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var podInformer cache.SharedIndexInformer
	var podSource *framework.FakeControllerSource
	var migrationInformer cache.SharedIndexInformer
	var migrationSource *framework.FakeControllerSource
	var kubeVirtSource *framework.FakeControllerSource
	var kubeVirtInformer cache.SharedIndexInformer
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var kubeClient *fake.Clientset
	var migrationFeeder *testutils.MigrationFeeder

	var controller *WorkloadUpdateController

	var expectedImage string

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go podInformer.Run(stop)
		go migrationInformer.Run(stop)
		go kubeVirtInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			migrationInformer.HasSynced,
			kubeVirtInformer.HasSynced,
		)).To(BeTrue())
	}

	addKubeVirt := func(kv *v1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		kubeVirtSource.Add(kv)
		mockQueue.Wait()
	}

	shouldExpectMultiplePodEvictions := func(evictionCount *int) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			if action.GetSubresource() == "eviction" {
				*evictionCount++
				return true, nil, nil
			}
			return false, nil, nil
		})
	}

	BeforeEach(func() {

		expectedImage = "cur-image"

		outdatedVirtualMachineInstanceWorkloads.Set(0.0)
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
		kubeVirtInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
			"node": func(obj interface{}) (strings []string, e error) {
				return []string{obj.(*v1.VirtualMachineInstance).Status.NodeName}, nil
			},
		})
		migrationInformer, migrationSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		recorder = record.NewFakeRecorder(200)
		config, _, _, _ := testutils.NewFakeClusterConfig(&v12.ConfigMap{})

		kubeVirtInformer, _ = testutils.NewFakeInformerFor(&v1.KubeVirt{})
		kubeVirtInformer, kubeVirtSource = testutils.NewFakeInformerFor(&v1.KubeVirt{})

		controller = NewWorkloadUpdateController(expectedImage, vmiInformer, podInformer, migrationInformer, kubeVirtInformer, recorder, virtClient, config)
		mockQueue = testutils.NewMockWorkQueue(controller.queue)
		controller.queue = mockQueue
		migrationFeeder = testutils.NewMigrationFeeder(mockQueue, migrationSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstanceMigration(v12.NamespaceDefault).Return(migrationInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(v12.NamespaceDefault).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().KubeVirt(v12.NamespaceDefault).Return(kubeVirtInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().PolicyV1beta1().Return(kubeClient.PolicyV1beta1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		syncCaches(stop)
	})

	Context("workload update in progress", func() {
		It("should migrate the VMI", func() {
			newVirtualMachine("testvm", true, "madeup", vmiSource, podSource)
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(1)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})

		It("should do nothing if deployment is updating", func() {
			newVirtualMachine("testvm", true, "madeup", vmiSource, podSource)
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(1)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			kv.Status.ObservedDeploymentID = "something new"
			controller.Execute()
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should update out of date value on kv and report prometheus metric", func() {

			By("Checking prometheus metric before sync")
			dto := &io_prometheus_client.Metric{}
			outdatedVirtualMachineInstanceWorkloads.Write(dto)

			zero := 0.0
			Expect(dto.GetGauge().Value).To(Equal(&zero), "outdated vmi workload reported should be equal to zero")

			reasons := []string{}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup", vmiSource, podSource)
			}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-non-migratable-%d", i), false, "madeup", vmiSource, podSource)
			}
			// add vmis that are not outdated to ensure they are not counted as outdated in count
			for i := 0; i < 100; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-up-to-date-%d", i), false, expectedImage, vmiSource, podSource)
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(0)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			kubeVirtInterface.EXPECT().PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(name string, pt types.PatchType, data []byte) {
				str := string(data)
				Expect(str).To(Equal("[{ \"op\": \"test\", \"path\": \"/status/outdatedVirtualMachineInstanceWorkloads\", \"value\": 0}, { \"op\": \"replace\", \"path\": \"/status/outdatedVirtualMachineInstanceWorkloads\", \"value\": 100}]"))

			}).Return(nil, nil).Times(1)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(int(virtconfig.ParallelMigrationsPerClusterDefault))

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)

			By("Checking prometheus metric")
			dto = &io_prometheus_client.Metric{}
			outdatedVirtualMachineInstanceWorkloads.Write(dto)

			val := 100.0

			Expect(dto.GetGauge().Value).To(Equal(&val))
			Expect(evictionCount).To(Equal(defaultBatchDeletionCount))

		})

		It("should migrate VMIs up to the global max migration count and delete up to delete batch count", func() {
			reasons := []string{}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup", vmiSource, podSource)
			}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-%d", i), false, "madeup", vmiSource, podSource)
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(100)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(int(virtconfig.ParallelMigrationsPerClusterDefault))
			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(defaultBatchDeletionCount))
		})

		It("should detect in-flight migrations when only migrate VMIs up to the global max migration count", func() {
			kv := newKubeVirt(50)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup", vmiSource, podSource)
				// create enough migrations to only allow one more active one to be created
				if i < int(virtconfig.ParallelMigrationsPerClusterDefault)-1 {
					migrationFeeder.Add(newMigration(fmt.Sprintf("vmim-%d", i), vmi.Name, v1.MigrationRunning))
				} else if i < int(virtconfig.ParallelMigrationsPerClusterDefault) {
					migrationFeeder.Add(newMigration(fmt.Sprintf("vmim-%d", i), vmi.Name, v1.MigrationSucceeded))
					// expect only a single migration to occur due to global limit
					reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
				} else {
					migrationFeeder.Add(newMigration(fmt.Sprintf("vmim-%d", i), vmi.Name, v1.MigrationSucceeded))
				}
			}

			// wait for informer to catch up since we aren't watching for vmis directly
			time.Sleep(1 * time.Second)

			//migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).AnyTimes()
			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(1)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should migrate/shutdown outdated VMIs and leave up to date VMIs alone", func() {
			reasons := []string{}
			newVirtualMachine("testvm-outdated-migratable", true, "madeup", vmiSource, podSource)
			reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)

			newVirtualMachine("testvm-outdated-non-migratable", false, "madeup", vmiSource, podSource)
			reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)

			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-up-to-date-migratable-%d", i), true, expectedImage, vmiSource, podSource)
				newVirtualMachine(fmt.Sprintf("testvm-up-to-date-non-migratable-%d", i), false, expectedImage, vmiSource, podSource)
			}

			// wait for informer to catch up since we aren't watching for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(1)
			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(1))
		})

		It("should do nothing if no method is set", func() {
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup", vmiSource, podSource)
			}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-%d", i), false, "madeup", vmiSource, podSource)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(100)
			addKubeVirt(kv)
			controller.Execute()
		})

		It("should shutdown VMIs and not migrate when only shutdown method is set", func() {
			reasons := []string{}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup", vmiSource, podSource)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(50)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(defaultBatchDeletionCount))
		})

		It("should not evict VMIs when an active migration is in flight", func() {
			kv := newKubeVirt(2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			addKubeVirt(kv)

			vmi := newVirtualMachine("testvm-migratable", true, "madeup", vmiSource, podSource)
			migrationFeeder.Add(newMigration("vmim-1", vmi.Name, v1.MigrationRunning))
			vmi = newVirtualMachine("testvm-nonmigratable", true, "madeup", vmiSource, podSource)
			migrationFeeder.Add(newMigration("vmim-2", vmi.Name, v1.MigrationRunning))

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)

			controller.Execute()
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should respect custom batch deletion count", func() {
			batchDeletions := 30
			reasons := []string{}
			for i := 0; i < 50; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, "madeup", vmiSource, podSource)
			}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(50)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize = &batchDeletions
			addKubeVirt(kv)

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(batchDeletions))
		})

		It("should respect custom batch interval", func() {
			controller.throttleIntervalSeconds = 0

			batchDeletions := 5
			batchInterval := time.Duration(2) * time.Second
			reasons := []string{}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			for i := 0; i < batchDeletions*2; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-1-%d", i), true, "madeup", vmiSource, podSource)
			}
			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(batchDeletions * 2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize = &batchDeletions
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionInterval = &metav1.Duration{
				Duration: batchInterval,
			}

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			addKubeVirt(kv)
			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)

			// Should do nothing this second execute due to interval
			addKubeVirt(kv)
			controller.Execute()
			Expect(recorder.Events).To(BeEmpty())

			// sleep to account for batch interval
			time.Sleep(3 * time.Second)

			// Should execute another batch of deletions after sleep
			addKubeVirt(kv)
			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(batchDeletions * 2))
		})

		It("should respect reconcile loop throttling", func() {
			controller.throttleIntervalSeconds = 5

			batchDeletions := 5
			batchInterval := time.Duration(2) * time.Second
			reasons := []string{}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulEvictVirtualMachineInstanceReason)
			}

			for i := 0; i < batchDeletions*2; i++ {
				newVirtualMachine(fmt.Sprintf("testvm-migratable-1-%d", i), true, "madeup", vmiSource, podSource)
			}
			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(batchDeletions * 2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodEvict}
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize = &batchDeletions
			kv.Spec.WorkloadUpdateStrategy.BatchEvictionInterval = &metav1.Duration{
				Duration: batchInterval,
			}

			evictionCount := 0
			shouldExpectMultiplePodEvictions(&evictionCount)

			addKubeVirt(kv)
			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)

			// Should do nothing this second execute due to interval
			addKubeVirt(kv)
			controller.Execute()
			Expect(recorder.Events).To(BeEmpty())

			// sleep to account for batch interval
			time.Sleep(3 * time.Second)

			// Should do nothing this third time due to reconcile loop throttle
			addKubeVirt(kv)
			controller.Execute()
			Expect(recorder.Events).To(BeEmpty())

			// sleep to account for throttle interval
			time.Sleep(3 * time.Second)

			// Should execute another batch of deletions after throttling expires
			addKubeVirt(kv)
			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
			Expect(evictionCount).To(Equal(batchDeletions * 2))
		})

	})

	AfterEach(func() {
		close(stop)

		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})
})

func newKubeVirt(expectedNumOutdated int) *v1.KubeVirt {
	return &v1.KubeVirt{
		ObjectMeta: v13.ObjectMeta{
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

func newVirtualMachine(name string, isMigratable bool, image string, vmiSource *framework.FakeControllerSource, podSource *framework.FakeControllerSource) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMI("testvm")
	vmi.Name = name
	vmi.Namespace = v12.NamespaceDefault
	vmi.Status.LauncherContainerImageVersion = image
	vmi.Status.Phase = v1.Running
	vmi.UID = "1234"
	if isMigratable {
		vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceIsMigratable, Status: v12.ConditionTrue}}
	}

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
		},
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: false, Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}}},
			},
		},
	}
	vmi.Status.ActivePods = map[types.UID]string{
		pod.UID: "node1",
	}

	vmiSource.Add(vmi)
	podSource.Add(pod)
	return vmi
}

func newMigration(name string, vmi string, phase v1.VirtualMachineInstanceMigrationPhase) *v1.VirtualMachineInstanceMigration {
	migration := kubecli.NewMinimalMigration(name)
	migration.Status.Phase = phase
	migration.Spec.VMIName = vmi
	migration.Namespace = v12.NamespaceDefault
	return migration
}
