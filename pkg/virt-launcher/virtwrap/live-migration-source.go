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

package virtwrap

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cpudedicated"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

const liveMigrationFailed = "Live migration failed."

type migrationDisks struct {
	shared         map[string]bool
	generated      map[string]bool
	localToMigrate map[string]bool
}

func generateMigrationFlags(isBlockMigration, migratePaused bool, options *cmdclient.MigrationOptions) libvirt.DomainMigrateFlags {
	migrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

	if isBlockMigration {
		migrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
	}
	if options.UnsafeMigration {
		migrateFlags |= libvirt.MIGRATE_UNSAFE
	}
	if options.AllowAutoConverge {
		migrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
	}
	if options.AllowPostCopy {
		migrateFlags |= libvirt.MIGRATE_POSTCOPY
	}
	if migratePaused {
		migrateFlags |= libvirt.MIGRATE_PAUSED
	}
	if shouldConfigureParallel, _ := shouldConfigureParallelMigration(options); shouldConfigureParallel {
		migrateFlags |= libvirt.MIGRATE_PARALLEL
	}

	return migrateFlags

}

func hotUnplugHostDevices(virConn cli.Connection, dom cli.VirDomain) error {
	domainSpec, err := util.GetDomainSpecWithFlags(dom, 0)
	if err != nil {
		return err
	}

	eventChan := make(chan interface{}, hostdevice.MaxConcurrentHotPlugDevicesEvents)
	var callback libvirt.DomainEventDeviceRemovedCallback = func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventDeviceRemoved) {
		eventChan <- event.DevAlias
	}

	if domainEvent := cli.NewDomainEventDeviceRemoved(virConn, dom, callback, eventChan); domainEvent != nil {
		const waitForDetachTimeout = 30 * time.Second
		err := sriov.SafelyDetachHostDevices(domainSpec, domainEvent, dom, waitForDetachTimeout)
		if err != nil {
			return err
		}
	}
	return nil
}

// This returns domain xml without the metadata section, as it is only relevant to the source domain
// Note: Unfortunately we can't just use UnMarshall + Marshall here, as that leads to unwanted XML alterations
func migratableDomXML(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec, libvirtHooksEnabled bool) (string, error) {
	var domain *api.Domain
	var err error

	xmlstr, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Live migration failed. Failed to get XML.")
		return "", err
	}
	domcfg := &libvirtxml.Domain{}
	if err := domcfg.Unmarshal(xmlstr); err != nil {
		return "", err
	}
	// TODO: Once LibvirtHooksServerAndClient feature gate is GA, remove
	// convertDisks, replaced by DiskSourcePathHook on the target.
	if !libvirtHooksEnabled {
		if err = convertDisks(domSpec, domcfg); err != nil {
			return "", err
		}
	}
	// TODO: Once the LibvirtHooksServerAndClient feature gate is GA,
	// this logic in the source can be removed, as XML modifications
	// for dedicated CPUs will always be handled on the target side.
	if !libvirtHooksEnabled && vmi.IsCPUDedicated() {
		// If the VMI has dedicated CPUs, we need to replace the old CPUs that were
		// assigned in the source node with the new CPUs assigned in the target node
		err = xml.Unmarshal([]byte(xmlstr), &domain)
		if err != nil {
			return "", err
		}
		domain, err := cpudedicated.GenerateDomainForTargetCPUSetAndTopology(vmi, domSpec)
		if err != nil {
			return "", err
		}
		if err = cpudedicated.ConvertCPUDedicatedFields(domain, domcfg); err != nil {
			return "", err
		}
	}
	// set slice size for local disks to migrate
	if err := configureLocalDiskToMigrate(domcfg, vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to set size for local disk.")
		return "", err
	}

	return domcfg.Marshal()
}

