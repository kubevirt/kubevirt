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
	"context"
	"fmt"
	"sync"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type domainNotifier interface {
	SendDomainEvent(event watch.Event) error
	SendK8sEvent(vmi *v1.VirtualMachineInstance, severity string, reason string, message string) error
}

type DomainPublisher struct {
	notifier       domainNotifier
	connection     cli.Connection
	vmi            *v1.VirtualMachineInstance
	domainName     string
	domainTemplate *api.Domain
	agentPoller    *agentpoller.AgentPoller
	metadataCache  *metadata.Cache
	nonRoot        bool

	queue                    workqueue.TypedInterface[any]
	domainSeen               bool
	domainStatus             api.LifeCycle
	domainStatusChangeReason api.StateChangeReason
	firstAdd                 sync.Once
	firstDelete              sync.Once
}

func NewDomainPublisher(
	notifier domainNotifier,
	connection cli.Connection,
	vmi *v1.VirtualMachineInstance,
	agentPoller *agentpoller.AgentPoller,
	metadataCache *metadata.Cache,
	nonRoot bool,
) *DomainPublisher {
	domainName := api.VMINamespaceKeyFunc(vmi)
	return &DomainPublisher{
		notifier:       notifier,
		connection:     connection,
		vmi:            vmi,
		domainName:     domainName,
		domainTemplate: util.NewDomainFromName(domainName, vmi.UID),
		agentPoller:    agentPoller,
		metadataCache:  metadataCache,
		nonRoot:        nonRoot,
		queue:          workqueue.NewTyped[any](),
	}
}

func (p *DomainPublisher) OnLifecycle(event *libvirt.DomainEventLifecycle) {
	p.queue.Add(event)
}

func (p *DomainPublisher) OnJobCompleted(event *libvirt.DomainEventJobCompleted) {
	if event.Info.Operation == libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT {
		storeCompletedMigrationStats(&event.Info, p.metadataCache)
	}
	p.queue.Add(event)
}

func (p *DomainPublisher) OnAgentLifecycle(event *libvirt.DomainEventAgentLifecycle) {
	p.queue.Add(event)
}

// Resync publishes a fresh notification of the domain once libvirt has defined it.
func (p *DomainPublisher) Resync() {
	p.queue.Add(resync{})
}

func (p *DomainPublisher) updateEvents(event watch.Event, domain *api.Domain, events chan watch.Event) {
	if event.Type == watch.Added {
		p.firstAdd.Do(func() { events <- event })
	} else if event.Type == watch.Modified && domain.ObjectMeta.DeletionTimestamp != nil {
		p.firstDelete.Do(func() { events <- event })
	}
}

func (p *DomainPublisher) printStatus(status *api.DomainStatus) {
	v := 2
	if status.Status == p.domainStatus && status.Reason == p.domainStatusChangeReason {
		// Status hasn't changed so log only in higher verbosity.
		v = 3
	}
	log.Log.V(v).Infof("kubevirt domain status: %v(%v) reason: %v(%v)", status.Status, p.domainStatus, status.Reason, p.domainStatusChangeReason)
}

func (p *DomainPublisher) updateStatus(status *api.DomainStatus) {
	p.domainStatus = status.Status
	p.domainStatusChangeReason = status.Reason
}

func (p *DomainPublisher) recordGuestPanic(eventDetail int) {
	if guestPanic, exists := p.metadataCache.GuestPanic.Load(); exists && guestPanic.Handled {
		log.Log.V(3).Info("Guest panic event already handled, skipping")
		return
	}

	logPath := util.GetQemuLogPath(p.domainName, p.nonRoot)

	panicInfo, err := util.ReadPanicInfoFromLog(logPath)

	var eventMessage string
	if panicInfo == nil || err != nil || panicInfo.Type == "" {
		if err != nil {
			log.Log.Reason(err).Warning("Failed to read panic info from log")
		}
		eventMessage = "GuestPanicked (details unavailable)"
		panicInfo = &api.GuestPanicInfo{Type: "unknown"}
	} else {
		eventMessage = util.FormatGuestPanicInfo(panicInfo)
	}
	log.Log.Infof("Guest panic detected: %s", eventMessage)

	// Only mark as handled for PANICKED events. CRASHLOADED indicates kdump-based
	// recovery where the guest reboots, so subsequent panic events should still fire.
	p.metadataCache.GuestPanic.WithSafeBlock(func(guestPanic *metadata.GuestPanicData, _ bool) {
		guestPanic.Handled = libvirt.DomainEventCrashedDetailType(eventDetail) == libvirt.DOMAIN_EVENT_CRASHED_PANICKED
		guestPanic.Count++
		guestPanic.Info = panicInfo
	})

	if err := p.notifier.SendK8sEvent(p.vmi, k8sv1.EventTypeWarning, "GuestPanicked",
		eventMessage); err != nil {
		log.Log.Reason(err).Warningf("Failed to send guest panic event")
	}
}

