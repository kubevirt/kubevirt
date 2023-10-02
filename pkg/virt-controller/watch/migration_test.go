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
	"context"
	"fmt"
	"strings"
	"time"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/api"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
)

var _ = Describe("Migration watcher", func() {

	var ctrl *gomock.Controller
	var migrationSource *framework.FakeControllerSource
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var podInformer cache.SharedIndexInformer
	var migrationInformer cache.SharedIndexInformer
	var nodeInformer cache.SharedIndexInformer
	var pdbInformer cache.SharedIndexInformer
	var migrationPolicyInformer cache.SharedIndexInformer
	var resourceQuotaInformer cache.SharedIndexInformer
	var namespaceInformer cache.SharedIndexInformer
	var storageClassInformer cache.SharedIndexInformer
	var storageProfileInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *MigrationController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var virtClient *kubecli.MockKubevirtClient
	var virtClientset *kubevirtfake.Clientset
	var kubeClient *fake.Clientset
	var networkClient *fakenetworkclient.Clientset
	var pvcInformer cache.SharedIndexInformer
	var qemuGid int64 = 107
	var namespace k8sv1.Namespace

	expectMigrationFinalizerRemoved := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Finalizers).To(BeEmpty())
	}

	expectPodCreation := func(namespace string, uid types.UID, migrationUid types.UID, expectedAntiAffinityCount int, expectedAffinityCount int, expectedNodeAffinityCount int) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", virtv1.MigrationJobLabel, string(migrationUid), virtv1.CreatedByLabel, string(uid)),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Spec.Affinity).ToNot(BeNil())
		Expect(pods.Items[0].Spec.Affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(pods.Items[0].Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(expectedAntiAffinityCount))
		if expectedAffinityCount > 0 {
			Expect(pods.Items[0].Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(expectedAffinityCount))
		}
		if expectedNodeAffinityCount > 0 {
			Expect(pods.Items[0].Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(expectedNodeAffinityCount))
		}
	}

	expectPodDoesNotExist := func(namespace, uid, migrationUid string) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", virtv1.MigrationJobLabel, migrationUid, virtv1.CreatedByLabel, uid),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).To(BeEmpty())
	}

	expectAttachmentPodCreation := func(namespace, migrationUid string) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", virtv1.MigrationJobLabel, migrationUid, virtv1.AppLabel, "hotplug-disk"),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Labels[virtv1.MigrationJobLabel]).To(Equal(migrationUid))
		Expect(pods.Items[0].Spec.Affinity).ToNot(BeNil())
		Expect(pods.Items[0].Spec.Affinity.NodeAffinity).ToNot(BeNil())
		Expect(pods.Items[0].Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
	}

	expectPDB := func(namespace, migrationName, vmiUID string) {
		pdbList, err := kubeClient.PolicyV1().PodDisruptionBudgets(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", virtv1.MigrationNameLabel, migrationName),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(pdbList.Items).To(HaveLen(1))
		Expect(pdbList.Items[0].Spec.MinAvailable.String()).To(Equal("2"))
		Expect(pdbList.Items[0].Spec.Selector).ToNot(BeNil())
		Expect(pdbList.Items[0].Spec.Selector.MatchLabels).To(HaveKeyWithValue(virtv1.CreatedByLabel, vmiUID))
	}

	expectPodAnnotationTimestamp := func(namespace, name, expectedTimestamp string) {
		updatedPod, err := kubeClient.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedPod).ToNot(BeNil())
		Expect(updatedPod.Annotations).To(HaveKeyWithValue(virtv1.MigrationTargetReadyTimestamp, expectedTimestamp))
	}

	expectMigrationSchedulingState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationScheduling))
	}

	expectMigrationPreparingTargetState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationPreparingTarget))
	}

	expectMigrationTargetReadyState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationTargetReady))
	}

	expectMigrationRunningState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationRunning))
	}

	expectMigrationCompletedState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationSucceeded))
	}

	expectMigrationPendingState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationPending))
	}

	expectMigrationFailedState := func(namespace, name string) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Phase).To(BeEquivalentTo(virtv1.MigrationFailed))
	}

	expectMigrationStateUpdated := func(namespace, name string, expectedState *virtv1.VirtualMachineInstanceMigrationState) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.MigrationState).To(Equal(expectedState))
	}

	expectVirtualMachineInstanceMigrationState := func(namespace, name string, matchers ...gomegaTypes.GomegaMatcher) {
		updatedVMI, err := virtClientset.KubevirtV1().VirtualMachineInstances(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI.Status.MigrationState).To(SatisfyAll(matchers...))
	}

	expectVirtualMachineInstanceLabels := func(namespace, name string, matchers ...gomegaTypes.GomegaMatcher) {
		updatedVMI, err := virtClientset.KubevirtV1().VirtualMachineInstances(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI.Labels).To(SatisfyAll(matchers...))
	}

	expectVirtualMachineInstanceMigrationConfiguration := func(namespace, name string, expectedConfiguration *virtv1.MigrationConfiguration) {
		updatedVMI, err := virtClientset.KubevirtV1().VirtualMachineInstances(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI.Status.MigrationState).ToNot(BeNil())
		Expect(updatedVMI.Status.MigrationState.MigrationConfiguration).ToNot(BeNil())
		Expect(updatedVMI.Status.MigrationState.MigrationConfiguration).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"NodeDrainTaintKey":                 Equal(expectedConfiguration.NodeDrainTaintKey),
			"ParallelOutboundMigrationsPerNode": Equal(expectedConfiguration.ParallelOutboundMigrationsPerNode),
			"ParallelMigrationsPerCluster":      Equal(expectedConfiguration.ParallelMigrationsPerCluster),
			"AllowAutoConverge":                 Equal(expectedConfiguration.AllowAutoConverge),
			"BandwidthPerMigration": WithTransform(func(qnt *resource.Quantity) string {
				return qnt.String()
			}, BeEquivalentTo(expectedConfiguration.BandwidthPerMigration.String())),
			"CompletionTimeoutPerGiB":      Equal(expectedConfiguration.CompletionTimeoutPerGiB),
			"ProgressTimeout":              Equal(expectedConfiguration.ProgressTimeout),
			"UnsafeMigrationOverride":      Equal(expectedConfiguration.UnsafeMigrationOverride),
			"AllowPostCopy":                Equal(expectedConfiguration.AllowPostCopy),
			"DisableTLS":                   Equal(expectedConfiguration.DisableTLS),
			"Network":                      Equal(expectedConfiguration.Network),
			"MatchSELinuxLevelOnMigration": Equal(expectedConfiguration.MatchSELinuxLevelOnMigration),
		})))
	}

	expectMigrationCondition := func(namespace, name string, conditionType virtv1.VirtualMachineInstanceMigrationConditionType) {
		updatedVMIM, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIM.Status.Conditions).To(ContainElement(
			MatchFields(IgnoreExtras, Fields{"Type": BeEquivalentTo(conditionType), "Status": Equal(k8sv1.ConditionTrue)}),
		))
	}

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go podInformer.Run(stop)
		go migrationInformer.Run(stop)
		go nodeInformer.Run(stop)
		go pdbInformer.Run(stop)
		go migrationPolicyInformer.Run(stop)
		go resourceQuotaInformer.Run(stop)
		go namespaceInformer.Run(stop)
		go storageClassInformer.Run(stop)
		go storageProfileInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			podInformer.HasSynced,
			migrationInformer.HasSynced,
			nodeInformer.HasSynced,
			pdbInformer.HasSynced,
			resourceQuotaInformer.HasSynced,
			namespaceInformer.HasSynced,
			storageClassInformer.HasSynced,
			storageProfileInformer.HasSynced,
			migrationPolicyInformer.HasSynced)).To(BeTrue())
	}

	initController := func(kvConfig *virtv1.KubeVirtConfiguration) {
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kvConfig)

		controller, _ = NewMigrationController(
			services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", resourceQuotaInformer.GetStore(), namespaceInformer.GetStore(), storageClassInformer.GetStore(), pvcInformer.GetIndexer(), storageProfileInformer.GetStore()),
			vmiInformer,
			podInformer,
			migrationInformer,
			nodeInformer,
			pvcInformer,
			pdbInformer,
			migrationPolicyInformer,
			resourceQuotaInformer,
			recorder,
			virtClient,
			config,
		)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClientset = kubevirtfake.NewSimpleClientset()

		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		migrationInformer, migrationSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstanceMigration{})
		podInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		pdbInformer, _ = testutils.NewFakeInformerFor(&policyv1.PodDisruptionBudget{})
		resourceQuotaInformer, _ = testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		namespaceInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		storageClassInformer, _ = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		storageProfileInformer, _ = testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
		migrationPolicyInformer, _ = testutils.NewFakeInformerFor(&migrationsv1.MigrationPolicy{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		nodeInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Node{})

		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})

		initController(&virtv1.KubeVirtConfiguration{})

		namespace = k8sv1.Namespace{
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: metav1.NamespaceDefault},
		}

		// Set up mock client
		kubeClient = fake.NewSimpleClientset(&namespace)
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachineInstances(k8sv1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().PolicyV1().Return(kubeClient.PolicyV1()).AnyTimes()
		networkClient = fakenetworkclient.NewSimpleClientset()
		virtClient.EXPECT().NetworkClient().Return(networkClient).AnyTimes()
		virtClient.EXPECT().MigrationPolicy().Return(virtClientset.MigrationsV1alpha1().MigrationPolicies()).AnyTimes()

		syncCaches(stop)
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
	})

	addPod := func(pod *k8sv1.Pod) {
		ExpectWithOffset(1, podInformer.GetStore().Add(pod)).To(Succeed())
		_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addVirtualMachineInstance := func(vmi *virtv1.VirtualMachineInstance) {
		// Annotations and Labels are defined as `omitempty`
		// This means that, if empty, they will be stored as nil in the fakeclient (as real clusters do).
		// Since we are storing the passed vmi resource in the vmiSource Store we need to clean
		// the annotations and the labels in case they are empty to reflect the reality.
		// Otherwise, we are going to create an inconsistency (unreal case) between the Store and
		// the resource in the fakeclient.
		// The latter could lead to failures composing the patch operations.
		if len(vmi.Annotations) == 0 {
			vmi.Annotations = nil
		}
		if len(vmi.Labels) == 0 {
			vmi.Labels = nil
		}
		mockQueue.ExpectAdds(1)
		vmiSource.Add(vmi)
		mockQueue.Wait()
		_, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addMigration := func(migration *virtv1.VirtualMachineInstanceMigration) {
		mockQueue.ExpectAdds(1)
		migrationSource.Add(migration)
		mockQueue.Wait()
		_, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addNode := func(node *k8sv1.Node) {
		err := nodeInformer.GetIndexer().Add(node)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = kubeClient.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addPDB := func(pdb *policyv1.PodDisruptionBudget) {
		err := pdbInformer.GetIndexer().Add(pdb)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = kubeClient.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Create(context.Background(), pdb, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addMigrationPolicies := func(policies ...migrationsv1.MigrationPolicy) {
		for _, policy := range policies {
			err := migrationPolicyInformer.GetIndexer().Add(&policy)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = virtClientset.MigrationsV1alpha1().MigrationPolicies().Create(context.Background(), &policy, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
	}

	getMigrationConfig := func(customConfigs ...*virtv1.MigrationConfiguration) *virtv1.MigrationConfiguration {
		Expect(customConfigs).To(Or(BeEmpty(), HaveLen(1)))

		var migrationConfiguration *virtv1.MigrationConfiguration

		if len(customConfigs) > 0 && customConfigs[0] != nil {
			migrationConfiguration = customConfigs[0]
		} else {
			migrationConfiguration = controller.clusterConfig.GetMigrationConfiguration()
			Expect(migrationConfiguration).ToNot(BeNil())
		}

		return migrationConfiguration
	}

	Context("Migration with hotplug volumes", func() {
		var (
			vmi           *virtv1.VirtualMachineInstance
			migration     *virtv1.VirtualMachineInstanceMigration
			sourcePod     *k8sv1.Pod
			targetPod     *k8sv1.Pod
			attachmentPod *k8sv1.Pod
		)

		BeforeEach(func() {
			vmi = newVirtualMachineWithHotplugVolume("testvmi", virtv1.Running)
			migration = newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			sourcePod = newSourcePodForVirtualMachine(vmi)
			targetPod = newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			attachmentPod = newAttachmentPodForVirtualMachine(targetPod, migration, k8sv1.PodRunning)
		})

		It("should create target attachment pod", func() {
			addMigration(migration)
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			addVirtualMachineInstance(vmi)
			addPod(sourcePod)
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectAttachmentPodCreation(migration.Namespace, string(migration.UID))
		})

		It("should set migration state to scheduling if attachment pod exists", func() {
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(sourcePod)
			addPod(targetPod)
			addPod(attachmentPod)

			controller.Execute()

			expectMigrationSchedulingState(migration.Namespace, migration.Name)
		})

		It("should hand pod over to target virt-handler if attachment pod is ready and running", func() {
			addNodeNameToVMI(vmi, "node02")
			migration.Status.Phase = virtv1.MigrationScheduled
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(sourcePod)
			addPod(targetPod)
			addPod(attachmentPod)

			controller.Execute()

			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"TargetNode":             Equal("node01"),
				"TargetPod":              Equal(targetPod.Name),
				"TargetAttachmentPodUID": Equal(attachmentPod.UID),
				"SourceNode":             Equal("node02"),
				"MigrationUID":           Equal(types.UID("testmigration")),
			})))
			expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
			expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name, HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"))
			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
		})

		It("should fail the migration if the attachment pod goes to final state", func() {
			attachmentPod.Status.Phase = k8sv1.PodFailed

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(sourcePod)
			addPod(targetPod)
			addPod(attachmentPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			expectMigrationFailedState(migration.Namespace, migration.Name)
		})
	})

	Context("Migration with hotplug", func() {
		var (
			vmi       *virtv1.VirtualMachineInstance
			migration *virtv1.VirtualMachineInstanceMigration
			sourcePod *k8sv1.Pod
			targetPod *k8sv1.Pod
		)

		BeforeEach(func() {
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration = newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			sourcePod = newSourcePodForVirtualMachine(vmi)
			targetPod = newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"
		})

		Context("CPU", func() {
			It("should annotate VMI with dedicated CPU limits", func() {
				vmi.Spec.Domain = virtv1.DomainSpec{
					CPU: &virtv1.CPU{
						DedicatedCPUPlacement: true,
						Cores:                 2,
						Sockets:               1,
						Threads:               1,
					},
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceVCPUChange,
						Status: k8sv1.ConditionTrue,
					})

				targetPod.Spec.Containers = append(targetPod.Spec.Containers, k8sv1.Container{
					Name: "compute",
					Resources: k8sv1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("4"),
						},
						Limits: k8sv1.ResourceList{
							k8sv1.ResourceCPU: resource.MustParse("4"),
						},
					},
				})
				targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}}

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				addPod(sourcePod)
				addPod(targetPod)

				controller.Execute()

				testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
				expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
					"TargetNode":   Equal("node01"),
					"TargetPod":    Equal(targetPod.Name),
					"SourceNode":   Equal("node02"),
					"MigrationUID": Equal(types.UID("testmigration")),
				})))
				expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
				expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name, HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"), HaveKeyWithValue(virtv1.VirtualMachinePodCPULimitsLabel, "4"))
			})
		})

		Context("Memory", func() {
			DescribeTable("should label VMI with target pod memory requests", func(hugepages *virtv1.Hugepages, expectedRequests string) {
				guestMemory := resource.MustParse("1Gi")
				vmi.Spec.Domain = virtv1.DomainSpec{
					Memory: &virtv1.Memory{
						Guest:     &guestMemory,
						Hugepages: hugepages,
					},
				}

				vmi.Status.Conditions = append(vmi.Status.Conditions,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceMemoryChange,
						Status: k8sv1.ConditionTrue,
					})

				targetPod.Spec.Containers = append(targetPod.Spec.Containers, k8sv1.Container{
					Name: "compute",
					Resources: k8sv1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("150Mi"),
						},
					},
				})
				targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}}

				if hugepages != nil {
					resourceName := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + hugepages.PageSize)
					targetPod.Spec.Containers[0].Resources.Requests[resourceName] = guestMemory
				}

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				addPod(sourcePod)
				addPod(targetPod)

				controller.Execute()

				testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
				expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
					"TargetNode":   Equal("node01"),
					"TargetPod":    Equal(targetPod.Name),
					"SourceNode":   Equal("node02"),
					"MigrationUID": Equal(types.UID("testmigration")),
				})))
				expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
				expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name,
					HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"),
					HaveKeyWithValue(virtv1.VirtualMachinePodMemoryRequestsLabel, expectedRequests),
				)
			},
				Entry("when using a common VM", nil, "150Mi"),
				Entry("when using 2Mi Hugepages", &virtv1.Hugepages{PageSize: "2Mi"}, "1174Mi"),
				Entry("when using 1Gi Hugepages", &virtv1.Hugepages{PageSize: "1Gi"}, "1174Mi"),
			)

			It("should mark migration as succeeded if memory hotplug failed", func() {
				vmi.Status.Conditions = append(vmi.Status.Conditions,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceMemoryChange,
						Status: k8sv1.ConditionFalse,
					},
				)

				addNodeNameToVMI(vmi, "node02")
				runningMigration := newMigration("testmigration", vmi.Name, virtv1.MigrationRunning)
				runningTargetPod := newTargetPodForVirtualMachine(vmi, runningMigration, k8sv1.PodRunning)
				runningTargetPod.Spec.NodeName = "node01"

				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					MigrationUID:      runningMigration.UID,
					TargetNode:        "node01",
					SourceNode:        "node02",
					TargetNodeAddress: "10.10.10.10:1234",
					StartTimestamp:    pointer.P(metav1.Now()),
					EndTimestamp:      pointer.P(metav1.Now()),
					Failed:            false,
					Completed:         true,
				}
				addMigration(runningMigration)
				addVirtualMachineInstance(vmi)
				addPod(newSourcePodForVirtualMachine(vmi))
				addPod(runningTargetPod)

				controller.Execute()

				testutils.ExpectEvent(recorder, virtcontroller.SuccessfulMigrationReason)
				expectMigrationCompletedState(migration.Namespace, migration.Name)
			})
		})
	})

	Context("Migration object in pending state", func() {
		It("should patch VMI with nonroot user", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			delete(vmi.Annotations, virtv1.DeprecatedNonRootVMIAnnotation)
			vmi.Status.RuntimeUser = 0
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
			updatedVMI, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVMI.Status.RuntimeUser).To(Equal(uint64(107)))
			Expect(updatedVMI.Annotations).To(HaveKeyWithValue(virtv1.DeprecatedNonRootVMIAnnotation, "true"))
		})

		It("should create target pod", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
		})

		It("should not create target pod if multiple pods exist in a non finalized state for VMI", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			pod1 := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			pod1.Labels[virtv1.MigrationJobLabel] = "some other job"
			pod2 := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			pod2.Labels[virtv1.MigrationJobLabel] = "some other job"
			addPod(pod1)
			addPod(pod2)
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			controller.Execute()

			pods, err := kubeClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).To(HaveLen(3))
		})

		It("should create another target pods if only 4 migrations are in progress", func() {
			// It should create a pod for this one
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			// Ensure that 4 migrations are there which are in non-final state
			for i := 0; i < 4; i++ {
				newVMI := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				addNodeNameToVMI(newVMI, fmt.Sprintf("node%v", i))
				migration := newMigration(fmt.Sprintf("testmigration%v", i), newVMI.Name, virtv1.MigrationScheduling)

				addMigration(migration)
				addVirtualMachineInstance(newVMI)
			}

			// Add two pending migrations without a target pod to see that tye get ignored
			for i := 0; i < 2; i++ {
				newVMI := newVirtualMachine(fmt.Sprintf("xtestvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("xtestmigration%v", i), newVMI.Name, virtv1.MigrationPending)
				addNodeNameToVMI(newVMI, fmt.Sprintf("node%v", i))

				addMigration(migration)
				addVirtualMachineInstance(newVMI)
			}

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
			for i := 0; i < 2; i++ {
				expectPodDoesNotExist(vmi.Namespace, fmt.Sprintf("xtestvmi%v", i), fmt.Sprintf("xtestmigration%v", i))
			}
		})

		It("should not overload the cluster and only run 5 migrations in parallel", func() {
			// It should create a pod for this one if we would not limit migrations
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			// Ensure that 5 migrations are there which are in non-final state
			for i := 0; i < 5; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)
				addNodeNameToVMI(vmi, fmt.Sprintf("node%v", i))

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			controller.Execute()

			expectPodDoesNotExist(vmi.Namespace, fmt.Sprintf("testvmi"), "testmigration")
		})

		It("should not overload the cluster and detect pending migrations as running if they have a target pod", func() {
			// It should create a pod for this one if we would not limit migrations
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			// Ensure that 3 migrations are there which are running
			for i := 0; i < 3; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)
				addNodeNameToVMI(vmi, fmt.Sprintf("node%v", i))

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			// Ensure that 2 migrations are pending but have a target pod
			for i := 0; i < 2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("xtestvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("xtestmigration%v", i), vmi.Name, virtv1.MigrationPending)
				pod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
				addNodeNameToVMI(vmi, fmt.Sprintf("node%v", i))

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				addPod(pod)
			}

			controller.Execute()

			expectPodDoesNotExist(vmi.Namespace, fmt.Sprintf("testvmi"), "testmigration")
		})

		It("should create another target pods if there is only one outbound migration on the node", func() {
			// It should create a pod for this one
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			// Ensure that 4 migrations are there which are in non-final state
			for i := 0; i < 1; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
		})

		It("should not overload the node and only run 2 outbound migrations in parallel", func() {
			// It should create a pod for this one if we would not limit migrations
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			// Ensure that 5 migrations are there which are in non-final state
			for i := 0; i < 2; i++ {
				vmi := newVirtualMachine(fmt.Sprintf("testvmi%v", i), virtv1.Running)
				migration := newMigration(fmt.Sprintf("testmigration%v", i), vmi.Name, virtv1.MigrationScheduling)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
			}

			controller.Execute()

			expectPodDoesNotExist(vmi.Namespace, fmt.Sprintf("testvmi"), "testmigration")
		})

		It("should create target pod and not override existing affinity rules", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			antiAffinityTerm := k8sv1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"somelabel": "somekey",
					},
				},
				TopologyKey: k8sv1.LabelHostname,
			}
			affinityTerm := k8sv1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"someotherlabel": "someotherkey",
					},
				},
				TopologyKey: k8sv1.LabelHostname,
			}
			antiAffinityRule := &k8sv1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{antiAffinityTerm},
			}
			affinityRule := &k8sv1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{affinityTerm},
			}

			nodeAffinityRule := &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      k8sv1.LabelHostname,
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{"somenode"},
								},
							},
						},
					},
				},
			}

			vmi.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity:    nodeAffinityRule,
				PodAntiAffinity: antiAffinityRule,
				PodAffinity:     affinityRule,
			}

			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 2, 1, 1)
		})

		It("should place migration in scheduling state if pod exists", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectMigrationSchedulingState(migration.Namespace, migration.Name)
		})

		DescribeTable("should handle pod stuck in unschedulable state", func(phase virtv1.VirtualMachineInstanceMigrationPhase, shouldTimeout bool, timeLapse int64, annotationVal string) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)

			if annotationVal != "" {
				migration.Annotations[virtv1.MigrationUnschedulablePodTimeoutSecondsAnnotation] = annotationVal
			}

			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)

			targetPod.Status.Conditions = append(targetPod.Status.Conditions, k8sv1.PodCondition{
				Type:   k8sv1.PodScheduled,
				Status: k8sv1.ConditionFalse,
				Reason: k8sv1.PodReasonUnschedulable,
			})
			targetPod.CreationTimestamp = metav1.NewTime(metav1.Now().Time.Add(time.Duration(-timeLapse) * time.Second))

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			if phase != virtv1.MigrationScheduled {
				testutils.ExpectEvent(recorder, virtcontroller.MigrationTargetPodUnschedulable)
			}

			if shouldTimeout {
				testutils.ExpectEvent(recorder, virtcontroller.SuccessfulDeletePodReason)
				expectPodDoesNotExist(vmi.Namespace, string(vmi.UID), string(migration.UID))
			}
		},
			Entry("in pending state", virtv1.MigrationPending, true, defaultUnschedulablePendingTimeoutSeconds, ""),
			Entry("in scheduling state", virtv1.MigrationScheduling, true, defaultUnschedulablePendingTimeoutSeconds, ""),
			Entry("in scheduled state", virtv1.MigrationScheduled, false, defaultUnschedulablePendingTimeoutSeconds, ""),
			Entry("in pending state but timeout not hit", virtv1.MigrationPending, false, defaultUnschedulablePendingTimeoutSeconds-10, ""),
			Entry("in pending state with custom timeout", virtv1.MigrationPending, true, int64(10), "10"),
			Entry("in pending state with custom timeout not hit", virtv1.MigrationPending, false, int64(8), "11"),
			Entry("in scheduling state but timeout not hit", virtv1.MigrationScheduling, false, defaultUnschedulablePendingTimeoutSeconds-10, ""),
		)

		DescribeTable("should handle pod stuck in pending phase for extended period of time", func(phase virtv1.VirtualMachineInstanceMigrationPhase, shouldTimeout bool, timeLapse int64, annotationVal string) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)

			targetPod.CreationTimestamp = metav1.NewTime(metav1.Now().Time.Add(time.Duration(-timeLapse) * time.Second))

			if annotationVal != "" {
				migration.Annotations[virtv1.MigrationPendingPodTimeoutSecondsAnnotation] = annotationVal
			}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			if shouldTimeout {
				testutils.ExpectEvent(recorder, virtcontroller.SuccessfulDeletePodReason)
				expectPodDoesNotExist(vmi.Namespace, string(vmi.UID), string(migration.UID))
			}
		},
			Entry("in pending state", virtv1.MigrationPending, true, defaultCatchAllPendingTimeoutSeconds, ""),
			Entry("in scheduling state", virtv1.MigrationScheduling, true, defaultCatchAllPendingTimeoutSeconds, ""),
			Entry("in scheduled state", virtv1.MigrationScheduled, false, defaultCatchAllPendingTimeoutSeconds, ""),
			Entry("in pending state but timeout not hit", virtv1.MigrationPending, false, defaultCatchAllPendingTimeoutSeconds-10, ""),
			Entry("in pending state with custom timeout", virtv1.MigrationPending, true, int64(10), "10"),
			Entry("in pending state with custom timeout not hit", virtv1.MigrationPending, false, int64(8), "11"),
			Entry("in scheduling state but timeout not hit", virtv1.MigrationScheduling, false, defaultCatchAllPendingTimeoutSeconds-10, ""),
		)
	})

	Context("Migration garbage collection", func() {
		DescribeTable("should garbage old finalized migration objects", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)

			phasesToGarbageCollect := []virtv1.VirtualMachineInstanceMigrationPhase{
				virtv1.MigrationFailed,
				virtv1.MigrationSucceeded,
			}

			phasesToKeep := []virtv1.VirtualMachineInstanceMigrationPhase{
				virtv1.MigrationPhaseUnset,
				virtv1.MigrationPending,
				virtv1.MigrationScheduling,
				virtv1.MigrationPreparingTarget,
				virtv1.MigrationTargetReady,
				virtv1.MigrationRunning,
			}

			for _, curPhase := range phasesToKeep {
				for i := 0; i < 100; i++ {
					mCopy := newMigration(fmt.Sprintf("should-keep-%s-%d", curPhase, i), vmi.Name, curPhase)
					mCopy.Finalizers = []string{}
					mCopy.Labels = map[string]string{"should-delete": "no"}

					mCopy.CreationTimestamp = metav1.Unix(int64(rand.Intn(100)), int64(0))

					Expect(migrationInformer.GetStore().Add(mCopy)).To(Succeed())
					_, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(mCopy.Namespace).Create(context.Background(), mCopy, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			}

			for _, curPhase := range phasesToGarbageCollect {
				for i := 0; i < 100; i++ {
					mCopy := newMigration(fmt.Sprintf("should-delete-%s-%d", curPhase, i), vmi.Name, curPhase)
					mCopy.Labels = map[string]string{"should-delete": "yes"}
					mCopy.CreationTimestamp = metav1.Unix(int64(rand.Intn(100)), int64(0))

					Expect(migrationInformer.GetStore().Add(mCopy)).To(Succeed())
					_, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(mCopy.Namespace).Create(context.Background(), mCopy, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			}

			keyMigration := newMigration("should-keep-key-migration", vmi.Name, phase)
			if keyMigration.IsFinal() {
				keyMigration.Labels = map[string]string{"should-delete": "yes"}
			}
			keyMigration.Finalizers = []string{}
			keyMigration.CreationTimestamp = metav1.Unix(int64(101), int64(0))
			addMigration(keyMigration)

			sourcePod := newSourcePodForVirtualMachine(vmi)
			Expect(podInformer.GetStore().Add(sourcePod)).To(Succeed())
			Expect(vmiInformer.GetStore().Add(vmi)).To(Succeed())

			controller.Execute()

			testutils.IgnoreEvents(recorder)
			migrationsStored, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).List(context.Background(), metav1.ListOptions{
				LabelSelector: "should-delete=yes",
			})
			Expect(err).ToNot(HaveOccurred())
			if keyMigration.IsFinal() {
				Expect(migrationsStored.Items).To(HaveLen(defaultFinalizedMigrationGarbageCollectionBuffer))
			} else {
				Expect(migrationsStored.Items).To(HaveLen(len(phasesToGarbageCollect) * 100))
			}
		},
			Entry("in failed phase", virtv1.MigrationFailed),
			Entry("in succeeded phase", virtv1.MigrationSucceeded),
			Entry("in unset phase", virtv1.MigrationPhaseUnset),
			Entry("in pending phase", virtv1.MigrationPending),
			Entry("in scheduling phase", virtv1.MigrationScheduling),
			Entry("in preparing target phase", virtv1.MigrationPreparingTarget),
			Entry("in target ready phase", virtv1.MigrationTargetReady),
			Entry("in running phase", virtv1.MigrationRunning),
		)
	})

	Context("Migration should immediately fail if", func() {
		DescribeTable("vmi moves to final state", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Succeeded)
			vmi.DeletionTimestamp = pointer.P(metav1.Now())
			migration := newMigration("testmigration", vmi.Name, phase)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
			}
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			expectMigrationFailedState(migration.Namespace, migration.Name)
		},
			Entry("in running state", virtv1.MigrationRunning),
			Entry("in unset state", virtv1.MigrationPhaseUnset),
			Entry("in pending state", virtv1.MigrationPending),
			Entry("in scheduled state", virtv1.MigrationScheduled),
			Entry("in scheduling state", virtv1.MigrationScheduling),
			Entry("in target ready state", virtv1.MigrationTargetReady),
		)

		DescribeTable("Pod moves to final state", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
			}
			if phase == virtv1.MigrationTargetReady {
				vmi.Status.MigrationState.StartTimestamp = pointer.P(metav1.Now())
			}
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodSucceeded)
			targetPod.Spec.NodeName = "node01"

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			expectMigrationFailedState(migration.Namespace, migration.Name)
		},
			Entry("in running state", virtv1.MigrationRunning),
			Entry("in unset state", virtv1.MigrationPhaseUnset),
			Entry("in pending state", virtv1.MigrationPending),
			Entry("in scheduled state", virtv1.MigrationScheduled),
			Entry("in scheduling state", virtv1.MigrationScheduling),
			Entry("in target ready state", virtv1.MigrationTargetReady),
		)

		DescribeTable("VMI's migrate state moves to final state", func(phase virtv1.VirtualMachineInstanceMigrationPhase) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, phase)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:   migration.UID,
				Failed:         true,
				Completed:      true,
				StartTimestamp: pointer.P(metav1.Now()),
				EndTimestamp:   pointer.P(metav1.Now()),
			}
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			expectMigrationFailedState(migration.Namespace, migration.Name)
		},
			Entry("in running state", virtv1.MigrationRunning),
			Entry("in unset state", virtv1.MigrationPhaseUnset),
			Entry("in pending state", virtv1.MigrationPending),
			Entry("in scheduled state", virtv1.MigrationScheduled),
			Entry("in scheduling state", virtv1.MigrationScheduling),
			Entry("in target ready state", virtv1.MigrationTargetReady),
		)
	})

	Context("Migration object ", func() {
		DescribeTable("should hand pod over to target virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = containerStatus

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"TargetNode":   Equal("node01"),
				"TargetPod":    Equal(targetPod.Name),
				"SourceNode":   Equal("node02"),
				"MigrationUID": Equal(types.UID("testmigration")),
			})))
			expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
			expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name, HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"))
		},
			Entry("with running compute container and no infra container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}},
			),
			Entry("with running compute container and no ready istio-proxy container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}, {Name: "istio-proxy", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}}, Ready: false}},
			),
		)

		DescribeTable("should not hand pod over to target virt-handler if pod is not ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = containerStatus

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, BeNil())
		},
			Entry("with not ready infra container and not ready compute container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}, {Name: "kubevirt-infra", Ready: false}},
			),
			Entry("with not ready compute container and no infra container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}},
			),
		)

		It("should hand pod over to target virt-handler with migration config", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)

			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"TargetNode":   Equal("node01"),
				"TargetPod":    Equal(targetPod.Name),
				"SourceNode":   Equal("node02"),
				"MigrationUID": Equal(types.UID("testmigration")),
			})))
			expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
			expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name, HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"))
		})

		It("should hand pod over to target virt-handler overriding previous state", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: "1111-2222-3333-4444",
			}
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"TargetNode":   Equal("node01"),
				"TargetPod":    Equal(targetPod.Name),
				"SourceNode":   Equal("node02"),
				"MigrationUID": Equal(types.UID("testmigration")),
			})))
			expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
			expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name, HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"))
		})

		It("should not hand pod over target pod that's already handed over", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
			}
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			targetPod.Spec.NodeName = "node01"

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"MigrationUID": Equal(migration.UID),
			})))
		})

		It("should not transition to PreparingTarget if VMI MigrationState is outdated", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"

			const oldMigrationUID = "oldmigrationuid"
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: types.UID(oldMigrationUID),
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"TargetNode":   Equal("node01"),
				"TargetPod":    Equal(targetPod.Name),
				"SourceNode":   Equal("node02"),
				"MigrationUID": Equal(types.UID("testmigration")),
			})))
			expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig())
			expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name, HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"))
		})

		It("should transition to preparing target phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
				TargetNode:   "node01",
				SourceNode:   "node02",
				TargetPod:    targetPod.Name,
			}
			vmi.Labels[virtv1.MigrationTargetNodeNameLabel] = "node01"
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectMigrationPreparingTargetState(migration.Namespace, migration.Name)
		})

		It("should transition to target prepared phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPreparingTarget)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			targetPod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectMigrationTargetReadyState(migration.Namespace, migration.Name)
		})

		It("should transition to running phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationTargetReady)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			targetPod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
				StartTimestamp:    pointer.P(metav1.Now()),
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectMigrationRunningState(migration.Namespace, migration.Name)
		})

		It("should transition to completed phase", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationRunning)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			targetPod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:                   migration.UID,
				TargetNode:                     "node01",
				SourceNode:                     "node02",
				TargetNodeAddress:              "10.10.10.10:1234",
				StartTimestamp:                 pointer.P(metav1.Now()),
				EndTimestamp:                   pointer.P(metav1.Now()),
				TargetNodeDomainReadyTimestamp: pointer.P(metav1.Now()),
				Failed:                         false,
				Completed:                      true,
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulMigrationReason)
			expectPodAnnotationTimestamp(targetPod.Namespace, targetPod.Name, vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp.String())
			expectMigrationCompletedState(migration.Namespace, migration.Name)
		})

		DescribeTable("should not transit to succeeded phase when VMI status has", func(conditions []virtv1.VirtualMachineInstanceConditionType) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationRunning)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			targetPod.Spec.NodeName = "node01"

			for _, c := range conditions {
				vmi.Status.Conditions = append(vmi.Status.Conditions,
					virtv1.VirtualMachineInstanceCondition{
						Type:          c,
						Status:        k8sv1.ConditionTrue,
						LastProbeTime: metav1.Now(),
					})
			}

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:                   migration.UID,
				TargetNode:                     "node01",
				SourceNode:                     "node02",
				TargetNodeAddress:              "10.10.10.10:1234",
				StartTimestamp:                 pointer.P(metav1.Now()),
				EndTimestamp:                   pointer.P(metav1.Now()),
				TargetNodeDomainReadyTimestamp: pointer.P(metav1.Now()),
				Failed:                         false,
				Completed:                      true,
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectMigrationRunningState(migration.Namespace, migration.Name)
			expectPodAnnotationTimestamp(targetPod.Namespace, targetPod.Name, vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp.String())
		},
			Entry("CPU change condition", []virtv1.VirtualMachineInstanceConditionType{virtv1.VirtualMachineInstanceVCPUChange}),
			Entry("Memory change condition", []virtv1.VirtualMachineInstanceConditionType{virtv1.VirtualMachineInstanceMemoryChange}),
			Entry("CPU and Memory change condition", []virtv1.VirtualMachineInstanceConditionType{virtv1.VirtualMachineInstanceMemoryChange, virtv1.VirtualMachineInstanceVCPUChange}),
		)

		It("should expect MigrationState to be updated on a completed migration", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationSucceeded)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"

			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
				StartTimestamp:    pointer.P(metav1.Now()),
				EndTimestamp:      pointer.P(metav1.Now()),
				Failed:            false,
				Completed:         true,
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			expectMigrationStateUpdated(migration.Namespace, migration.Name, vmi.Status.MigrationState)
			expectMigrationFinalizerRemoved(migration.Namespace, migration.Name)
		})

		It("should delete itself if VMI no longer exists", func() {
			migration := newMigration("testmigration", "somevmi", virtv1.MigrationRunning)
			addMigration(migration)

			controller.Execute()

			_, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
		})

		It("should abort the migration", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationRunning)
			condition := virtv1.VirtualMachineInstanceMigrationCondition{
				Type:          virtv1.VirtualMachineInstanceMigrationAbortRequested,
				Status:        k8sv1.ConditionTrue,
				LastProbeTime: metav1.Now(),
			}
			migration.Status.Conditions = append(migration.Status.Conditions, condition)
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			targetPod.Spec.NodeName = "node01"
			migration.DeletionTimestamp = pointer.P(metav1.Now())
			vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID:      migration.UID,
				TargetNode:        "node01",
				SourceNode:        "node02",
				TargetNodeAddress: "10.10.10.10:1234",
				StartTimestamp:    pointer.P(metav1.Now()),
			}
			controller.addHandOffKey(virtcontroller.MigrationKey(migration))
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulAbortMigrationReason)
			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
				"TargetNode":     Equal(vmi.Status.MigrationState.TargetNode),
				"SourceNode":     Equal(vmi.Status.MigrationState.SourceNode),
				"MigrationUID":   Equal(vmi.Status.MigrationState.MigrationUID),
				"AbortRequested": BeTrue(),
			})))
		})

		DescribeTable("should finalize migration on VMI if target pod fails before migration starts", func(phase virtv1.VirtualMachineInstanceMigrationPhase, hasPod bool, podPhase k8sv1.PodPhase, initializeMigrationState bool) {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, "node02")
			migration := newMigration("testmigration", vmi.Name, phase)

			vmi.Status.MigrationState = nil
			if initializeMigrationState {
				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					MigrationUID: migration.UID,
					TargetNode:   "node01",
					SourceNode:   "node02",
				}
			}
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			if hasPod {
				targetPod := newTargetPodForVirtualMachine(vmi, migration, podPhase)
				targetPod.Spec.NodeName = "node01"
				addPod(targetPod)
			}

			controller.Execute()

			// in this case, we have two failed events. one for the VMI and one on the Migration object.
			if initializeMigrationState {
				expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, Fields{
					"TargetNode":     Equal("node01"),
					"SourceNode":     Equal("node02"),
					"MigrationUID":   Equal(types.UID("testmigration")),
					"Completed":      BeTrue(),
					"Failed":         BeTrue(),
					"StartTimestamp": Not(BeNil()),
					"EndTimestamp":   Not(BeNil()),
				})))
				testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			}

			if phase == virtv1.MigrationFailed {
				if initializeMigrationState {
					expectMigrationStateUpdated(migration.Namespace, migration.Name, vmi.Status.MigrationState)
				}
				expectMigrationFinalizerRemoved(migration.Namespace, migration.Name)
			} else {
				testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
				expectMigrationFailedState(migration.Namespace, migration.Name)
			}
		},
			Entry("in preparing target state", virtv1.MigrationPreparingTarget, true, k8sv1.PodFailed, true),
			Entry("in target ready state", virtv1.MigrationTargetReady, true, k8sv1.PodFailed, true),
			Entry("in failed state", virtv1.MigrationFailed, true, k8sv1.PodFailed, true),
			Entry("in failed state before pod is created", virtv1.MigrationFailed, false, k8sv1.PodFailed, false),
			Entry("in failed state and pod does not exist", virtv1.MigrationFailed, false, k8sv1.PodFailed, false),
		)

		DescribeTable("with CPU mode which is", func(toDefineHostModelCPU bool) {
			const nodeName = "testNode"

			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, nodeName)
			if toDefineHostModelCPU {
				vmi.Spec.Domain.CPU = &virtv1.CPU{Model: virtv1.CPUModeHostModel}
			}

			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			node := newNode(nodeName)
			if toDefineHostModelCPU {
				node.ObjectMeta.Labels = map[string]string{
					virtv1.HostModelCPULabel + "fake":              "true",
					virtv1.SupportedHostModelMigrationCPU + "fake": "true",
					virtv1.HostModelRequiredFeaturesLabel + "fake": "true",
				}
			}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addNode(node)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 1)
			pods, err := kubeClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", virtv1.MigrationJobLabel, string(migration.UID), virtv1.CreatedByLabel, string(vmi.UID)),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))
			podHasCpuModeLabelSelector := false
			for key := range pods.Items[0].Spec.NodeSelector {
				if strings.Contains(key, virtv1.SupportedHostModelMigrationCPU) {
					podHasCpuModeLabelSelector = true
					break
				}
			}
			Expect(podHasCpuModeLabelSelector).To(Equal(toDefineHostModelCPU))
		},
			Entry("host-model should be targeted only to nodes which support the model", true),
			Entry("non-host-model should not be targeted to nodes which support the model", false),
		)
	})

	Context("Migration with protected VMI (PDB)", func() {
		It("should update PDB before starting the migration", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Spec.EvictionStrategy = pointer.P(virtv1.EvictionStrategyLiveMigrate)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			pdb := newPDB("pdb-test", vmi, 1)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPDB(pdb)

			controller.Execute()

			testutils.ExpectEvents(recorder, successfulUpdatePodDisruptionBudgetReason)
			expectPDB(migration.Namespace, migration.Name, string(vmi.UID))
		})

		It("should create the target Pod after the k8s PDB controller processed the PDB mutation", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Spec.EvictionStrategy = pointer.P(virtv1.EvictionStrategyLiveMigrate)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			pdb := newPDB("pdb-test", vmi, 2)
			pdb.Generation = 42
			pdb.Status.DesiredHealthy = int32(pdb.Spec.MinAvailable.IntValue())
			pdb.Status.ObservedGeneration = pdb.Generation
			pdb.Labels = map[string]string{
				virtv1.MigrationNameLabel: migration.Name,
			}

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPDB(pdb)

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
		})

		Context("when cluster EvictionStrategy is set to 'LiveMigrate'", func() {
			BeforeEach(func() {
				initController(&virtv1.KubeVirtConfiguration{EvictionStrategy: pointer.P(virtv1.EvictionStrategyLiveMigrate)})
			})

			It("should update PDB", func() {
				vmi := newVirtualMachine("testvmi", virtv1.Running)
				migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
				pdb := newPDB("pdb-test", vmi, 1)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				addPod(newSourcePodForVirtualMachine(vmi))
				addPDB(pdb)

				controller.Execute()

				testutils.ExpectEvents(recorder, successfulUpdatePodDisruptionBudgetReason)
				expectPDB(migration.Namespace, migration.Name, string(vmi.UID))
			})
		})
	})

	Context("Migration policy", func() {
		var vmi *virtv1.VirtualMachineInstance
		var stubNumber int64
		var stubResourceQuantity resource.Quantity
		var targetPod *k8sv1.Pod

		BeforeEach(func() {
			stubNumber = 33425
			stubResourceQuantity = resource.MustParse("25Mi")
		})

		Context("matching and precedence", func() {

			BeforeEach(func() {
				vmi = newVirtualMachine("testvmi", virtv1.Running)
			})

			type policyInfo struct {
				name                    string
				vmiMatchingLabels       int
				namespaceMatchingLabels int
			}

			DescribeTable("must be done correctly", func(expectedMatchedPolicyName string, policiesToDefine ...policyInfo) {
				policies := make([]migrationsv1.MigrationPolicy, 0)

				for _, info := range policiesToDefine {
					policy := preparePolicyAndVMIWithNSAndVMILabels(vmi, &namespace, info.vmiMatchingLabels, info.namespaceMatchingLabels)
					policy.Name = info.name
					policies = append(policies, *policy)
				}

				policyList := kubecli.NewMinimalMigrationPolicyList(policies...)
				actualMatchedPolicy := MatchPolicy(policyList, vmi, &namespace)

				Expect(actualMatchedPolicy).ToNot(BeNil())
				Expect(actualMatchedPolicy.Name).To(Equal(expectedMatchedPolicyName))
			},
				Entry("only one policy should be matched", "one", policyInfo{"one", 1, 4}),
				Entry("most detail policy should be matched", "two",
					policyInfo{"one", 1, 4}, policyInfo{"two", 4, 2}),
				Entry("if two policies are detailed at the same level, matching policy should be the first name in lexicographic order (1)", "aa",
					policyInfo{"aa", 2, 2}, policyInfo{"zz", 2, 2}),
				Entry("if two policies are detailed at the same level, matching policy should be the first name in lexicographic order (2)", "aa",
					policyInfo{"zz", 2, 2}, policyInfo{"aa", 2, 2}),
			)

			It("policy with one non-fitting label should not match", func() {
				const labelKeyFmt = "%s-key-0"

				policy := preparePolicyAndVMIWithNSAndVMILabels(vmi, &namespace, 4, 3)
				_, exists := policy.Spec.Selectors.VirtualMachineInstanceSelector[fmt.Sprintf(labelKeyFmt, policy.Name)]
				Expect(exists).To(BeTrue())

				By("Changing one of the policy's labels to it won't match to VMI")
				policy.Spec.Selectors.VirtualMachineInstanceSelector[fmt.Sprintf(labelKeyFmt, policy.Name)] = "XYZ"
				policyList := kubecli.NewMinimalMigrationPolicyList(*policy)

				matchedPolicy := MatchPolicy(policyList, vmi, &namespace)
				Expect(matchedPolicy).To(BeNil())
			})

			It("when no policies exist, MatchPolicy() should return nil", func() {
				policyList := kubecli.NewMinimalMigrationPolicyList()
				matchedPolicy := MatchPolicy(policyList, vmi, &namespace)
				Expect(matchedPolicy).To(BeNil())
			})

			It("VMI labels should have precedence over namespace labels", func() {
				numberOfLabels := rand.Intn(5) + 1

				By(fmt.Sprintf("Defining two policies with %d labels, one with VMI labels and one with NS labels", numberOfLabels))
				policyWithNSLabels := preparePolicyAndVMIWithNSAndVMILabels(vmi, &namespace, 0, numberOfLabels)
				policyWithVmiLabels := preparePolicyAndVMIWithNSAndVMILabels(vmi, &namespace, numberOfLabels, 0)

				policyList := kubecli.NewMinimalMigrationPolicyList(*policyWithNSLabels, *policyWithVmiLabels)

				By("Expecting VMI labels policy to be matched")
				matchedPolicy := MatchPolicy(policyList, vmi, &namespace)
				Expect(matchedPolicy.Name).To(Equal(policyWithVmiLabels.Name), "policy with VMI labels should match")
			})
		})

		DescribeTable("should override cluster-wide migration configurations when", func(defineMigrationPolicy func(*migrationsv1.MigrationPolicySpec), testMigrationConfigs func(configuration *virtv1.MigrationConfiguration), expectConfigUpdate bool) {
			By("Initialize VMI and migration")
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduled)

			targetPod = newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodRunning)
			targetPod.Spec.NodeName = "node01"
			targetPod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}

			By("Defining migration policy, matching it to vmi to posting it into the cluster")
			migrationPolicy := generatePolicyAndAlignVMI(vmi)
			defineMigrationPolicy(&migrationPolicy.Spec)

			addMigrationPolicies(*migrationPolicy)
			addMigration(migration)
			addPod(targetPod)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			By("Calculating new migration config and validating it")
			expectedConfigs := getDefaultMigrationConfiguration()
			isConfigUpdated, err := migrationPolicy.GetMigrationConfByPolicy(expectedConfigs)
			Expect(err).ToNot(HaveOccurred())
			Expect(isConfigUpdated).To(Equal(expectConfigUpdate))
			testMigrationConfigs(expectedConfigs)

			By("Running the controller")
			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulHandOverPodReason)
			fields := Fields{
				"TargetNode":          Equal("node01"),
				"TargetPod":           Equal(targetPod.Name),
				"SourceNode":          Equal("tefwegwrerg"),
				"MigrationUID":        Equal(types.UID("testmigration")),
				"MigrationPolicyName": BeNil(),
			}
			if expectConfigUpdate {
				fields["MigrationPolicyName"] = Equal(pointer.P(migrationPolicy.Name))
			}
			expectVirtualMachineInstanceMigrationState(vmi.Namespace, vmi.Name, PointTo(MatchFields(IgnoreExtras, fields)))
			expectVirtualMachineInstanceMigrationConfiguration(vmi.Namespace, vmi.Name, getMigrationConfig(expectedConfigs))
			expectVirtualMachineInstanceLabels(vmi.Namespace, vmi.Name,
				HaveKeyWithValue(virtv1.MigrationTargetNodeNameLabel, "node01"),
				HaveKeyWithValue(fmt.Sprintf("%s-key-0", migrationPolicy.Name), fmt.Sprintf("%s-value-0", migrationPolicy.Name)),
			)
		},
			Entry("allow auto coverage",
				func(p *migrationsv1.MigrationPolicySpec) { p.AllowAutoConverge = pointer.P(true) },
				func(c *virtv1.MigrationConfiguration) {
					Expect(c.AllowAutoConverge).ToNot(BeNil())
					Expect(*c.AllowAutoConverge).To(BeTrue())
				},
				true,
			),
			Entry("deny auto coverage",
				func(p *migrationsv1.MigrationPolicySpec) { p.AllowAutoConverge = pointer.P(false) },
				func(c *virtv1.MigrationConfiguration) {
					Expect(c.AllowAutoConverge).ToNot(BeNil())
					Expect(*c.AllowAutoConverge).To(BeFalse())
				},
				true,
			),
			Entry("set bandwidth per migration",
				func(p *migrationsv1.MigrationPolicySpec) { p.BandwidthPerMigration = &stubResourceQuantity },
				func(c *virtv1.MigrationConfiguration) {
					Expect(c.BandwidthPerMigration).ToNot(BeNil())
					Expect(c.BandwidthPerMigration.Equal(stubResourceQuantity)).To(BeTrue())
				},
				true,
			),
			Entry("set completion time per GiB",
				func(p *migrationsv1.MigrationPolicySpec) { p.CompletionTimeoutPerGiB = &stubNumber },
				func(c *virtv1.MigrationConfiguration) {
					Expect(c.CompletionTimeoutPerGiB).ToNot(BeNil())
					Expect(*c.CompletionTimeoutPerGiB).To(Equal(stubNumber))
				},
				true,
			),
			Entry("deny post copy",
				func(p *migrationsv1.MigrationPolicySpec) { p.AllowPostCopy = pointer.P(false) },
				func(c *virtv1.MigrationConfiguration) {
					Expect(c.AllowPostCopy).ToNot(BeNil())
					Expect(*c.AllowPostCopy).To(BeFalse())
				},
				true,
			),
			Entry("nothing is changed",
				func(p *migrationsv1.MigrationPolicySpec) {},
				func(c *virtv1.MigrationConfiguration) {},
				false,
			),
		)
	})

	Context("Migration of host-model VMI", func() {
		It("should trigger alert when no node supports host-model", func() {
			const nodeName = "testNode"

			By("Defining node (that does not support host model)")
			node := newNode(nodeName)

			By("Defining VMI")
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			addNodeNameToVMI(vmi, nodeName)
			vmi.Spec.Domain.CPU = &virtv1.CPU{Model: virtv1.CPUModeHostModel}

			By("Defining migration")
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationScheduling)
			migration.Annotations[virtv1.MigrationUnschedulablePodTimeoutSecondsAnnotation] = "1"

			By("Defining target pod")
			targetPod := newTargetPodForVirtualMachine(vmi, migration, k8sv1.PodPending)
			if targetPod.Spec.NodeSelector == nil {
				targetPod.Spec.NodeSelector = make(map[string]string)
			}
			targetPod.Spec.NodeSelector[virtv1.HostModelCPULabel+"fake-model"] = "true"
			if node.Labels == nil {
				node.Labels = make(map[string]string)
			}
			node.Labels[virtv1.HostModelCPULabel+"other-fake-model"] = "true"
			targetPod.CreationTimestamp = metav1.NewTime(pointer.P(metav1.Now()).Time.Add(time.Duration(-defaultUnschedulablePendingTimeoutSeconds) * time.Second))
			targetPod.Status.Conditions = append(targetPod.Status.Conditions, k8sv1.PodCondition{
				Type:   k8sv1.PodScheduled,
				Status: k8sv1.ConditionFalse,
				Reason: k8sv1.PodReasonUnschedulable,
			})

			By("Adding objects to mocked cluster")
			addNode(node)
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addPod(targetPod)

			controller.Execute()

			testutils.ExpectEvent(recorder, virtcontroller.NoSuitableNodesForHostModelMigration)
			testutils.ExpectEvent(recorder, virtcontroller.MigrationTargetPodUnschedulable)
			testutils.ExpectEvent(recorder, virtcontroller.SuccessfulDeletePodReason)
			expectPodDoesNotExist(vmi.Namespace, string(vmi.UID), string(migration.UID))
		})
	})

	Context("Migration abortion before hand-off to virt-handler", func() {
		var vmi *virtv1.VirtualMachineInstance
		var migration *virtv1.VirtualMachineInstanceMigration

		BeforeEach(func() {
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			migration = newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
			migration.DeletionTimestamp = pointer.P(metav1.Now())

			Expect(controller.isMigrationHandedOff(migration, vmi)).To(BeFalse(), "this test assumes migration was not handed off yet")
			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
		})

		It("expect abort condition", func() {
			controller.Execute()
			testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			expectMigrationCondition(migration.Namespace, migration.Name, virtv1.VirtualMachineInstanceMigrationAbortRequested)
		})

		It("expect failure phase", func() {
			controller.Execute()
			testutils.ExpectEvent(recorder, virtcontroller.FailedMigrationReason)
			expectMigrationFailedState(migration.Namespace, migration.Name)
		})
	})

	Context("Migration backoff", func() {
		var vmi *virtv1.VirtualMachineInstance

		It("should be applied after an evacuation migration fails", func() {
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			failedMigration := newMigration("testmigration", vmi.Name, virtv1.MigrationFailed)
			pendingMigration := newMigration("testmigration2", vmi.Name, virtv1.MigrationPending)
			setEvacuationAnnotation(failedMigration, pendingMigration)

			failedMigration.Status.PhaseTransitionTimestamps = []virtv1.VirtualMachineInstanceMigrationPhaseTransitionTimestamp{
				{
					Phase:                    virtv1.MigrationFailed,
					PhaseTransitionTimestamp: failedMigration.CreationTimestamp,
				},
			}
			pendingMigration.CreationTimestamp = metav1.NewTime(failedMigration.CreationTimestamp.Add(time.Second * 1))

			addMigration(pendingMigration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addMigration(failedMigration)

			controller.Execute()

			testutils.ExpectEvent(recorder, "MigrationBackoff")
			expectMigrationPendingState(pendingMigration.Namespace, pendingMigration.Name)
		})

		It("should not be applied if it is not an evacuation", func() {
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			failedMigration := newMigration("testmigration", vmi.Name, virtv1.MigrationFailed)
			pendingMigration := newMigration("testmigration2", vmi.Name, virtv1.MigrationPending)
			setEvacuationAnnotation(failedMigration)

			failedMigration.Status.PhaseTransitionTimestamps = []virtv1.VirtualMachineInstanceMigrationPhaseTransitionTimestamp{
				{
					Phase:                    virtv1.MigrationFailed,
					PhaseTransitionTimestamp: failedMigration.CreationTimestamp,
				},
			}
			pendingMigration.CreationTimestamp = metav1.NewTime(failedMigration.CreationTimestamp.Add(time.Second * 1))

			addMigration(pendingMigration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addMigration(failedMigration)

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, pendingMigration.UID, 1, 0, 0)
		})

		It("should be cleared when a migration succeeds", func() {
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			failedMigration := newMigration("testmigration", vmi.Name, virtv1.MigrationFailed)
			successfulMigration := newMigration("testmigration2", vmi.Name, virtv1.MigrationSucceeded)
			pendingMigration := newMigration("testmigration3", vmi.Name, virtv1.MigrationPending)
			setEvacuationAnnotation(failedMigration, pendingMigration, successfulMigration)

			failedMigration.Status.PhaseTransitionTimestamps = []virtv1.VirtualMachineInstanceMigrationPhaseTransitionTimestamp{
				{
					Phase:                    virtv1.MigrationFailed,
					PhaseTransitionTimestamp: failedMigration.CreationTimestamp,
				},
			}
			successfulMigration.CreationTimestamp = metav1.NewTime(failedMigration.CreationTimestamp.Add(time.Second * 1))
			pendingMigration.CreationTimestamp = metav1.NewTime(successfulMigration.CreationTimestamp.Add(time.Second * 1))

			addMigration(pendingMigration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))
			addMigration(failedMigration)
			addMigration(successfulMigration)

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, pendingMigration.UID, 1, 0, 0)
		})
	})

	Context("Descheduler annotations", func() {
		var vmi *virtv1.VirtualMachineInstance

		It("should not add eviction-in-progress annotation in case of non evacuation migration", func() {
			vmi = newVirtualMachine("testvmi", virtv1.Running)
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			sourcePod := newSourcePodForVirtualMachine(vmi)
			addPod(sourcePod)

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
			updatedSourcePod, err := kubeClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), sourcePod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedSourcePod.Annotations).ToNot(HaveKey(descheduler.EvictionInProgressAnnotation))
		})

		Context("with an evacuation migration", func() {
			It("should add eviction-in-progress annotation only to source virt-launcher pod", func() {
				vmi = newVirtualMachine("testvmi", virtv1.Running)
				migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)
				setEvacuationAnnotation(migration)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				sourcePod := newSourcePodForVirtualMachine(vmi)
				addPod(sourcePod)

				controller.Execute()

				testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
				expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
				updatedSourcePod, err := kubeClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), sourcePod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedSourcePod.Annotations).To(HaveKeyWithValue(descheduler.EvictionInProgressAnnotation, "kubevirt"))

				pods, err := kubeClient.CoreV1().Pods(migration.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s,%s=%s", virtv1.MigrationJobLabel, string(migration.UID), virtv1.CreatedByLabel, string(vmi.UID)),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(pods.Items).To(HaveLen(1))
				targetPod := pods.Items[0]
				Expect(targetPod.Annotations).ToNot(HaveKey(descheduler.EvictionInProgressAnnotation))
			})

			It("should remove eviction-in-progress annotation from source virt-launcher pod in case of failure", func() {
				By("Create a pending migration")
				vmi = newVirtualMachine("testvmi", virtv1.Running)
				sourcePod := newSourcePodForVirtualMachine(vmi)
				sourcePod.Annotations[descheduler.EvictionInProgressAnnotation] = "kubevirt"
				migration := newMigration("testmigration", vmi.Name, virtv1.MigrationFailed)
				migration.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					SourcePod: sourcePod.Name,
				}
				setEvacuationAnnotation(migration)

				addMigration(migration)
				addVirtualMachineInstance(vmi)
				addPod(sourcePod)

				controller.Execute()

				updatedSourcePod, err := kubeClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), sourcePod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedSourcePod.Annotations).ToNot(HaveKeyWithValue(descheduler.EvictionInProgressAnnotation, ""))
			})
		})
	})

	Context("Migration target SELinux level", func() {
		expectTargetPodWithSELinuxLevel := func(namespace string, uid types.UID, migrationUid types.UID, level string) {
			pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", virtv1.MigrationJobLabel, string(migrationUid), virtv1.CreatedByLabel, string(uid)),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))
			Expect(pods.Items[0].Spec.Affinity).ToNot(BeNil())

			if level != "" {
				Expect(pods.Items[0].Spec.SecurityContext.SELinuxOptions.Level).To(Equal(level))
			} else {
				if pods.Items[0].Spec.SecurityContext != nil && pods.Items[0].Spec.SecurityContext.SELinuxOptions != nil {
					Expect(pods.Items[0].Spec.SecurityContext.SELinuxOptions.Level).To(BeEmpty())
				}
			}
		}

		It("should be forced to the SELinux level of the source by default", func() {
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
			expectTargetPodWithSELinuxLevel(vmi.Namespace, vmi.UID, migration.UID, "s0:c1,c2")
		})

		It("should not be forced to the SELinux level of the source if the CR option is set to false", func() {
			initController(&virtv1.KubeVirtConfiguration{
				MigrationConfiguration: &virtv1.MigrationConfiguration{
					MatchSELinuxLevelOnMigration: pointer.P(false),
				},
			})
			vmi := newVirtualMachine("testvmi", virtv1.Running)
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			migration := newMigration("testmigration", vmi.Name, virtv1.MigrationPending)

			addMigration(migration)
			addVirtualMachineInstance(vmi)
			addPod(newSourcePodForVirtualMachine(vmi))

			controller.Execute()

			testutils.ExpectEvents(recorder, virtcontroller.SuccessfulCreatePodReason)
			expectPodCreation(vmi.Namespace, vmi.UID, migration.UID, 1, 0, 0)
			expectTargetPodWithSELinuxLevel(vmi.Namespace, vmi.UID, migration.UID, "")
		})
	})
})

