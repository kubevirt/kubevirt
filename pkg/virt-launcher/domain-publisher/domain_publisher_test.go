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

package domainpublisher_test

import (
	"context"
	"encoding/xml"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	domainpublisher "kubevirt.io/kubevirt/pkg/virt-launcher/domain-publisher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var _ = Describe("DomainPublisher", func() {
	var (
		mockLibvirt   *testing.Libvirt
		metadataCache *metadata.Cache
		h             *harness
	)

	BeforeEach(func() {
		mockLibvirt = testing.NewLibvirt(gomock.NewController(GinkgoT()))
		metadataCache = metadata.NewCache()
		metadataCache.UID.Set("1234")
		h = startPublisher(mockLibvirt, metadataCache)
	})

	domainXML := func() string {
		x, err := xml.Marshal(api.NewMinimalDomain("test").Spec)
		Expect(err).ToNot(HaveOccurred())
		return string(x)
	}

	domainExists := func(state libvirt.DomainState, reason int) {
		mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(mockLibvirt.VirtDomain, nil).AnyTimes()
		mockLibvirt.DomainEXPECT().GetState().Return(state, reason, nil).AnyTimes()
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(domainXML(), nil).AnyTimes()
		mockLibvirt.DomainEXPECT().Free().AnyTimes()
	}

	domainGone := func() {
		mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}).AnyTimes()
	}

	resumed := func(p *domainpublisher.DomainPublisher) {
		p.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_RESUMED})
	}

	domainRemoved := func() {
		mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(mockLibvirt.VirtDomain, nil)
		mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, -1, nil)
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(domainXML(), nil)
		mockLibvirt.DomainEXPECT().Free()
		resumed(h.publisher)
		h.receive()
		domainGone()
	}

	stoppedMigrated := func(p *domainpublisher.DomainPublisher) {
		p.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STOPPED, Detail: int(libvirt.DOMAIN_EVENT_STOPPED_MIGRATED)})
	}

	DescribeTable("should notify the domain state on lifecycle events", func(state libvirt.DomainState, event libvirt.DomainEventType, expectedState api.LifeCycle) {
		domainExists(state, -1)

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: event})

		notification := h.receive()
		Expect(notification.Type).To(Equal(watch.Modified))
		domain := notification.Object.(*api.Domain)
		Expect(domain.Status.Status).To(Equal(expectedState))
		expectedSpec := api.NewMinimalDomain("test").Spec
		expectedSpec.Metadata.KubeVirt.UID = "1234"
		domain.Spec.XMLName = xml.Name{}
		Expect(equality.Semantic.DeepEqual(expectedSpec, domain.Spec)).To(BeTrue())
	},
		Entry("crashed", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed),
		Entry("shut down", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff),
		Entry("stopped", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff),
		Entry("started", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running),
		Entry("defined", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff),
	)

	It("should not notify a resync before the domain is defined", func() {
		domainGone()

		h.publisher.Resync()

		Consistently(h.notifier.domainEvents, 500*time.Millisecond).ShouldNot(Receive())
		Expect(h.publisher.DomainDefined()).ToNot(BeClosed())
	})

	It("should signal the domain definition once libvirt has defined it", func() {
		domainExists(libvirt.DOMAIN_SHUTOFF, -1)

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_DEFINED})

		h.receive()
		Expect(h.publisher.DomainDefined()).To(BeClosed())
	})

	It("should publish the removal of a domain that was gone before its first lookup", func() {
		domainGone()

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STARTED, Detail: int(libvirt.DOMAIN_EVENT_STARTED_MIGRATED)})
		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STOPPED, Detail: int(libvirt.DOMAIN_EVENT_STOPPED_FAILED)})

		Eventually(h.latest).Should(And(
			HaveField("Status.Reason", api.ReasonNonExistent),
			HaveField("DeletionTimestamp", Not(BeNil())),
		))
		Expect(h.publisher.DomainDefined()).To(BeClosed())
		Eventually(h.publisher.FinalNotificationSent()).Should(BeClosed())
	})

	It("should not notify guest agent updates before the domain is defined", func() {
		domainGone()

		h.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, api.GuestOSInfo{Name: "fedora"})

		Consistently(h.notifier.domainEvents, 500*time.Millisecond).ShouldNot(Receive())
	})

	It("should notify every lifecycle event of a burst", func() {
		const burst = 20
		domainExists(libvirt.DOMAIN_RUNNING, -1)

		for range burst {
			resumed(h.publisher)
		}

		for range burst {
			h.receive()
		}
	})

	It("should stop publishing once its context is cancelled", func() {
		h.stop()

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STARTED})

		Consistently(h.notifier.domainEvents, 500*time.Millisecond).ShouldNot(Receive())
	})

	It("should notify the domain again on a resync", func() {
		domainExists(libvirt.DOMAIN_RUNNING, -1)
		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STARTED})
		h.receive()

		h.publisher.Resync()

		notification := h.receive()
		Expect(notification.Type).To(Equal(watch.Modified))
		Expect(notification.Object.(*api.Domain).Status.Status).To(Equal(api.Running))
	})

	It("should send a final delete notification when the domain is undefined", func() {
		domainRemoved()

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED})

		notification := h.receive()
		Expect(notification.Type).To(Equal(watch.Modified))
		Expect(notification.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
		Expect(notification.Object.(*api.Domain).DeletionTimestamp).ToNot(BeNil())
		Eventually(h.publisher.FinalNotificationSent()).Should(BeClosed())
	})

	It("should retry a notification that failed to send", func() {
		domainExists(libvirt.DOMAIN_RUNNING, -1)
		h.notifier.sendErrors <- errors.New("virt-handler unreachable")

		resumed(h.publisher)

		Expect(h.receive().Object.(*api.Domain).Status.Status).To(Equal(api.Running))
	})

	It("should retry a notification that failed to read the domain", func() {
		mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(nil, libvirt.Error{Code: libvirt.ERR_INTERNAL_ERROR})
		domainExists(libvirt.DOMAIN_RUNNING, -1)

		resumed(h.publisher)

		Expect(h.receive().Object.(*api.Domain).Status.Status).To(Equal(api.Running))
	})

	It("should signal the final notification only once it was sent", func() {
		domainRemoved()
		h.notifier.sendErrors <- errors.New("virt-handler unreachable")

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED})

		Consistently(h.publisher.FinalNotificationSent(), 500*time.Millisecond).ShouldNot(BeClosed())
		Expect(h.receive().Object.(*api.Domain).DeletionTimestamp).ToNot(BeNil())
		Eventually(h.publisher.FinalNotificationSent()).Should(BeClosed())
	})

	It("should keep the deletion timestamp in later notifications", func() {
		domainRemoved()
		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED})
		removed := h.receive().Object.(*api.Domain)
		Expect(removed.DeletionTimestamp).ToNot(BeNil())

		h.publisher.Resync()

		Expect(h.receive().Object.(*api.Domain).DeletionTimestamp).To(Equal(removed.DeletionTimestamp))
	})

	It("should complete a backup job after the domain is gone", func() {
		metadataCache.Backup.Set(api.BackupMetadata{Name: "backup"})
		domainRemoved()

		h.publisher.OnJobCompleted(&libvirt.DomainEventJobCompleted{Info: libvirt.DomainJobInfo{
			Type:      libvirt.DOMAIN_JOB_COMPLETED,
			Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
		}})

		Expect(h.receive().Object.(*api.Domain).Spec.Metadata.KubeVirt.Backup.Completed).To(BeTrue())
	})

	DescribeTable("should not mark a gone domain deleted on", func(fire func(*domainpublisher.DomainPublisher)) {
		metadataCache.Migration.Set(api.MigrationMetadata{UID: "migration-1"})
		domainRemoved()

		fire(h.publisher)

		domain := h.receive().Object.(*api.Domain)
		Expect(domain.Status.Reason).To(Equal(api.ReasonNonExistent))
		Expect(domain.DeletionTimestamp).To(BeNil())
		Consistently(h.publisher.FinalNotificationSent(), 200*time.Millisecond).ShouldNot(BeClosed())
	},
		Entry("a stopped-migrated event", stoppedMigrated),
		Entry("a migration job completion", func(p *domainpublisher.DomainPublisher) {
			p.OnJobCompleted(&libvirt.DomainEventJobCompleted{Info: libvirt.DomainJobInfo{
				Type:      libvirt.DOMAIN_JOB_COMPLETED,
				Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT,
			}})
		}),
	)

	DescribeTable("should notify guest agent data", func(store func(*agentpoller.AsyncAgentStore), verify func(*api.Domain)) {
		domainExists(libvirt.DOMAIN_RUNNING, -1)
		resumed(h.publisher)
		h.receive()

		store(h.agentStore)

		verify(h.receive().Object.(*api.Domain))
	},
		Entry("interfaces", func(s *agentpoller.AsyncAgentStore) {
			s.Store(libvirt.DOMAIN_GUEST_INFO_INTERFACES, []api.InterfaceStatus{{Ip: "1.1.1.1/24", Mac: "1", InterfaceName: "eth1"}})
		}, func(d *api.Domain) {
			Expect(d.Status.Interfaces).To(Equal([]api.InterfaceStatus{{Ip: "1.1.1.1/24", Mac: "1", InterfaceName: "eth1"}}))
		}),
		Entry("guest OS info", func(s *agentpoller.AsyncAgentStore) {
			s.Store(libvirt.DOMAIN_GUEST_INFO_OS, api.GuestOSInfo{Name: "TestGuestOS"})
		}, func(d *api.Domain) {
			Expect(d.Status.OSInfo).To(Equal(api.GuestOSInfo{Name: "TestGuestOS"}))
		}),
		Entry("filesystem freeze status", func(s *agentpoller.AsyncAgentStore) {
			s.Store(agentpoller.GetFSFreezeStatus, api.FSFreeze{Status: "frozen"})
		}, func(d *api.Domain) {
			Expect(d.Status.FSFreezeStatus).To(Equal(api.FSFreeze{Status: "frozen"}))
		}),
	)

	Context("with a domain paused on an I/O error", func() {
		BeforeEach(func() {
			domainExists(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_IOERROR))
		})

		DescribeTable("should report an I/O error that pauses the VM as a Kubernetes event", func(reason, expectedMessage string) {
			h.publisher.OnIOError(&libvirt.DomainEventIOErrorReason{
				DevAlias: "ua-disk0", Action: libvirt.DOMAIN_EVENT_IO_ERROR_PAUSE, Reason: reason,
			})

			h.receive()
			Expect(h.notifier.k8sEvents).To(Receive(Equal(k8sEvent{severity: "Warning", reason: "IOerror", message: expectedMessage})))
		},
			Entry("out of space", "enospc", "VM Paused due to not enough space on volume: disk0"),
			Entry("any other error", "message", "VM Paused due to IO error at the volume: disk0"),
		)

		It("should not report an I/O error that does not pause the VM", func() {
			h.publisher.OnIOError(&libvirt.DomainEventIOErrorReason{
				DevAlias: "ua-disk0", Action: libvirt.DOMAIN_EVENT_IO_ERROR_REPORT, Reason: "message",
			})

			h.receive()
			Expect(h.notifier.k8sEvents).ToNot(Receive())
		})

		It("should consolidate the I/O error state and agent data into one notification", func() {
			h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_SUSPENDED, Detail: int(libvirt.DOMAIN_EVENT_SUSPENDED_IOERROR)})
			h.receive()

			h.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_INTERFACES, []api.InterfaceStatus{{Ip: "10.0.0.1", InterfaceName: "eth0"}})

			domain := h.receive().Object.(*api.Domain)
			Expect(domain.Status.Reason).To(Equal(api.ReasonPausedIOError))
			Expect(domain.Status.Interfaces).To(Equal([]api.InterfaceStatus{{Ip: "10.0.0.1", InterfaceName: "eth0"}}))
		})

		It("should complete a backup job", func() {
			jobInfo := libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_COMPLETED, Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP}
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DOMAIN_JOB_STATS_COMPLETED|libvirt.DOMAIN_JOB_STATS_KEEP_COMPLETED).Return(&jobInfo, nil)
			metadataCache.Backup.Set(api.BackupMetadata{})

			h.publisher.OnJobCompleted(&libvirt.DomainEventJobCompleted{Info: jobInfo})

			Expect(h.receive().Object.(*api.Domain).Spec.Metadata.KubeVirt.Backup.Completed).To(BeTrue())
		})
	})

	Context("completed migration stats", func() {
		migrationCompleted := func(info libvirt.DomainJobInfo) {
			info.Operation = libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT
			h.publisher.OnJobCompleted(&libvirt.DomainEventJobCompleted{Info: info})
		}

		BeforeEach(func() {
			domainRemoved()
		})

		DescribeTable("should attach the downtime libvirt reports to its migration", func(info libvirt.DomainJobInfo, expected *api.CompletedMigrationStats) {
			metadataCache.Migration.Set(api.MigrationMetadata{UID: "migration-1"})

			migrationCompleted(info)

			Expect(h.receive().Object.(*api.Domain).Status.CompletedMigrationStats).To(Equal(expected))
		},
			Entry("reported downtime", libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150}, &api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}),
			Entry("reported zero downtime", libvirt.DomainJobInfo{DowntimeSet: true}, &api.CompletedMigrationStats{DowntimeSet: true}),
			Entry("unreported downtime", libvirt.DomainJobInfo{Downtime: 150}, nil),
		)

		It("should not attach downtime without a tracked migration", func() {

			migrationCompleted(libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150})

			Expect(h.receive().Object.(*api.Domain).Status.CompletedMigrationStats).To(BeNil())
		})

		DescribeTable("should not attach downtime to another migration", func(next api.MigrationMetadata) {
			metadataCache.Migration.Set(api.MigrationMetadata{UID: "migration-1"})
			migrationCompleted(libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150})
			h.receive()

			metadataCache.Migration.Store(next)

			Eventually(h.latest).Should(And(
				HaveField("Spec.Metadata.KubeVirt.Migration", Equal(&next)),
				HaveField("Status.CompletedMigrationStats", BeNil()),
			))
		},
			Entry("a later migration", api.MigrationMetadata{UID: "migration-2"}),
			Entry("a migration without a UID", api.MigrationMetadata{}),
		)

		It("should drop downtime once the completed migration is reset", func() {
			metadataCache.Migration.Set(api.MigrationMetadata{UID: "migration-1"})
			migrationCompleted(libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150})
			Expect(h.receive().Object.(*api.Domain).Status.CompletedMigrationStats).ToNot(BeNil())

			metadataCache.CompletedMigration.Store(metadata.CompletedMigrationData{})

			Eventually(h.latest).Should(HaveField("Status.CompletedMigrationStats", BeNil()))
		})
	})

	It("should include migration results updated while the domain XML is being read", func() {
		startTime := metav1.NewTime(time.Unix(100, 0))
		end := metav1.NewTime(time.Unix(200, 0))
		metadataCache.Migration.Set(api.MigrationMetadata{UID: "migration-1", StartTimestamp: &startTime})
		mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(mockLibvirt.VirtDomain, nil).AnyTimes()
		mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTOFF, int(libvirt.DOMAIN_SHUTOFF_MIGRATED), nil).AnyTimes()
		mockLibvirt.DomainEXPECT().Free().AnyTimes()
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).DoAndReturn(func(_ libvirt.DomainXMLFlags) (string, error) {
			metadataCache.Migration.WithSafeBlock(func(migration *api.MigrationMetadata, _ bool) {
				migration.EndTimestamp = &end
			})
			return domainXML(), nil
		}).AnyTimes()

		h.publisher.OnJobCompleted(&libvirt.DomainEventJobCompleted{Info: libvirt.DomainJobInfo{
			Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT, DowntimeSet: true, Downtime: 150,
		}})

		domain := h.receive().Object.(*api.Domain)
		Expect(domain.Spec.Metadata.KubeVirt.Migration.EndTimestamp).To(Equal(&end))
		Expect(domain.Status.CompletedMigrationStats).To(Equal(&api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}))
	})

	DescribeTable("should keep the KubeVirt metadata in every notification", func(domainMissing bool, xmlError error, fire func(*domainpublisher.DomainPublisher), failed bool) {
		if domainMissing {
			domainRemoved()
		} else {
			mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTOFF, int(libvirt.DOMAIN_SHUTOFF_MIGRATED), nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(domainXML(), xmlError).AnyTimes()
			mockLibvirt.DomainEXPECT().Free().AnyTimes()
		}
		backup := api.BackupMetadata{Name: "backup", Completed: true}
		metadataCache.Backup.Set(backup)
		startTime := metav1.NewTime(time.Unix(100, 0))
		end := metav1.NewTime(time.Unix(200, 0))
		expected := api.MigrationMetadata{
			UID:            "migration-1",
			StartTimestamp: &startTime,
			EndTimestamp:   &end,
			Mode:           v1.MigrationPostCopy,
			AbortStatus:    string(v1.MigrationAbortFailed),
			Failed:         failed,
		}
		if failed {
			expected.FailureReason = "migration failed"
		}
		metadataCache.Migration.Set(expected)
		metadataCache.CompletedMigration.Set(metadata.CompletedMigrationData{
			Stats:        api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150},
			MigrationUID: "migration-1",
		})

		for range 2 {
			fire(h.publisher)

			domain := h.receive().Object.(*api.Domain)
			Expect(domain.Spec.Metadata.KubeVirt.Migration).To(Equal(&expected))
			Expect(string(domain.Spec.Metadata.KubeVirt.UID)).To(Equal("1234"))
			Expect(domain.Spec.Metadata.KubeVirt.Backup).To(Equal(&backup))
			Expect(domain.Status.CompletedMigrationStats).To(Equal(&api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150}))
		}
	},
		Entry("with the domain present", false, nil, resumed, false),
		Entry("after the domain disappeared", true, nil, resumed, false),
		Entry("while the XML is unavailable", false, libvirt.Error{Code: libvirt.ERR_OPERATION_INVALID}, resumed, false),
		Entry("when the domain disappears before its XML is read", false, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}, resumed, false),
		Entry("for a failed migration", false, nil, resumed, true),
		Entry("on a delayed job-completed event", true, nil, func(p *domainpublisher.DomainPublisher) {
			p.OnJobCompleted(&libvirt.DomainEventJobCompleted{Info: libvirt.DomainJobInfo{Operation: libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT}})
		}, false),
		Entry("on a stopped-migrated event", true, nil, stoppedMigrated, false),
		Entry("on the final undefined event", true, nil, func(p *domainpublisher.DomainPublisher) {
			p.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED})
		}, false),
	)

	Context("guest panic", func() {
		crashed := func(detail libvirt.DomainEventCrashedDetailType) {
			h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_CRASHED, Detail: int(detail)})
		}

		BeforeEach(func() {
			domainExists(libvirt.DOMAIN_CRASHED, int(libvirt.DOMAIN_CRASHED_PANICKED))
		})

		It("should report unknown panic details when the QEMU log is missing", func() {
			crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED)

			domain := h.receive().Object.(*api.Domain)
			Expect(domain.Status.GuestPanicInfo).To(Equal(&api.GuestPanicInfo{Type: "unknown"}))
			Expect(domain.Status.PanicCount).To(Equal(1))
			Eventually(h.notifier.k8sEvents).Should(Receive(Equal(k8sEvent{
				severity: "Warning", reason: "GuestPanicked", message: "GuestPanicked (details unavailable)",
			})))
		})

		It("should count a PANICKED panic once", func() {
			crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED)
			crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED)

			Eventually(h.latest).Should(HaveField("Status.PanicCount", 1))
			Consistently(h.latest, 200*time.Millisecond).Should(Or(BeNil(), HaveField("Status.PanicCount", 1)))
		})

		It("should count every CRASHLOADED panic", func() {
			crashed(libvirt.DOMAIN_EVENT_CRASHED_CRASHLOADED)
			crashed(libvirt.DOMAIN_EVENT_CRASHED_CRASHLOADED)

			Eventually(h.latest).Should(HaveField("Status.PanicCount", 2))
		})

		It("should keep panic details in later notifications", func() {
			crashed(libvirt.DOMAIN_EVENT_CRASHED_PANICKED)
			h.receive()

			h.publisher.Resync()

			domain := h.receive().Object.(*api.Domain)
			Expect(domain.Status.GuestPanicInfo).To(Equal(&api.GuestPanicInfo{Type: "unknown"}))
			Expect(domain.Status.PanicCount).To(Equal(1))
		})
	})
})