func (p *DomainPublisher) eventCallback(event any, events chan watch.Event) {
	// Handle guest panic event early, before domain lookup which may fail if VM is already gone
	if lifecycle, ok := event.(*libvirt.DomainEventLifecycle); ok && lifecycle.Event == libvirt.DOMAIN_EVENT_CRASHED {
		p.recordGuestPanic(lifecycle.Detail)
	}
	// libvirt emits lifecycle events only for a domain it has defined.
	if _, ok := event.(*libvirt.DomainEventLifecycle); ok {
		p.domainSeen = true
	}

	domain := p.domainTemplate.DeepCopy()
	d, err := p.connection.LookupDomainByName(p.domainName)
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			log.Log.Reason(err).Error("Could not fetch the Domain.")
			return
		}
		if !p.domainSeen {
			log.Log.Object(p.vmi).V(3).Info("Domain is not defined yet, nothing to publish")
			return
		}
		domain.SetState(api.NoState, api.ReasonNonExistent)
	} else {
		defer d.Free()
		p.domainSeen = true

		// No matter which event, try to fetch the domain xml
		// and the state. If we get a IsNotFound error, that
		// means that the VirtualMachineInstance was removed.
		status, reason, err := d.GetState()
		if err != nil {
			if !domainerrors.IsNotFound(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain state.")
				return
			}
			domain.SetState(api.NoState, api.ReasonNonExistent)
		} else {
			domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		}

		kubevirtMetadata := metadata.LoadKubevirtMetadata(p.metadataCache)
		spec, err := util.GetDomainSpecWithRuntimeInfo(d)
		if err != nil {
			// NOTE: Getting domain metadata for a live-migrating VM isn't allowed
			if !domainerrors.IsNotFound(err) && !domainerrors.IsInvalidOperation(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain specification.")
				return
			}
		} else {
			domain.ObjectMeta.UID = kubevirtMetadata.UID
		}

		if spec != nil {
			spec.Metadata.KubeVirt = kubevirtMetadata
			domain.Spec = *spec
		}

		p.printStatus(&domain.Status)
		p.updateStatus(&domain.Status)
	}

	eventType := watch.Modified

	switch domain.Status.Reason {
	case api.ReasonPausedIOError:
		domainDisksWithErrors, err := d.GetDiskErrors(0)
		if err != nil {
			log.Log.Reason(err).Error("Could not get disks with errors")
		}
		for _, disk := range domainDisksWithErrors {
			volumeName := getVolumeNameByTarget(domain, disk.Disk)
			var reasonError string
			switch disk.Error {
			case libvirt.DOMAIN_DISK_ERROR_NONE:
				continue
			case libvirt.DOMAIN_DISK_ERROR_UNSPEC:
				reasonError = fmt.Sprintf("VM Paused due to IO error at the volume: %s", volumeName)
			case libvirt.DOMAIN_DISK_ERROR_NO_SPACE:
				reasonError = fmt.Sprintf("VM Paused due to not enough space on volume: %s", volumeName)
			}
			err = p.notifier.SendK8sEvent(p.vmi, "Warning", "IOerror", reasonError)
			if err != nil {
				log.Log.Reason(err).Error(fmt.Sprintf("Could not send k8s event"))
			}
		}
	}

	switch e := event.(type) {
	case *libvirt.DomainEventJobCompleted:
		if shouldAdd := processJobCompletedEvent(domain, d, e, p.metadataCache); shouldAdd {
			eventType = watch.Added
		}
	case *libvirt.DomainEventLifecycle:
		if shouldAdd := processLifecycleEvent(domain, e, p.metadataCache, p.connection, p.vmi); shouldAdd {
			eventType = watch.Added
		}
	}

	agentStore := p.agentPoller.Store()
	if interfaces := agentStore.GetInterfaceStatus(); interfaces != nil {
		domain.Status.Interfaces = interfaces
	}
	if osInfo := agentStore.GetGuestOSInfo(); osInfo != nil {
		domain.Status.OSInfo = *osInfo
	}
	if fsFreezeStatus := agentStore.GetFSFreezeStatus(); fsFreezeStatus != nil {
		domain.Status.FSFreezeStatus = *fsFreezeStatus
	}
	// Lifecycle metadata can advance after JOB_COMPLETED and remains available
	// even after the source domain is gone.
	domain.Spec.Metadata.KubeVirt.Migration = nil
	if migration, exists := p.metadataCache.Migration.Load(); exists {
		domain.Spec.Metadata.KubeVirt.Migration = &migration
	}
	applyCompletedMigrationStats(domain, p.metadataCache)
	if guestPanic, exists := p.metadataCache.GuestPanic.Load(); exists {
		domain.Status.GuestPanicInfo = guestPanic.Info
		domain.Status.PanicCount = guestPanic.Count
	}

	watchEvent := watch.Event{Type: eventType, Object: domain}

	if err := p.notifier.SendDomainEvent(watchEvent); err != nil {
		log.Log.Reason(err).Error("Could not send domain notify event.")
	}
	p.updateEvents(watchEvent, domain, events)
}

