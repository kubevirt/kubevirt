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
	"fmt"
	"sync"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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

const (
	cantDetermineLibvirtDomainName = "Could not determine name of libvirt domain in event callback."
	libvirtEventChannelFull        = "Libvirt event channel is full, dropping event."
)

type libvirtEvent struct {
	Domain            string
	Event             *libvirt.DomainEventLifecycle
	AgentEvent        *libvirt.DomainEventAgentLifecycle
	JobCompletedEvent *libvirt.DomainEventJobCompleted
}

type domainNotifier interface {
	SendDomainEvent(event watch.Event) error
	SendK8sEvent(vmi *v1.VirtualMachineInstance, severity string, reason string, message string) error
}

type DomainPublisher struct {
	notifier      domainNotifier
	connection    cli.Connection
	vmi           *v1.VirtualMachineInstance
	domainName    string
	agentPoller   *agentpoller.AgentPoller
	metadataCache *metadata.Cache
	nonRoot       bool

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
	return &DomainPublisher{
		notifier:      notifier,
		connection:    connection,
		vmi:           vmi,
		domainName:    api.VMINamespaceKeyFunc(vmi),
		agentPoller:   agentPoller,
		metadataCache: metadataCache,
		nonRoot:       nonRoot,
	}
}

func (p *DomainPublisher) Start(deleteNotificationSent chan watch.Event) error {
	eventChan := make(chan libvirtEvent, 10)

	reconnectChan := make(chan bool, 10)

	p.connection.SetReconnectChan(reconnectChan)

	// Run the event process logic in a separate go-routine to not block libvirt
	go p.worker(eventChan, deleteNotificationSent, reconnectChan)

	return registerLibvirtCallbacks(eventChan, p.metadataCache, p.domainName, p.connection)
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

func isGuestPanicEvent(event *libvirt.DomainEventLifecycle) bool {
	return event != nil && event.Event == libvirt.DOMAIN_EVENT_CRASHED
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

func (p *DomainPublisher) eventCallback(domain *api.Domain, libvirtEvent libvirtEvent, events chan watch.Event,
	interfaceStatus []api.InterfaceStatus, osInfo *api.GuestOSInfo, fsFreezeStatus *api.FSFreeze) {
	// Handle guest panic event early, before domain lookup which may fail if VM is already gone
	if isGuestPanicEvent(libvirtEvent.Event) {
		p.recordGuestPanic(libvirtEvent.Event.Detail)
	}

	d, err := p.connection.LookupDomainByName(util.DomainFromNamespaceName(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name))
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			log.Log.Reason(err).Error("Could not fetch the Domain.")
			return
		}
		domain.SetState(api.NoState, api.ReasonNonExistent)
	} else {
		defer d.Free()

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

	switch {
	case libvirtEvent.JobCompletedEvent != nil:
		if shouldAdd := processJobCompletedEvent(domain, d, libvirtEvent.JobCompletedEvent, p.metadataCache); shouldAdd {
			eventType = watch.Added
		}
	case libvirtEvent.Event != nil:
		if shouldAdd := processLifecycleEvent(domain, libvirtEvent.Event, p.metadataCache, p.connection, p.vmi); shouldAdd {
			eventType = watch.Added
		}
	}

	if interfaceStatus != nil {
		domain.Status.Interfaces = interfaceStatus
	}
	if osInfo != nil {
		domain.Status.OSInfo = *osInfo
	}

	if fsFreezeStatus != nil {
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

	event := watch.Event{Type: eventType, Object: domain}

	if err := p.notifier.SendDomainEvent(event); err != nil {
		log.Log.Reason(err).Error("Could not send domain notify event.")
	}
	p.updateEvents(event, domain, events)
}

func registerLibvirtCallbacks(eventChan chan libvirtEvent, metadataCache *metadata.Cache, domainName string, domainConn cli.Connection) error {
	domainEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {

		log.Log.Infof("DomainLifecycle event %s with event id %d reason %d received", event.String(), event.Event, event.Detail)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}
		select {
		case eventChan <- libvirtEvent{Event: event, Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	domainEventDeviceAddedCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceAdded) {
		log.Log.Infof("Domain Device Added event received")
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}
		select {
		case eventChan <- libvirtEvent{Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	domainEventDeviceRemovedCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceRemoved) {
		log.Log.Infof("Domain Device Removed event received")
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}

		select {
		case eventChan <- libvirtEvent{Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	domainEventMemoryDeviceSizeChange := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventMemoryDeviceSizeChange) {
		log.Log.Infof("Domain Memory Device size-change event received")
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}

		select {
		case eventChan <- libvirtEvent{Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}
	domainEventJobCompletedCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventJobCompleted) {
		log.Log.Infof("Domain Job Completed event type %v received. Job operation: %v, success reported: %t, succeeded: %t", event.Info.Type, event.Info.Operation, event.Info.JobSuccessSet, event.Info.JobSuccess)
		if event.Info.Operation == libvirt.DOMAIN_JOB_OPERATION_MIGRATION_OUT {
			storeCompletedMigrationStats(&event.Info, metadataCache)
		}
		// The domain may already be gone when the terminal callback runs, so use the name captured when callbacks were registered.
		select {
		case eventChan <- libvirtEvent{JobCompletedEvent: event, Domain: domainName}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}

	err := domainConn.DomainEventLifecycleRegister(domainEventLifecycleCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}

	err = domainConn.DomainEventDeviceAddedRegister(domainEventDeviceAddedCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register device added event callback with libvirt")
		return err
	}
	err = domainConn.DomainEventDeviceRemovedRegister(domainEventDeviceRemovedCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register device removed event callback with libvirt")
		return err
	}
	err = domainConn.DomainEventMemoryDeviceSizeChangeRegister(domainEventMemoryDeviceSizeChange)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register memory device size change event callback with libvirt")
		return err
	}
	err = domainConn.DomainEventJobCompletedRegister(domainEventJobCompletedCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event job completed callback with libvirt")
		return err
	}

	agentEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventAgentLifecycle) {
		log.Log.Infof("GuestAgentLifecycle event state %d with reason %d received", event.State, event.Reason)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info(cantDetermineLibvirtDomainName)
		}
		select {
		case eventChan <- libvirtEvent{AgentEvent: event, Domain: name}:
		default:
			log.Log.Infof(libvirtEventChannelFull)
		}
	}
	err = domainConn.AgentEventLifecycleRegister(agentEventLifecycleCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return err
	}

	log.Log.Infof("Registered libvirt event notify callback")
	return nil
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

func (p *DomainPublisher) worker(eventChan chan libvirtEvent, deleteNotificationSent chan watch.Event, reconnectChan chan bool) {
	var interfaceStatuses []api.InterfaceStatus
	var guestOsInfo *api.GuestOSInfo
	var fsFreezeStatus *api.FSFreeze
	var domainCache *api.Domain

	for {
		select {
		case event := <-eventChan:
			p.metadataCache.ResetNotification()
			domainCache = util.NewDomainFromName(event.Domain, p.vmi.UID)
			p.eventCallback(domainCache, event, deleteNotificationSent, interfaceStatuses, guestOsInfo, fsFreezeStatus)
			log.Log.Infof("Domain name event: %v", domainCache.Spec.Name)
			switch {
			case event.Event != nil:
				p.agentPoller.OnDomainLifecycle(event.Event.Event)
			case event.AgentEvent != nil:
				p.agentPoller.OnAgentLifecycle(event.AgentEvent.State)
			}
		case agentUpdate := <-p.agentPoller.Store().AgentUpdated:
			p.metadataCache.ResetNotification()
			interfaceStatuses = agentUpdate.DomainInfo.Interfaces
			guestOsInfo = agentUpdate.DomainInfo.OSInfo
			fsFreezeStatus = agentUpdate.DomainInfo.FSFreezeStatus

			p.eventCallback(domainCache, libvirtEvent{}, deleteNotificationSent, interfaceStatuses, guestOsInfo, fsFreezeStatus)
		case <-reconnectChan:
			log.Log.Infof("Libvirt reconnected, domain %s. Event callbacks re-registered, triggering immediate reconciliation.", p.domainName)
			if domainCache != nil {
				p.eventCallback(domainCache, libvirtEvent{}, deleteNotificationSent, interfaceStatuses, guestOsInfo, fsFreezeStatus)
			}
		case <-p.metadataCache.Listen():
			// Metadata cache updates should be processed only *after* at least one
			// libvirt event arrived (which creates the first domainCache).
			if domainCache != nil {
				domainCache = util.NewDomainFromName(
					util.DomainFromNamespaceName(domainCache.ObjectMeta.Namespace, domainCache.ObjectMeta.Name),
					p.vmi.UID,
				)
				p.eventCallback(domainCache, libvirtEvent{}, deleteNotificationSent, interfaceStatuses, guestOsInfo, fsFreezeStatus)
			} else {
				log.Log.Object(p.vmi).Warning("Dropping metadata cache notification")
			}
		}
	}
}
