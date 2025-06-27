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
 * Copyright The KubeVirt Authors.
 *
 */

package virthandler

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/certificates"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	hotplugvolume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	launcherclients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VirtualMachineInstance migration target", func() {
	var (
		client         *cmdclient.MockLauncherClient
		virtClient     *kubecli.MockKubevirtClient
		virtfakeClient *kubevirtfake.Clientset
		controller     *MigrationTargetController
		mockQueue      *testutils.MockWorkQueue[string]

		vmiTestUUID types.UID
		podTestUUID types.UID

		sockFile                 string
		stop                     chan struct{}
		wg                       *sync.WaitGroup
		eventChan                chan watch.Event
		recorder                 *record.FakeRecorder
		mockHotplugVolumeMounter *hotplugvolume.MockVolumeMounter

		networkBindingPluginMemoryCalculator *stubNetBindingPluginMemoryCalculator
		migrationTargetPasstRepairHandler    *stubTargetPasstRepairHandler
	)

	const (
		host                           = "master"
		migratableNetworkBindingPlugin = "mig_plug"
	)

	addDomain := func(domain *api.Domain) {
		Expect(controller.domainStore.Add(domain)).To(Succeed())
		key, err := virtcontroller.KeyFunc(domain)
		Expect(err).ToNot(HaveOccurred())
		controller.queue.Add(key)
	}

	createVMI := func(vmi *v1.VirtualMachineInstance) {
		Expect(controller.vmiStore.Add(vmi)).To(Succeed())
		_, err := virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		key, err := virtcontroller.KeyFunc(vmi)
		Expect(err).ToNot(HaveOccurred())
		controller.queue.Add(key)
	}

	addVMI := func(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
		addDomain(domain)
		createVMI(vmi)
	}

	sanityExecute := func() {
		controllertesting.SanityExecute(controller, []cache.Store{
			controller.domainStore, controller.vmiStore,
		}, Default)
	}

	BeforeEach(func() {
		networkBindingPluginMemoryCalculator = &stubNetBindingPluginMemoryCalculator{}
		diskutils.MockDefaultOwnershipManager()

		wg = &sync.WaitGroup{}
		stop = make(chan struct{})
		eventChan = make(chan watch.Event, 100)
		shareDir := GinkgoT().TempDir()
		privateDir := GinkgoT().TempDir()
		podsDir, err := os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, podsDir)
		certDir := GinkgoT().TempDir()

		vmiShareDir := GinkgoT().TempDir()
		ghostCacheDir := GinkgoT().TempDir()

		_ = virtcache.InitializeGhostRecordCache(virtcache.NewIterableCheckpointManager(ghostCacheDir))

		Expect(os.MkdirAll(filepath.Join(vmiShareDir, "var", "run", "kubevirt"), 0755)).To(Succeed())

		cmdclient.SetPodsBaseDir(podsDir)

		store, err := certificates.GenerateSelfSignedCert(certDir, "test", "test")
		Expect(err).ToNot(HaveOccurred())

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
				return store.Current()
			},
		}

		vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		domainInformer, _ := testutils.NewFakeInformerFor(&api.Domain{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		k8sfakeClient := fake.NewSimpleClientset()
		virtfakeClient = kubevirtfake.NewSimpleClientset()
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().CoreV1().Return(k8sfakeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		kv := &v1.KubeVirtConfiguration{
			NetworkConfiguration: &v1.NetworkConfiguration{Binding: map[string]v1.InterfaceBindingPlugin{
				migratableNetworkBindingPlugin: {Migration: &v1.InterfaceBindingMigration{}},
			}},
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.PasstIPStackMigration},
			},
		}

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kv)

		Expect(os.MkdirAll(filepath.Join(vmiShareDir, "dev"), 0755)).To(Succeed())
		f, err := os.OpenFile(filepath.Join(vmiShareDir, "dev", "kvm"), os.O_CREATE, 0755)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Close()).To(Succeed())

		mockIsolationResult := isolation.NewMockIsolationResult(ctrl)
		mockIsolationResult.EXPECT().Pid().Return(1).AnyTimes()
		rootDir, err := safepath.JoinAndResolveWithRelativeRoot(vmiShareDir)
		Expect(err).ToNot(HaveOccurred())
		mockIsolationResult.EXPECT().MountRoot().Return(rootDir, nil).AnyTimes()

		mockIsolationDetector := isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationDetector.EXPECT().Detect(gomock.Any()).Return(mockIsolationResult, nil).AnyTimes()
		mockIsolationDetector.EXPECT().AdjustResources(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		mockHotplugVolumeMounter = hotplugvolume.NewMockVolumeMounter(ctrl)

		migrationProxy := migrationproxy.NewMigrationProxyManager(tlsConfig, tlsConfig, config)
		launcherClientManager := &launcherclients.MockLauncherClientManager{
			Initialized: true,
		}
		migrationTargetPasstRepairHandler = &stubTargetPasstRepairHandler{isHandleMigrationTargetCalled: false}

		controller, _ = NewMigrationTargetController(
			recorder,
			virtClient,
			host,
			privateDir,
			podsDir,
			"127.1.1.1", // migration ip address
			launcherClientManager,
			vmiInformer,
			domainInformer,
			config,
			mockIsolationDetector,
			migrationProxy,
			"/tmp/%d",
			nil, // capabilities
			&netConfStub{},
			&netStatStub{},
			networkBindingPluginMemoryCalculator,
			migrationTargetPasstRepairHandler,
		)

		controller.hotplugVolumeMounter = mockHotplugVolumeMounter

		vmiTestUUID = uuid.NewUUID()
		podTestUUID = uuid.NewUUID()
		sockFile = cmdclient.SocketFilePathOnHost(string(podTestUUID))
		Expect(os.MkdirAll(filepath.Dir(sockFile), 0755)).To(Succeed())
		f, err = os.Create(sockFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Close()).To(Succeed())

		mockQueue = testutils.NewMockWorkQueue(controller.queue)
		controller.queue = mockQueue

		wg.Add(1)

		go func() {
			err = notifyserver.RunServer(shareDir, stop, eventChan, nil, nil)
			wg.Done()
			Expect(err).ToNot(HaveOccurred())
		}()
		client = cmdclient.NewMockLauncherClient(ctrl)
		clientInfo := &virtcache.LauncherClientInfo{
			Client:             client,
			SocketFile:         sockFile,
			DomainPipeStopChan: make(chan struct{}),
			Ready:              true,
		}
		launcherClientManager.Client = client
		launcherClientManager.ClientInfo = clientInfo
	})

	AfterEach(func() {
		close(stop)
		wg.Wait()
	})

	It("should prepare migration target", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = "othernode"
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:   host,
			SourceNode:   "othernode",
			MigrationUID: "123",
		}
		vmi = addActivePods(vmi, podTestUUID, host)

		// something has to be listening to the cmd socket
		// for the proxy to work.
		Expect(os.MkdirAll(cmdclient.SocketDirectoryOnHost(string(podTestUUID)), os.ModePerm)).To(Succeed())

		socketFile := cmdclient.SocketFilePathOnHost(string(podTestUUID))
		Expect(os.RemoveAll(socketFile)).To(Succeed())
		socket, err := net.Listen("unix", socketFile)
		Expect(err).NotTo(HaveOccurred())
		defer socket.Close()

		// since a random port is generated, we have to create the proxy
		// here in order to know what port will be in the update.
		err = controller.handleTargetMigrationProxy(vmi)
		Expect(err).NotTo(HaveOccurred())

		destSrcPorts := controller.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
		updatedVmi := vmi.DeepCopy()
		updatedVmi.Status.MigrationState.TargetNodeAddress = controller.migrationIpAddress
		updatedVmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPorts

		client.EXPECT().SyncMigrationTarget(vmi, gomock.Any())
		createVMI(vmi)
		sanityExecute()
		testutils.ExpectEvent(recorder, VMIMigrationTargetPrepared)
		testutils.ExpectEvent(recorder, "Migration Target is listening")
		Expect(migrationTargetPasstRepairHandler.isHandleMigrationTargetCalled).Should(BeTrue())
	})

	It("should abort target prep if VMI is deleted", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = "othernode"
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:   host,
			SourceNode:   "othernode",
			MigrationUID: "123",
		}
		now := metav1.Time{Time: time.Now()}
		vmi.DeletionTimestamp = &now
		vmi = addActivePods(vmi, podTestUUID, host)

		createVMI(vmi)

		// something has to be listening to the cmd socket
		// for the proxy to work.
		Expect(os.MkdirAll(cmdclient.SocketDirectoryOnHost(string(podTestUUID)), os.ModePerm)).To(Succeed())

		socketFile := cmdclient.SocketFilePathOnHost(string(podTestUUID))
		Expect(os.RemoveAll(socketFile)).To(Succeed())
		socket, err := net.Listen("unix", socketFile)
		Expect(err).NotTo(HaveOccurred())
		defer socket.Close()

		// since a random port is generated, we have to create the proxy
		// here in order to know what port will be in the update.
		err = controller.handleTargetMigrationProxy(vmi)
		Expect(err).NotTo(HaveOccurred())

		destSrcPorts := controller.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
		updatedVmi := vmi.DeepCopy()
		updatedVmi.Status.MigrationState.TargetNodeAddress = controller.migrationIpAddress
		updatedVmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPorts

		sanityExecute()
	})

	It("should hotplug memory in post-migration when target pod has the required conditions", func() {
		conditionManager := virtcontroller.NewVirtualMachineInstanceConditionManager()

		initialMemory := resource.MustParse("512Mi")
		requestedMemory := resource.MustParse("1Gi")

		vmi := api2.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			Guest: &requestedMemory,
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = requestedMemory
		vmi.Status.Memory = &v1.MemoryStatus{
			GuestAtBoot:    &initialMemory,
			GuestCurrent:   &initialMemory,
			GuestRequested: &initialMemory,
		}

		targetPodMemory := services.GetMemoryOverhead(vmi, runtime.GOARCH, nil)
		targetPodMemory.Add(requestedMemory)
		vmi.Labels = map[string]string{
			v1.VirtualMachinePodMemoryRequestsLabel: targetPodMemory.String(),
		}

		condition := &v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceMemoryChange,
			Status: k8sv1.ConditionTrue,
		}
		conditionManager.UpdateCondition(vmi, condition)

		client.EXPECT().SyncVirtualMachineMemory(vmi, gomock.Any())

		Expect(controller.hotplugMemory(vmi, client)).To(Succeed())

		Expect(conditionManager.HasCondition(vmi, v1.VirtualMachineInstanceMemoryChange)).To(BeFalse())
		Expect(v1.VirtualMachinePodMemoryRequestsLabel).ToNot(BeKeyOf(vmi.Labels))
		Expect(vmi.Status.Memory.GuestRequested).To(Equal(vmi.Spec.Domain.Memory.Guest))
		Expect(networkBindingPluginMemoryCalculator.calculatedMemoryOverhead).To(BeTrue())
	})

	It("should not hotplug memory if target pod does not have enough memory", func() {
		conditionManager := virtcontroller.NewVirtualMachineInstanceConditionManager()

		initialMemory := resource.MustParse("512Mi")
		requestedMemory := resource.MustParse("1Gi")

		vmi := api2.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			Guest: &requestedMemory,
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = requestedMemory

		vmi.Status.Memory = &v1.MemoryStatus{
			GuestAtBoot:    &initialMemory,
			GuestCurrent:   &initialMemory,
			GuestRequested: &initialMemory,
		}
		vmi.Labels = map[string]string{
			v1.VirtualMachinePodMemoryRequestsLabel: initialMemory.String(),
		}

		condition := &v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceMemoryChange,
			Status: k8sv1.ConditionTrue,
		}
		conditionManager.UpdateCondition(vmi, condition)

		Expect(controller.hotplugMemory(vmi, client)).ToNot(Succeed())

		Expect(conditionManager.HasCondition(vmi, v1.VirtualMachineInstanceMemoryChange)).To(BeFalse())
		Expect(v1.VirtualMachinePodMemoryRequestsLabel).ToNot(BeKeyOf(vmi.Labels))
		Expect(vmi.Status.Memory.GuestRequested).ToNot(Equal(vmi.Spec.Domain.Memory.Guest))
	})

	It("should set HotMemoryChange condition to False if memory hotplug fails", func() {
		conditionManager := virtcontroller.NewVirtualMachineInstanceConditionManager()

		initialMemory := resource.MustParse("512Mi")
		requestedMemory := resource.MustParse("1Gi")

		vmi := api2.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			Guest: &requestedMemory,
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = requestedMemory
		vmi.Status.Memory = &v1.MemoryStatus{
			GuestAtBoot:    &initialMemory,
			GuestCurrent:   &initialMemory,
			GuestRequested: &initialMemory,
		}
		vmi.Spec.Architecture = "amd64"

		targetPodMemory := services.GetMemoryOverhead(vmi, runtime.GOARCH, nil)
		targetPodMemory.Add(requestedMemory)
		vmi.Labels = map[string]string{
			v1.VirtualMachinePodMemoryRequestsLabel: targetPodMemory.String(),
		}

		condition := &v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceMemoryChange,
			Status: k8sv1.ConditionTrue,
		}
		conditionManager.UpdateCondition(vmi, condition)

		client.EXPECT().SyncVirtualMachineMemory(vmi, gomock.Any()).Return(fmt.Errorf("hotplug failure"))

		Expect(controller.hotplugMemory(vmi, client)).ToNot(Succeed())

		Expect(conditionManager.HasConditionWithStatusAndReason(vmi, v1.VirtualMachineInstanceMemoryChange, k8sv1.ConditionFalse, "Memory Hotplug Failed")).To(BeTrue())
		Expect(v1.VirtualMachinePodMemoryRequestsLabel).ToNot(BeKeyOf(vmi.Labels))
		Expect(vmi.Status.Memory.GuestRequested).ToNot(Equal(vmi.Spec.Domain.Memory.Guest))
	})

	It("migration should be acked when successful", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = host
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
		controller.host = "othernode"
		nowTimeStamp := metav1.Now()
		startTimestamp := metav1.NewTime(nowTimeStamp.Add(-1 * time.Minute))
		vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:                     "othernode",
			TargetNodeAddress:              "127.0.0.1:12345",
			SourceNode:                     host,
			MigrationUID:                   "123",
			TargetNodeDomainDetected:       true,
			TargetNodeDomainReadyTimestamp: &nowTimeStamp,
			StartTimestamp:                 &startTimestamp,
		}

		domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
		domain.Status.Status = api.Running
		domain.Status.Reason = api.ReasonMigrated

		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			UID:            "123",
			StartTimestamp: &startTimestamp,
			EndTimestamp:   pointer.P(nowTimeStamp),
		}

		addVMI(vmi, domain)

		sanityExecute()

		testutils.ExpectEvent(recorder, "The VirtualMachineInstance migrated to node")
		updatedVMI, err := virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedVMI.Labels).To(HaveKeyWithValue(v1.NodeNameLabel, "othernode"))
		Expect(updatedVMI.Labels).ToNot(HaveKey(v1.OutdatedLauncherImageLabel))
		Expect(updatedVMI.Status.LauncherContainerImageVersion).To(BeEmpty())
		Expect(updatedVMI.Status.NodeName).To(Equal("othernode"))
		Expect(updatedVMI.Status.EvacuationNodeName).To(BeEmpty())
		Expect(updatedVMI.Status.MigrationState.Completed).To(BeFalse())
		Expect(updatedVMI.Status.MigrationTransport).To(Equal(v1.MigrationTransportUnix))
		Expect(updatedVMI.Status.Interfaces).To(BeEmpty())
	})

	It("should always remove the VirtualMachineInstanceVCPUChange condition even if hotplug CPU has failed", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = host
		vmi.Labels[v1.NodeNameLabel] = host
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host

		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:                     host,
			TargetNodeAddress:              "127.0.0.1:12345",
			SourceNode:                     "othernode",
			MigrationUID:                   "123",
			TargetNodeDomainDetected:       true,
			TargetNodeDomainReadyTimestamp: pointer.P(metav1.Now()),
			StartTimestamp:                 pointer.P(metav1.NewTime(metav1.Now().Add(-1 * time.Minute))),
			EndTimestamp:                   pointer.P(metav1.Now()),
		}

		cpuTopology := &v1.CPU{
			Sockets: 1,
			Cores:   1,
			Threads: 1,
		}
		vmi.Spec.Domain.CPU = cpuTopology
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceVCPUChange,
			Status: k8sv1.ConditionTrue,
		})
		vmi.Status.CurrentCPUTopology = &v1.CPUTopology{
			Cores:   cpuTopology.Cores,
			Sockets: cpuTopology.Sockets,
			Threads: cpuTopology.Threads,
		}

		domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
		domain.Status.Status = api.Running

		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			UID:            "123",
			StartTimestamp: pointer.P(metav1.NewTime(metav1.Now().Add(-1 * time.Minute))),
			EndTimestamp:   pointer.P(metav1.Now()),
		}

		addVMI(vmi, domain)

		client.EXPECT().Ping().AnyTimes()
		client.EXPECT().FinalizeVirtualMachineMigration(gomock.Any(), gomock.Any())
		client.EXPECT().SyncVirtualMachineCPUs(gomock.Any(), gomock.Any()).Return(fmt.Errorf("some error"))

		sanityExecute()

		testutils.ExpectEvent(recorder, "failed to change vCPUs")
		updatedVMI, err := virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedVMI.Status.CurrentCPUTopology).NotTo(BeNil())
		Expect(updatedVMI.Status.Conditions).NotTo(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type": Equal(v1.VirtualMachineInstanceVCPUChange),
		})))
	})

	It("should hotplug CPU in post-migration when target pod has the required conditions", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = host
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
		pastTime := metav1.NewTime(metav1.Now().Add(time.Duration(-10) * time.Second))
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:                     host,
			TargetNodeAddress:              "127.0.0.1:12345",
			SourceNode:                     "othernode",
			MigrationUID:                   "123",
			TargetNodeDomainDetected:       true,
			TargetNodeDomainReadyTimestamp: pointer.P(metav1.Now()),
			StartTimestamp:                 &pastTime,
			EndTimestamp:                   pointer.P(metav1.Now()),
		}
		cpuTopology := &v1.CPU{
			Sockets: 1,
			Cores:   1,
			Threads: 1,
		}

		vmi.Spec.Domain.CPU = cpuTopology

		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceVCPUChange,
			Status: k8sv1.ConditionTrue,
		})

		domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
		domain.Status.Status = api.Running

		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			UID:            "123",
			StartTimestamp: &pastTime,
			EndTimestamp:   pointer.P(metav1.Now()),
		}

		addVMI(vmi, domain)

		client.EXPECT().Ping().AnyTimes()
		client.EXPECT().FinalizeVirtualMachineMigration(gomock.Any(), gomock.Any())
		client.EXPECT().SyncVirtualMachineCPUs(gomock.Any(), gomock.Any())

		sanityExecute()

		updatedVMI, err := virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedVMI.Status.CurrentCPUTopology).NotTo(BeNil())
		Expect(updatedVMI.Status.Conditions).NotTo(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type": Equal(v1.VirtualMachineInstanceVCPUChange),
		})))
	})

	It("migration should be marked as completed after finalization", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = host
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
		vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:                     host,
			TargetNodeAddress:              "127.0.0.1:12345",
			SourceNode:                     "othernode",
			MigrationUID:                   "123",
			TargetNodeDomainDetected:       true,
			TargetNodeDomainReadyTimestamp: pointer.P(metav1.Now()),
			StartTimestamp:                 pointer.P(metav1.NewTime(metav1.Now().Add(-1 * time.Minute))),
			EndTimestamp:                   pointer.P(metav1.Now()),
		}

		domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
		domain.Status.Status = api.Running

		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			UID:            "123",
			StartTimestamp: pointer.P(metav1.NewTime(metav1.Now().Add(-1 * time.Minute))),
			EndTimestamp:   pointer.P(metav1.Now()),
		}

		addVMI(vmi, domain)

		client.EXPECT().Ping().AnyTimes()
		client.EXPECT().FinalizeVirtualMachineMigration(gomock.Any(), gomock.Any()).Return(nil)

		sanityExecute()

		updatedVMI, err := virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedVMI.Status.MigrationState.Completed).To(BeTrue())
	})

	It("should signal target pod to early exit on failed migration", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = "othernode"
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:   host,
			SourceNode:   "othernode",
			MigrationUID: "123",
			Failed:       true,
			EndTimestamp: pointer.P(metav1.Now()),
		}
		vmi = addActivePods(vmi, podTestUUID, host)

		createVMI(vmi)

		client.EXPECT().SignalTargetPodCleanup(vmi)
		sanityExecute()
	})
})

type stubTargetPasstRepairHandler struct {
	isHandleMigrationTargetCalled bool
}

func (s *stubTargetPasstRepairHandler) HandleMigrationTarget(*v1.VirtualMachineInstance, func(*v1.VirtualMachineInstance) (string, error)) error {
	s.isHandleMigrationTargetCalled = true
	return nil
}
