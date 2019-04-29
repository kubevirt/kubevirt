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

package virthandler

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/certificates"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/testutils"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

var _ = Describe("VirtualMachineInstance", func() {
	var client *cmdclient.MockLauncherClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var virtClient *kubecli.MockKubevirtClient

	var ctrl *gomock.Controller
	var controller *VirtualMachineController
	var vmiSource *framework.FakeControllerSource
	var vmiSourceInformer cache.SharedIndexInformer
	var vmiTargetInformer cache.SharedIndexInformer
	var domainSource *framework.FakeControllerSource
	var domainInformer cache.SharedIndexInformer
	var gracefulShutdownInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue
	var mockWatchdog *MockWatchdog
	var mockGracefulShutdown *MockGracefulShutdown

	var vmiFeeder *testutils.VirtualMachineFeeder
	var domainFeeder *testutils.DomainFeeder

	var recorder record.EventRecorder

	var err error
	var shareDir string
	var testUUID types.UID
	var stop chan struct{}

	var host string

	log.Log.SetIOWriter(GinkgoWriter)

	var certDir string

	BeforeEach(func() {
		stop = make(chan struct{})
		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())
		certDir, err = ioutil.TempDir("", "migrationproxytest")
		Expect(err).ToNot(HaveOccurred())

		store, err := certificates.GenerateSelfSignedCert(certDir, "test", "test")

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
				return store.Current()
			},
		}
		Expect(err).ToNot(HaveOccurred())

		host = "master"
		podIpAddress := "10.10.10.10"

		Expect(err).ToNot(HaveOccurred())

		vmiSourceInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		vmiTargetInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		domainInformer, domainSource = testutils.NewFakeInformerFor(&api.Domain{})
		gracefulShutdownInformer, _ = testutils.NewFakeInformerFor(&api.Domain{})
		recorder = record.NewFakeRecorder(100)

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()

		mockWatchdog = &MockWatchdog{shareDir}
		mockGracefulShutdown = &MockGracefulShutdown{shareDir}
		config, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})

		controller = NewController(recorder,
			virtClient,
			host,
			podIpAddress,
			shareDir,
			vmiSourceInformer,
			vmiTargetInformer,
			domainInformer,
			gracefulShutdownInformer,
			1,
			10,
			config,
			tlsConfig,
		)

		testUUID = uuid.NewUUID()
		client = cmdclient.NewMockLauncherClient(ctrl)
		sockFile := cmdclient.SocketFromUID(shareDir, string(testUUID))
		controller.addLauncherClient(client, sockFile)

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
		domainFeeder = testutils.NewDomainFeeder(mockQueue, domainSource)

		go vmiSourceInformer.Run(stop)
		go vmiTargetInformer.Run(stop)
		go domainInformer.Run(stop)
		go gracefulShutdownInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmiSourceInformer.HasSynced, vmiTargetInformer.HasSynced, domainInformer.HasSynced, gracefulShutdownInformer.HasSynced)).To(BeTrue())
	})

	AfterEach(func() {
		close(stop)
		ctrl.Finish()
		os.RemoveAll(shareDir)
		os.RemoveAll(certDir)
	})

	initGracePeriodHelper := func(gracePeriod int64, vmi *v1.VirtualMachineInstance, dom *api.Domain) {
		vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
		dom.Spec.Features = &api.Features{
			ACPI: &api.FeatureEnabled{},
		}
		dom.Spec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{}
		dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = gracePeriod
	}

	Context("VirtualMachineInstance controller gets informed about a Domain change through the Domain controller", func() {

		It("should delete non-running Domains if no cluster wide equivalent and no grace period info exists", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().DeleteDomain(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))
			controller.Execute()
		})

		It("should delete running Domains if no cluster wide equivalent exists and no grace period info exists", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))

			controller.Execute()
		})

		It("should perform cleanup of local ephemeral data if domain and vmi are deleted", func() {
			mockSockFile := cmdclient.SocketFromUID(shareDir, "")
			controller.addLauncherClient(client, mockSockFile)

			mockQueue.Add("default/testvmi")
			client.EXPECT().Close()
			controller.Execute()
		})

		It("should not attempt graceful shutdown of Domain if domain is already down.", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.Status.Phase = v1.Running

			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Crashed

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().DeleteDomain(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)
		It("should attempt graceful shutdown of Domain if trigger file exists.", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.Status.Phase = v1.Running

			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should attempt graceful shutdown of Domain if no cluster wide equivalent exists", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should do nothing if vmi and domain do not match", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			oldVMI := v1.NewMinimalVMI("testvmi")
			oldVMI.UID = "other uuid"
			domain := api.NewMinimalDomainWithUUID("testvmi", "other uuid")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(oldVMI)
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(oldVMI))
			Expect(os.IsNotExist(err)).To(BeFalse())
		}, 3)

		It("should cleanup if vmi and domain do not match and watchdog is expired", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			oldVMI := v1.NewMinimalVMI("testvmi")
			oldVMI.UID = "other uuid"
			domain := api.NewMinimalDomainWithUUID("testvmi", "other uuid")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(oldVMI)
			// the domain is dead because the watchdog is expired
			mockWatchdog.Expire(oldVMI)
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(oldVMI))
			Expect(os.IsNotExist(err)).To(BeTrue())
		}, 3)

		It("should attempt force terminate Domain if grace period expires", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			metav1.Now()
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix()-3, 0)}
			domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now

			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should immediately kill domain with grace period of 0", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID

			initGracePeriodHelper(0, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", testUUID))
			domainFeeder.Add(domain)
			controller.Execute()
		}, 3)

		It("should re-enqueue if the Key is unparseable", func() {
			Expect(mockQueue.Len()).Should(Equal(0))
			mockQueue.Add("a/b/c/d/e")
			controller.Execute()
			Expect(mockQueue.NumRequeues("a/b/c/d/e")).To(Equal(1))
		})

		It("should create the Domain if it sees the first time on a new VirtualMachineInstance", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			client.EXPECT().SyncVirtualMachine(vmi)

			controller.Execute()
		})

		It("should update from Scheduled to Running, if it sees a running Domain", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Phase = v1.Running
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			updatedVMI.Status.MigrationMethod = v1.LiveMigration

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(updatedVMI)

			node := &k8sv1.Node{
				Status: k8sv1.NodeStatus{
					Addresses: []k8sv1.NodeAddress{
						{
							Type:    k8sv1.NodeInternalIP,
							Address: "127.0.0.1",
						},
					},
				},
			}
			fakeClient := fake.NewSimpleClientset(node).CoreV1()
			virtClient.EXPECT().CoreV1().Return(fakeClient).AnyTimes()

			controller.Execute()
		})

		It("should add guest agent condition when sees the channel connected", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			domain.Spec.Devices.Channels = []api.Channel{
				{
					Type: "unix",
					Target: &api.ChannelTarget{
						Name:  "org.qemu.guest_agent.0",
						State: "connected",
					},
				},
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
				{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi)
			vmiInterface.EXPECT().Update(NewVMICondMatcher(*updatedVMI))

			controller.Execute()
		})

		It("should remove guest agent condition when there is no channel connected", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			domain.Spec.Devices.Channels = []api.Channel{
				{
					Type: "unix",
					Target: &api.ChannelTarget{
						Name:  "org.qemu.guest_agent.0",
						State: "disconnected",
					},
				},
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi)
			vmiInterface.EXPECT().Update(NewVMICondMatcher(*updatedVMI))

			controller.Execute()
		})

		It("should move VirtualMachineInstance from Scheduled to Failed if watchdog file is missing", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			controller.Execute()
		})
		It("should move VirtualMachineInstance from Scheduled to Failed if watchdog file is expired", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			time.Sleep(2 * time.Second)
			controller.Execute()
		}, 2)

		It("should move VirtualMachineInstance from Running to Failed if domain does not exist in cache", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			controller.Execute()
		})

		It("should remove an error condition if a synchronization run succeeds", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceSynchronized,
					Status: k8sv1.ConditionFalse,
				},
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			updatedVMI.Status.MigrationMethod = v1.LiveMigration

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			client.EXPECT().SyncVirtualMachine(vmi)
			vmiInterface.EXPECT().Update(updatedVMI)

			controller.Execute()
		})

		table.DescribeTable("should leave the VirtualMachineInstance alone if it is in the final phase", func(phase v1.VirtualMachineInstancePhase) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = phase
			vmiFeeder.Add(vmi)
			controller.Execute()
			// expect no errors and no mock interactions
			Expect(mockQueue.NumRequeues("default/testvmi")).To(Equal(0))
		},
			table.Entry("succeeded", v1.Succeeded),
			table.Entry("failed", v1.Failed),
		)

		It("should leave VirtualMachineInstance phase alone if not the current active node", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Status.NodeName = "othernode"

			// no domain would result in a failure, but the NodeName is not
			// equal to controller.host's node, so we know that this node
			// does not own the vmi right now.

			vmiFeeder.Add(vmi)
			controller.Execute()
		})

		It("should prepare migration target", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
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

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			// something has to be listening to the cmd socket
			// for the proxy to work.
			os.MkdirAll(cmdclient.SocketsDirectory(shareDir), os.ModePerm)
			portsList := []int{0, 49152}
			for _, port := range portsList {
				key := string(vmi.UID)
				if port != 0 {
					key += fmt.Sprintf("-%d", port)
				}
				socketFile := cmdclient.SocketFromUID(shareDir, key)
				socket, err := net.Listen("unix", socketFile)
				Expect(err).NotTo(HaveOccurred())
				defer socket.Close()
			}
			// since a random port is generated, we have to create the proxy
			// here in order to know what port will be in the update.
			err = controller.handleMigrationProxy(vmi)
			Expect(err).NotTo(HaveOccurred())
			err = controller.handlePostSyncMigrationProxy(vmi)
			Expect(err).NotTo(HaveOccurred())

			destSrcPorts := controller.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
			fmt.Println("destSrcPorts: ", destSrcPorts)
			updatedVmi := vmi.DeepCopy()
			updatedVmi.Status.MigrationState.TargetNodeAddress = controller.ipAddress
			updatedVmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPorts

			client.EXPECT().Ping()
			client.EXPECT().SyncMigrationTarget(vmi)
			vmiInterface.EXPECT().Update(updatedVmi)
			controller.Execute()
		}, 3)

		It("should migrate vmi once target address is known", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[int]int{49152: 12132},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)
			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         150,
				CompletionTimeoutPerGiB: 800,
				UnsafeMigration:         false,
			}
			client.EXPECT().MigrateVirtualMachine(vmi, options)
			controller.Execute()
		}, 3)

		It("should abort vmi migration vmi when migration object indicates deletion", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				AbortRequested:                 true,
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[int]int{49152: 12132},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &now,
			}
			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			client.EXPECT().CancelVirtualMachineMigration(vmi)
			vmiInterface.EXPECT().Update(gomock.Any())
			controller.Execute()
		}, 3)

		It("Handoff domain to other node after completed migration", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:               "othernode",
				TargetNodeAddress:        "127.0.0.1:12345",
				SourceNode:               host,
				MigrationUID:             "123",
				TargetNodeDomainDetected: true,
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Shutoff
			domain.Status.Reason = api.ReasonMigrated

			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &now,
				EndTimestamp:   &now,
				Completed:      true,
			}

			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			vmiUpdated := vmi.DeepCopy()
			vmiUpdated.Status.MigrationState.Completed = true
			vmiUpdated.Status.MigrationState.StartTimestamp = &now
			vmiUpdated.Status.MigrationState.EndTimestamp = &now
			vmiUpdated.Status.NodeName = "othernode"
			vmiUpdated.Labels[v1.NodeNameLabel] = "othernode"

			vmiInterface.EXPECT().Update(vmiUpdated)

			controller.Execute()
		}, 3)
	})

	Context("check VMI disks for migration", func() {
		var testBlockPvc *k8sv1.PersistentVolumeClaim

		BeforeEach(func() {
			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			// create a test block pvc
			mode := k8sv1.PersistentVolumeBlock
			testBlockPvc = &k8sv1.PersistentVolumeClaim{
				TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: k8sv1.NamespaceDefault, Name: "testblock"},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					VolumeMode:  &mode,
					AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
				},
			}

		})
		It("should block migrate non-shared disks ", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(BeNil())
		})
		It("should migrate shared disks without blockMigration flag", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
			}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeFalse())
			Expect(err).To(BeNil())
		})
		It("should fail migration for non-shared PVCs", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
			}

			testBlockPvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI with non-shared PVCs")))
		})
		It("should fail migration for non-shared data volume PVCs", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "testblock",
						},
					},
				},
			}

			testBlockPvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI with non-shared PVCs")))
		})
		It("should be allowed to migrate a mix of shared and non-shared disks", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
			}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(BeNil())
		})
		It("should be allowed to migrate a mix of non-shared and shared disks", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
			}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(BeNil())
		})
		It("should be allowed to live-migrate shared HostDisks ", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "myvolume",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_true,
						},
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeFalse())
			Expect(err).To(BeNil())
		})
		It("should not be allowed to live-migrate shared and non-shared HostDisks ", func() {
			_true := true
			_false := false
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_true,
						},
					},
				},
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume31/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_false,
						},
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI with non-shared HostDisk")))
		})
	})
	Context("VirtualMachineInstance controller gets informed about interfaces in a Domain", func() {
		It("should update existing interface with MAC", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			interface_name := "interface_name"

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				v1.VirtualMachineInstanceNetworkInterface{
					IP:   "1.1.1.1",
					MAC:  "C0:01:BE:E7:15:G0:0D",
					Name: interface_name,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			new_MAC := "1C:CE:C0:01:BE:E7"

			domain.Spec.Devices.Interfaces = []api.Interface{
				api.Interface{
					MAC:   &api.MAC{MAC: new_MAC},
					Alias: &api.Alias{Name: interface_name},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(new_MAC))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should update existing interface with IPs", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			interfaceName := "interface_name"
			mac := "C0:01:BE:E7:15:G0:0D"
			ip := "2.2.2.2/24"
			ips := []string{"2.2.2.2/24", "3.3.3.3/16"}

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				v1.VirtualMachineInstanceNetworkInterface{
					IP:   "1.1.1.1",
					MAC:  mac,
					Name: interfaceName,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			domain.Spec.Devices.Interfaces = []api.Interface{
				api.Interface{
					MAC:   &api.MAC{MAC: mac},
					Alias: &api.Alias{Name: interfaceName},
				},
			}
			domain.Status.Interfaces = []api.InterfaceStatus{
				api.InterfaceStatus{
					Name: interfaceName,
					Mac:  mac,
					Ip:   ip,
					IPs:  ips,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(interfaceName))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(mac))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IP).To(Equal(ip))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IPs).To(Equal(ips))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should add new vmi interfaces for new domain interfaces", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			old_interface_name := "old_interface_name"
			old_MAC := "C0:01:BE:E7:15:G0:0D"

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				v1.VirtualMachineInstanceNetworkInterface{
					IP:   "1.1.1.1",
					MAC:  old_MAC,
					Name: old_interface_name,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			new_interface_name := "new_interface_name"
			new_MAC := "1C:CE:C0:01:BE:E7"

			domain.Spec.Devices.Interfaces = []api.Interface{
				api.Interface{
					MAC:   &api.MAC{MAC: old_MAC},
					Alias: &api.Alias{Name: old_interface_name},
				},
				api.Interface{
					MAC:   &api.MAC{MAC: new_MAC},
					Alias: &api.Alias{Name: new_interface_name},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(2))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(old_interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(old_MAC))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[1].Name).To(Equal(new_interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[1].MAC).To(Equal(new_MAC))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should update name on status interfaces with no name", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = testUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			interface_name := "interface_name"

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				v1.VirtualMachineInstanceNetworkInterface{
					IP:  "1.1.1.1",
					MAC: "C0:01:BE:E7:15:G0:0D",
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", testUUID)
			domain.Status.Status = api.Running

			new_MAC := "1C:CE:C0:01:BE:E7"

			domain.Spec.Devices.Interfaces = []api.Interface{
				api.Interface{
					MAC:   &api.MAC{MAC: new_MAC},
					Alias: &api.Alias{Name: interface_name},
				},
			}

			vmi.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "other_name",
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}},
				},
				v1.Network{
					Name:          interface_name,
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(new_MAC))
			}).Return(vmi, nil)

			controller.Execute()
		})
	})
})