func convertDisks(domSpec *api.DomainSpec, domcfg *libvirtxml.Domain) error {
	if domcfg == nil || domcfg.Devices == nil || domSpec == nil {
		return nil
	}
	if len(domSpec.Devices.Disks) != len(domcfg.Devices.Disks) {
		return fmt.Errorf("spec and domain have different disks count")
	}
	for i, disk := range domSpec.Devices.Disks {
		domcfgDisk := (&domcfg.Devices.Disks[i])
		diskName := disk.Alias.GetName()

		if disk.Source.File != "" {
			if domcfgDisk.Source == nil || domcfgDisk.Source.File == nil {
				return fmt.Errorf("disk %s: spec has file source but domain is missing it", diskName)
			}
			log.Log.Infof("Updating disk %s source file from %s to %s", diskName, domcfgDisk.Source.File.File, disk.Source.File)
			domcfgDisk.Source.File.File = disk.Source.File
		}
		if disk.Source.DataStore != nil && disk.Source.DataStore.Source != nil && disk.Source.DataStore.Source.File != "" {
			if domcfgDisk.Source == nil || domcfgDisk.Source.DataStore == nil || domcfgDisk.Source.DataStore.Source == nil || domcfgDisk.Source.DataStore.Source.File == nil {
				return fmt.Errorf("disk %s: spec has DataStore file source but domain is missing it", diskName)
			}
			log.Log.Infof("Updating disk %s datastore backend from %s to %s", diskName, domcfgDisk.Source.DataStore.Source.File.File, disk.Source.DataStore.Source.File)
			domcfgDisk.Source.DataStore.Source.File.File = disk.Source.DataStore.Source.File
		}
	}
	return nil
}

func (d *migrationDisks) isSharedVolume(name string) bool {
	_, shared := d.shared[name]
	return shared
}

func (d *migrationDisks) isGeneratedVolume(name string) bool {
	_, generated := d.generated[name]
	return generated
}

func (d *migrationDisks) isLocalVolumeToMigrate(name string) bool {
	_, migrate := d.localToMigrate[name]
	return migrate
}

func classifyVolumesForMigration(vmi *v1.VirtualMachineInstance) *migrationDisks {
	// This method collects all VMI volumes that should not be copied during
	// live migration. It also collects all generated disks suck as cloudinit, secrets, ServiceAccount and ConfigMaps
	// to make sure that these are being copied during migration.

	disks := &migrationDisks{
		shared:         make(map[string]bool),
		generated:      make(map[string]bool),
		localToMigrate: make(map[string]bool),
	}
	migrateDisks := make(map[string]bool)
	for _, v := range vmi.Status.MigratedVolumes {
		migrateDisks[v.VolumeName] = true
	}
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		switch {
		case volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil:
			if _, ok := migrateDisks[volume.Name]; ok {
				disks.localToMigrate[volume.Name] = true
			} else {
				disks.shared[volume.Name] = true
			}
		case volSrc.HostDisk != nil:
			if _, ok := migrateDisks[volume.Name]; ok {
				disks.localToMigrate[volume.Name] = true
			} else if volSrc.HostDisk.Shared != nil && *volSrc.HostDisk.Shared {
				disks.shared[volume.Name] = true
			}
		case volSrc.ConfigMap != nil || volSrc.Secret != nil || volSrc.DownwardAPI != nil ||
			volSrc.ServiceAccount != nil || volSrc.CloudInitNoCloud != nil ||
			volSrc.CloudInitConfigDrive != nil || volSrc.ContainerDisk != nil:
			disks.generated[volume.Name] = true
		}
	}

	return disks
}

func getDiskTargetsForMigration(dom cli.VirDomain, vmi *v1.VirtualMachineInstance) []string {
	// This method collects all VMI disks that needs to be copied during live migration
	// and returns a list of its target device names.
	// Shared volues are being excluded.
	var copyDisks []string
	migrationVols := classifyVolumesForMigration(vmi)
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to parse domain XML to get disks.")
	}
	// the name of the volume should match the alias
	for _, disk := range disks {
		// explicitly skip cd-rom drives
		if disk.Device == "cdrom" {
			continue
		}
		if disk.ReadOnly != nil && !migrationVols.isGeneratedVolume(disk.Alias.GetName()) {
			continue
		}
		if (disk.Type != "file" && disk.Type != "block") || migrationVols.isSharedVolume(disk.Alias.GetName()) {
			continue
		}
		copyDisks = append(copyDisks, disk.Target.Device)
	}
	return copyDisks
}

