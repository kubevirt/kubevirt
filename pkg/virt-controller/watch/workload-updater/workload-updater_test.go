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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

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
	var migrationInformer cache.SharedIndexInformer
	var migrationSource *framework.FakeControllerSource
	var kubeVirtSource *framework.FakeControllerSource
	var kubeVirtInformer cache.SharedIndexInformer
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var kubeClient *fake.Clientset
	var migrationFeeder *testutils.MigrationFeeder

	var controller *WorkloadUpdateController

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
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

	BeforeEach(func() {
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
		recorder = record.NewFakeRecorder(200)
		config, _, _, _ := testutils.NewFakeClusterConfig(&v12.ConfigMap{
			Data: map[string]string{"feature-gates": "AutomatedWorkloadUpdate"},
		})

		kubeVirtInformer, _ = testutils.NewFakeInformerFor(&v1.KubeVirt{})
		kubeVirtInformer, kubeVirtSource = testutils.NewFakeInformerFor(&v1.KubeVirt{})

		controller = NewWorkloadUpdateController(vmiInformer, migrationInformer, kubeVirtInformer, recorder, virtClient, config)
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
			vmi := newVirtualMachine("testvm", true, true)
			vmiSource.Add(vmi)
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(1)
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineInstanceMigrationReason)
		})

		It("should do nothing if deployment is updating", func() {
			vmi := newVirtualMachine("testvm", true, true)
			vmiSource.Add(vmi)
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(1)
			addKubeVirt(kv)

			kv.Status.ObservedDeploymentID = "something new"
			controller.Execute()
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should update out of date value on kv", func() {
			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-non-migratable-%d", i), false, true)
				vmiSource.Add(vmi)
			}
			// add vmis that are not outdated to ensure they are not counted as outdated in count
			for i := 0; i < 100; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-up-to-date-%d", i), false, false)
				vmiSource.Add(vmi)
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(0)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodShutdown}
			addKubeVirt(kv)

			kubeVirtInterface.EXPECT().PatchStatus(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(name string, pt types.PatchType, data []byte) {
				str := string(data)
				Expect(str).To(Equal("[{ \"op\": \"test\", \"path\": \"/status/outdatedVMIWorkloads\", \"value\": 0}, { \"op\": \"replace\", \"path\": \"/status/outdatedVMIWorkloads\", \"value\": 100}]"))

			}).Return(nil, nil).Times(1)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(int(virtconfig.ParallelMigrationsPerClusterDefault))
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(defaultBatchDeletionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should migrate VMIs up to the global max migration count and delete up to delete batch count", func() {
			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-%d", i), false, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(100)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodShutdown}
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(int(virtconfig.ParallelMigrationsPerClusterDefault))
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(defaultBatchDeletionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should detect in-flight migrations when only migrate VMIs up to the global max migration count", func() {
			kv := newKubeVirt(50)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodShutdown}
			addKubeVirt(kv)

			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, true)
				vmiSource.Add(vmi)
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
			vmi := newVirtualMachine("testvm-outdated-migratable", true, true)
			reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			vmiSource.Add(vmi)

			vmi = newVirtualMachine("testvm-outdated-non-migratable", false, true)
			reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			vmiSource.Add(vmi)

			for i := 0; i < 50; i++ {
				vmi = newVirtualMachine(fmt.Sprintf("testvm-up-to-date-migratable-%d", i), true, false)
				vmiSource.Add(vmi)
				vmi = newVirtualMachine(fmt.Sprintf("testvm-up-to-date-non-migratable-%d", i), false, false)
				vmiSource.Add(vmi)
			}

			// wait for informer to catch up since we aren't watching for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodShutdown}
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(1)
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should only migrate VMIs and leave non migratable alone when shutdown method isn't set", func() {
			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-%d", i), false, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < int(virtconfig.ParallelMigrationsPerClusterDefault); i++ {
				reasons = append(reasons, SuccessfulCreateVirtualMachineInstanceMigrationReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(100)
			addKubeVirt(kv)

			migrationInterface.EXPECT().Create(gomock.Any()).Return(&v1.VirtualMachineInstanceMigration{ObjectMeta: v13.ObjectMeta{Name: "something"}}, nil).Times(int(virtconfig.ParallelMigrationsPerClusterDefault))

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should shutdown VMIs and not migrate when only shutdown method is set", func() {
			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < defaultBatchDeletionCount; i++ {
				reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(50)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodShutdown}
			addKubeVirt(kv)

			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(defaultBatchDeletionCount)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should respect custom batch deletion count", func() {
			batchDeletions := 30
			reasons := []string{}
			for i := 0; i < 50; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			}

			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(50)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodShutdown}
			kv.Spec.WorkloadUpdateStrategy.BatchShutdownCount = &batchDeletions
			addKubeVirt(kv)

			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(batchDeletions)

			controller.Execute()
			testutils.ExpectEvents(recorder, reasons...)
		})

		It("should respect custom batch interval", func() {
			controller.throttleIntervalSeconds = 0

			batchDeletions := 5
			batchInterval := time.Duration(2) * time.Second
			reasons := []string{}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			}

			for i := 0; i < batchDeletions*2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-1-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(batchDeletions * 2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodShutdown}
			kv.Spec.WorkloadUpdateStrategy.BatchShutdownCount = &batchDeletions
			kv.Spec.WorkloadUpdateStrategy.BatchShutdownInterval = &metav1.Duration{
				Duration: batchInterval,
			}

			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(batchDeletions * 2)

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
		})

		It("should respect reconcile loop throttling", func() {
			controller.throttleIntervalSeconds = 5

			batchDeletions := 5
			batchInterval := time.Duration(2) * time.Second
			reasons := []string{}
			for i := 0; i < batchDeletions; i++ {
				reasons = append(reasons, SuccessfulDeleteVirtualMachineInstanceReason)
			}

			for i := 0; i < batchDeletions*2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvm-migratable-1-%d", i), true, true)
				vmiSource.Add(vmi)
			}
			// wait for informer to catch up since we aren't watching
			// for vmis directly
			time.Sleep(1 * time.Second)
			kv := newKubeVirt(batchDeletions * 2)
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodShutdown}
			kv.Spec.WorkloadUpdateStrategy.BatchShutdownCount = &batchDeletions
			kv.Spec.WorkloadUpdateStrategy.BatchShutdownInterval = &metav1.Duration{
				Duration: batchInterval,
			}

			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(batchDeletions * 2)

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
			Phase:                v1.KubeVirtPhaseDeployed,
			OutdatedVMIWorkloads: &expectedNumOutdated,
		},
	}
}

func newVirtualMachine(name string, isMigratable bool, isOutdated bool) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMI("testvm")
	vmi.Name = name
	vmi.Namespace = v12.NamespaceDefault
	if isOutdated {
		vmi.Labels = map[string]string{v1.OutdatedLauncherImageLabel: ""}
	}
	vmi.Status.Phase = v1.Running
	vmi.UID = "1234"
	if isMigratable {
		vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceIsMigratable, Status: v12.ConditionTrue}}
	}
	return vmi
}

func newMigration(name string, vmi string, phase v1.VirtualMachineInstanceMigrationPhase) *v1.VirtualMachineInstanceMigration {
	migration := kubecli.NewMinimalMigration(name)
	migration.Status.Phase = phase
	migration.Spec.VMIName = vmi
	migration.Namespace = v12.NamespaceDefault
	return migration
}