func newPDB(name string, vmi *virtv1.VirtualMachineInstance, pods int) *policyv1.PodDisruptionBudget {
	minAvailable := intstr.FromInt(pods)

	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind),
			},
			Name:      name,
			Namespace: vmi.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					virtv1.CreatedByLabel: string(vmi.UID),
				},
			},
		},
	}
}

func newMigration(name string, vmiName string, phase virtv1.VirtualMachineInstanceMigrationPhase) *virtv1.VirtualMachineInstanceMigration {

	migration := &virtv1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k8sv1.NamespaceDefault,
			Annotations: map[string]string{
				virtv1.ControllerAPILatestVersionObservedAnnotation:  virtv1.ApiLatestVersion,
				virtv1.ControllerAPIStorageVersionObservedAnnotation: virtv1.ApiStorageVersion,
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: virtv1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
	migration.TypeMeta = metav1.TypeMeta{
		APIVersion: virtv1.GroupVersion.String(),
		Kind:       "VirtualMachineInstanceMigration",
	}
	migration.UID = types.UID(name)
	migration.Status.Phase = phase
	return migration
}

func newVirtualMachine(name string, phase virtv1.VirtualMachineInstancePhase) *virtv1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI(name)
	vmi.UID = types.UID(name)
	vmi.Status.Phase = phase
	vmi.Status.NodeName = "tefwegwrerg"
	vmi.Status.SelinuxContext = "system_u:object_r:container_file_t:s0:c1,c2"
	vmi.ObjectMeta.Labels = make(map[string]string)
	// This would be set by mutation webhook
	vmi.Status.RuntimeUser = 107
	vmi.ObjectMeta.Annotations = map[string]string{
		virtv1.DeprecatedNonRootVMIAnnotation: "true",
	}
	return vmi
}