func (l *LibvirtDomainManager) startMigration(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {
	if vmi.Status.ChangedBlockTracking != nil && vmi.Status.ChangedBlockTracking.BackupStatus != nil {
		return fmt.Errorf("cannot migrate VMI until backup %s is completed", vmi.Status.ChangedBlockTracking.BackupStatus.BackupName)
	}
	if vmi.Status.MigrationState == nil {
		return fmt.Errorf("cannot migrate VMI until migrationState is ready")
	}

	inProgress, err := l.initializeMigrationMetadata(vmi, v1.MigrationPreCopy)
	if err != nil {
		return err
	}
	if inProgress {
		return nil
	}

	go l.migrate(vmi, options)
	return nil
}

func (l *LibvirtDomainManager) initializeMigrationMetadata(vmi *v1.VirtualMachineInstance, migrationMode v1.MigrationMode) (bool, error) {
	migrationMetadata, exists := l.metadataCache.Migration.Load()
	uid := MigrationUID(vmi)
	if exists && migrationMetadata.UID == uid {
		if migrationMetadata.EndTimestamp == nil {
			// don't stop on currently executing migrations
			return true, nil
		}

		// Don't allow the same migration UID to be executed twice.
		// Migration attempts are like pods. One shot.
		return false, fmt.Errorf("migration job %v already executed, finished at %v, failed: %t, abortStatus: %s",
			migrationMetadata.UID, *migrationMetadata.EndTimestamp, migrationMetadata.Failed, migrationMetadata.AbortStatus)
	}

	now := metav1.Now()
	m := api.MigrationMetadata{
		UID:            uid,
		StartTimestamp: &now,
		Mode:           migrationMode,
	}
	l.metadataCache.Migration.Store(m)
	log.Log.V(4).Infof("initialize migration metadata: %v", m)
	return false, nil
}

func (l *LibvirtDomainManager) cancelMigration(vmi *v1.VirtualMachineInstance) error {
	l.metadataCache.Migration.WithSafeBlock(func(migration *api.MigrationMetadata, _ bool) {
		if migration.EndTimestamp != nil || migration.Failed || migration.StartTimestamp == nil {
			log.Log.Object(vmi).Infof("cancel migration ignored: vmi is not migrating")
			return
		}

		switch v1.MigrationAbortStatus(migration.AbortStatus) {
		case v1.MigrationAbortInProgress:
			log.Log.Object(vmi).Infof("cancel migration ignored: abort is already in progress")
			return
		case v1.MigrationAbortSucceeded:
			log.Log.Object(vmi).Infof("cancel migration ignored: abort already succeeded")
			return
		}

		migration.AbortStatus = string(v1.MigrationAbortInProgress)
		l.asyncMigrationAbort(vmi)
	})
	return nil
}

func (l *LibvirtDomainManager) setMigrationResult(failed bool, reason string) {
	migrationMetadata, exists := l.metadataCache.Migration.Load()
	if !exists {
		log.Log.Errorf("setMigrationResult called but migration metadata is empty")
		return
	}

	if migrationMetadata.EndTimestamp != nil {
		log.Log.Errorf("setMigrationResult called but migration result has already been reported at %v", migrationMetadata.EndTimestamp)
		return
	}

	if failed {
		switch {
		case strings.Contains(reason, "canceled by client"):
			reason = "Live migration has been aborted"
		case strings.Contains(standardizeSpaces(reason), "has to be smaller or equal to the actual size of the containing file"):
			reason = fmt.Sprintf("Volume migration cannot be performed because the destination volume is smaller than the source volume: %v", reason)
		}
	}

	l.metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
		if failed {
			migrationMetadata.Failed = true
			migrationMetadata.FailureReason = reason
		}

		migrationMetadata.EndTimestamp = pointer.P(metav1.Now())
	})

	logger := log.Log.V(2)
	if !failed {
		logger = logger.V(4)
	}
	logger.Infof("set migration result in metadata: %s", l.metadataCache.Migration.String())
}

func (l *LibvirtDomainManager) setMigrationAbortStatus(abortStatus v1.MigrationAbortStatus) {
	l.metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
		migrationMetadata.AbortStatus = string(abortStatus)
	})
	log.Log.V(2).Infof("set migration abort status in metadata: %s", l.metadataCache.Migration.String())
}

