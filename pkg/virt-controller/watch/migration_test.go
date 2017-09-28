/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package watch

import (
	"net/http"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	clientv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("Migration", func() {
	var recorder *record.FakeRecorder
	var (
		app            VirtControllerApp = VirtControllerApp{}
		server         *ghttp.Server
		migration      *v1.Migration
		vm             *v1.VirtualMachine
		pod            *clientv1.Pod
		podList        clientv1.PodList
		migrationKey   interface{}
		srcIp          clientv1.NodeAddress
		destIp         kubev1.NodeAddress
		srcNodeWithIp  kubev1.Node
		destNodeWithIp kubev1.Node
		srcNode        kubev1.Node
		destNode       kubev1.Node
	)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	destinationNodeName := "mynode"
	sourceNodeName := "sourcenode"

	app.launcherImage = "kubevirt/virt-launcher"
	app.migratorImage = "kubevirt/virt-handler"

	BeforeEach(func() {

		server = ghttp.NewServer()
		app.clientSet, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.restClient = app.clientSet.RestClient()

		app.vmCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)

		app.migrationCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		app.migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		recorder = record.NewFakeRecorder(100)
		app.migrationRecorder = recorder

		app.initCommon()
		// Create a VM which is being scheduled

		vm = v1.NewMinimalVM("testvm")
		vm.Status.Phase = v1.Running
		vm.ObjectMeta.SetUID(uuid.NewUUID())

		migration = v1.NewMinimalMigration(vm.ObjectMeta.Name+"-migration", vm.ObjectMeta.Name)
		migration.ObjectMeta.SetUID(uuid.NewUUID())
		migration.Spec.NodeSelector = map[string]string{"beta.kubernetes.io/arch": "amd64"}

		// Create a target Pod for the VM
		templateService, err := services.NewTemplateService("whatever", "whatever", "whatever")
		Expect(err).ToNot(HaveOccurred())
		pod, err = templateService.RenderLaunchManifest(vm)
		Expect(err).ToNot(HaveOccurred())

		pod.Spec.NodeName = destinationNodeName
		pod.Status.Phase = clientv1.PodSucceeded
		pod.Labels[v1.DomainLabel] = migration.ObjectMeta.Name

		podList = clientv1.PodList{}
		podList.Items = []clientv1.Pod{*pod}

		srcIp = kubev1.NodeAddress{
			Type:    kubev1.NodeInternalIP,
			Address: "127.0.0.2",
		}
		destIp = kubev1.NodeAddress{
			Type:    kubev1.NodeInternalIP,
			Address: "127.0.0.3",
		}
		srcNodeWithIp = kubev1.Node{
			Status: kubev1.NodeStatus{
				Addresses: []kubev1.NodeAddress{srcIp},
			},
		}
		destNodeWithIp = kubev1.Node{
			Status: kubev1.NodeStatus{
				Addresses: []kubev1.NodeAddress{destIp},
			},
		}

		srcNode = clientv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: sourceNodeName,
			},
			Status: clientv1.NodeStatus{
				Addresses: []clientv1.NodeAddress{srcIp, destIp},
			},
		}
		destNode = clientv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: destinationNodeName,
			},
			Status: clientv1.NodeStatus{
				Addresses: []clientv1.NodeAddress{destIp, srcIp},
			},
		}
		migrationKey, err = cache.DeletionHandlingMetaNamespaceKeyFunc(migration)
		Expect(err).ToNot(HaveOccurred())

	})

	buildExpectedVM := func(phase v1.VMPhase) *v1.VirtualMachine {

		obj, err := conversion.NewCloner().DeepCopy(vm)
		Expect(err).ToNot(HaveOccurred())

		expectedVM := obj.(*v1.VirtualMachine)
		expectedVM.Status.Phase = phase
		expectedVM.Status.MigrationNodeName = pod.Spec.NodeName
		expectedVM.Spec.NodeSelector = map[string]string{"beta.kubernetes.io/arch": "amd64"}
		expectedVM.ObjectMeta.Labels = map[string]string{}

		return expectedVM
	}

	handlePutMigration := func(migration *v1.Migration, expectedStatus v1.MigrationPhase) http.HandlerFunc {

		obj, err := conversion.NewCloner().DeepCopy(migration)
		Expect(err).ToNot(HaveOccurred())

		expectedMigration := obj.(*v1.Migration)
		expectedMigration.Status.Phase = expectedStatus

		return ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
			ghttp.VerifyJSONRepresenting(expectedMigration),
			ghttp.RespondWithJSONEncoded(http.StatusOK, expectedMigration),
		)
	}

	Context("Running Migration target Pod for a running VM given", func() {
		It("should update the VM with the migration target node of the running Pod", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(podList),
				handleCreatePod(pod),
				handlePutMigration(migration, v1.MigrationRunning),
			)
			app.migrationQueue.Add(migrationKey)
			app.migrationCache.Add(migration)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("failed GET oF VM should requeue", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVMAuthError(buildExpectedVM(v1.Running)),
			)

			app.migrationQueue.Add(migrationKey)
			app.migrationCache.Add(migration)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("failed GET oF Pod List should requeue", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodListAuthError(podList),
			)

			app.migrationQueue.Add(migrationKey)
			app.migrationCache.Add(migration)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("Should Mark Migration as failed if VM Not found.", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVMNotFound(),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			app.migrationQueue.Add(migrationKey)
			app.migrationCache.Add(migration)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should requeue if VM Not found and Migration update error.", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVMNotFound(),
				handlePutMigrationAuthError(),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("Should mark Migration failed if VM not running ", func() {
			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Scheduled)),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("Should Requeue if VM not running and updateMigratio0n Failure", func() {
			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Scheduled)),
				handlePutMigrationAuthError(),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should requeue if Migration update fails", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(podList),
				handleCreatePod(pod),
				handlePutMigrationAuthError(),
			)
			app.migrationCache.Add(migration)

			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should fail if conflicting VM and Migration have conflicting Node Selectors", func() {
			vm := buildExpectedVM(v1.Running)
			vm.Spec.NodeSelector = map[string]string{"beta.kubernetes.io/arch": "i386"}

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(vm),
			)
			app.migrationCache.Add(migration)

			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should requeue if create of the Target Pod fails ", func() {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(podList),
				handleCreatePodAuthError(),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			app.migrationCache.Add(migration)

			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())
		})

		It("should fail if another migration is in process.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			currentMigration := v1.NewMinimalMigration(vm.ObjectMeta.Name+"-current", vm.ObjectMeta.Name)
			unmatchedPodList.Items = []clientv1.Pod{mockPod(1, "bogus"), mockPod(2, "bogus")}
			unmatchedPodList.Items[0].Labels[v1.MigrationLabel] = currentMigration.GetObjectMeta().GetName()
			app.migrationCache.Add(currentMigration)

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(recorder.Events).To(BeEmpty())
			close(done)
		}, 10)

		It("should requeue if another migration is in process and migration update fails.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			currentMigration := v1.NewMinimalMigration(vm.ObjectMeta.Name+"-current", vm.ObjectMeta.Name)
			unmatchedPodList.Items = []clientv1.Pod{mockPod(1, "bogus"), mockPod(2, "bogus")}
			unmatchedPodList.Items[0].Labels[v1.MigrationLabel] = currentMigration.GetObjectMeta().GetName()
			app.migrationCache.Add(currentMigration)

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(unmatchedPodList),
				handlePutMigrationAuthError(),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			Expect(recorder.Events).To(BeEmpty())

			close(done)
		}, 10)

		It("should succeed if many migrations, and this is one.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.NodeName = sourceNodeName

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationRunning),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(recorder.Events).To(BeEmpty())

			close(done)
		}, 10)

		It("should create migration Pod if migration and pod not created.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)

			migrationPodList := clientv1.PodList{}
			migrationPodList.Items = []clientv1.Pod{
				*mockMigrationPod(expectedVM0),
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationRunning),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(recorder.Events).To(BeEmpty())

			close(done)
		}, 10)

		It("should set vm.Status.MigrationNodeName if Not set.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			migrationLabel := string(migration.GetObjectMeta().GetUID())
			migration.Status.Phase = v1.MigrationRunning

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.MigrationNodeName = ""

			expectedVM2 := buildExpectedVM(v1.Migrating)
			expectedVM2.Status.MigrationNodeName = destinationNodeName

			migrationPodList := clientv1.PodList{}
			migrationPodList.Items = []clientv1.Pod{
				*mockMigrationPod(expectedVM0),
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handlePutVM(expectedVM2),
				handleGetPodList(migrationPodList),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(<-recorder.Events).To(ContainSubstring(v1.StartedVirtualMachineMigration.String()))
			Expect(recorder.Events).To(BeEmpty())

			close(done)
		}, 10)

		It("should mark migration as successful if migration pod completes successfully.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}
			migration.Status.Phase = v1.MigrationRunning

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.NodeName = sourceNodeName
			expectedVM1 := buildExpectedVM(v1.Migrating)
			expectedVM1.Status.MigrationNodeName = destinationNodeName
			expectedVM2 := buildExpectedVM(v1.Running)
			expectedVM2.Status.MigrationNodeName = ""
			expectedVM2.Status.NodeName = destinationNodeName
			expectedVM2.ObjectMeta.Labels = map[string]string{v1.NodeNameLabel: destinationNodeName}

			migrationPod := *mockMigrationPod(expectedVM2)
			migrationPod.Status.Phase = clientv1.PodSucceeded

			migrationPodList := clientv1.PodList{
				Items: []clientv1.Pod{
					migrationPod,
				},
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handleGetPodList(migrationPodList),
				handlePutVM(expectedVM2),
				handlePutMigration(migration, v1.MigrationSucceeded),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(5))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(<-recorder.Events).To(ContainSubstring(v1.SucceededVirtualMachineMigration.String()))
			Expect(recorder.Events).To(BeEmpty())

			close(done)
		}, 10)

		It("should mark migration as failed if migration pod fails.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}
			migration.Status.Phase = v1.MigrationRunning

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.NodeName = sourceNodeName
			expectedVM1 := buildExpectedVM(v1.Running)
			expectedVM1.Status.MigrationNodeName = ""
			expectedVM1.Status.NodeName = sourceNodeName

			migrationPod := *mockMigrationPod(expectedVM1)
			migrationPod.Status.Phase = clientv1.PodFailed

			migrationPodList := clientv1.PodList{
				Items: []clientv1.Pod{
					migrationPod,
				},
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handleGetPodList(migrationPodList),
				handlePutVM(expectedVM1),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			app.migrationCache.Add(migration)
			app.migrationQueue.Add(migrationKey)
			app.migrationController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(5))
			Expect(app.migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			Expect(<-recorder.Events).To(ContainSubstring(v1.FailedVirtualMachineMigration.String()))
			Expect(recorder.Events).To(BeEmpty())

			close(done)
		}, 10)

	})

	Context("Pod Investigation", func() {
		var (
			podList   kubev1.PodList
			migration *v1.Migration
		)

		BeforeEach(func() {
			pod1 := kubev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod1",
					Labels: map[string]string{
						v1.MigrationUIDLabel: "ce662d9f-34c0-40fd-a4e4-abe4146a1457",
						v1.MigrationLabel:    "test-migration1",
					},
					Namespace: "test",
				},
			}
			pod2 := kubev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod2",
					Labels: map[string]string{
						v1.MigrationUIDLabel: "99a8ac71-4ced-48fa-9bd4-0b4322dcc3dd",
						v1.MigrationLabel:    "test-migration1",
					},
					Namespace: "test",
				},
			}
			pod3 := kubev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod3",
					Labels: map[string]string{
						v1.MigrationUIDLabel: "7efc4067-039e-4c21-a494-0b52c09fe6fb",
						v1.MigrationLabel:    "test-migration2",
					},
					Namespace: "test",
				},
			}
			podList.Items = []kubev1.Pod{pod1, pod2, pod3}

			app.migrationCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			migration = new(v1.Migration)
			migration.ObjectMeta.Name = "test-migration1"
			migration.ObjectMeta.Namespace = "test"
			migration.Status.Phase = v1.MigrationFailed
		})

		It("should count exact matches", func() {
			migration.ObjectMeta.UID = "ce662d9f-34c0-40fd-a4e4-abe4146a1457"
			podList.Items = []kubev1.Pod{podList.Items[0]}
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, app.migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(1))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[0].ObjectMeta.Name))
		})

		It("should count uncached pods", func() {
			migration.ObjectMeta.UID = "99a8ac71-4ced-48fa-9bd4-0b4322dcc3dd"
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, app.migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(3))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[1].ObjectMeta.Name))
		})

		It("should ignore finalized migrations", func() {
			migration.ObjectMeta.UID = "ce662d9f-34c0-40fd-a4e4-abe4146a1457"
			app.migrationCache.Add(migration)
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, app.migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(2))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[0].ObjectMeta.Name))
		})

		It("should not count pods without MigrationLabels", func() {
			migration.ObjectMeta.UID = "ce662d9f-34c0-40fd-a4e4-abe4146a1457"
			app.migrationCache.Add(migration)
			delete(podList.Items[2].Labels, v1.MigrationLabel)
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, app.migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(1))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[0].ObjectMeta.Name))
		})
	})

	AfterEach(func() {
		server.Close()
	})

})