func addNodeNameToVMI(vmi *virtv1.VirtualMachineInstance, nodeName string) {
	vmi.Status.NodeName = nodeName
	vmi.Labels[virtv1.NodeNameLabel] = nodeName
}

func newVirtualMachineWithHotplugVolume(name string, phase virtv1.VirtualMachineInstancePhase) *virtv1.VirtualMachineInstance {
	vmi := newVirtualMachine(name, phase)
	vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
		{
			HotplugVolume: &virtv1.HotplugVolumeStatus{},
		},
	}
	return vmi
}

func newSourcePodForVirtualMachine(vmi *virtv1.VirtualMachineInstance) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: vmi.Namespace,
			Labels: map[string]string{
				virtv1.AppLabel:       "virt-launcher",
				virtv1.CreatedByLabel: string(vmi.UID),
			},
			Annotations: map[string]string{
				virtv1.DomainAnnotation: vmi.Name,
			},
		},
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "test"},
			},
		},
		Spec: k8sv1.PodSpec{
			NodeName: vmi.Status.NodeName,
			Volumes:  []k8sv1.Volume{},
		},
	}
}

func newTargetPodForVirtualMachine(vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: vmi.Namespace,
			Labels: map[string]string{
				virtv1.AppLabel:          "virt-launcher",
				virtv1.CreatedByLabel:    string(vmi.UID),
				virtv1.MigrationJobLabel: string(migration.UID),
			},
			Annotations: map[string]string{
				virtv1.DomainAnnotation:           vmi.Name,
				virtv1.MigrationJobNameAnnotation: migration.Name,
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "test"},
			},
		},
	}
}