func processJobCompletedEvent(domain *api.Domain, d cli.VirDomain, jobCompletedEvent *libvirt.DomainEventJobCompleted, metadataCache *metadata.Cache) bool {
	switch jobCompletedEvent.Info.Operation {
	case libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT:
		if migration, exists := metadataCache.Migration.Load(); exists {
			virtwrap.LogMigrationInfo(log.Log, migration.UID, &jobCompletedEvent.Info)
		} else {
			log.Log.Warning("Received migration completed event, but no migration is being tracked")
		}
		return false
	case libvirt.DOMAIN_JOB_OPERATION_BACKUP:
		storage.HandleBackupJobCompletedEvent(d, jobCompletedEvent, metadataCache)
		if value, exists := metadataCache.Backup.Load(); exists {
			domain.Spec.Metadata.KubeVirt.Backup = &value
		}
		return true
	default:
		log.Log.V(3).Infof("Received a job completion event for operation %v", jobCompletedEvent.Info.Operation)
		return false
	}
}

func storeCompletedMigrationStats(jobInfo *libvirt.DomainJobInfo, metadataCache *metadata.Cache) {
	// Migration-out completion is emitted only on success; JobSuccess is not emitted for migration stats.
	if !jobInfo.DowntimeSet {
		log.Log.Warning("Ignoring completed migration stats because libvirt did not report a downtime")
		return
	}
	migration, exists := metadataCache.Migration.Load()
	if !exists || migration.UID == "" {
		log.Log.Error("Ignoring completed migration stats because no migration is being tracked")
		return
	}

	metadataCache.CompletedMigration.Store(metadata.CompletedMigrationData{
		Stats: api.CompletedMigrationStats{
			DowntimeSet: jobInfo.DowntimeSet,
			Downtime:    jobInfo.Downtime,
		},
		MigrationUID: migration.UID,
	})
}

func applyCompletedMigrationStats(domain *api.Domain, metadataCache *metadata.Cache) {
	domain.Status.CompletedMigrationStats = nil
	completedMigration, exists := metadataCache.CompletedMigration.Load()
	if !exists || !completedMigration.Stats.DowntimeSet {
		return
	}

	// Match the snapshot being sent, not a separate read of the advancing cache.
	migration := domain.Spec.Metadata.KubeVirt.Migration
	if migration == nil || completedMigration.MigrationUID == "" || migration.UID != completedMigration.MigrationUID {
		return
	}

	domain.Status.CompletedMigrationStats = &completedMigration.Stats
}

