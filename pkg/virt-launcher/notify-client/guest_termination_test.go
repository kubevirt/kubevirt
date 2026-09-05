package eventsclient

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func fixedTerminationEventTime() metav1.Time {
	return metav1.NewTime(time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC))
}

var _ = Describe("Termination events", func() {
	It("marks guest initiated shutdown events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonGuestShutdown))
		Expect(event.Timestamp.IsZero()).To(BeFalse())
	})

	It("marks host initiated shutdown events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_HOST),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonHostShutdown))
	})

	It("marks fresh platform requested shutdown events", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: now,
		})

		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_HOST),
			},
		}, cache, now)

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonPlatformRequestedShutdown))
	})

	It("consumes fresh platform termination intents after classification", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: now,
		})

		firstEvent := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, cache, now)
		secondEvent := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, cache, now)

		Expect(firstEvent).ToNot(BeNil())
		Expect(firstEvent.Reason).To(Equal(api.TerminationReasonPlatformRequestedShutdown))
		Expect(secondEvent).ToNot(BeNil())
		Expect(secondEvent.Reason).To(Equal(api.TerminationReasonGuestShutdown))
	})

	It("does not reuse a platform termination intent after a crash and domain restart", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: now,
		})
		domain := api.NewMinimalDomain("test")

		crashEvent := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_CRASHED,
				Detail: int(libvirt.DOMAIN_EVENT_CRASHED_PANICKED),
			},
		}, cache, now)

		clearStaleTerminationStateOnDomainStart(domain, cache, libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STARTED,
				Detail: int(libvirt.DOMAIN_EVENT_STARTED_BOOTED),
			},
		})

		shutdownEvent := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, cache, now)

		Expect(crashEvent).ToNot(BeNil())
		Expect(crashEvent.Reason).To(Equal(api.TerminationReasonGuestCrashed))
		Expect(shutdownEvent).ToNot(BeNil())
		Expect(shutdownEvent.Reason).To(Equal(api.TerminationReasonGuestShutdown))
	})

	It("keeps observed termination events on later low-signal domain notifications", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: now,
		})
		domain := api.NewMinimalDomain("test")

		terminationEvent := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_HOST),
			},
		}, cache, now)
		cacheAndSetTerminationEvent(domain, cache, terminationEvent)

		lowSignalEvent := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STOPPED,
				Detail: int(libvirt.DOMAIN_EVENT_STOPPED_DESTROYED),
			},
		}, cache, now)
		nextDomainNotification := api.NewMinimalDomain("test")
		setCachedTerminationEvent(nextDomainNotification, cache)

		Expect(terminationEvent).ToNot(BeNil())
		Expect(terminationEvent.Reason).To(Equal(api.TerminationReasonPlatformRequestedShutdown))
		Expect(lowSignalEvent).To(BeNil())
		Expect(nextDomainNotification.Status.TerminationEvent).ToNot(BeNil())
		Expect(nextDomainNotification.Status.TerminationEvent.Reason).To(Equal(api.TerminationReasonPlatformRequestedShutdown))
		Expect(nextDomainNotification.Status.TerminationEvent.Timestamp).To(Equal(now))
	})

	It("clears stale platform termination intents without using them", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: metav1.NewTime(now.Add(-platformTerminationIntentFreshness - time.Second)),
		})

		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, cache, now)
		intent, exists := cache.PendingPlatformTermination.Load()

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonGuestShutdown))
		Expect(exists).To(BeTrue())
		Expect(intent).To(Equal(api.PendingPlatformTerminationIntent{}))
	})

	It("keeps platform termination intents fresh within the configured grace period", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.GracePeriod.Store(api.GracePeriodMetadata{
			DeletionGracePeriodSeconds: int64((5 * time.Minute).Seconds()),
		})
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: metav1.NewTime(now.Add(-platformTerminationIntentFreshness - time.Second)),
		})

		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, cache, now)

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonPlatformRequestedShutdown))
	})

	It("clears platform termination intents stale beyond the configured grace period", func() {
		now := fixedTerminationEventTime()
		cache := metadata.NewCache()
		cache.GracePeriod.Store(api.GracePeriodMetadata{
			DeletionGracePeriodSeconds: int64((5 * time.Minute).Seconds()),
		})
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: metav1.NewTime(now.Add(-5*time.Minute - platformTerminationIntentGracePeriodMargin - time.Second)),
		})

		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_SHUTDOWN,
				Detail: int(libvirt.DOMAIN_EVENT_SHUTDOWN_GUEST),
			},
		}, cache, now)
		intent, exists := cache.PendingPlatformTermination.Load()

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonGuestShutdown))
		Expect(exists).To(BeTrue())
		Expect(intent).To(Equal(api.PendingPlatformTerminationIntent{}))
	})

	It("marks guest panics as guest crashed events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_CRASHED,
				Detail: int(libvirt.DOMAIN_EVENT_CRASHED_PANICKED),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonGuestCrashed))
		Expect(event.Timestamp.IsZero()).To(BeFalse())
	})

	It("marks guest crashloaded events as guest crashed events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_CRASHED,
				Detail: int(libvirt.DOMAIN_EVENT_CRASHED_CRASHLOADED),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonGuestCrashed))
	})

	It("marks stopped crashed events as guest crashed events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STOPPED,
				Detail: int(libvirt.DOMAIN_EVENT_STOPPED_CRASHED),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonGuestCrashed))
		Expect(event.Timestamp.IsZero()).To(BeFalse())
	})

	It("marks stopped failed events as host stopped failed events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STOPPED,
				Detail: int(libvirt.DOMAIN_EVENT_STOPPED_FAILED),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).ToNot(BeNil())
		Expect(event.Reason).To(Equal(api.TerminationReasonHostStoppedFailed))
		Expect(event.Timestamp.IsZero()).To(BeFalse())
	})

	It("clears stale termination state on started lifecycle events", func() {
		domain := api.NewMinimalDomain("test")
		domain.Status.TerminationEvent = &api.TerminationEvent{
			Reason:    api.TerminationReasonGuestShutdown,
			Timestamp: metav1.Now(),
		}
		cache := metadata.NewCache()
		cache.PendingPlatformTermination.Set(api.PendingPlatformTerminationIntent{
			Timestamp: fixedTerminationEventTime(),
		})
		cache.ObservedTerminationEvent.Set(api.TerminationEvent{
			Reason:    api.TerminationReasonGuestShutdown,
			Timestamp: fixedTerminationEventTime(),
		})

		clearStaleTerminationStateOnDomainStart(domain, cache, libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STARTED,
				Detail: int(libvirt.DOMAIN_EVENT_STARTED_BOOTED),
			},
		})
		intent, exists := cache.PendingPlatformTermination.Load()
		terminationEvent, terminationEventExists := cache.ObservedTerminationEvent.Load()

		Expect(domain.Status.TerminationEvent).To(BeNil())
		Expect(exists).To(BeTrue())
		Expect(intent).To(Equal(api.PendingPlatformTerminationIntent{}))
		Expect(terminationEventExists).To(BeTrue())
		Expect(terminationEvent).To(Equal(api.TerminationEvent{}))
	})

	It("keeps termination state on non-started lifecycle events", func() {
		timestamp := metav1.Now()
		domain := api.NewMinimalDomain("test")
		domain.Status.TerminationEvent = &api.TerminationEvent{
			Reason:    api.TerminationReasonGuestShutdown,
			Timestamp: timestamp,
		}
		cache := metadata.NewCache()
		intent := api.PendingPlatformTerminationIntent{
			Timestamp: fixedTerminationEventTime(),
		}
		terminationEvent := api.TerminationEvent{
			Reason:    api.TerminationReasonGuestShutdown,
			Timestamp: timestamp,
		}
		cache.PendingPlatformTermination.Set(intent)
		cache.ObservedTerminationEvent.Set(terminationEvent)

		clearStaleTerminationStateOnDomainStart(domain, cache, libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_STOPPED,
				Detail: int(libvirt.DOMAIN_EVENT_STOPPED_SHUTDOWN),
			},
		})
		pendingIntent, exists := cache.PendingPlatformTermination.Load()
		pendingTerminationEvent, terminationEventExists := cache.ObservedTerminationEvent.Load()

		Expect(domain.Status.TerminationEvent).To(Equal(&api.TerminationEvent{
			Reason:    api.TerminationReasonGuestShutdown,
			Timestamp: timestamp,
		}))
		Expect(exists).To(BeTrue())
		Expect(pendingIntent).To(Equal(intent))
		Expect(terminationEventExists).To(BeTrue())
		Expect(pendingTerminationEvent).To(Equal(terminationEvent))
	})

	DescribeTable("ignores low-signal stopped events",
		func(detail libvirt.DomainEventStoppedDetailType) {
			event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
				Event: &libvirt.DomainEventLifecycle{
					Event:  libvirt.DOMAIN_EVENT_STOPPED,
					Detail: int(detail),
				},
			}, metadata.NewCache(), fixedTerminationEventTime())

			Expect(event).To(BeNil())
		},
		Entry("destroyed", libvirt.DOMAIN_EVENT_STOPPED_DESTROYED),
		Entry("shutdown", libvirt.DOMAIN_EVENT_STOPPED_SHUTDOWN),
		Entry("migrated", libvirt.DOMAIN_EVENT_STOPPED_MIGRATED),
	)

	It("ignores unsupported events", func() {
		event := guestTerminationEventFromLibvirtEventAt(libvirtEvent{
			Event: &libvirt.DomainEventLifecycle{
				Event:  libvirt.DOMAIN_EVENT_DEFINED,
				Detail: int(libvirt.DOMAIN_EVENT_DEFINED_ADDED),
			},
		}, metadata.NewCache(), fixedTerminationEventTime())

		Expect(event).To(BeNil())
	})
})