func (l *LibvirtDomainManager) asyncMigrationAbort(vmi *v1.VirtualMachineInstance) {
	// Libvirt calls (GetJobInfo, AbortJob) can hang indefinitely if the QEMU
	// monitor is unresponsive. Wrap with a timeout.
	l.abortWg.Add(1)
	go func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			domName := api.VMINamespaceKeyFunc(vmi)
			dom, err := l.virConn.LookupDomainByName(domName)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Warning("failed to cancel migration, domain not found")
				l.setMigrationAbortStatus(v1.MigrationAbortFailed)
				return
			}
			defer dom.Free()
			jobInfo, err := dom.GetJobInfo()
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("failed to get domain job info")
				l.setMigrationAbortStatus(v1.MigrationAbortFailed)
				return
			}
			if jobInfo.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
				if err := dom.AbortJob(); err != nil {
					log.Log.Object(vmi).Reason(err).Error("failed to cancel migration")
					l.setMigrationAbortStatus(v1.MigrationAbortFailed)
					return
				}
				l.setMigrationAbortStatus(v1.MigrationAbortSucceeded)
				log.Log.Object(vmi).Info("Live migration abort succeeded")
			} else {
				log.Log.Object(vmi).Infof("migration job is not active (type=%d), nothing to abort", jobInfo.Type)
				l.setMigrationAbortStatus(v1.MigrationAbortFailed)
			}
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			log.Log.Object(vmi).Error("migration abort timed out waiting for libvirt")
			l.setMigrationAbortStatus(v1.MigrationAbortFailed)
		}
		l.abortWg.Done()
	}()
}

func generateDomainName(vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.Namespace
	name := vmi.Name
	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.DomainName != nil &&
		*vmi.Status.MigrationState.TargetState.DomainName != "" &&
		vmi.Status.MigrationState.TargetState.DomainNamespace != nil &&
		*vmi.Status.MigrationState.TargetState.DomainNamespace != "" {
		name = *vmi.Status.MigrationState.TargetState.DomainName
		namespace = *vmi.Status.MigrationState.TargetState.DomainNamespace
	}
	domainName := util.DomainFromNamespaceName(namespace, name)
	log.Log.Object(vmi).Infof("generated target domain name %s", domainName)
	return domainName
}

func updateFilePathsToNewDomain(vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec) {
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetState != nil && vmi.Status.MigrationState.TargetState.DomainNamespace != nil {
		targetNS := *vmi.Status.MigrationState.TargetState.DomainNamespace
		// Modify the domain XML to update paths to the target volumes to match the new domain
		for i, disk := range domSpec.Devices.Disks {
			if disk.Source.DataStore != nil &&
				disk.Source.DataStore.Source != nil &&
				strings.Contains(disk.Source.DataStore.Source.File, vmi.Namespace) {
				oldPath := disk.Source.DataStore.Source.File
				domSpec.Devices.Disks[i].Source.DataStore.Source.File = strings.Replace(disk.Source.DataStore.Source.File, vmi.Namespace, targetNS, 1)
				log.Log.Object(vmi).V(4).Infof("Updated disk %s datastore backend path from %s to %s", disk.Alias.GetName(), oldPath, domSpec.Devices.Disks[i].Source.DataStore.Source.File)
			}
			if disk.Source.File != "" && strings.Contains(disk.Source.File, vmi.Namespace) {
				oldPath := disk.Source.File
				domSpec.Devices.Disks[i].Source.File = strings.Replace(disk.Source.File, vmi.Namespace, targetNS, 1)
				log.Log.Object(vmi).V(4).Infof("Updated disk %s source path from %s to %s", disk.Alias.GetName(), oldPath, domSpec.Devices.Disks[i].Source.File)
			}
			if bp := disksource.Resolve(domSpec.Devices.Disks[i]).BackendPath(); bp != "" {
				log.Log.Object(vmi).V(4).Infof("Paths of disk %s: %s", disk.Alias.GetName(), bp)
			}
		}
	}
}

