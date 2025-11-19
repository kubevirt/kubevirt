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
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
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
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
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
		controller     *MigrationSourceController
		mockQueue      *testutils.MockWorkQueue[string]

		vmiTestUUID types.UID
		podTestUUID types.UID

		sockFile                          string
		stop                              chan struct{}
		wg                                *sync.WaitGroup
		eventChan                         chan watch.Event
		recorder                          *record.FakeRecorder
		migrationSourcePasstRepairHandler *stubSourcePasstRepairHandler
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
		diskutils.MockDefaultOwnershipManager()

		wg = &sync.WaitGroup{}
		stop = make(chan struct{})
		eventChan = make(chan watch.Event, 100)
		shareDir := GinkgoT().TempDir()
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

		virtfakeClient = kubevirtfake.NewSimpleClientset()
		ctrl := gomock.NewController(GinkgoT())
		kv := &v1.KubeVirtConfiguration{
			NetworkConfiguration: &v1.NetworkConfiguration{Binding: map[string]v1.InterfaceBindingPlugin{
				migratableNetworkBindingPlugin: {Migration: &v1.InterfaceBindingMigration{}},
			}},
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.PasstIPStackMigration},
			},
		}
		k8sfakeClient := fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().CoreV1().Return(k8sfakeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
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
		mockIsolationDetector.EXPECT().AdjustResources(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		migrationProxy := migrationproxy.NewMigrationProxyManager(tlsConfig, tlsConfig, config)
		launcherClientManager := &launcherclients.MockLauncherClientManager{
			Initialized: true,
		}
		migrationSourcePasstRepairHandler = &stubSourcePasstRepairHandler{isHandleMigrationSourceCalled: false}

		controller, _ = NewMigrationSourceController(
			recorder,
			virtClient,
			host,
			launcherClientManager,
			vmiInformer,
			domainInformer,
			config,
			mockIsolationDetector,
			migrationProxy,
			"/tmp/%d",
			&netStatStub{},
			migrationSourcePasstRepairHandler,
		)

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
		var events []string
		// Ensure that we add checks for expected events to every test
		for len(recorder.Events) > 0 {
			events = append(events, <-recorder.Events)
		}
		Expect(events).To(BeEmpty(), "unexpected events: %+v", events)
	})

	Context("During upgrades, virt-handler", func() {
		It("should be able to read migrationConfiguration from older controller and start the migration", func() {
			// NOTE! When a new field in the migration configuration is added,
			// it is great to run this test without any change in the MigrationConfiguration below,
			// but adjusting the expected cmdclient.MigrationOptions.
			// This ensures that during upgrades, the migrationConfigurations that
			// are set by the older migration controller can be read by the updated virt-handler, without
			// panic.
			var migrationConfiguration = &v1.MigrationConfiguration{
				BandwidthPerMigration:   pointer.P(resource.MustParse("0Mi")),
				ProgressTimeout:         pointer.P(int64(150)),
				AllowAutoConverge:       pointer.P(false),
				CompletionTimeoutPerGiB: pointer.P(int64(50)),
				UnsafeMigrationOverride: pointer.P(false),
				AllowPostCopy:           pointer.P(true),
			}
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
				MigrationConfiguration:         migrationConfiguration,
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			addVMI(vmi, domain)
			expectedOptions := &cmdclient.MigrationOptions{
				Bandwidth:                resource.MustParse("0Mi"),
				ProgressTimeout:          150,
				CompletionTimeoutPerGiB:  50,
				UnsafeMigration:          false,
				AllowPostCopy:            true,
				AllowWorkloadDisruption:  true,
				AllowAutoConverge:        false,
				ParallelMigrationThreads: nil,
			}
			client.EXPECT().MigrateVirtualMachine(vmi, expectedOptions)
			sanityExecute()
			testutils.ExpectEvent(recorder, VMIMigrating)
		})
	})
	Context("setMigrationProgressStatus", func() {
		newDomainMigrationKubevirtMetadata := func(miguid types.UID, end *metav1.Time, completed, failed bool, mode v1.MigrationMode) *api.Domain {
			d := api.NewMinimalDomainWithUUID("test", "1234")
			d.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:          miguid,
				EndTimestamp: end,
				Failed:       failed,
			}
			return d
		}
		DescribeTable("should leave the VMI untouched", func(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

			vmiCopy := vmi.DeepCopy()
			controller.setMigrationProgressStatus(vmi, domain)
			Expect(vmi).To(Equal(vmiCopy))
		},
			Entry("with empty domain", libvmi.New(), nil),
			Entry("without any migration metadata", libvmi.New(libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{}),
			))), api.NewMinimalDomain("test")),
			Entry("without any migration state", libvmi.New(libvmistatus.WithStatus(libvmistatus.New())),
				newDomainMigrationKubevirtMetadata("1234", nil, false, false, v1.MigrationPreCopy)),
			Entry("when the source of the migration is running on the other node", libvmi.New(libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
				MigrationUID:      "1234",
				SourceNode:        "othernode",
				TargetNodeAddress: host,
				Completed:         false,
			})))), newDomainMigrationKubevirtMetadata("1234", nil, false, false, v1.MigrationPreCopy)),
			Entry("when the migration UID in the metadata doesn't correspond to the one in the status", libvmi.New(libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
				MigrationUID:      "4321",
				SourceNode:        host,
				TargetNodeAddress: host,
				Completed:         false,
			})))), newDomainMigrationKubevirtMetadata("1234", nil, false, false, v1.MigrationPreCopy)),
			Entry("when the migration is marked as completed in the VMI", libvmi.New(libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
				MigrationUID:      "1234",
				SourceNode:        host,
				TargetNodeAddress: "othernode",
				Completed:         true,
			})))), newDomainMigrationKubevirtMetadata("1234", nil, true, false, v1.MigrationPreCopy)),
		)

		It("should set the vmi migration state to the same state as the metadata", func() {
			d := newDomainMigrationKubevirtMetadata("1234", pointer.P(metav1.NewTime(time.Now())),
				true, false, v1.MigrationPreCopy)
			vmi := libvmi.New(libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
					MigrationUID:      "1234",
					SourceNode:        host,
					TargetNodeAddress: "othernode",
					Completed:         false,
				}))))
			controller.setMigrationProgressStatus(vmi, d)
			Expect(vmi.Status.MigrationState.StartTimestamp).To(Equal(d.Spec.Metadata.KubeVirt.Migration.StartTimestamp))
			Expect(vmi.Status.MigrationState.Failed).To(Equal(d.Spec.Metadata.KubeVirt.Migration.Failed))
			Expect(vmi.Status.MigrationState.AbortStatus).To(Equal(v1.MigrationAbortStatus(
				d.Spec.Metadata.KubeVirt.Migration.AbortStatus)))
		})

		It("should send an event if the migration failed", func() {
			d := newDomainMigrationKubevirtMetadata("1234", pointer.P(metav1.NewTime(time.Now())),
				true, true, v1.MigrationPreCopy)
			vmi := libvmi.New(libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
					MigrationUID:      "1234",
					SourceNode:        host,
					TargetNodeAddress: "othernode",
					Completed:         false,
				}), libvmistatus.WithNodeName(host)),
			))
			d.Spec.Metadata.KubeVirt.Migration.FailureReason = "some failure happened"

			controller.setMigrationProgressStatus(vmi, d)

			Expect(vmi.Status.MigrationState.StartTimestamp).To(Equal(d.Spec.Metadata.KubeVirt.Migration.StartTimestamp))
			Expect(vmi.Status.MigrationState.EndTimestamp).To(Equal(d.Spec.Metadata.KubeVirt.Migration.EndTimestamp))
			Expect(vmi.Status.MigrationState.Failed).To(Equal(d.Spec.Metadata.KubeVirt.Migration.Failed))
			Expect(vmi.Status.MigrationState.AbortStatus).To(Equal(v1.MigrationAbortStatus(
				d.Spec.Metadata.KubeVirt.Migration.AbortStatus)))
			Expect(vmi.Status.MigrationState.FailureReason).To(Equal(d.Spec.Metadata.KubeVirt.Migration.FailureReason))
			testutils.ExpectEvent(recorder, v1.Migrated.String())
		})
	})

	Context("handleMigrationAbort", func() {
		DescribeTable("should abort the migration with an abort request", func(vmi *v1.VirtualMachineInstance) {
			client.EXPECT().CancelVirtualMachineMigration(vmi)
			Expect(controller.handleMigrationAbort(vmi, client)).To(Succeed())
			testutils.ExpectEvent(recorder, VMIAbortingMigration)
		},
			Entry("when the request failed", libvmi.New(libvmi.WithUID(vmiTestUUID),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						AbortRequested: true,
						AbortStatus:    v1.MigrationAbortFailed,
					})))),
			),
			Entry("when the request the abort status isn't set", libvmi.New(libvmi.WithUID(vmiTestUUID),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						AbortRequested: true,
					})))),
			),
		)
		DescribeTable("should do nothing", func(vmi *v1.VirtualMachineInstance) {

			Expect(controller.handleMigrationAbort(vmi, client)).To(Succeed())
		},
			Entry("when the request succeeded", libvmi.New(libvmi.WithUID(vmiTestUUID),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						AbortRequested: true,
						AbortStatus:    v1.MigrationAbortInProgress,
					})))),
			),
			Entry("when the request is in progress", libvmi.New(libvmi.WithUID(vmiTestUUID),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						AbortRequested: true,
						AbortStatus:    v1.MigrationAbortSucceeded,
					})))),
			),
		)

		It("should return an error if the migration cancellation failed", func() {
			const errMsg = "some error"
			vmi := libvmi.New(libvmi.WithUID(vmiTestUUID),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithMigrationState(v1.VirtualMachineInstanceMigrationState{
						AbortRequested: true,
					}))))
			client.EXPECT().CancelVirtualMachineMigration(vmi).Return(fmt.Errorf(errMsg))
			Expect(controller.handleMigrationAbort(vmi, client)).To(MatchError(errMsg))
		})

		It("should abort vmi migration vmi when migration object indicates deletion", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
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
				TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &now,
			}

			addVMI(vmi, domain)

			client.EXPECT().CancelVirtualMachineMigration(vmi)
			sanityExecute()
			testutils.ExpectEvent(recorder, VMIAbortingMigration)
		})
	})

	Context("Migration options", func() {
		Context("multi-threaded qemu migrations", func() {

			var (
				vmi    *v1.VirtualMachineInstance
				domain *api.Domain
			)

			BeforeEach(func() {
				vmi = api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Status.NodeName = host
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					TargetNode:                     "othernode",
					TargetNodeAddress:              "127.0.0.1:12345",
					SourceNode:                     host,
					MigrationUID:                   "123",
					TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
				}
				vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
					{
						Type:   v1.VirtualMachineInstanceIsMigratable,
						Status: k8sv1.ConditionTrue,
					},
				}
				vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
				vmi = addActivePods(vmi, podTestUUID, host)

				domain = api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				addVMI(vmi, domain)
			})

			DescribeTable("should configure 8 threads when CPU is not limited", func(cpuQuantity *resource.Quantity) {
				if cpuQuantity == nil {
					vmi.Spec.Domain.Resources.Limits = nil
				} else {
					vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = *cpuQuantity
				}

				client.EXPECT().MigrateVirtualMachine(gomock.Any(), gomock.Any()).Do(func(_ *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) {
					Expect(options).ToNot(BeNil())
					Expect(options.ParallelMigrationThreads).ToNot(BeNil())
					Expect(*options.ParallelMigrationThreads).To(Equal(parallelMultifdMigrationThreads))
				}).Times(1).Return(nil)

				controller.Execute()
				testutils.ExpectEvent(recorder, VMIMigrating)
			},
				Entry("with a nil CPU quantity", nil),
				Entry("with a zero CPU quantity", pointer.P(resource.MustParse("0"))),
			)

			DescribeTable("should not configure multiple threads", func(allowPostcopy bool, vmiLimits k8sv1.ResourceList) {
				var migrationConfiguration = &v1.MigrationConfiguration{
					BandwidthPerMigration:   pointer.P(resource.MustParse("0Mi")),
					ProgressTimeout:         pointer.P(int64(150)),
					AllowAutoConverge:       pointer.P(false),
					CompletionTimeoutPerGiB: pointer.P(int64(50)),
					UnsafeMigrationOverride: pointer.P(false),
					AllowPostCopy:           pointer.P(allowPostcopy),
					AllowWorkloadDisruption: pointer.P(true),
				}
				vmi.Status.MigrationState.MigrationConfiguration = migrationConfiguration
				vmi.Spec.Domain.Resources.Limits = vmiLimits

				client.EXPECT().MigrateVirtualMachine(gomock.Any(), gomock.Any()).Do(func(_ *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) {
					Expect(options.ParallelMigrationThreads).To(BeNil())
				}).Times(1).Return(nil)

				controller.Execute()
				testutils.ExpectEvent(recorder, VMIMigrating)
			},
				Entry("if CPU is limited", false, k8sv1.ResourceList{k8sv1.ResourceCPU: resource.MustParse("4")}),
				Entry("if post-copy is enabled", true, k8sv1.ResourceList{}),
			)
		})
	})

	It("should migrate vmi once target address is known", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = host
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
		vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:                     "othernode",
			TargetNodeAddress:              "127.0.0.1:12345",
			SourceNode:                     host,
			MigrationUID:                   "123",
			TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
		}
		vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
			{
				Type:   v1.VirtualMachineInstanceIsMigratable,
				Status: k8sv1.ConditionTrue,
			},
		}
		vmi = addActivePods(vmi, podTestUUID, host)

		domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
		domain.Status.Status = api.Running
		const testIfaceName = "eth0"
		domain.Status.Interfaces = []api.InterfaceStatus{
			{InterfaceName: testIfaceName},
		}
		addVMI(vmi, domain)
		options := &cmdclient.MigrationOptions{
			Bandwidth:                resource.MustParse("0Mi"),
			ProgressTimeout:          virtconfig.MigrationProgressTimeout,
			CompletionTimeoutPerGiB:  virtconfig.MigrationCompletionTimeoutPerGiB,
			UnsafeMigration:          virtconfig.DefaultUnsafeMigrationOverride,
			AllowPostCopy:            virtconfig.MigrationAllowPostCopy,
			ParallelMigrationThreads: pointer.P(parallelMultifdMigrationThreads),
		}
		client.EXPECT().MigrateVirtualMachine(vmi, options)
		sanityExecute()
		testutils.ExpectEvent(recorder, VMIMigrating)
		Expect(migrationSourcePasstRepairHandler.isHandleMigrationSourceCalled).Should(BeTrue())
		updatedVMI, err := virtfakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedVMI.Status.Interfaces[0].InterfaceName).To(Equal(testIfaceName))
	})
})

type stubSourcePasstRepairHandler struct {
	isHandleMigrationSourceCalled bool
}

func (s *stubSourcePasstRepairHandler) HandleMigrationSource(*v1.VirtualMachineInstance, func(*v1.VirtualMachineInstance) (string, error)) error {
	s.isHandleMigrationSourceCalled = true
	return nil
}