type harness struct {
	publisher  *domainpublisher.DomainPublisher
	notifier   *fakeNotifier
	agentStore *agentpoller.AsyncAgentStore
	stop       func()
}

func startPublisher(lib *testing.Libvirt, metadataCache *metadata.Cache) *harness {
	agentStore := agentpoller.NewAsyncAgentStore()
	h := &harness{
		notifier:   newFakeNotifier(),
		agentStore: &agentStore,
	}
	vmi := libvmi.New(libvmi.WithName("test"), libvmi.WithNamespace(k8sv1.NamespaceDefault), libvmi.WithUID("1234"))
	agentPoller := agentpoller.CreatePoller(lib.VirtConnection, vmi.UID, "default_test", h.agentStore,
		10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second)
	h.publisher = domainpublisher.NewDomainPublisher(h.notifier, lib.VirtConnection, vmi, agentPoller, metadataCache, false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		h.publisher.Run(ctx)
		close(done)
	}()
	h.stop = func() {
		cancel()
		Eventually(done).Should(BeClosed())
	}
	DeferCleanup(h.stop)
	return h
}

func (h *harness) latest() *api.Domain {
	var latest *api.Domain
	for {
		select {
		case event := <-h.notifier.domainEvents:
			latest = event.Object.(*api.Domain)
		default:
			return latest
		}
	}
}

func (h *harness) receive() watch.Event {
	var event watch.Event
	EventuallyWithOffset(1, h.notifier.domainEvents, 2*time.Second).Should(Receive(&event))
	return event
}

type k8sEvent struct {
	severity string
	reason   string
	message  string
}

type fakeNotifier struct {
	domainEvents chan watch.Event
	k8sEvents    chan k8sEvent
	sendErrors   chan error
}

func newFakeNotifier() *fakeNotifier {
	return &fakeNotifier{
		domainEvents: make(chan watch.Event, 100),
		k8sEvents:    make(chan k8sEvent, 100),
		sendErrors:   make(chan error, 10),
	}
}

func (f *fakeNotifier) SendDomainEvent(_ context.Context, event watch.Event) error {
	select {
	case err := <-f.sendErrors:
		return err
	default:
	}
	f.domainEvents <- event
	return nil
}

func (f *fakeNotifier) SendK8sEvent(_ *v1.VirtualMachineInstance, severity string, reason string, message string) error {
	f.k8sEvents <- k8sEvent{severity: severity, reason: reason, message: message}
	return nil
}