func generateMigrationParams(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions, virtShareDir string, domSpec *api.DomainSpec, libvirtHooksEnabled bool) (*libvirt.DomainMigrateParameters, error) {
	bandwidth, err := vcpu.QuantityToMebiByte(options.Bandwidth)
	if err != nil {
		return nil, err
	}

	// TODO: Once LibvirtHooksServerAndClient feature gate is GA, remove
	// updateFilePathsToNewDomain, replaced by DiskSourcePathHook on the target.
	if !libvirtHooksEnabled {
		updateFilePathsToNewDomain(vmi, domSpec)
	}
	xmlstr, err := migratableDomXML(dom, vmi, domSpec, libvirtHooksEnabled)
	if err != nil {
		return nil, err
	}
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetState != nil &&
		vmi.Status.MigrationState.TargetState.VirtualMachineInstanceUID != nil {
		log.Log.Object(vmi).Infof("Replacing VMI UID %s with target VMI UID %s in the XML", vmi.UID, *vmi.Status.MigrationState.TargetState.VirtualMachineInstanceUID)
		// Replace all occurences of the VMI UID in the XML with the target UID.
		xmlstr = strings.ReplaceAll(xmlstr, string(vmi.UID), string(*vmi.Status.MigrationState.TargetState.VirtualMachineInstanceUID))
	}

	parallelMigrationSet, parallelMigrationThreads := shouldConfigureParallelMigration(options)

	key := migrationproxy.ConstructProxyKey(string(vmi.UID), migrationproxy.LibvirtDirectMigrationPort)
	migrURI := fmt.Sprintf("unix://%s", migrationproxy.SourceUnixFile(virtShareDir, key))
	log.Log.Object(vmi).V(5).Infof("migration URI: %s", migrURI)
	params := &libvirt.DomainMigrateParameters{
		URI:                    migrURI,
		URISet:                 true,
		Bandwidth:              bandwidth, // MiB/s
		BandwidthSet:           bandwidth > 0,
		DestXML:                xmlstr,
		DestXMLSet:             true,
		PersistXML:             xmlstr,
		PersistXMLSet:          true,
		ParallelConnectionsSet: parallelMigrationSet,
		ParallelConnections:    parallelMigrationThreads,
		DestName:               generateDomainName(vmi),
		DestNameSet:            true,
	}

	copyDisks := getDiskTargetsForMigration(dom, vmi)
	if len(copyDisks) != 0 {
		params.MigrateDisks = copyDisks
		params.MigrateDisksSet = true
		// add a socket for live block migration
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), migrationproxy.LibvirtBlockMigrationPort)
		disksURI := fmt.Sprintf("unix://%s", migrationproxy.SourceUnixFile(virtShareDir, key))
		params.DisksURI = disksURI
		params.DisksURISet = true
		params.MigrateDisksDetectZeroesList = copyDisks
		params.MigrateDisksDetectZeroesSet = true
	}

	log.Log.Object(vmi).Infof("generated migration parameters: %+v", params)
	return params, nil
}

type getDiskVirtualSizeFuncType func(disk *libvirtxml.DomainDisk) (int64, error)

var getDiskVirtualSizeFunc getDiskVirtualSizeFuncType

func init() {
	getDiskVirtualSizeFunc = getDiskVirtualSize
}

// getDiskVirtualSize return the size of a local volume to migrate.
// See suggestion in: https://issues.redhat.com/browse/RHEL-4607
func getDiskVirtualSize(disk *libvirtxml.DomainDisk) (int64, error) {
	path, err := getDiskPathFromSource(disk.Source)
	if err != nil {
		return -1, err
	}
	info, err := osdisk.GetDiskInfo(path)
	if err != nil {
		return -1, err
	}
	return info.VirtualSize, nil
}

func getDiskPathFromSource(source *libvirtxml.DomainDiskSource) (string, error) {
	var path string
	if source == nil {
		return "", fmt.Errorf("empty source for the disk")
	}

	if source.DataStore != nil {
		if source.DataStore.Source == nil {
			return "", fmt.Errorf("disk has initialized datastore with no source")
		}
		source = source.DataStore.Source
	}

	switch {
	case source.File != nil:
		path = source.File.File
	case source.Block != nil:
		path = source.Block.Dev
	default:
		return "", fmt.Errorf("no path set")
	}

	return path, nil
}

func getDiskName(disk *libvirtxml.DomainDisk) string {
	if disk == nil {
		return ""
	}
	n := disk.Alias.Name
	if len(n) < 3 {
		return n
	}
	// Trim the ua- prefix
	if strings.HasPrefix(n, "ua-") {
		return n[3:]
	}
	return n
}

func getMigrateVolumeForCondition(vmi *v1.VirtualMachineInstance, condition func(info *v1.StorageMigratedVolumeInfo) bool) map[string]bool {
	res := make(map[string]bool)

	for _, v := range vmi.Status.MigratedVolumes {
		if v.SourcePVCInfo == nil || v.DestinationPVCInfo == nil {
			continue
		}
		if v.SourcePVCInfo.VolumeMode == nil || v.DestinationPVCInfo.VolumeMode == nil {
			continue
		}
		if condition(&v) {
			res[v.VolumeName] = true
		}
	}
	return res
}