func newAttachmentPodForVirtualMachine(ownerPod *k8sv1.Pod, migration *virtv1.VirtualMachineInstanceMigration, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10),
			Namespace: ownerPod.Namespace,
			UID:       "test-uid",
			Labels: map[string]string{
				virtv1.AppLabel:          "hotplug-disk",
				virtv1.MigrationJobLabel: string(migration.UID),
			},
			Annotations: map[string]string{
				virtv1.MigrationJobNameAnnotation: migration.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   k8sv1.SchemeGroupVersion.Group,
					Version: k8sv1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true, Name: "test"},
			},
		},
	}
}

func newNode(name string) *k8sv1.Node {
	node := &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: virtv1.GroupVersion.String(),
		},
	}

	node.Status.Phase = k8sv1.NodeRunning

	return node
}

func getDefaultMigrationConfiguration() *virtv1.MigrationConfiguration {
	nodeTaintKey := "kubevirt.io/drain"
	parallelOutboundMigrationsPerNode := virtconfig.ParallelOutboundMigrationsPerNodeDefault
	parallelMigrationsPerCluster := virtconfig.ParallelMigrationsPerClusterDefault
	allowAutoConverge := virtconfig.MigrationAllowAutoConverge
	bandwidthPerMigration := resource.MustParse(virtconfig.BandwidthPerMigrationDefault)
	completionTimeoutPerGiB := virtconfig.MigrationCompletionTimeoutPerGiB
	progressTimeout := virtconfig.MigrationProgressTimeout
	unsafeMigrationOverride := virtconfig.DefaultUnsafeMigrationOverride
	allowPostCopy := virtconfig.MigrationAllowPostCopy

	return &virtv1.MigrationConfiguration{
		NodeDrainTaintKey:                 &nodeTaintKey,
		ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNode,
		ParallelMigrationsPerCluster:      &parallelMigrationsPerCluster,
		AllowAutoConverge:                 &allowAutoConverge,
		BandwidthPerMigration:             &bandwidthPerMigration,
		CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
		ProgressTimeout:                   &progressTimeout,
		UnsafeMigrationOverride:           &unsafeMigrationOverride,
		AllowPostCopy:                     &allowPostCopy,
	}
}

