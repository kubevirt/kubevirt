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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtwrap

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	eventsclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"

	libvirt "github.com/libvirt/libvirt-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type DomainManager interface {
	SyncVMI(*v1.VirtualMachineInstance, bool) (*api.DomainSpec, error)
	KillVMI(*v1.VirtualMachineInstance) error
	DeleteVMI(*v1.VirtualMachineInstance) error
	SignalShutdownVMI(*v1.VirtualMachineInstance) error
	ListAllDomains() ([]*api.Domain, error)
	MigrateVMI(*v1.VirtualMachineInstance) error
	PrepareMigrationTarget(*v1.VirtualMachineInstance, bool) error
}

type LibvirtDomainManager struct {
	virConn cli.Connection

	// Anytime a get and a set is done on the domain, this lock must be held.
	domainModifyLock sync.Mutex

	virtShareDir           string
	notifier               *eventsclient.NotifyClient
	lessPVCSpaceToleration int
}

func NewLibvirtDomainManager(connection cli.Connection, virtShareDir string, notifier *eventsclient.NotifyClient, lessPVCSpaceToleration int) (DomainManager, error) {
	manager := LibvirtDomainManager{
		virConn:                connection,
		virtShareDir:           virtShareDir,
		notifier:               notifier,
		lessPVCSpaceToleration: lessPVCSpaceToleration,
	}

	return &manager, nil
}

func (l *LibvirtDomainManager) initializeMigrationMetadata(vmi *v1.VirtualMachineInstance) (bool, error) {
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
	}
	_, err = l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return false, err
	}
	return false, nil
}

func (l *LibvirtDomainManager) setMigrationResult(vmi *v1.VirtualMachineInstance, failed bool, reason string) error {

	connectionInterval := 10 * time.Second
	connectionTimeout := 60 * time.Second

	err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
		err = l.setMigrationResultHelper(vmi, failed, reason)
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

func (l *LibvirtDomainManager) setMigrationResultHelper(vmi *v1.VirtualMachineInstance, failed bool, reason string) error {

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

	if failed {
		domainSpec.Metadata.KubeVirt.Migration.Failed = true
		domainSpec.Metadata.KubeVirt.Migration.FailureReason = reason
	} else {
		domainSpec.Metadata.KubeVirt.Migration.Completed = true
	}
	domainSpec.Metadata.KubeVirt.Migration.EndTimestamp = &now

	_, err = l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}
	return nil

}

func prepateMigrationFlags(isBlockMigration bool) libvirt.DomainMigrateFlags {
	migrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_TUNNELLED

	if isBlockMigration {
		migrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
	}
	return migrateFlags

}

func (l *LibvirtDomainManager) asyncMigrate(vmi *v1.VirtualMachineInstance) {

	go func(l *LibvirtDomainManager, vmi *v1.VirtualMachineInstance) {

		// Start local migration proxy.
		//
		// Right now Libvirt won't let us perform a migration using a unix socket, so
		// we have to create this local host tcp server that forwards the traffic
		// to libvirt in order to trick libvirt into doing what we want.
		migrationProxy := migrationproxy.NewTargetProxy("127.0.0.1", 22222, migrationproxy.SourceUnixFile(l.virtShareDir, string(vmi.UID)))

		err := migrationProxy.StartListening()
		if err != nil {
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err))
			return
		}

		defer migrationProxy.StopListening()

		// For a tunnelled migration, this is always the uri
		dstUri := "qemu+tcp://127.0.0.1:22222/system"

		destConn, err := libvirt.NewConnect(dstUri)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Failed to establish connection with target pod's libvirtd for migration.")
			return
		}

		domName := api.VMINamespaceKeyFunc(vmi)
		dom, err := l.virConn.LookupDomainByName(domName)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Live migration failed.")
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err))
			return
		}

		isBlockMigration := false
		if vmi.Status.MigrationMethod == v1.BlockMigration {
			isBlockMigration = true
		}
		migrateFlags := prepateMigrationFlags(isBlockMigration)
		_, err = dom.Migrate(destConn, migrateFlags, "", "", 0)
		if err != nil {

			log.Log.Object(vmi).Reason(err).Error("Live migration failed.")
			l.setMigrationResult(vmi, true, fmt.Sprintf("%v", err))
			return
		}

		log.Log.Object(vmi).Infof("Live migration succeeded.")
		l.setMigrationResult(vmi, false, "")
	}(l, vmi)
}

