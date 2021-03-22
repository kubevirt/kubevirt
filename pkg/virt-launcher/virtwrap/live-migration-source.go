package virtwrap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	libvirt "libvirt.org/libvirt-go"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/sriov"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

// Only used for testing, migration proxy ports are 'well-known' ports and should not be randomized in production
var osChosenMigrationProxyPort = false

func setOSChosenMigrationProxyPort(val bool) {
	osChosenMigrationProxyPort = val
}

type migrationDisks struct {
	shared    map[string]bool
	generated map[string]bool
}

func prepareMigrationFlags(isBlockMigration, isUnsafeMigration, allowAutoConverge, allowPostyCopy bool, migratePaused bool) libvirt.DomainMigrateFlags {
	migrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

	if isBlockMigration {
		migrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
	}
	if isUnsafeMigration {
		migrateFlags |= libvirt.MIGRATE_UNSAFE
	}
	if allowAutoConverge {
		migrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
	}
	if allowPostyCopy {
		migrateFlags |= libvirt.MIGRATE_POSTCOPY
	}
	if migratePaused {
		migrateFlags |= libvirt.MIGRATE_PAUSED
	}

	return migrateFlags

}

func hotUnplugHostDevices(virConn cli.Connection, dom cli.VirDomain) error {
	domainSpec, err := util.GetDomainSpecWithFlags(dom, 0)
	if err != nil {
		return err
	}

	eventChan := make(chan interface{}, sriov.MaxConcurrentHotPlugDevicesEvents)
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

// This returns domain xml without the metadata sections
func migratableDomXML(dom cli.VirDomain, vmi *v1.VirtualMachineInstance) (string, error) {
	xmlstr, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Live migration failed. Failed to get XML.")
		return "", err
	}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlstr)))
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)

	depth := 0
	inMeta := false
	inMetaKV := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Log.Object(vmi).Errorf("error getting token: %v\n", err)
			break
		}

		switch v := token.(type) {
		case xml.StartElement:
			if depth == 1 && v.Name.Local == "metadata" {
				inMeta = true
			} else if inMeta && depth == 2 && v.Name.Local == "kubevirt" {
				inMetaKV = true
			}
			depth++
		case xml.EndElement:
			depth--
			if inMetaKV && depth == 2 && v.Name.Local == "kubevirt" {
				inMetaKV = false
				continue // Skip </kubevirt>
			}
			if inMeta && depth == 1 && v.Name.Local == "metadata" {
				inMeta = false
			}
		}
		if inMetaKV {
			continue // We're inside metadata/kubevirt, continuing to skip elements
		}

		if err := encoder.EncodeToken(xml.CopyToken(token)); err != nil {
			log.Log.Object(vmi).Reason(err)
		}
	}

	if err := encoder.Flush(); err != nil {
		log.Log.Object(vmi).Reason(err)
	}

	return string(buf.Bytes()), nil
}

func (d *migrationDisks) isSharedVolume(name string) bool {
	_, shared := d.shared[name]
	return shared
}

func (d *migrationDisks) isGeneratedVolume(name string) bool {
	_, generated := d.generated[name]
	return generated
}

func classifyVolumesForMigration(vmi *v1.VirtualMachineInstance) *migrationDisks {
	// This method collects all VMI volumes that should not be copied during
	// live migration. It also collects all generated disks suck as cloudinit, secrets, ServiceAccount and ConfigMaps
	// to make sure that these are being copied during migration.
	// Persistent volume claims without ReadWriteMany access mode
	// should be filtered out earlier in the process

	disks := &migrationDisks{
		shared:    make(map[string]bool),
		generated: make(map[string]bool),
	}
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		if volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil ||
			(volSrc.HostDisk != nil && *volSrc.HostDisk.Shared) {
			disks.shared[volume.Name] = true
		}
		if volSrc.ConfigMap != nil || volSrc.Secret != nil || volSrc.DownwardAPI != nil ||
			volSrc.ServiceAccount != nil || volSrc.CloudInitNoCloud != nil ||
			volSrc.CloudInitConfigDrive != nil || volSrc.ContainerDisk != nil {
			disks.generated[volume.Name] = true
		}
	}
	return disks
}

func getDiskTargetsForMigration(dom cli.VirDomain, vmi *v1.VirtualMachineInstance) []string {
	// This method collects all VMI disks that needs to be copied during live migration
	// and returns a list of its target device names.
	// Shared volues are being excluded.
	copyDisks := []string{}
	migrationVols := classifyVolumesForMigration(vmi)
	disks, err := getAllDomainDisks(dom)
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

	if vmi.Status.MigrationState == nil {
		return fmt.Errorf("cannot migration VMI until migrationState is ready")
	}

	inProgress, err := l.initializeMigrationMetadata(vmi, v1.MigrationPreCopy)
	if err != nil {
		return err
	}
	if inProgress {
		return nil
	}

	if err := updateHostsFile(fmt.Sprintf("%s %s\n", ip.GetLoopbackAddress(), vmi.Status.MigrationState.TargetPod)); err != nil {
		return fmt.Errorf("failed to update the hosts file: %v", err)
	}
	l.asyncMigrate(vmi, options)

	return nil

}