func getFsSrcBlockDstVols(vmi *v1.VirtualMachineInstance) map[string]bool {
	return getMigrateVolumeForCondition(vmi, func(v *v1.StorageMigratedVolumeInfo) bool {
		if v == nil {
			return false
		}
		if *v.SourcePVCInfo.VolumeMode == k8sv1.PersistentVolumeFilesystem &&
			*v.DestinationPVCInfo.VolumeMode == k8sv1.PersistentVolumeBlock {
			return true
		}
		return false
	})
}

func getBlockSrcFsDstVols(vmi *v1.VirtualMachineInstance) map[string]bool {
	return getMigrateVolumeForCondition(vmi, func(v *v1.StorageMigratedVolumeInfo) bool {
		if v == nil {
			return false
		}
		if *v.SourcePVCInfo.VolumeMode == k8sv1.PersistentVolumeBlock &&
			*v.DestinationPVCInfo.VolumeMode == k8sv1.PersistentVolumeFilesystem {
			return true
		}
		return false
	})
}

// configureLocalDiskToMigrate modifies the domain XML for the volume migration. For example, it sets the slice to allow the migration to a destination
// volume with different size then the source, or it adjust the XML configuration when it migrates from a filesystem source to a block destination or
// vice versa.
func configureLocalDiskToMigrate(dom *libvirtxml.Domain, vmi *v1.VirtualMachineInstance) error {
	if dom.Devices == nil {
		return nil
	}

	migDisks := classifyVolumesForMigration(vmi)
	fsSrcBlockDstVols := getFsSrcBlockDstVols(vmi)
	blockSrcFsDstVols := getBlockSrcFsDstVols(vmi)
	hotplugVols := make(map[string]bool)

	for _, v := range vmi.Spec.Volumes {
		if storagetypes.IsHotplugVolume(&v) {
			hotplugVols[v.Name] = true
		}
	}

	for i, d := range dom.Devices.Disks {
		if d.Alias == nil {
			return fmt.Errorf("empty alias")
		}
		name := getDiskName(&d)
		if !migDisks.isLocalVolumeToMigrate(name) {
			continue
		}
		// Calculate the size of the volume to migrate
		size, err := getDiskVirtualSizeFunc(&d)
		if err != nil {
			return err
		}
		// Configure the slice to enable to migrate the volume to a destination with different size
		// See suggestion in: https://issues.redhat.com/browse/RHEL-4607
		var source *libvirtxml.DomainDiskSource
		if dom.Devices.Disks[i].Source.DataStore != nil && dom.Devices.Disks[i].Source.DataStore.Source != nil {
			source = dom.Devices.Disks[i].Source.DataStore.Source
		} else {
			source = dom.Devices.Disks[i].Source
		}

		if source.Slices == nil {
			source.Slices = &libvirtxml.DomainDiskSlices{
				Slices: []libvirtxml.DomainDiskSlice{
					{
						Type:   "storage",
						Offset: 0,
						Size:   uint(size),
					},
				}}
		}
		var path string
		_, hotplugVol := hotplugVols[name]
		// Adjust the XML configuration when it migrates from a filesystem source to a block destination or vice versa
		if _, ok := fsSrcBlockDstVols[name]; ok {
			log.Log.V(2).Infof("Replace filesystem source with block destination for volume %s", name)
			if hotplugVol {
				path = hotplugdisk.GetVolumeMountDir(name)
			} else {
				path = filepath.Join(string(filepath.Separator), "dev", name)
			}
			source.Block = &libvirtxml.DomainDiskSourceBlock{Dev: path}
			source.File = nil
		}
		if _, ok := blockSrcFsDstVols[name]; ok {
			log.Log.V(2).Infof("Replace block source with destination for volume %s", name)
			if hotplugVol {
				path = hotplugdisk.GetVolumeMountDir(name) + ".img"
			} else {
				path = filepath.Join(hostdisk.GetMountedHostDiskDir(name), "disk.img")
			}
			source.File = &libvirtxml.DomainDiskSourceFile{File: path}
			source.Block = nil
		}
	}

	return nil
}