func preparePolicyAndVMIWithNSAndVMILabels(vmi *virtv1.VirtualMachineInstance, namespace *k8sv1.Namespace, matchingVmiLabels, matchingNSLabels int) *migrationsv1.MigrationPolicy {
	ExpectWithOffset(1, vmi).ToNot(BeNil())
	if matchingNSLabels > 0 {
		ExpectWithOffset(1, namespace).ToNot(BeNil())
	}

	var policyName string
	policyName = fmt.Sprintf("testpolicy-%s", rand.String(5))
	policy := kubecli.NewMinimalMigrationPolicy(policyName)

	if policy.Labels == nil {
		policy.Labels = map[string]string{}
	}

	var namespaceLabels map[string]string
	if namespace != nil {
		if namespace.Labels == nil {
			namespace.Labels = make(map[string]string)
		}

		namespaceLabels = namespace.Labels
	}

	if vmi.Labels == nil {
		vmi.Labels = make(map[string]string)
	}

	if policy.Spec.Selectors == nil {
		policy.Spec.Selectors = &migrationsv1.Selectors{
			VirtualMachineInstanceSelector: migrationsv1.LabelSelector{},
			NamespaceSelector:              migrationsv1.LabelSelector{},
		}
	} else if policy.Spec.Selectors.VirtualMachineInstanceSelector == nil {
		policy.Spec.Selectors.VirtualMachineInstanceSelector = migrationsv1.LabelSelector{}
	} else if policy.Spec.Selectors.NamespaceSelector == nil {
		policy.Spec.Selectors.NamespaceSelector = migrationsv1.LabelSelector{}
	}

	labelKeyPattern := policyName + "-key-%d"
	labelValuePattern := policyName + "-value-%d"

	applyLabels := func(policyLabels, vmiOrNSLabels map[string]string, labelCount int) {
		for i := 0; i < labelCount; i++ {
			labelKey := fmt.Sprintf(labelKeyPattern, i)
			labelValue := fmt.Sprintf(labelValuePattern, i)

			vmiOrNSLabels[labelKey] = labelValue
			policyLabels[labelKey] = labelValue
		}
	}

	applyLabels(policy.Spec.Selectors.VirtualMachineInstanceSelector, vmi.Labels, matchingVmiLabels)

	if namespace != nil {
		applyLabels(policy.Spec.Selectors.NamespaceSelector, namespaceLabels, matchingNSLabels)
		namespace.Labels = namespaceLabels
	}

	return policy
}

func generatePolicyAndAlignVMI(vmi *virtv1.VirtualMachineInstance) *migrationsv1.MigrationPolicy {
	return preparePolicyAndVMIWithNSAndVMILabels(vmi, nil, 1, 0)
}

func setEvacuationAnnotation(migrations ...*virtv1.VirtualMachineInstanceMigration) {
	for _, m := range migrations {
		if m.Annotations == nil {
			m.Annotations = map[string]string{
				virtv1.EvacuationMigrationAnnotation: m.Name,
			}
		} else {
			m.Annotations[virtv1.EvacuationMigrationAnnotation] = m.Name
		}
	}
}