func (l *LibvirtDomainManager) initializeMigrationMetadata(vmi *v1.VirtualMachineInstance, migrationMode v1.MigrationMode) (bool, error) {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain for migration failed.")
		return false, err
	}

	defer dom.Free()
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return false, err
	}

	migrationMetadata := domainSpec.Metadata.KubeVirt.Migration
	if migrationMetadata != nil && migrationMetadata.UID == vmi.Status.MigrationState.MigrationUID {
		if migrationMetadata.EndTimestamp == nil {
			// don't stomp on currently executing migrations
			return true, nil

		} else {
			// Don't allow the same migration UID to be executed twice.
			// Migration attempts are like pods. One shot.
			return false, fmt.Errorf("migration job already executed")
		}
	}

	now := metav1.Now()
	domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
		UID:            vmi.Status.MigrationState.MigrationUID,
		StartTimestamp: &now,
		Mode:           migrationMode,
	}
	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return false, err
	}
	defer d.Free()
	return false, nil
}

func (l *LibvirtDomainManager) cancelMigration(vmi *v1.VirtualMachineInstance) error {
	if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.Completed ||
		vmi.Status.MigrationState.Failed || vmi.Status.MigrationState.StartTimestamp == nil {

		return fmt.Errorf("failed to cancel migration - vmi is not migrating")
	}
	err := l.setMigrationAbortStatus(vmi, v1.MigrationAbortInProgress)
	if err != nil {
		if err == domainerrors.MigrationAbortInProgressError {
			return nil
		}
		return err
	}

	l.asyncMigrationAbort(vmi)
	return nil
}

func (l *LibvirtDomainManager) setMigrationResultHelper(vmi *v1.VirtualMachineInstance, failed bool, completed bool, reason string, abortStatus v1.MigrationAbortStatus) error {

	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil

		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain for completed migration failed.")
		}
		return err
	}

	defer dom.Free()
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return err
	}
	migrationMetadata := domainSpec.Metadata.KubeVirt.Migration
	if migrationMetadata == nil {
		// nothing to report if migration metadata is empty
		return nil
	}

	now := metav1.Now()
	if abortStatus != "" {
		metaAbortStatus := domainSpec.Metadata.KubeVirt.Migration.AbortStatus
		if metaAbortStatus == string(abortStatus) && metaAbortStatus == string(v1.MigrationAbortInProgress) {
			return domainerrors.MigrationAbortInProgressError
		}
		domainSpec.Metadata.KubeVirt.Migration.AbortStatus = string(abortStatus)
	}

	if failed {
		domainSpec.Metadata.KubeVirt.Migration.Failed = true
		domainSpec.Metadata.KubeVirt.Migration.FailureReason = reason
	}
	if completed {
		domainSpec.Metadata.KubeVirt.Migration.Completed = true
		domainSpec.Metadata.KubeVirt.Migration.EndTimestamp = &now
	}
	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}
	defer d.Free()
	return nil

}

func (l *LibvirtDomainManager) setMigrationResult(vmi *v1.VirtualMachineInstance, failed bool, reason string, abortStatus v1.MigrationAbortStatus) error {
	connectionInterval := 10 * time.Second
	connectionTimeout := 60 * time.Second

	err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		err = l.setMigrationResultHelper(vmi, failed, true, reason, abortStatus)
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Unable to post migration results to libvirt after multiple tries")
		return err
	}
	return nil

}