func handleCreatePod(pod *clientv1.Pod) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
		//TODO: Validate that posted Pod request is sane
		ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
	)
}

func handleCreatePodAuthError() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
		//TODO: Validate that posted Pod request is sane
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, ""),
	)
}

func handlePutMigrationAuthError() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, ""),
	)
}

func handleGetPodList(podList clientv1.PodList) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
		ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
	)
}

func handleGetPodListAuthError(podList clientv1.PodList) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, podList),
	)
}

func handleGetTestVM(expectedVM *v1.VirtualMachine) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
		ghttp.RespondWithJSONEncoded(http.StatusOK, expectedVM),
	)
}

func handleGetTestVMNotFound() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
		ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
	)
}

func handleGetTestVMAuthError(expectedVM *v1.VirtualMachine) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, expectedVM),
	)
}

func mockPod(i int, label string) clientv1.Pod {
	return clientv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-migration" + strconv.Itoa(i),
			Labels: map[string]string{
				v1.DomainLabel:       "testvm",
				v1.AppLabel:          "virt-launcher",
				v1.MigrationUIDLabel: label,
			},
		},
		Status: clientv1.PodStatus{
			Phase: clientv1.PodRunning,
		},
	}
}

func mockMigrationPod(vm *v1.VirtualMachine) *kubev1.Pod {
	temlateService, err := services.NewTemplateService("whatever", "whatever", "whatever")
	Expect(err).ToNot(HaveOccurred())
	pod, err := temlateService.RenderLaunchManifest(vm)
	Expect(err).ToNot(HaveOccurred())
	pod.Spec.NodeName = "targetNode"
	pod.Labels[v1.MigrationLabel] = "testvm-migration"
	return pod
}

func handlePutVM(vm *v1.VirtualMachine) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/"+vm.ObjectMeta.Name),
		ghttp.VerifyJSONRepresenting(vm),
		ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
	)
}

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration")
}
