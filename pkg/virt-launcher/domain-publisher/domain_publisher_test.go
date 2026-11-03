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

package domainpublisher

import (
	"encoding/xml"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	api2 "kubevirt.io/client-go/api"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

var _ = Describe("DomainPublisher", func() {

	Describe("Domain Events", func() {

		var notifier *fakeNotifier
		var deleteNotificationSent chan watch.Event
		var mockLibvirt *testing.Libvirt

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockLibvirt = testing.NewLibvirt(ctrl)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			notifier = newFakeNotifier()
			deleteNotificationSent = make(chan watch.Event, 100)
		})

		metadataCache := func() *metadata.Cache { return metadata.NewCache() }

		newPublisher := func(conn cli.Connection, mc *metadata.Cache) *DomainPublisher {
			vmi := api2.NewMinimalVMI("test")
			vmi.UID = "1234"
			agentStore := agentpoller.NewAsyncAgentStore()
			agentPoller := agentpoller.CreatePoller(conn, vmi.UID, "default_test", &agentStore,
				10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second)
			return NewDomainPublisher(notifier, conn, vmi, agentPoller, mc, false)
		}

		Context("server", func() {
			DescribeTable("should accept Domain notify events", func(state libvirt.DomainState, event libvirt.DomainEventType, kubevirtState api.LifeCycle, kubeEventType watch.EventType) {
				domain := api.NewMinimalDomain("test")
				x, err := xml.Marshal(domain.Spec)
				Expect(err).ToNot(HaveOccurred())

				mockLibvirt.DomainEXPECT().GetState().Return(state, -1, nil)
				mockLibvirt.DomainEXPECT().Free()
				mockLibvirt.DomainEXPECT().GetName().Return("test", nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil)

				newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: event}}, deleteNotificationSent, nil, nil, nil)

				var eventReceived watch.Event
				Eventually(notifier.domainEvents).Within(2 * time.Second).Should(Receive(&eventReceived))
				newDomain, ok := eventReceived.Object.(*api.Domain)
				newDomain.Spec.XMLName = xml.Name{}
				Expect(ok).To(BeTrue(), "should typecase domain")
				Expect(equality.Semantic.DeepEqual(domain.Spec, newDomain.Spec)).To(BeTrue())
				Expect(eventReceived.Type).To(Equal(kubeEventType))
			},
				Entry("modified for crashed VMIs", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed, watch.Modified),
				Entry("modified for stopped VMIs with shutoff reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff, watch.Modified),
				Entry("modified for stopped VMIs with stopped reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff, watch.Modified),
				Entry("modified for running VMIs", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running, watch.Modified),
				Entry("added for defined VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff, watch.Added),
			)

			It("should not send watch.Error event on libvirt reconnect", func() {
				By("Starting the publisher with a mock connection")
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

				Expect(newPublisher(mockConn.VirtConnection, metadataCache()).Start(deleteNotificationSent)).To(Succeed())

				By("Triggering a libvirt reconnect")
				Expect(reconnectChan).ToNot(BeNil())
				reconnectChan <- true

				By("Verifying no watch.Error event is sent to virt-handler")
				Consistently(notifier.domainEvents, 500*time.Millisecond).ShouldNot(Receive())
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
				newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{}, deleteNotificationSent, nil, nil, nil)

				var event watch.Event
				Eventually(notifier.domainEvents, 2*time.Second).Should(Receive(&event))
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

				newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED}}, deleteNotificationSent, nil, nil, nil)

				var event watch.Event
				Eventually(notifier.domainEvents).Within(2 * time.Second).Should(Receive(&event))
				Expect(event.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
				Expect(event.Object.(*api.Domain).ObjectMeta.DeletionTimestamp).ToNot(BeNil())
				Expect(event.Type).To(Equal(watch.Modified))

				// virt-launcher waits in a final delete notification to be sent before exiting.
				Eventually(deleteNotificationSent).Within(2 * time.Second).Should(Receive())
			})

		It("should not set DeletionTimestamp for stopped/migrated event when domain is already gone", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockLib := testing.NewLibvirt(ctrl)
			mockLib.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}).AnyTimes()

			domain := util.NewDomainFromName("test", "1234")
			newPublisher(mockLib.VirtConnection, metadataCache()).eventCallback(domain, libvirtEvent{Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STOPPED,
				Detail: int(libvirt.DOMAIN_EVENT_STOPPED_MIGRATED),
			}}, deleteNotificationSent, nil, nil, nil)

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
			newPublisher(mockLib.VirtConnection, mc).eventCallback(domain, libvirtEvent{
				JobCompletedEvent: &libvirt.DomainEventJobCompleted{
					Info: libvirt.DomainJobInfo{
						Type:      libvirt.DOMAIN_JOB_COMPLETED,
						Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT,
					},
				},
			}, deleteNotificationSent, nil, nil, nil)

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

				newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{}, deleteNotificationSent, interfaceStatus, nil, nil)

				var event watch.Event
				Eventually(notifier.domainEvents).Within(2 * time.Second).Should(Receive(&event))
				newDomain, _ := event.Object.(*api.Domain)
				newInterfaceStatuses := newDomain.Status.Interfaces
				Expect(newInterfaceStatuses).To(HaveLen(1))
				Expect(equality.Semantic.DeepEqual(interfaceStatus, newInterfaceStatuses)).To(BeTrue())
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

				newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{}, deleteNotificationSent, nil, &osInfoStatus, nil)

				var event watch.Event
				Eventually(notifier.domainEvents).Within(2 * time.Second).Should(Receive(&event))
				newDomain, _ := event.Object.(*api.Domain)
				newOSStatus := newDomain.Status.OSInfo
				Expect(equality.Semantic.DeepEqual(osInfoStatus, newOSStatus)).To(BeTrue())
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

				newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{}, deleteNotificationSent, nil, nil, &fsFreezeStatus)

				var event watch.Event
				Eventually(notifier.domainEvents).Within(2 * time.Second).Should(Receive(&event))
				newDomain, _ := event.Object.(*api.Domain)
				newFSFreezeStatus := newDomain.Status.FSFreezeStatus
				Expect(equality.Semantic.DeepEqual(fsFreezeStatus, newFSFreezeStatus)).To(BeTrue())
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

			metadataCache := metadata.NewCache()
			interfaceStatus := []api.InterfaceStatus{{Ip: "10.0.0.1", InterfaceName: "eth0"}}
			newPublisher(mockLibvirt.VirtConnection, metadataCache).eventCallback(domain, libvirtEvent{}, deleteNotificationSent, interfaceStatus, nil, nil)

			var event watch.Event
			Eventually(notifier.domainEvents, 2*time.Second).Should(Receive(&event))

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
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DOMAIN_JOB_STATS_COMPLETED|libvirt.DOMAIN_JOB_STATS_KEEP_COMPLETED).Return(&domainJobInfo, nil)

			metadataCache := metadata.NewCache()
			metadataCache.Backup.Store(api.BackupMetadata{})
			libvirtEvent := libvirtEvent{
				JobCompletedEvent: &libvirt.DomainEventJobCompleted{
					Info: domainJobInfo,
				},
			}

			newPublisher(mockLibvirt.VirtConnection, metadataCache).eventCallback(domain, libvirtEvent, deleteNotificationSent, nil, nil, nil)
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

			newPublisher(mockLib.VirtConnection, metadataCache).eventCallback(domain, libvirtEvent{}, deleteNotificationSent, nil, nil, nil)

			var event watch.Event
			Eventually(notifier.domainEvents, 2*time.Second).Should(Receive(&event))
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

			newPublisher(mockLibvirt.VirtConnection, metadataCache).eventCallback(util.NewDomainFromName("test", "1234"), libvirtEvent{}, deleteNotificationSent, nil, nil, nil)

			var event watch.Event
			Eventually(notifier.domainEvents, 2*time.Second).Should(Receive(&event))
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

			publisher := newPublisher(mockLib.VirtConnection, metadataCache)
			for range 2 {
				publisher.eventCallback(domain, notification, deleteNotificationSent, nil, nil, nil)

				var event watch.Event
				Eventually(notifier.domainEvents, 2*time.Second).Should(Receive(&event))
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

		Context("guest panic", func() {
			var publisher *DomainPublisher

			crashed := func(detail libvirt.DomainEventCrashedDetailType) libvirtEvent {
				return libvirtEvent{Event: &libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_CRASHED, Detail: int(detail)}}
			}
			notify := func(event libvirtEvent) *api.Domain {
				publisher.eventCallback(api.NewMinimalDomain("test"), event, deleteNotificationSent, nil, nil, nil)
				var sent watch.Event
				Eventually(notifier.domainEvents).Should(Receive(&sent))
				return sent.Object.(*api.Domain)
			}

			BeforeEach(func() {
				x, err := xml.Marshal(api.NewMinimalDomain("test").Spec)
				Expect(err).ToNot(HaveOccurred())
				mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_CRASHED, int(libvirt.DOMAIN_CRASHED_PANICKED), nil).AnyTimes()
				mockLibvirt.DomainEXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(x), nil).AnyTimes()
				mockLibvirt.DomainEXPECT().Free().AnyTimes()
				publisher = newPublisher(mockLibvirt.VirtConnection, metadataCache())
			})

			It("should report unknown panic details when the QEMU log is missing", func() {
				domain := notify(crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED))

				Expect(domain.Status.GuestPanicInfo).To(Equal(&api.GuestPanicInfo{Type: "unknown"}))
				Expect(domain.Status.PanicCount).To(Equal(1))
				Expect(notifier.k8sEvents).To(Receive(Equal(k8sEvent{severity: "Warning", reason: "GuestPanicked", message: "GuestPanicked (details unavailable)"})))
			})

			It("should count a PANICKED panic once", func() {
				notify(crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED))

				Expect(notify(crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED)).Status.PanicCount).To(Equal(1))
			})

			It("should count every CRASHLOADED panic", func() {
				notify(crashed(libvirt.DOMAIN_EVENT_CRASHED_CRASHLOADED))

				Expect(notify(crashed(libvirt.DOMAIN_EVENT_CRASHED_CRASHLOADED)).Status.PanicCount).To(Equal(2))
			})

			It("should keep panic details in later notifications", func() {
				notify(crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED))

				domain := notify(libvirtEvent{})
				Expect(domain.Status.GuestPanicInfo).To(Equal(&api.GuestPanicInfo{Type: "unknown"}))
				Expect(domain.Status.PanicCount).To(Equal(1))
			})
		})

		It("should generate a k8s event on IO errors", func() {
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

			newPublisher(mockLibvirt.VirtConnection, metadataCache()).eventCallback(domain, libvirtEvent{}, deleteNotificationSent, nil, nil, nil)
			Expect(notifier.k8sEvents).To(Receive(Equal(k8sEvent{severity: "Warning", reason: "IOerror", message: "VM Paused due to not enough space on volume: "})))
		})
	})
})

type k8sEvent struct {
	severity string
	reason   string
	message  string
}

type fakeNotifier struct {
	domainEvents chan watch.Event
	k8sEvents    chan k8sEvent
}

func newFakeNotifier() *fakeNotifier {
	return &fakeNotifier{
		domainEvents: make(chan watch.Event, 100),
		k8sEvents:    make(chan k8sEvent, 100),
	}
}

func (f *fakeNotifier) SendDomainEvent(event watch.Event) error {
	f.domainEvents <- event
	return nil
}

func (f *fakeNotifier) SendK8sEvent(_ *v1.VirtualMachineInstance, severity string, reason string, message string) error {
	f.k8sEvents <- k8sEvent{severity: severity, reason: reason, message: message}
	return nil
}