func (l *LibvirtDomainManager) MigrateVMI(vmi *v1.VirtualMachineInstance) error {

	if vmi.Status.MigrationState == nil {
		return fmt.Errorf("cannot migration VMI until migrationState is ready")
	}

	inProgress, err := l.initializeMigrationMetadata(vmi)
	if err != nil {
		return err
	}

	if inProgress {
		return nil
	}

	l.asyncMigrate(vmi)

	return nil
}

// Prepares the target pod environment by executing the preStartHook
func (l *LibvirtDomainManager) PrepareMigrationTarget(vmi *v1.VirtualMachineInstance, useEmulation bool) error {

	logger := log.Log.Object(vmi)

	domain := &api.Domain{}
	podCPUSet, err := util.GetPodCPUSet()
	if err != nil {
		logger.Reason(err).Error("failed to read pod cpuset.")
		return fmt.Errorf("failed to read pod cpuset: %v", err)
	}

	// Map the VirtualMachineInstance to the Domain
	c := &api.ConverterContext{
		VirtualMachine: vmi,
		UseEmulation:   useEmulation,
		CPUSet:         podCPUSet,
	}
	if err := api.Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c); err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	dom, err := l.preStartHook(vmi, domain)
	if err != nil {
		return fmt.Errorf("pre-start pod-setup failed: %v", err)
	}
	// TODO this should probably a OnPrepareMigration hook or something.
	// Right now we need to call OnDefineDomain, so that additional setup, which might be done
	// by the hook can also be done for the new target pod
	hooksManager := hooks.GetManager()
	_, err = hooksManager.OnDefineDomain(&dom.Spec, vmi)
	if err != nil {
		return fmt.Errorf("executing custom preStart hooks failed: %v", err)
	}
	return nil
}

// All local environment setup that needs to occur before VirtualMachineInstance starts
// can be done in this function. This includes things like...
//
// - storage prep
// - network prep
// - cloud-init
//
// The Domain.Spec can be alterned in this function and any changes
// made to the domain will get set in libvirt after this function exits.
func (l *LibvirtDomainManager) preStartHook(vmi *v1.VirtualMachineInstance, domain *api.Domain) (*api.Domain, error) {

	logger := log.Log.Object(vmi)

	logger.Info("Executing PreStartHook on VMI pod environment")
	// ensure registry disk files have correct ownership privileges
	err := containerdisk.SetFilePermissions(vmi)
	if err != nil {
		return domain, fmt.Errorf("setting registry-disk file permissions failed: %v", err)
	}

	// generate cloud-init data
	cloudInitData := cloudinit.GetCloudInitNoCloudSource(vmi)

	// Pass cloud-init data to PreCloudInitIso hook
	logger.Info("Starting PreCloudInitIso hook")
	hooksManager := hooks.GetManager()
	cloudInitData, err = hooksManager.PreCloudInitIso(vmi, cloudInitData)
	if err != nil {
		return domain, fmt.Errorf("PreCloudInitIso hook failed: %v", err)
	}

	if cloudInitData != nil {
		hostname := dns.SanitizeHostname(vmi)

		err := cloudinit.GenerateLocalData(vmi.Name, hostname, vmi.Namespace, cloudInitData)
		if err != nil {
			return domain, fmt.Errorf("generating local cloud-init data failed: %v", err)
		}
	}

	// setup networking
	err = network.SetupPodNetwork(vmi, domain)
	if err != nil {
		return domain, fmt.Errorf("preparing the pod network failed: %v", err)
	}

	// create disks images on the cluster lever
	// or initalize disks images for empty PVC
	hostDiskCreator := hostdisk.NewHostDiskCreator(l.notifier, l.lessPVCSpaceToleration)
	err = hostDiskCreator.Create(vmi)
	if err != nil {
		return domain, fmt.Errorf("preparing host-disks failed: %v", err)
	}

	// Create images for volumes that are marked ephemeral.
	err = ephemeraldisk.CreateEphemeralImages(vmi)
	if err != nil {
		return domain, fmt.Errorf("preparing ephemeral images failed: %v", err)
	}
	// create empty disks if they exist
	if err := emptydisk.CreateTemporaryDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating empty disks failed: %v", err)
	}
	// create ConfigMap disks if they exists
	if err := config.CreateConfigMapDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating config map disks failed: %v", err)
	}
	// create Secret disks if they exists
	if err := config.CreateSecretDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating secret disks failed: %v", err)
	}
	// create ServiceAccount disk if exists
	if err := config.CreateServiceAccountDisk(vmi); err != nil {
		return domain, fmt.Errorf("creating service account disk failed: %v", err)
	}

	// set drivers cache mode
	for i := range domain.Spec.Devices.Disks {
		err := api.SetDriverCacheMode(&domain.Spec.Devices.Disks[i])
		if err != nil {
			return domain, err
		}
	}

	return domain, err
}