func (l *LibvirtDomainManager) setMigrationAbortStatus(vmi *v1.VirtualMachineInstance, abortStatus v1.MigrationAbortStatus) error {
	connectionInterval := 10 * time.Second
	connectionTimeout := 60 * time.Second

	err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		err = l.setMigrationResultHelper(vmi, false, false, "", abortStatus)
		if err != nil {
			if err == domainerrors.MigrationAbortInProgressError {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Unable to post migration results to libvirt after multiple tries")
		return err
	}
	return nil

}

func liveMigrationMonitor(vmi *v1.VirtualMachineInstance, l *LibvirtDomainManager, options *cmdclient.MigrationOptions, migrationErr chan error) {

	logger := log.Log.Object(vmi)

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		logger.Reason(err).Error("Live migration failed.")
		l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
		return
	}
	defer dom.Free()

	start := time.Now().UTC().Unix()
	lastProgressUpdate := start
	progressWatermark := int64(0)

	// update timeouts from migration config
	progressTimeout := options.ProgressTimeout
	completionTimeoutPerGiB := options.CompletionTimeoutPerGiB

	acceptableCompletionTime := completionTimeoutPerGiB * getVMIMigrationDataSize(vmi)
monitorLoop:
	for {

		select {
		case passedErr := <-migrationErr:
			if passedErr != nil {
				logger.Reason(passedErr).Error("Live migration failed")
				var abortStatus v1.MigrationAbortStatus
				if strings.Contains(passedErr.Error(), "canceled by client") {
					abortStatus = v1.MigrationAbortSucceeded
				}
				l.setMigrationResult(vmi, true, fmt.Sprintf("Live migration failed %v", passedErr), abortStatus)
				break monitorLoop
			}
		default:
		}

		stats, err := dom.GetJobInfo()
		if err != nil {
			logger.Reason(err).Error("failed to get domain job info")
			break
		}

		remainingData := int64(stats.DataRemaining)
		switch stats.Type {
		case libvirt.DOMAIN_JOB_UNBOUNDED:
			// Migration is running
			now := time.Now().UTC().Unix()
			elapsed := now - start

			if (progressWatermark == 0) ||
				(progressWatermark > remainingData) {
				progressWatermark = remainingData
				lastProgressUpdate = now
			}

			domainSpec, err := l.getDomainSpec(dom)
			if err != nil {
				logger.Reason(err).Error("failed to get domain spec info")
				break
			}

			// check if the migration is progressing
			progressDelay := now - lastProgressUpdate
			if progressTimeout != 0 &&
				progressDelay > progressTimeout {
				logger.Warningf("Live migration stuck for %d sec", progressDelay)

				// If the migration is in post copy mode we should not abort the job as the migration would lose state
				if domainSpec.Metadata.KubeVirt.Migration != nil && domainSpec.Metadata.KubeVirt.Migration.Mode == v1.MigrationPostCopy {
					break
				}

				err := dom.AbortJob()
				if err != nil {
					logger.Reason(err).Error("failed to abort migration")
				}
				l.setMigrationResult(vmi, true, fmt.Sprintf("Live migration stuck for %d sec and has been aborted", progressDelay), v1.MigrationAbortSucceeded)
				break monitorLoop
			}

			// check the overall migration time
			if shouldTriggerTimeout(acceptableCompletionTime, elapsed, domainSpec) {

				if options.AllowPostCopy {
					err = dom.MigrateStartPostCopy(uint32(0))
					if err != nil {
						logger.Reason(err).Error("failed to start post migration")
					}

					err = l.updateVMIMigrationMode(dom, vmi, v1.MigrationPostCopy)
					if err != nil {
						log.Log.Object(vmi).Reason(err).Error("Unable to update migration mode on domain xml")
					}

					break
				}

				logger.Warningf("Live migration is not completed after %d sec",
					acceptableCompletionTime)

				err := dom.AbortJob()
				if err != nil {
					logger.Reason(err).Error("failed to abort migration")
				}
				l.setMigrationResult(vmi, true, fmt.Sprintf("Live migration is not completed after %d sec and has been aborted", acceptableCompletionTime), v1.MigrationAbortSucceeded)
				break monitorLoop
			}

		case libvirt.DOMAIN_JOB_NONE:
			logger.Info("Migration job didn't start yet")
		case libvirt.DOMAIN_JOB_COMPLETED:
			logger.Info("Migration has been completed")
			l.setMigrationResult(vmi, false, "", "")
			break monitorLoop
		case libvirt.DOMAIN_JOB_FAILED:
			logger.Info("Migration job failed")
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
			break monitorLoop
		case libvirt.DOMAIN_JOB_CANCELLED:
			logger.Info("Migration was canceled")
			l.setMigrationResult(vmi, true, "Live migration aborted ", v1.MigrationAbortSucceeded)
			break monitorLoop
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func (l *LibvirtDomainManager) asyncMigrationAbort(vmi *v1.VirtualMachineInstance) {
	go func(l *LibvirtDomainManager, vmi *v1.VirtualMachineInstance) {

		domName := api.VMINamespaceKeyFunc(vmi)
		dom, err := l.virConn.LookupDomainByName(domName)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Warning("failed to cancel migration, domain not found ")
			return
		}
		defer dom.Free()
		stats, err := dom.GetJobInfo()
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("failed to get domain job info")
			return
		}
		if stats.Type == libvirt.DOMAIN_JOB_UNBOUNDED {
			err := dom.AbortJob()
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("failed to cancel migration")
				l.setMigrationAbortStatus(vmi, v1.MigrationAbortFailed)
				return
			}
			log.Log.Object(vmi).Info("Live migration abort succeeded")
		}
		return
	}(l, vmi)
}

