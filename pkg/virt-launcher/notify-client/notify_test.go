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

package eventsclient

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"libvirt.org/go/libvirt"

	api2 "kubevirt.io/client-go/api"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("Notify", func() {

	Describe("Domain Events", func() {

		var eventChan chan watch.Event
		var deleteNotificationSent chan watch.Event
		var client *Notifier
		var vmiStore cache.Store
		var recorder *record.FakeRecorder

		var mockLibvirt *testing.Libvirt
		var e *eventCaller

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt = testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()

			stop := make(chan struct{})
			eventChan = make(chan watch.Event, 100)
			deleteNotificationSent = make(chan watch.Event, 100)
			stopped := false
			shareDir, err := os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())
			recorder = record.NewFakeRecorder(10)
			recorder.IncludeObject = true
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()
			e = &eventCaller{}

			go func(ec chan watch.Event, rec *record.FakeRecorder, vs cache.Store) {
				notifyserver.RunServer(shareDir, stop, ec, rec, vs)
			}(eventChan, recorder, vmiStore)
			// mimic pipe
			notifyServer := filepath.Join(shareDir, "domain-notify.sock")
			pipePath := filepath.Join(shareDir, "domain-notify-pipe.sock")
			Expect(os.Symlink(notifyServer, pipePath)).To(Succeed())

			client = NewNotifier(shareDir)

			DeferCleanup(
				func() {
					if stopped == false {
						close(stop)
					}
					client.Close()
					os.RemoveAll(shareDir)
				},
			)
		})

		metadataCache := func() *metadata.Cache { return metadata.NewCache() }

		Context("server", func() {
			DescribeTable("should accept Domain notify events", func(state libvirt.DomainState, event libvirt.DomainEventType, kubevirtState api.LifeCycle, kubeEventType watch.EventType) {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())

				mockLibvirt.DomainEXPECT().GetState().Return(state, -1, nil)
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: event}}, client, deleteNotificationSent, nil, nil, nil, nil, metadataCache(), false)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, ok := event.Object.(*api.Domain)
					newDomain.Spec.XMLName = xml.Name{}
					Expect(ok).To(BeTrue(), "should typecase domain")
					Expect(equality.Semantic.DeepEqual(domain.Spec, newDomain.Spec)).To(BeTrue())
					Expect(event.Type).To(Equal(kubeEventType))
				}
				Expect(timedOut).To(BeFalse(), "should not time out")
			},
				Entry("modified for crashed VMIs", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed, watch.Modified),
				Entry("modified for stopped VMIs with shutoff reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff, watch.Modified),
				Entry("modified for stopped VMIs with stopped reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff, watch.Modified),
				Entry("modified for running VMIs", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running, watch.Modified),
				Entry("added for defined VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff, watch.Added),
			)

			It("should not send watch.Error event on libvirt reconnect", func() {
				By("Setting up StartDomainNotifier with a mock connection")
				ctrl := gomock.NewController(GinkgoT())
				mockConn := testing.NewLibvirt(ctrl)

				var reconnectChan chan bool
				mockConn.ConnectionEXPECT().SetReconnectChan(gomock.Any()).Do(func(ch chan bool) {
					reconnectChan = ch
				})
				mockConn.ConnectionEXPECT().DomainEventLifecycleRegister(gomock.Any()).Return(nil)
				mockConn.ConnectionEXPECT().DomainEventDeviceAddedRegister(gomock.Any()).Return(nil)
				mockConn.ConnectionEXPECT().DomainEventDeviceRemovedRegister(gomock.Any()).Return(nil)
				mockConn.ConnectionEXPECT().DomainEventMemoryDeviceSizeChangeRegister(gomock.Any()).Return(nil)
				mockConn.ConnectionEXPECT().DomainEventJobCompletedRegister(gomock.Any()).Return(nil)
				mockConn.ConnectionEXPECT().AgentEventLifecycleRegister(gomock.Any()).Return(nil)

				vmi := api2.NewMinimalVMI("test-vmi")
				vmi.UID = "1234"
				agentStore := agentpoller.NewAsyncAgentStore()

				err := client.StartDomainNotifier(
					mockConn.VirtConnection,
					deleteNotificationSent,
					vmi,
					"test_domain",
					&agentStore,
					10*time.Second,
					10*time.Second,
					10*time.Second,
					10*time.Second,
					10*time.Second,
					metadataCache(),
					false,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Triggering a libvirt reconnect")
				Expect(reconnectChan).ToNot(BeNil())
				reconnectChan <- true

				By("Verifying no watch.Error event is sent to virt-handler")
				Consistently(eventChan, 500*time.Millisecond).ShouldNot(Receive())
			})

			It("should send watch.Modified event on libvirt reconnect when domain cache is populated", func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())

				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				// Exercises the reconnect handler's code path when domainCache is non-nil.
				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, nil, nil, nil, nil, metadataCache(), false)

				var event watch.Event
				Eventually(eventChan, 2*time.Second).Should(Receive(&event))
				Expect(event.Type).To(Equal(watch.Modified))
				newDomain, ok := event.Object.(*api.Domain)
				Expect(ok).To(BeTrue())
				Expect(newDomain.Status.Status).To(Equal(api.Running))
			})
		})

		It("should receive a delete event when a VirtualMachineInstance is undefined",
			func() {
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return("", libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_NOSTATE, -1, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}}, client, deleteNotificationSent, nil, nil, nil, nil, metadataCache(), false)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case e := <-eventChan:
					Expect(e.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
					Expect(e.Object.(*api.Domain).ObjectMeta.DeletionTimestamp).ToNot(BeNil())
					Expect(e.Type).To(Equal(watch.Modified))

				}
				Expect(timedOut).To(BeFalse())

				select {
				case <-timeout:
					timedOut = true
				case <-deleteNotificationSent:
					// virt-launcher waits in a final delete notification to be sent before exiting.
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should not set DeletionTimestamp for stopped/migrated event when domain is already gone", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockLib := testing.NewLibvirt(ctrl)
			mockLib.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}).AnyTimes()

			domain := util.NewDomainFromName("test", "1234")
			e.eventCallback(mockLib.VirtConnection, domain, libvirtEvent{Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STOPPED,
				Detail: int(libvirt.DOMAIN_EVENT_STOPPED_MIGRATED),
			}}, client, deleteNotificationSent, nil, nil, nil, nil, metadataCache(), false)

			Expect(domain.Status.Reason).To(Equal(api.ReasonNonExistent))
			Expect(domain.ObjectMeta.DeletionTimestamp).To(BeNil())
			Consistently(deleteNotificationSent, 200*time.Millisecond).ShouldNot(Receive())
		})

		It("should not set DeletionTimestamp for Job Completed event when domain is already gone", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockLib := testing.NewLibvirt(ctrl)
			mockLib.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}).AnyTimes()

			mc := metadataCache()
			mc.Migration.Store(api.MigrationMetadata{UID: "test-migration-uid"})

			domain := util.NewDomainFromName("test", "1234")
			e.eventCallback(mockLib.VirtConnection, domain, libvirtEvent{
				JobCompletedEvent: &libvirt.DomainEventJobCompleted{
					Info: libvirt.DomainJobInfo{
						Type:      libvirt.DOMAIN_JOB_COMPLETED,
						Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT,
					},
				},
			}, client, deleteNotificationSent, nil, nil, nil, nil, mc, false)

			Expect(domain.Status.Reason).To(Equal(api.ReasonNonExistent))
			Expect(domain.ObjectMeta.DeletionTimestamp).To(BeNil())
			Consistently(deleteNotificationSent, 200*time.Millisecond).ShouldNot(Receive())
		})

		It("should update Interface status",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				interfaceStatus := []api.InterfaceStatus{
					{
						Ip: "1.1.1.1/24", Mac: "1", InterfaceName: "eth1",
					},
				}

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, interfaceStatus, nil, nil, nil, metadataCache(), false)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newInterfaceStatuses := newDomain.Status.Interfaces
					Expect(newInterfaceStatuses).To(HaveLen(1))
					Expect(equality.Semantic.DeepEqual(interfaceStatus, newInterfaceStatuses)).To(BeTrue())
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should update Guest OS Info",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				guestOsName := "TestGuestOS"
				osInfoStatus := api.GuestOSInfo{
					Name: guestOsName,
				}

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, nil, &osInfoStatus, nil, nil, metadataCache(), false)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newOSStatus := newDomain.Status.OSInfo
					Expect(equality.Semantic.DeepEqual(osInfoStatus, newOSStatus)).To(BeTrue())
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should update Guest FSFreeze status",
			func() {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				fsFrozenStatus := "frozen"
				fsFreezeStatus := api.FSFreeze{
					Status: fsFrozenStatus,
				}

				e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{}, client, deleteNotificationSent, nil, nil, nil, &fsFreezeStatus, metadataCache(), false)

				timedOut := false
				timeout := time.After(2 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-eventChan:
					newDomain, _ := event.Object.(*api.Domain)
					newFSFreezeStatus := newDomain.Status.FSFreezeStatus
					Expect(equality.Semantic.DeepEqual(fsFreezeStatus, newFSFreezeStatus)).To(BeTrue())
				}
				Expect(timedOut).To(BeFalse())
			})

		It("should consolidate I/O error status and Agent updates into a single watch event", func() {
			faultDisk := []libvirt.DomainDiskError{
				{
					Disk:  "vda",
					Error: libvirt.DOMAIN_DISK_ERROR_NO_SPACE,
				},
			}
			domain := api.NewMinimalDomain("test")
			domain.Status.Reason = api.ReasonPausedIOError
			x, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt := testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_IOERROR), nil)
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
			mockLibvirt.DomainEXPECT().GetDiskErrors(uint32(0)).Return(faultDisk, nil)

			vmi := api2.NewMinimalVMI("test-vmi")
			vmi.UID = "1234"
			vmiStore.Add(vmi)

			metadataCache := metadata.NewCache()
			interfaceStatus := []api.InterfaceStatus{{Ip: "10.0.0.1", InterfaceName: "eth0"}}
			e.eventCallback(mockLibvirt.VirtConnection, domain, libvirtEvent{}, client, deleteNotificationSent, interfaceStatus, nil, vmi, nil, metadataCache, false)

			var event watch.Event
			Eventually(eventChan, 2*time.Second).Should(Receive(&event))

			newDomain, ok := event.Object.(*api.Domain)
			Expect(ok).To(BeTrue())
			Expect(newDomain.Status.Reason).To(Equal(api.ReasonPausedIOError))
			Expect(newDomain.Status.Interfaces).To(HaveLen(1))
			Expect(newDomain.Status.Interfaces[0].Ip).To(Equal("10.0.0.1"))
		})

		It("should process job completion event even if the domain is paused due to an I/O error", func() {
			faultDisk := []libvirt.DomainDiskError{
				{
					Disk:  "vda",
					Error: libvirt.DOMAIN_DISK_ERROR_NO_SPACE,
				},
			}
			domain := api.NewMinimalDomain("test")
			domain.Status.Reason = api.ReasonPausedIOError
			x, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			domainJobInfo := libvirt.DomainJobInfo{
				Type:      libvirt.DOMAIN_JOB_COMPLETED,
				Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
			}
			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt := testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_IOERROR), nil)
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
			mockLibvirt.DomainEXPECT().GetDiskErrors(uint32(0)).Return(faultDisk, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(1)).Return(&domainJobInfo, nil)

			metadataCache := metadata.NewCache()
			metadataCache.Backup.Store(api.BackupMetadata{})
			libvirtEvent := libvirtEvent{
				JobCompletedEvent: &libvirt.DomainEventJobCompleted{
					Info: domainJobInfo,
				},
			}
			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			e.eventCallback(mockLibvirt.VirtConnection, domain, libvirtEvent, client, deleteNotificationSent, nil, nil, vmi, nil, metadataCache, false)
			backupMeta, ok := metadataCache.Backup.Load()
			Expect(ok).To(BeTrue())
			Expect(backupMeta.Completed).To(BeTrue())
		})

		It("should convert and persist completed migration stats", func() {
			domainJobInfo := libvirt.DomainJobInfo{
				Operation:   libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT,
				DowntimeSet: true,
				Downtime:    150,
			}
			metadataCache := metadata.NewCache()
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1"})

			storeCompletedMigrationStats(&domainJobInfo, metadataCache)

			completedMigration, exists := metadataCache.CompletedMigration.Load()
			Expect(exists).To(BeTrue())
			Expect(completedMigration.Stats.DowntimeSet).To(BeTrue())
			Expect(completedMigration.Stats.Downtime).To(Equal(uint64(150)))
			Expect(string(completedMigration.MigrationUID)).To(Equal("migration-1"))
		})

		It("should not persist stats without a reported downtime", func() {
			domainJobInfo := libvirt.DomainJobInfo{
				Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT,
			}
			metadataCache := metadata.NewCache()
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1"})

			storeCompletedMigrationStats(&domainJobInfo, metadataCache)

			_, exists := metadataCache.CompletedMigration.Load()
			Expect(exists).To(BeFalse())
		})

		It("should not persist stats without migration metadata", func() {
			metadataCache := metadata.NewCache()

			storeCompletedMigrationStats(&libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150}, metadataCache)

			_, exists := metadataCache.CompletedMigration.Load()
			Expect(exists).To(BeFalse())
		})

		DescribeTable("should attach completed stats only to their outgoing migration", func(
			captured api.MigrationMetadata, outgoing *api.MigrationMetadata, stats api.CompletedMigrationStats, attach bool,
		) {
			metadataCache := metadata.NewCache()
			metadataCache.Migration.Store(captured)
			storeCompletedMigrationStats(&libvirt.DomainJobInfo{DowntimeSet: stats.DowntimeSet, Downtime: stats.Downtime}, metadataCache)
			// A cache update after the outgoing snapshot was taken must not change the association.
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-2"})
			domain := api.NewMinimalDomain("test")
			domain.Spec.Metadata.KubeVirt.Migration = outgoing
			domain.Status.CompletedMigrationStats = &api.CompletedMigrationStats{DowntimeSet: true, Downtime: 99}
			expected := domain.DeepCopy()
			expected.Status.CompletedMigrationStats = nil
			if attach {
				expected.Status.CompletedMigrationStats = &stats
			}

			applyCompletedMigrationStats(domain, metadataCache)

			Expect(domain).To(Equal(expected))
		},
			Entry("matching outgoing snapshot despite a newer cache value",
				api.MigrationMetadata{UID: "migration-1"}, &api.MigrationMetadata{UID: "migration-1"},
				api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}, true),
			Entry("old stats must not overwrite a newer migration",
				api.MigrationMetadata{UID: "migration-1"}, &api.MigrationMetadata{UID: "migration-2"},
				api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}, false),
			Entry("new stats must not be attached to an older outgoing snapshot",
				api.MigrationMetadata{UID: "migration-2"}, &api.MigrationMetadata{UID: "migration-1"},
				api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}, false),
			Entry("missing outgoing migration",
				api.MigrationMetadata{UID: "migration-1"}, nil,
				api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}, false),
			Entry("missing outgoing migration UID",
				api.MigrationMetadata{UID: "migration-1"}, &api.MigrationMetadata{},
				api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}, false),
			Entry("missing captured migration UID",
				api.MigrationMetadata{}, &api.MigrationMetadata{},
				api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}, false),
			Entry("unreported downtime clears a reused domain's stats",
				api.MigrationMetadata{UID: "migration-1"}, &api.MigrationMetadata{UID: "migration-1"},
				api.CompletedMigrationStats{Downtime: 150}, false),
			Entry("reported zero is valid",
				api.MigrationMetadata{UID: "migration-1"}, &api.MigrationMetadata{UID: "migration-1"},
				api.CompletedMigrationStats{DowntimeSet: true}, true),
		)

		It("should clear completed stats from a reused domain after the cache is reset", func() {
			metadataCache := metadata.NewCache()
			migration := api.MigrationMetadata{UID: "migration-1"}
			metadataCache.Migration.Store(migration)
			storeCompletedMigrationStats(&libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150}, metadataCache)
			domain := api.NewMinimalDomain("test")
			domain.Spec.Metadata.KubeVirt.Migration = &migration
			applyCompletedMigrationStats(domain, metadataCache)
			Expect(domain.Status.CompletedMigrationStats).ToNot(BeNil())

			metadataCache.CompletedMigration.Set(metadata.CompletedMigrationData{})
			applyCompletedMigrationStats(domain, metadataCache)

			Expect(domain.Status.CompletedMigrationStats).To(BeNil())
			Expect(domain.Spec.Metadata.KubeVirt.Migration).To(Equal(&migration))
		})

		DescribeTable("should not publish stale downtime without matching authoritative metadata", func(migration *api.MigrationMetadata) {
			metadataCache := metadata.NewCache()
			previous := api.MigrationMetadata{UID: "migration-1"}
			stats := api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}
			if migration != nil {
				metadataCache.Migration.Store(*migration)
			}
			// Simulate an earlier completion being stored after the next migration's cache reset.
			metadataCache.CompletedMigration.Store(metadata.CompletedMigrationData{Stats: stats, MigrationUID: previous.UID})
			domain := util.NewDomainFromName("test", "1234")
			domain.Spec.Metadata.KubeVirt.Migration = &previous
			domain.Status.CompletedMigrationStats = &stats
			ctrl := gomock.NewController(GinkgoT())
			mockLib := testing.NewLibvirt(ctrl)
			mockLib.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			e.eventCallback(mockLib.VirtConnection, domain, libvirtEvent{}, client, deleteNotificationSent,
				nil, nil, nil, nil, metadataCache, false)

			var event watch.Event
			Eventually(eventChan, 2*time.Second).Should(Receive(&event))
			domainEvent, ok := event.Object.(*api.Domain)
			Expect(ok).To(BeTrue())
			Expect(domainEvent.Spec.Metadata.KubeVirt.Migration).To(Equal(migration))
			Expect(domainEvent.Status.CompletedMigrationStats).To(BeNil())
			Expect(string(domainEvent.Spec.Metadata.KubeVirt.UID)).To(Equal("1234"))
		},
			Entry("missing metadata", nil),
			Entry("missing migration UID", &api.MigrationMetadata{}),
			Entry("later migration with residual old stats", &api.MigrationMetadata{UID: "migration-2"}),
		)

		It("should include migration results updated while domain XML is being read", func() {
			metadataCache := metadata.NewCache()
			start := metav1.NewTime(time.Unix(100, 0))
			end := metav1.NewTime(time.Unix(200, 0))
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1", StartTimestamp: &start})
			storeCompletedMigrationStats(&libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150}, metadataCache)
			x, err := xml.Marshal(api.NewMinimalDomain("test").Spec)
			Expect(err).ToNot(HaveOccurred())
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTOFF, int(libvirt.DOMAIN_SHUTOFF_MIGRATED), nil)
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).DoAndReturn(func(_ libvirt.DomainXMLFlags) (string, error) {
				metadataCache.Migration.WithSafeBlock(func(migration *api.MigrationMetadata, _ bool) {
					migration.EndTimestamp = &end
				})
				return string(x), nil
			})

			e.eventCallback(mockLibvirt.VirtConnection, util.NewDomainFromName("test", "1234"), libvirtEvent{},
				client, deleteNotificationSent, nil, nil, nil, nil, metadataCache, false)

			var event watch.Event
			Eventually(eventChan, 2*time.Second).Should(Receive(&event))
			domainEvent, ok := event.Object.(*api.Domain)
			Expect(ok).To(BeTrue())
			current, exists := metadataCache.Migration.Load()
			Expect(exists).To(BeTrue())
			Expect(domainEvent.Spec.Metadata.KubeVirt.Migration).To(Equal(&current))
			Expect(domainEvent.Spec.Metadata.KubeVirt.Migration.EndTimestamp).To(Equal(&end))
			Expect(domainEvent.Status.CompletedMigrationStats).To(Equal(&api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}))
		})

		DescribeTable("should preserve updated migration metadata in domain notify events", func(
			domainMissing bool, xmlError error, notification libvirtEvent, failed bool,
		) {
			ctrl := gomock.NewController(GinkgoT())
			mockLib := testing.NewLibvirt(ctrl)
			if domainMissing {
				mockLib.ConnectionEXPECT().LookupDomainByName(gomock.Any()).
					Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}).Times(2)
			} else {
				x, err := xml.Marshal(api.NewMinimalDomain("test").Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLib.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLib.VirtDomain, nil).Times(2)
				mockLib.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTOFF, int(libvirt.DOMAIN_SHUTOFF_MIGRATED), nil).Times(2)
				mockLib.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(x), xmlError).Times(2)
				mockLib.DomainEXPECT().Free().Times(2)
			}
			metadataCache := metadata.NewCache()
			metadataCache.UID.Set("1234")
			backup := api.BackupMetadata{Name: "existing-backup", Completed: true}
			metadataCache.Backup.Set(backup)
			start := metav1.NewTime(time.Unix(100, 0))
			end := metav1.NewTime(time.Unix(200, 0))
			metadataCache.Migration.Store(api.MigrationMetadata{
				UID: "migration-1", StartTimestamp: &start, Mode: v1.MigrationPreCopy,
			})
			storeCompletedMigrationStats(&libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150}, metadataCache)
			// Reproduce JOB_COMPLETED arriving before the result writer, without timing-dependent goroutines.
			metadataCache.Migration.WithSafeBlock(func(migration *api.MigrationMetadata, _ bool) {
				migration.EndTimestamp = &end
				migration.Mode = v1.MigrationPostCopy
				migration.AbortStatus = string(v1.MigrationAbortFailed)
				migration.Failed = failed
				if failed {
					migration.FailureReason = "migration failed"
				}
			})
			expected, exists := metadataCache.Migration.Load()
			Expect(exists).To(BeTrue())
			domain := util.NewDomainFromName("test", "1234")
			domain.Spec.Metadata.KubeVirt.Backup = &backup

			for range 2 {
				e.eventCallback(mockLib.VirtConnection, domain, notification, client, deleteNotificationSent,
					nil, nil, nil, nil, metadataCache, false)

				var event watch.Event
				Eventually(eventChan, 2*time.Second).Should(Receive(&event))
				Expect(event.Type).To(Equal(watch.Modified))
				domainEvent, ok := event.Object.(*api.Domain)
				Expect(ok).To(BeTrue())
				Expect(domainEvent.Spec.Metadata.KubeVirt.Migration).To(Equal(&expected))
				Expect(string(domainEvent.Spec.Metadata.KubeVirt.UID)).To(Equal("1234"))
				Expect(domainEvent.Spec.Metadata.KubeVirt.Backup).To(Equal(&backup))
				Expect(domainEvent.Status.CompletedMigrationStats).To(Equal(&api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}))
				current, _ := metadataCache.Migration.Load()
				Expect(current).To(Equal(expected))
			}
			if notification.Event != nil && notification.Event.Event == libvirt.DOMAIN_EVENT_UNDEFINED {
				Expect(domain.DeletionTimestamp).ToNot(BeNil())
				Expect(deleteNotificationSent).To(HaveLen(1))
			} else {
				Expect(domain.DeletionTimestamp).To(BeNil())
				Expect(deleteNotificationSent).To(BeEmpty())
			}
		},
			Entry("metadata notification with the domain present", false, nil, libvirtEvent{}, false),
			Entry("metadata notification after the domain disappeared", true, nil, libvirtEvent{}, false),
			Entry("XML temporarily unavailable", false, libvirt.Error{Code: libvirt.ERR_OPERATION_INVALID}, libvirtEvent{}, false),
			Entry("domain disappears before XML is read", false, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}, libvirtEvent{}, false),
			Entry("preserve failure result fields", false, nil, libvirtEvent{}, true),
			Entry("delayed job-completed notification", true, nil, libvirtEvent{
				JobCompletedEvent: &libvirt.DomainEventJobCompleted{
					Info: libvirt.DomainJobInfo{Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT},
				},
			}, false),
			Entry("stopped-migrated notification", true, nil, libvirtEvent{
				Event: &libvirt.DomainEventLifecycle{
					Event: libvirt.DOMAIN_EVENT_STOPPED, Detail: int(libvirt.DOMAIN_EVENT_STOPPED_MIGRATED),
				},
			}, false),
			Entry("final undefined notification", true, nil, libvirtEvent{
				Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED},
			}, false),
		)
	})

	Describe("K8s Events", func() {
		var deleteNotificationSent chan watch.Event
		var client *Notifier
		var recorder *record.FakeRecorder
		var vmiStore cache.Store
		var e *eventCaller

		BeforeEach(func() {
			stop := make(chan struct{})
			eventChan := make(chan watch.Event, 100)
			deleteNotificationSent = make(chan watch.Event, 100)
			shareDir, err := os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			recorder = record.NewFakeRecorder(10)
			recorder.IncludeObject = true
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()
			e = &eventCaller{}

			go func(rec record.EventRecorder, store cache.Store) {
				notifyserver.RunServer(shareDir, stop, eventChan, rec, store)
			}(recorder, vmiStore)
			// mimic pipe
			notifyServer := filepath.Join(shareDir, "domain-notify.sock")
			pipePath := filepath.Join(shareDir, "domain-notify-pipe.sock")
			Expect(os.Symlink(notifyServer, pipePath)).To(Succeed())

			client = NewNotifier(shareDir)

			DeferCleanup(func() {
				close(stop)
				client.Close()
				os.RemoveAll(shareDir)
			})
		})

		It("Should send a k8s event", func() {

			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			err := client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

			event := <-recorder.Events
			Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
		})

		It("Should generate a k8s event on IO errors", func() {
			faultDisk := []libvirt.DomainDiskError{
				{
					Disk:  "vda",
					Error: libvirt.DOMAIN_DISK_ERROR_NO_SPACE,
				},
			}
			domain := api.NewMinimalDomain("test")
			domain.Status.Reason = api.ReasonPausedIOError
			x, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt := testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_IOERROR), nil)
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)
			mockLibvirt.DomainEXPECT().GetDiskErrors(uint32(0)).Return(faultDisk, nil)

			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)
			eventType := "Warning"
			eventReason := "IOerror"
			eventMessage := "VM Paused due to not enough space on volume: "
			metadataCache := metadata.NewCache()
			e.eventCallback(mockLibvirt.VirtConnection, domain, libvirtEvent{}, client, deleteNotificationSent, nil, nil, vmi, nil, metadataCache, false)
			event := <-recorder.Events
			Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
		})

		Context("handleGuestPanicEvent", func() {
			It("should return GuestPanicInfo with type unknown when log file is missing", func() {
				vmi := api2.NewMinimalVMI("test-vmi")
				vmi.Namespace = "test-ns"
				vmi.UID = "1234"
				vmiStore.Add(vmi)

				cache := metadata.NewCache()

				panicInfo := e.handleGuestPanicEvent(client, vmi, cache, int(libvirt.DOMAIN_EVENT_CRASHED_PANICKED), false)

				Expect(panicInfo).ToNot(BeNil())
				Expect(panicInfo.Type).To(Equal("unknown"))
			})

			It("should mark panic as handled for PANICKED events", func() {
				vmi := api2.NewMinimalVMI("test-vmi")
				vmi.Namespace = "test-ns"
				vmi.UID = "1234"
				vmiStore.Add(vmi)

				cache := metadata.NewCache()

				e.handleGuestPanicEvent(client, vmi, cache, int(libvirt.DOMAIN_EVENT_CRASHED_PANICKED), false)

				handled, exists := cache.GuestPanicHandled.Load()
				Expect(exists).To(BeTrue())
				Expect(handled).To(BeTrue())
			})

			It("should not mark panic as handled for CRASHLOADED events", func() {
				vmi := api2.NewMinimalVMI("test-vmi")
				vmi.Namespace = "test-ns"
				vmi.UID = "1234"
				vmiStore.Add(vmi)

				cache := metadata.NewCache()

				e.handleGuestPanicEvent(client, vmi, cache, int(libvirt.DOMAIN_EVENT_CRASHED_CRASHLOADED), false)

				_, exists := cache.GuestPanicHandled.Load()
				Expect(exists).To(BeFalse())
			})

			It("should return nil when VMI is nil", func() {
				cache := metadata.NewCache()
				panicInfo := e.handleGuestPanicEvent(client, nil, cache, int(libvirt.DOMAIN_EVENT_CRASHED_PANICKED), false)
				Expect(panicInfo).To(BeNil())
			})

			It("should skip already handled panic events", func() {
				vmi := api2.NewMinimalVMI("test-vmi")
				vmi.Namespace = "test-ns"
				vmi.UID = "1234"
				vmiStore.Add(vmi)

				cache := metadata.NewCache()
				cache.GuestPanicHandled.Set(true)

				panicInfo := e.handleGuestPanicEvent(client, vmi, cache, int(libvirt.DOMAIN_EVENT_CRASHED_PANICKED), false)
				Expect(panicInfo).To(BeNil())
			})
		})

	})

	Describe("Version mismatch", func() {

		var err error
		var ctrl *gomock.Controller
		var infoClient *info.MockNotifyInfoClient

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			infoClient = info.NewMockNotifyInfoClient(ctrl)
		})

		It("Should report error when server version mismatches", func() {

			fakeResponse := info.NotifyInfoResponse{
				SupportedNotifyVersions: []uint32{42},
			}
			infoClient.EXPECT().Info(gomock.Any(), gomock.Any()).Return(&fakeResponse, nil)

			By("Initializing the notifier")
			_, err = negotiateVersion(infoClient)

			Expect(err).To(HaveOccurred(), "Should have returned error about incompatible versions")
			Expect(err.Error()).To(ContainSubstring("no compatible version found"), "Expected error message to contain 'no compatible version found'")

		})
	})

	Describe("isTransientError", func() {
		DescribeTable("should classify errors correctly",
			func(err error, expected bool) {
				Expect(isTransientError(err)).To(Equal(expected))
			},
			Entry("nil error", nil, false),
			Entry("Unavailable is transient", status.Errorf(codes.Unavailable, "unavailable"), true),
			Entry("DeadlineExceeded is transient", status.Errorf(codes.DeadlineExceeded, "timeout"), true),
			Entry("Canceled is not transient", status.Errorf(codes.Canceled, "canceled"), false),
			Entry("Aborted is transient", status.Errorf(codes.Aborted, "aborted"), true),
			Entry("InvalidArgument is not transient", status.Errorf(codes.InvalidArgument, "bad request"), false),
			Entry("Internal is not transient", status.Errorf(codes.Internal, "internal error"), false),
			Entry("NotFound is not transient", status.Errorf(codes.NotFound, "not found"), false),
			Entry("PermissionDenied is not transient", status.Errorf(codes.PermissionDenied, "denied"), false),
			Entry("non-gRPC error is transient", errors.New("connection refused"), true),
			Entry("context.DeadlineExceeded is transient", context.DeadlineExceeded, true),
		)
	})

	Describe("gRPC status error handling", func() {
		newTestNotifier := func(fake *fakeNotifyClient) *Notifier {
			return &Notifier{
				client: &notifyClient{
					v1client:        fake,
					conn:            &grpc.ClientConn{},
					intervalTimeout: 100 * time.Millisecond,
					sendTimeout:     1 * time.Second,
					totalTimeout:    3 * time.Second,
				},
			}
		}

		Context("SendDomainEvent", func() {
			It("should propagate business errors immediately", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventErr: status.Errorf(codes.InvalidArgument, "invalid domain event"),
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).To(MatchError(ContainSubstring("InvalidArgument")))
				Expect(err).To(MatchError(ContainSubstring("invalid domain event")))
			})

			It("should propagate Internal errors from server", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventErr: status.Errorf(codes.Internal, "server crashed"),
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).To(MatchError(ContainSubstring("Internal")))
				Expect(err).To(MatchError(ContainSubstring("server crashed")))
			})

			It("should succeed when server returns no error", func() {
				client := newTestNotifier(&fakeNotifyClient{})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).ToNot(HaveOccurred())
			})

			It("should fall back to Response.Success for old servers", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventResponse: &notifyv1.Response{Success: false, Message: "legacy error"},
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).To(MatchError(ContainSubstring("legacy error")))
			})

			It("should succeed when a migrated server returns an empty response without Success field", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventResponse: &notifyv1.Response{},
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("SendK8sEvent", func() {
			It("should propagate business errors immediately", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventErr: status.Errorf(codes.InvalidArgument, "invalid k8s event"),
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).To(MatchError(ContainSubstring("InvalidArgument")))
				Expect(err).To(MatchError(ContainSubstring("invalid k8s event")))
			})

			It("should propagate Internal errors from server", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventErr: status.Errorf(codes.Internal, "handler error"),
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).To(MatchError(ContainSubstring("Internal")))
				Expect(err).To(MatchError(ContainSubstring("handler error")))
			})

			It("should succeed when server returns no error", func() {
				client := newTestNotifier(&fakeNotifyClient{})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).ToNot(HaveOccurred())
			})

			It("should fall back to Response.Success for old servers", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventResponse: &notifyv1.Response{Success: false, Message: "legacy k8s error"},
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).To(MatchError(ContainSubstring("legacy k8s error")))
			})

			It("should succeed when a migrated server returns an empty response without Success field", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventResponse: &notifyv1.Response{},
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("transient error retry", func() {
			var shareDir string
			var grpcServer *grpc.Server
			var sock net.Listener
			var retryClient *Notifier

			startRetryServer := func(server notifyv1.NotifyServer) *Notifier {
				shareDir = GinkgoT().TempDir()

				grpcServer = grpc.NewServer()
				info.RegisterNotifyInfoServer(grpcServer, &testInfoServer{})
				notifyv1.RegisterNotifyServer(grpcServer, server)

				var err error
				sock, err = net.Listen("unix", filepath.Join(shareDir, "domain-notify.sock"))
				Expect(err).ToNot(HaveOccurred())
				go func() { _ = grpcServer.Serve(sock) }()

				Expect(os.Symlink(
					filepath.Join(shareDir, "domain-notify.sock"),
					filepath.Join(shareDir, "domain-notify-pipe.sock"),
				)).To(Succeed())

				retryClient = NewNotifier(shareDir)
				retryClient.SetCustomTimeouts(50*time.Millisecond, 1*time.Second, 5*time.Second)
				return retryClient
			}

			AfterEach(func() {
				if retryClient != nil {
					retryClient.Close()
				}
				if grpcServer != nil {
					grpcServer.Stop()
				}
				if sock != nil {
					sock.Close()
				}
			})

			It("SendDomainEvent should succeed after a transient error on retry", func() {
				var callCount atomic.Int32
				server := &callbackNotifyServer{
					onDomainEvent: func(_ context.Context, _ *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
						if callCount.Add(1) == 1 {
							return nil, status.Errorf(codes.Unavailable, "temporary failure")
						}
						return &notifyv1.Response{Success: true}, nil
					},
				}
				startRetryServer(server)

				domain := api.NewMinimalDomain("test")
				err := retryClient.SendDomainEvent(watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).ToNot(HaveOccurred())
				Expect(callCount.Load()).To(BeNumerically(">=", int32(2)))
			})

			It("SendK8sEvent should succeed after a transient error on retry", func() {
				var callCount atomic.Int32
				server := &callbackNotifyServer{
					onK8SEvent: func(_ context.Context, _ *notifyv1.K8SEventRequest) (*notifyv1.Response, error) {
						if callCount.Add(1) == 1 {
							return nil, status.Errorf(codes.Unavailable, "temporary failure")
						}
						return &notifyv1.Response{Success: true}, nil
					},
				}
				startRetryServer(server)

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := retryClient.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).ToNot(HaveOccurred())
				Expect(callCount.Load()).To(BeNumerically(">=", int32(2)))
			})
		})
	})
})