type MockGracefulShutdown struct {
	baseDir string
}

func (m *MockGracefulShutdown) TriggerShutdown(vmi *v1.VirtualMachineInstance) {
	Expect(os.MkdirAll(virtlauncher.GracefulShutdownTriggerDir(m.baseDir), os.ModePerm)).To(Succeed())

	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	triggerFile := virtlauncher.GracefulShutdownTriggerFromNamespaceName(m.baseDir, namespace, domain)
	err := virtlauncher.GracefulShutdownTriggerInitiate(triggerFile)
	Expect(err).NotTo(HaveOccurred())
}

type MockWatchdog struct {
	baseDir string
}

func (m *MockWatchdog) CreateFile(vmi *v1.VirtualMachineInstance) {
	Expect(os.MkdirAll(watchdog.WatchdogFileDirectory(m.baseDir), os.ModePerm)).To(Succeed())
	err := watchdog.WatchdogFileUpdate(
		watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name), string(vmi.UID),
	)
	Expect(err).NotTo(HaveOccurred())
}

func (m *MockWatchdog) Expire(vmi *v1.VirtualMachineInstance) {
	watchdogFile := watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name)
	expireDate := time.Now().Add(time.Duration(-1) * time.Minute)
	os.Chtimes(watchdogFile, expireDate, expireDate)
}

func (m *MockWatchdog) File(vmi *v1.VirtualMachineInstance) string {
	return watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name)
}

type vmiCondMatcher struct {
	vmi v1.VirtualMachineInstance
}

func NewVMICondMatcher(vmi v1.VirtualMachineInstance) gomock.Matcher {
	return &vmiCondMatcher{vmi}
}

func (m *vmiCondMatcher) Matches(x interface{}) bool {
	vmi, ok := x.(*v1.VirtualMachineInstance)
	if !ok {
		return false
	}

	if len(m.vmi.Status.Conditions) != len(vmi.Status.Conditions) {
		return false
	}

	for ind, _ := range vmi.Status.Conditions {
		if m.vmi.Status.Conditions[ind].Type != vmi.Status.Conditions[ind].Type {
			return false
		}
	}

	return true
}

func (m *vmiCondMatcher) String() string {
	return "conditions matches on vmis"
}