func (l *LibvirtDomainManager) asyncMigrate(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) {

	go func(l *LibvirtDomainManager, vmi *v1.VirtualMachineInstance) {

		// Start local migration proxy.
		//
		// Right now Libvirt won't let us perform a migration using a unix socket, so
		// we have to create a local host tcp server (on port 22222) that forwards the traffic
		// to libvirt in order to trick libvirt into doing what we want.
		// This also creates a tcp server for each additional direct migration connections
		// that will be proxied to the destination pod

		isBlockMigration := (vmi.Status.MigrationMethod == v1.BlockMigration)
		migrationPortsRange := migrationproxy.GetMigrationPortsList(isBlockMigration)

		loopbackAddress := ip.GetLoopbackAddress()
		// Create a tcp server for each direct connection proxy
		for _, port := range migrationPortsRange {
			if osChosenMigrationProxyPort {
				// this is only set to 0 during unit tests
				port = 0
			}
			key := migrationproxy.ConstructProxyKey(string(vmi.UID), port)
			migrationProxy := migrationproxy.NewTargetProxy(loopbackAddress, port, nil, nil, migrationproxy.SourceUnixFile(l.virtShareDir, key))
			defer migrationProxy.StopListening()
			err := migrationProxy.StartListening()
			if err != nil {
				l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
				return
			}
		}

		//  proxy incoming migration requests on port 22222 to the vmi's existing libvirt connection
		tcpBindPort := LibvirtLocalConnectionPort
		if osChosenMigrationProxyPort {
			// this is only set to 0 during unit tests
			tcpBindPort = 0
		}
		libvirtConnectionProxy := migrationproxy.NewTargetProxy(loopbackAddress, tcpBindPort, nil, nil, migrationproxy.SourceUnixFile(l.virtShareDir, string(vmi.UID)))
		defer libvirtConnectionProxy.StopListening()
		err := libvirtConnectionProxy.StartListening()
		if err != nil {
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
			return
		}

		// For a tunnelled migration, this is always the uri
		dstURI := fmt.Sprintf("qemu+tcp://%s/system", net.JoinHostPort(loopbackAddress, strconv.Itoa(LibvirtLocalConnectionPort)))
		migrURI := fmt.Sprintf("tcp://%s", ip.NormalizeIPAddress(loopbackAddress))

		domName := api.VMINamespaceKeyFunc(vmi)
		dom, err := l.virConn.LookupDomainByName(domName)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Live migration failed.")
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
			return
		}
		defer dom.Free()

		bandwidth, err := converter.QuantityToMebiByte(options.Bandwidth)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Live migration failed. Invalid bandwidth supplied.")
			return
		}

		if err := hotUnplugHostDevices(l.virConn, dom); err != nil {
			log.Log.Object(vmi).Reason(err).Error(fmt.Sprintf("Live migration failed."))
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err), "")
		}

		xmlstr, err := migratableDomXML(dom, vmi)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Live migration failed. Could not compute target XML.")
			return
		}

		params := &libvirt.DomainMigrateParameters{
			Bandwidth:  bandwidth, // MiB/s
			URI:        migrURI,
			URISet:     true,
			DestXML:    xmlstr,
			DestXMLSet: true,
		}
		copyDisks := getDiskTargetsForMigration(dom, vmi)
		if len(copyDisks) != 0 {
			params.MigrateDisks = copyDisks
			params.MigrateDisksSet = true
		}
		// start live migration tracking
		migrationErrorChan := make(chan error, 1)
		defer close(migrationErrorChan)
		go liveMigrationMonitor(vmi, l, options, migrationErrorChan)

		migratePaused, err := isDomainPaused(dom)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Live migration failed: can't retrive state")
			migrationErrorChan <- err
			return
		}

		migrateFlags := prepareMigrationFlags(isBlockMigration, options.UnsafeMigration, options.AllowAutoConverge, options.AllowPostCopy, migratePaused)
		if options.UnsafeMigration {
			log.Log.Object(vmi).Info("UNSAFE_MIGRATION flag is set, libvirt's migration checks will be disabled!")
		}

		err = dom.MigrateToURI3(dstURI, params, migrateFlags)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Live migration failed.")
			migrationErrorChan <- err
			return
		}
		log.Log.Object(vmi).Infof("Live migration succeeded.")
	}(l, vmi)
}

func (l *LibvirtDomainManager) updateVMIMigrationMode(dom cli.VirDomain, vmi *v1.VirtualMachineInstance, mode v1.MigrationMode) error {
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return err
	}

	if domainSpec.Metadata.KubeVirt.Migration == nil {
		domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{}
	}

	domainSpec.Metadata.KubeVirt.Migration.Mode = mode

	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}
	defer d.Free()

	return nil
}
