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

	stoppedMigrated := func(p *domainpublisher.DomainPublisher) {
		p.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_STOPPED, Detail: int(libvirt.DOMAIN_EVENT_STOPPED_MIGRATED)})
	}

	DescribeTable("should notify the domain state on lifecycle events", func(state libvirt.DomainState, event libvirt.DomainEventType, expectedState api.LifeCycle, expectedType watch.EventType) {
		domainExists(state, -1)

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: event})

		notification := h.receive()
		Expect(notification.Type).To(Equal(expectedType))
		domain := notification.Object.(*api.Domain)
		Expect(domain.Status.Status).To(Equal(expectedState))
		expectedSpec := api.NewMinimalDomain("test").Spec
		domain.Spec.XMLName = xml.Name{}
		Expect(equality.Semantic.DeepEqual(expectedSpec, domain.Spec)).To(BeTrue())
	},
		Entry("modified for crashed VMIs", libvirt.DOMAIN_CRASHED, libvirt.DOMAIN_EVENT_CRASHED, api.Crashed, watch.Modified),
		Entry("modified for stopped VMIs with shutoff reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_SHUTDOWN, api.Shutoff, watch.Modified),
		Entry("modified for stopped VMIs with stopped reason", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_STOPPED, api.Shutoff, watch.Modified),
		Entry("modified for running VMIs", libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_EVENT_STARTED, api.Running, watch.Modified),
		Entry("added for defined VMIs", libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_EVENT_DEFINED, api.Shutoff, watch.Added),
	)

	It("should not notify a resync before the first libvirt event", func() {

		h.publisher.Resync()

		Consistently(h.notifier.domainEvents, 500*time.Millisecond).ShouldNot(Receive())
	})

	It("should not notify guest agent updates before the first libvirt event", func() {
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
		domainGone()

		h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_UNDEFINED})

		notification := h.receive()
		Expect(notification.Type).To(Equal(watch.Modified))
		Expect(notification.Object.(*api.Domain).Status.Status).To(Equal(api.NoState))
		Expect(notification.Object.(*api.Domain).DeletionTimestamp).ToNot(BeNil())
		Eventually(h.deleteNotificationSent).Should(Receive())
	})

	DescribeTable("should not mark a gone domain deleted on", func(fire func(*domainpublisher.DomainPublisher)) {
		metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1"})
		domainGone()

		fire(h.publisher)

		domain := h.receive().Object.(*api.Domain)
		Expect(domain.Status.Reason).To(Equal(api.ReasonNonExistent))
		Expect(domain.DeletionTimestamp).To(BeNil())
		Consistently(h.deleteNotificationSent, 200*time.Millisecond).ShouldNot(Receive())
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
			mockLibvirt.DomainEXPECT().GetDiskErrors(uint32(0)).Return([]libvirt.DomainDiskError{
				{Disk: "vda", Error: libvirt.DOMAIN_DISK_ERROR_NO_SPACE},
			}, nil).AnyTimes()
		})

		It("should report the I/O error as a Kubernetes event", func() {

			h.publisher.OnLifecycle(&libvirt.DomainEventLifecycle{Event: libvirt.DOMAIN_EVENT_SUSPENDED, Detail: int(libvirt.DOMAIN_EVENT_SUSPENDED_IOERROR)})

			h.receive()
			Eventually(h.notifier.k8sEvents).Should(Receive(Equal(k8sEvent{
				severity: "Warning", reason: "IOerror", message: "VM Paused due to not enough space on volume: ",
			})))
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
			metadataCache.Backup.Store(api.BackupMetadata{})

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
			domainGone()
		})

		DescribeTable("should attach the downtime libvirt reports to its migration", func(info libvirt.DomainJobInfo, expected *api.CompletedMigrationStats) {
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1"})

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
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1"})
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
			metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1"})
			migrationCompleted(libvirt.DomainJobInfo{DowntimeSet: true, Downtime: 150})
			Expect(h.receive().Object.(*api.Domain).Status.CompletedMigrationStats).ToNot(BeNil())

			metadataCache.CompletedMigration.Store(metadata.CompletedMigrationData{})

			Eventually(h.latest).Should(HaveField("Status.CompletedMigrationStats", BeNil()))
		})
	})

	It("should include migration results updated while the domain XML is being read", func() {
		startTime := metav1.NewTime(time.Unix(100, 0))
		end := metav1.NewTime(time.Unix(200, 0))
		metadataCache.Migration.Store(api.MigrationMetadata{UID: "migration-1", StartTimestamp: &startTime})
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

	DescribeTable("should keep the tracked migration result in every notification", func(domainMissing bool, xmlError error, fire func(*domainpublisher.DomainPublisher), failed bool) {
		if domainMissing {
			domainGone()
		} else {
			mockLibvirt.ConnectionEXPECT().LookupDomainByName("default_test").Return(mockLibvirt.VirtDomain, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTOFF, int(libvirt.DOMAIN_SHUTOFF_MIGRATED), nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(domainXML(), xmlError).AnyTimes()
			mockLibvirt.DomainEXPECT().Free().AnyTimes()
		}
		metadataCache.UID.Set("1234")
		startTime := metav1.NewTime(time.Unix(100, 0))
		end := metav1.NewTime(time.Unix(200, 0))
		metadataCache.Migration.Set(api.MigrationMetadata{UID: "migration-1", StartTimestamp: &startTime, Mode: v1.MigrationPreCopy})
		metadataCache.CompletedMigration.Set(metadata.CompletedMigrationData{
			Stats:        api.CompletedMigrationStats{DowntimeSet: true, Downtime: 150},
			MigrationUID: "migration-1",
		})
		metadataCache.Migration.WithSafeBlock(func(migration *api.MigrationMetadata, _ bool) {
			migration.EndTimestamp = &end
			migration.Mode = v1.MigrationPostCopy
			migration.AbortStatus = string(v1.MigrationAbortFailed)
			migration.Failed = failed
			if failed {
				migration.FailureReason = "migration failed"
			}
		})
		expected, _ := metadataCache.Migration.Load()

		for range 2 {
			fire(h.publisher)

			domain := h.receive().Object.(*api.Domain)
			Expect(domain.Spec.Metadata.KubeVirt.Migration).To(Equal(&expected))
			Expect(string(domain.Spec.Metadata.KubeVirt.UID)).To(Equal("1234"))
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
	publisher              *domainpublisher.DomainPublisher
	notifier               *fakeNotifier
	agentStore             *agentpoller.AsyncAgentStore
	deleteNotificationSent chan watch.Event
	stop                   func()
}

func startPublisher(lib *testing.Libvirt, metadataCache *metadata.Cache) *harness {
	agentStore := agentpoller.NewAsyncAgentStore()
	h := &harness{
		notifier:               newFakeNotifier(),
		agentStore:             &agentStore,
		deleteNotificationSent: make(chan watch.Event, 100),
	}
	vmi := libvmi.New(libvmi.WithName("test"), libvmi.WithNamespace(k8sv1.NamespaceDefault), libvmi.WithUID("1234"))
	agentPoller := agentpoller.CreatePoller(lib.VirtConnection, vmi.UID, "default_test", h.agentStore,
		10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second, 10*time.Second)
	h.publisher = domainpublisher.NewDomainPublisher(h.notifier, lib.VirtConnection, vmi, agentPoller, metadataCache, false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		h.publisher.Run(ctx, h.deleteNotificationSent)
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