// This function parses variables that are set by SR-IOV device plugin listing
// PCI IDs for devices allocated to the pod. It also parses variables that
// virt-controller sets mapping network names to their respective resource
// names (if any).
//
// Format for PCI ID variables set by SR-IOV DP is:
// "": for no allocated devices
// PCIDEVICE_<resourceName>="0000:81:11.1": for a single device
// PCIDEVICE_<resourceName>="0000:81:11.1 0000:81:11.2[ ...]": for multiple devices
//
// Since special characters in environment variable names are not allowed,
// resourceName is mutated as follows:
// 1. All dots and slashes are replaced with underscore characters.
// 2. The result is upper cased.
//
// Example: PCIDEVICE_INTEL_COM_SRIOV_TEST=... for intel.com/sriov_test resources.
//
// Format for network to resource mapping variables is:
// KUBEVIRT_RESOURCE_NAME_<networkName>=<resourceName>
//
func resourceNameToEnvvar(resourceName string) string {
	varName := strings.ToUpper(resourceName)
	varName = strings.Replace(varName, "/", "_", -1)
	varName = strings.Replace(varName, ".", "_", -1)
	return fmt.Sprintf("PCIDEVICE_%s", varName)
}

func getSRIOVPCIAddresses(ifaces []v1.Interface) map[string][]string {
	networkToAddressesMap := map[string][]string{}
	for _, iface := range ifaces {
		if iface.SRIOV == nil {
			continue
		}
		networkToAddressesMap[iface.Name] = []string{}
		varName := fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", iface.Name)
		resourceName, isSet := os.LookupEnv(varName)
		if isSet {
			varName := resourceNameToEnvvar(resourceName)
			pciAddrString, isSet := os.LookupEnv(varName)
			if isSet {
				addrs := strings.Split(pciAddrString, ",")
				naddrs := len(addrs)
				if naddrs > 0 {
					if addrs[naddrs-1] == "" {
						addrs = addrs[:naddrs-1]
					}
				}
				networkToAddressesMap[iface.Name] = addrs
			} else {
				log.DefaultLogger().Warningf("%s not set for SR-IOV interface %s", varName, iface.Name)
			}
		} else {
			log.DefaultLogger().Warningf("%s not set for SR-IOV interface %s", varName, iface.Name)
		}
	}
	return networkToAddressesMap
}

func (l *LibvirtDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, useEmulation bool) (*api.DomainSpec, error) {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	logger := log.Log.Object(vmi)

	domain := &api.Domain{}
	podCPUSet, err := util.GetPodCPUSet()
	if err != nil {
		logger.Reason(err).Error("failed to read pod cpuset.")
		return nil, err
	}

	// Check if PVC volumes are block volumes
	isBlockPVCMap := make(map[string]bool)
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			isBlockPVC, err := isBlockDeviceVolume(volume.Name)
			if err != nil {
				logger.Reason(err).Errorf("failed to detect volume mode for Volume %v and PVC %v.",
					volume.Name, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
				return nil, err
			}
			isBlockPVCMap[volume.Name] = isBlockPVC
		}
	}

	// Map the VirtualMachineInstance to the Domain
	c := &api.ConverterContext{
		VirtualMachine: vmi,
		UseEmulation:   useEmulation,
		CPUSet:         podCPUSet,
		IsBlockPVC:     isBlockPVCMap,
		SRIOVDevices:   getSRIOVPCIAddresses(vmi.Spec.Domain.Devices.Interfaces),
	}
	if err := api.Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c); err != nil {
		logger.Error("Conversion failed.")
		return nil, err
	}

	// Set defaults which are not coming from the cluster
	api.SetObjectDefaults_Domain(domain)

	dom, err := l.virConn.LookupDomainByName(domain.Spec.Name)
	newDomain := false
	if err != nil {
		// We need the domain but it does not exist, so create it
		if domainerrors.IsNotFound(err) {
			newDomain = true
			domain, err = l.preStartHook(vmi, domain)
			if err != nil {
				logger.Reason(err).Error("pre start setup for VirtualMachineInstance failed.")
				return nil, err
			}
			dom, err = l.setDomainSpecWithHooks(vmi, &domain.Spec)
			if err != nil {
				return nil, err
			}
			logger.Info("Domain defined.")
		} else {
			logger.Reason(err).Error("Getting the domain failed.")
			return nil, err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("Getting the domain state failed.")
		return nil, err
	}

	// To make sure, that we set the right qemu wrapper arguments,
	// we update the domain XML whenever a VirtualMachineInstance was already defined but not running
	if !newDomain && cli.IsDown(domState) {
		dom, err = l.setDomainSpecWithHooks(vmi, &domain.Spec)
		if err != nil {
			return nil, err
		}
	}

	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	// TODO blocked state
	if cli.IsDown(domState) && !vmi.IsRunning() && !vmi.IsFinal() {
		err = dom.Create()
		if err != nil {
			logger.Reason(err).Error("Starting the VirtualMachineInstance failed.")
			return nil, err
		}
		logger.Info("Domain started.")
	} else if cli.IsPaused(domState) {
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logger.Reason(err).Error("Resuming the VirtualMachineInstance failed.")
			return nil, err
		}
		logger.Info("Domain resumed.")
	} else {
		// Nothing to do
	}

	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	var newSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		logger.Reason(err).Error("Parsing domain XML failed.")
		return nil, err
	}

	// TODO: check if VirtualMachineInstance Spec and Domain Spec are equal or if we have to sync
	return &newSpec, nil
}