type fakeNotifyClient struct {
	domainEventErr      error
	domainEventResponse *notifyv1.Response
	k8sEventErr         error
	k8sEventResponse    *notifyv1.Response
}

func (f *fakeNotifyClient) HandleDomainEvent(
	_ context.Context, _ *notifyv1.DomainEventRequest, _ ...grpc.CallOption,
) (*notifyv1.Response, error) {
	if f.domainEventErr != nil {
		return nil, f.domainEventErr
	}
	if f.domainEventResponse != nil {
		return f.domainEventResponse, nil
	}
	return &notifyv1.Response{Success: true}, nil
}

func (f *fakeNotifyClient) HandleK8SEvent(
	_ context.Context, _ *notifyv1.K8SEventRequest, _ ...grpc.CallOption,
) (*notifyv1.Response, error) {
	if f.k8sEventErr != nil {
		return nil, f.k8sEventErr
	}
	if f.k8sEventResponse != nil {
		return f.k8sEventResponse, nil
	}
	return &notifyv1.Response{Success: true}, nil
}

type testInfoServer struct{}

func (t *testInfoServer) Info(_ context.Context, _ *info.NotifyInfoRequest) (*info.NotifyInfoResponse, error) {
	return &info.NotifyInfoResponse{
		SupportedNotifyVersions: []uint32{1},
	}, nil
}

type callbackNotifyServer struct {
	onDomainEvent func(context.Context, *notifyv1.DomainEventRequest) (*notifyv1.Response, error)
	onK8SEvent    func(context.Context, *notifyv1.K8SEventRequest) (*notifyv1.Response, error)
}

func (s *callbackNotifyServer) HandleDomainEvent(ctx context.Context, req *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
	if s.onDomainEvent != nil {
		return s.onDomainEvent(ctx, req)
	}
	return &notifyv1.Response{Success: true}, nil
}

func (s *callbackNotifyServer) HandleK8SEvent(ctx context.Context, req *notifyv1.K8SEventRequest) (*notifyv1.Response, error) {
	if s.onK8SEvent != nil {
		return s.onK8SEvent(ctx, req)
	}
	return &notifyv1.Response{Success: true}, nil
}