func (l *LibvirtDomainManager) migrateHelper(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {

	var err error
	var params *libvirt.DomainMigrateParameters

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		return err
	}
	defer dom.Free()

	migratePaused, err := isDomainPaused(dom)
	if err != nil {
		return fmt.Errorf("failed to retrive domain state")
	}
	migrateFlags := generateMigrationFlags(vmi.IsBlockMigration(), migratePaused, options)

	// anything that modifies the domain needs to be performed with the domainModifyLock held
	// The domain params and unHotplug need to be performed in a critical section together.
	critSection := func() error {
		l.domainModifyLock.Lock()
		defer l.domainModifyLock.Unlock()

		if err := prepareDomainForMigration(l.virConn, dom); err != nil {
			return fmt.Errorf("error encountered during preparing domain for migration: %v", err)
		}
		domSpec, err := l.getDomainSpec(dom)
		if err != nil {
			return fmt.Errorf("failed to get domain spec: %v", err)
		}
		params, err = generateMigrationParams(dom, vmi, options, l.virtShareDir, domSpec, l.libvirtHooksServerAndClientEnabled)
		if err != nil {
			return fmt.Errorf("error encountered while generating migration parameters: %v", err)
		}

		return nil
	}
	err = critSection()
	if err != nil {
		return err
	}

	// initiate the live migration
	var dstURI string
	if vmitrait.IsNonRoot(vmi) {
		dstURI = fmt.Sprintf("qemu+unix:///session?socket=%s", migrationproxy.SourceUnixFile(l.virtShareDir, string(vmi.UID)))
	} else {
		dstURI = fmt.Sprintf("qemu+unix:///system?socket=%s", migrationproxy.SourceUnixFile(l.virtShareDir, string(vmi.UID)))
	}

	err = dom.MigrateToURI3(dstURI, params, migrateFlags)
	l.abortWg.Wait() // wait for in-flight cancellation
	if err != nil {
		log.Log.Object(vmi).Errorf("error encountered during MigrateToURI3 libvirt api call: %v", err)
		return err
	}

	return nil
}

// prepareDomainForMigration perform necessary operation
// on the source domain just before migration
func prepareDomainForMigration(virtConn cli.Connection, domain cli.VirDomain) error {
	return hotUnplugHostDevices(virtConn, domain)
}

func shouldImmediatelyFailMigration(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Annotations == nil {
		return false
	}

	_, shouldFail := vmi.Annotations[v1.FuncTestForceLauncherMigrationFailureAnnotation]
	return shouldFail
}

func (l *LibvirtDomainManager) migrate(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) {
	if shouldImmediatelyFailMigration(vmi) {
		log.Log.Object(vmi).Error("Live migration failed. Failure is forced by functional tests suite.")
		l.setMigrationResult(true, "Failed migration to satisfy functional test condition")
		return
	}

	migrationDone := make(chan struct{})
	defer close(migrationDone)

	log.Log.Object(vmi).Infof("Initiating live migration.")
	if options.UnsafeMigration {
		log.Log.Object(vmi).Info("UNSAFE_MIGRATION flag is set, libvirt's migration checks will be disabled!")
	}

	monitor := newMigrationMonitor(vmi, l, options, migrationDone)
	monitorReady := make(chan error, 1)
	go monitor.startMonitor(monitorReady)

	if err := <-monitorReady; err != nil {
		log.Log.Object(vmi).Reason(err).Error("migration monitor failed to start")
		l.setMigrationResult(true, err.Error())
		return
	}

	err := l.migrateHelper(vmi, options)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(liveMigrationFailed)
		l.setMigrationResult(true, err.Error())
		return
	}

	log.Log.Object(vmi).Infof("Live migration succeeded.")
	l.setMigrationResult(false, "")
}

func (l *LibvirtDomainManager) updateVMIMigrationMode(mode v1.MigrationMode) {
	l.metadataCache.Migration.WithSafeBlock(func(migrationMetadata *api.MigrationMetadata, _ bool) {
		migrationMetadata.Mode = mode
	})
	log.Log.V(4).Infof("Migration mode set in metadata: %s", l.metadataCache.Migration.String())
}

func shouldConfigureParallelMigration(options *cmdclient.MigrationOptions) (shouldConfigure bool, threadsCount int) {
	if options == nil {
		return
	}
	if options.ParallelMigrationThreads == nil {
		return
	}

	shouldConfigure = true
	threadsCount = int(*options.ParallelMigrationThreads)
	return
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