func isBlockDeviceVolume(volumeName string) (bool, error) {
	// check for block device
	path := api.GetBlockDeviceVolumePath(volumeName)
	fileInfo, err := os.Stat(path)
	if err == nil {
		if (fileInfo.Mode() & os.ModeDevice) != 0 {
			return true, nil
		}
		return false, fmt.Errorf("found %v, but it's not a block device", path)
	}
	if os.IsNotExist(err) {
		// cross check: is it a filesystem volume
		path = api.GetFilesystemVolumePath(volumeName)
		fileInfo, err := os.Stat(path)
		if err == nil {
			if fileInfo.Mode().IsRegular() {
				return false, nil
			}
			return false, fmt.Errorf("found %v, but it's not a regular file", path)
		}
		if os.IsNotExist(err) {
			return false, fmt.Errorf("neither found block device nor regular file for volume %v", volumeName)
		}
	}
	return false, fmt.Errorf("error checking for block device: %v", err)
}

func (l *LibvirtDomainManager) getDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	state, _, err := dom.GetState()
	if err != nil {
		return nil, err
	}
	return util.GetDomainSpec(state, dom)
}

func (l *LibvirtDomainManager) SignalShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := util.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed during graceful shutdown.")
			return err
		}
	}
	defer dom.Free()

	domState, _, err := dom.GetState()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		domSpec, err := l.getDomainSpec(dom)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Unable to retrieve domain xml")
			return err
		}

		if domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp == nil {
			err = dom.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Signalling graceful shutdown failed.")
				return err
			}
			log.Log.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())

			now := metav1.Now()
			domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now
			_, err = l.setDomainSpecWithHooks(vmi, domSpec)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Unable to update grace period start time on domain xml")
				return err
			}
		}
	}

	return nil
}

func (l *LibvirtDomainManager) KillVMI(vmi *v1.VirtualMachineInstance) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		log.Log.Object(vmi).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED || domState == libvirt.DOMAIN_SHUTDOWN {
		err = dom.DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				return nil
			}
			log.Log.Object(vmi).Reason(err).Error("Destroying the domain state failed.")
			return err
		}
		log.Log.Object(vmi).Info("Domain stopped.")
		return nil
	}

	log.Log.Object(vmi).Info("Domain not running or paused, nothing to do.")
	return nil
}

func (l *LibvirtDomainManager) DeleteVMI(vmi *v1.VirtualMachineInstance) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the domain does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()

	err = dom.Undefine()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Undefining the domain failed.")
		return err
	}
	log.Log.Object(vmi).Info("Domain undefined.")
	return nil
}

func (l *LibvirtDomainManager) ListAllDomains() ([]*api.Domain, error) {

	doms, err := l.virConn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, err
	}

	var list []*api.Domain
	for _, dom := range doms {
		domain, err := util.NewDomain(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		spec, err := l.getDomainSpec(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.Spec = *spec
		status, reason, err := dom.GetState()
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		list = append(list, domain)
		dom.Free()
	}

	return list, nil
}

func (l *LibvirtDomainManager) setDomainSpecWithHooks(vmi *v1.VirtualMachineInstance, spec *api.DomainSpec) (cli.VirDomain, error) {

	hooksManager := hooks.GetManager()
	domainSpec, err := hooksManager.OnDefineDomain(spec, vmi)
	if err != nil {
		return nil, err
	}
	return util.SetDomainSpecStr(l.virConn, vmi, domainSpec)
}