func processLifecycleEvent(domain *api.Domain, lifecycleEvent *libvirt.DomainEventLifecycle, metadataCache *metadata.Cache, c cli.Connection, vmi *v1.VirtualMachineInstance) bool {
	// Set DeletionTimestamp when the domain is truly gone from libvirt.
	// Undefined is terminal for persistent domains (emitted by explicit undefine).
	// Stopped is terminal for transient domains (e.g. migration target).
	// Stopped(Migrated) is excluded because it only fires on the persistent source
	// after a successful outbound migration, where Undefined follows.
	if (lifecycleEvent.Event == libvirt.DOMAIN_EVENT_UNDEFINED) ||
		(lifecycleEvent.Event == libvirt.DOMAIN_EVENT_STOPPED && libvirt.DomainEventStoppedDetailType(lifecycleEvent.Detail) != libvirt.DOMAIN_EVENT_STOPPED_MIGRATED) {
		if domain.Status.Reason == api.ReasonNonExistent {
			now := metav1.Now()
			domain.ObjectMeta.DeletionTimestamp = &now
		}
		return false
	}
	if lifecycleEvent.Event == libvirt.DOMAIN_EVENT_DEFINED &&
		libvirt.DomainEventDefinedDetailType(lifecycleEvent.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
		return true
	}
	if lifecycleEvent.Event == libvirt.DOMAIN_EVENT_STARTED &&
		libvirt.DomainEventStartedDetailType(lifecycleEvent.Detail) == libvirt.DOMAIN_EVENT_STARTED_MIGRATED {
		return true
	}
	if (lifecycleEvent.Event == libvirt.DOMAIN_EVENT_RESUMED && libvirt.DomainEventResumedDetailType(lifecycleEvent.Detail) == libvirt.DOMAIN_EVENT_RESUMED_MIGRATED) ||
		(lifecycleEvent.Event == libvirt.DOMAIN_EVENT_SUSPENDED && libvirt.DomainEventSuspendedDetailType(lifecycleEvent.Detail) == libvirt.DOMAIN_EVENT_SUSPENDED_PAUSED) {
		// This is a libvirt event that only the target can see, and it means that the migration has completed
		// we just set the EndTimestamp here so that the source can finalize the migration.
		// Usually this is performed by the source launcher/handler. However, in case of upgrade, this is not
		// guaranteed as the cluster will have an updated virt-handler together with outdated launchers, this
		// makes sure that migrations actually finish in those cases.
		monitor := virtwrap.NewTargetMigrationMonitor(c, log.Log.Object(vmi), util.DomainFromNamespaceName(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name), metadataCache)
		monitor.StartMonitor()
	}
	return false
}

// getVolumeNameByTarget returns the volume name associated to the device target in the domain (e.g vda)
// One should not rely on target (vda) as API, use disk serials instead.
func getVolumeNameByTarget(domain *api.Domain, target string) string {
	for _, d := range domain.Spec.Devices.Disks {
		if d.Target.Device == target {
			return d.Alias.GetName()
		}
	}
	return ""
}

type resync struct{}

// Run publishes domain notifications until ctx is cancelled.
func (p *DomainPublisher) Run(ctx context.Context, deleteNotificationSent chan watch.Event) {
	defer p.agentPoller.Stop()

	go p.resyncOnSignals(ctx)
	go func() {
		<-ctx.Done()
		p.queue.ShutDown()
	}()

	for p.processNext(deleteNotificationSent) {
	}
}

func (p *DomainPublisher) processNext(deleteNotificationSent chan watch.Event) bool {
	item, quit := p.queue.Get()
	if quit {
		return false
	}
	defer p.queue.Done(item)

	p.eventCallback(item, deleteNotificationSent)

	switch e := item.(type) {
	case *libvirt.DomainEventLifecycle:
		p.agentPoller.OnDomainLifecycle(e.Event)
	case *libvirt.DomainEventAgentLifecycle:
		p.agentPoller.OnAgentLifecycle(e.State)
	}
	return true
}

func (p *DomainPublisher) resyncOnSignals(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.agentPoller.Store().AgentUpdated:
			p.Resync()
		case <-p.metadataCache.Listen():
			p.Resync()
		}
	}
}
