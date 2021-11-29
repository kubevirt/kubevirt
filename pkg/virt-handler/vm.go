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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virthandler

import (
	"context"
	goerror "errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/config"

	"github.com/opencontainers/runc/libcontainer/cgroups"

	nodelabellerapi "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	netcache "kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	"kubevirt.io/kubevirt/pkg/util"

	"kubevirt.io/kubevirt/pkg/virt-handler/heartbeat"

	"kubevirt.io/kubevirt/pkg/util/migrations"

	container_disk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	hotplug_volume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	pvctypes "kubevirt.io/kubevirt/pkg/util/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

const (
	//VolumeReadyReason is the reason set when the volume is ready.
	VolumeReadyReason = "VolumeReady"
	//VolumeUnMountedFromPodReason is the reason set when the volume is unmounted from the virtlauncher pod
	VolumeUnMountedFromPodReason = "VolumeUnMountedFromPod"
	//VolumeMountedToPodReason is the reason set when the volume is mounted to the virtlauncher pod
	VolumeMountedToPodReason = "VolumeMountedToPod"
	//VolumeUnplugged is the reason set when the volume is completely unplugged from the VMI
	VolumeUnplugged = "VolumeUnplugged"
	//VMIDefined is the reason set when a VMI is defined
	VMIDefined = "VirtualMachineInstance defined."
	//VMIStarted is the reason set when a VMI is started
	VMIStarted = "VirtualMachineInstance started."
	//VMIShutdown is the reason set when a VMI is shutdown
	VMIShutdown = "The VirtualMachineInstance was shut down."
	//VMICrashed is the reason set when a VMI crashed
	VMICrashed = "The VirtualMachineInstance crashed."
	//VMIAbortingMigration is the reason set when migration is being aborted
	VMIAbortingMigration = "VirtualMachineInstance is aborting migration."
	//VMIMigrating in the reason set when the VMI is migrating
	VMIMigrating = "VirtualMachineInstance is migrating."
	//VMIMigrationTargetPrepared is the reason set when the migration target has been prepared
	VMIMigrationTargetPrepared = "VirtualMachineInstance Migration Target Prepared."
	//VMIStopping is the reason set when the VMI is stopping
	VMIStopping = "VirtualMachineInstance stopping"
	//VMIGracefulShutdown is the reason set when the VMI is gracefully shut down
	VMIGracefulShutdown = "Signaled Graceful Shutdown"
	//VMISignalDeletion is the reason set when the VMI has signal deletion
	VMISignalDeletion = "Signaled Deletion"
)

var RequiredGuestAgentCommands = []string{
	"guest-ping",
	"guest-get-time",
	"guest-info",
	"guest-shutdown",
	"guest-network-get-interfaces",
	"guest-get-fsinfo",
	"guest-get-host-name",
	"guest-get-users",
	"guest-get-timezone",
	"guest-get-osinfo",
}

var SSHRelatedGuestAgentCommands = []string{
	"guest-exec-status",
	"guest-exec",
	"guest-file-open",
	"guest-file-close",
	"guest-file-read",
	"guest-file-write",
}

var PasswordRelatedGuestAgentCommands = []string{
	"guest-set-user-password",
}

func NewController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	ipAddress string,
	virtShareDir string,
	virtPrivateDir string,
	vmiSourceInformer cache.SharedIndexInformer,
	vmiTargetInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	gracefulShutdownInformer cache.SharedIndexInformer,
	watchdogTimeoutSeconds int,
	maxDevices int,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	migrationProxy migrationproxy.ProxyManager,
	capabilities *nodelabellerapi.Capabilities,
) *VirtualMachineController {

	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-handler-vm")

	c := &VirtualMachineController{
		Queue:                       queue,
		recorder:                    recorder,
		clientset:                   clientset,
		host:                        host,
		ipAddress:                   ipAddress,
		virtShareDir:                virtShareDir,
		vmiSourceInformer:           vmiSourceInformer,
		vmiTargetInformer:           vmiTargetInformer,
		domainInformer:              domainInformer,
		gracefulShutdownInformer:    gracefulShutdownInformer,
		heartBeatInterval:           1 * time.Minute,
		watchdogTimeoutSeconds:      watchdogTimeoutSeconds,
		migrationProxy:              migrationProxy,
		podIsolationDetector:        podIsolationDetector,
		containerDiskMounter:        container_disk.NewMounter(podIsolationDetector, virtPrivateDir+"/container-disk-mount-state", clusterConfig),
		hotplugVolumeMounter:        hotplug_volume.NewVolumeMounter(podIsolationDetector, virtPrivateDir+"/hotplug-volume-mount-state"),
		clusterConfig:               clusterConfig,
		networkCacheStoreFactory:    netcache.NewInterfaceCacheFactory(),
		virtLauncherFSRunDirPattern: "/proc/%d/root/var/run",
		capabilities:                capabilities,
		vmiExpectations:             controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	vmiSourceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	vmiTargetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDomainFunc,
		DeleteFunc: c.deleteDomainFunc,
		UpdateFunc: c.updateDomainFunc,
	})

	gracefulShutdownInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	c.launcherClients = virtcache.LauncherClientInfoByVMI{}
	c.phase1NetworkSetupCache = virtcache.LauncherPIDByVMI{}
	c.podInterfaceCache = virtcache.PodInterfaceByVMIAndName{}

	c.domainNotifyPipes = make(map[string]string)

	permissions := "rw"
	if cgroups.IsCgroup2UnifiedMode() {
		// Need 'rwm' permissions otherwise ebpf filtering program attached by runc
		// will deny probing the device file with 'access' syscall. That in turn
		// will lead to libvirtd failure on VM startup.
		// This has been fixed upstream:
		//   https://github.com/opencontainers/runc/pull/2796
		// but the workaround is still needed to support previous versions without
		// the patch.
		permissions = "rwm"
	}

	c.deviceManagerController = device_manager.NewDeviceController(c.host, maxDevices, permissions, clusterConfig)
	c.heartBeat = heartbeat.NewHeartBeat(clientset.CoreV1(), c.deviceManagerController, clusterConfig, host)

	return c
}

type VirtualMachineController struct {
	recorder                 record.EventRecorder
	clientset                kubecli.KubevirtClient
	host                     string
	ipAddress                string
	virtShareDir             string
	virtPrivateDir           string
	Queue                    workqueue.RateLimitingInterface
	vmiSourceInformer        cache.SharedIndexInformer
	vmiTargetInformer        cache.SharedIndexInformer
	domainInformer           cache.SharedInformer
	gracefulShutdownInformer cache.SharedIndexInformer
	launcherClients          virtcache.LauncherClientInfoByVMI
	heartBeatInterval        time.Duration
	watchdogTimeoutSeconds   int
	deviceManagerController  *device_manager.DeviceController
	migrationProxy           migrationproxy.ProxyManager
	podIsolationDetector     isolation.PodIsolationDetector
	containerDiskMounter     container_disk.Mounter
	hotplugVolumeMounter     hotplug_volume.VolumeMounter
	clusterConfig            *virtconfig.ClusterConfig

	// records if pod network phase1 has completed
	// phase1 involves cycling an entire posix thread
	// so for performance, knowing phase1 is complete
	// prevents cycling an unncessary posix thread.
	phase1NetworkSetupCache virtcache.LauncherPIDByVMI

	// key is the file path, value is the contents.
	// if key exists, then don't read directly from file.
	podInterfaceCache virtcache.PodInterfaceByVMIAndName

	domainNotifyPipes           map[string]string
	networkCacheStoreFactory    netcache.InterfaceCacheFactory
	virtLauncherFSRunDirPattern string
	heartBeat                   *heartbeat.HeartBeat
	capabilities                *nodelabellerapi.Capabilities
	vmiExpectations             *controller.UIDTrackingControllerExpectations
}

type virtLauncherCriticalSecurebootError struct {
	msg string
}

func (e *virtLauncherCriticalSecurebootError) Error() string { return e.msg }

func handleDomainNotifyPipe(domainPipeStopChan chan struct{}, ln net.Listener, virtShareDir string, vmi *v1.VirtualMachineInstance) {

	fdChan := make(chan net.Conn, 100)

	// Close listener and exit when stop encountered
	go func() {
		<-domainPipeStopChan
		log.Log.Object(vmi).Infof("closing notify pipe listener for vmi")
		if err := ln.Close(); err != nil {
			log.Log.Object(vmi).Infof("failed closing notify pipe listener for vmi: %v", err)
		}
	}()

	// Listen for new connections,
	go func(vmi *v1.VirtualMachineInstance, ln net.Listener, domainPipeStopChan chan struct{}) {
		for {
			fd, err := ln.Accept()
			if err != nil {
				if goerror.Is(err, net.ErrClosed) {
					// As Accept blocks, closing it is our mechanism to exit this loop
					return
				}
				log.Log.Reason(err).Error("Domain pipe accept error encountered.")
				// keep listening until stop invoked
				time.Sleep(1 * time.Second)
			} else {
				fdChan <- fd
			}
		}
	}(vmi, ln, domainPipeStopChan)

	// Process new connections
	// exit when stop encountered
	go func(vmi *v1.VirtualMachineInstance, fdChan chan net.Conn, domainPipeStopChan chan struct{}) {
		for {
			select {
			case <-domainPipeStopChan:
				return
			case fd := <-fdChan:
				go func(vmi *v1.VirtualMachineInstance) {
					defer fd.Close()

					// pipe the VMI domain-notify.sock to the virt-handler domain-notify.sock
					// so virt-handler receives notifications from the VMI
					conn, err := net.Dial("unix", filepath.Join(virtShareDir, "domain-notify.sock"))
					if err != nil {
						log.Log.Reason(err).Error("error connecting to domain-notify.sock for proxy connection")
						return
					}
					defer conn.Close()

					log.Log.Object(vmi).Infof("Accepted new notify pipe connection for vmi")
					copyErr := make(chan error, 2)
					go func() {
						_, err := io.Copy(fd, conn)
						copyErr <- err
					}()
					go func() {
						_, err := io.Copy(conn, fd)
						copyErr <- err
					}()

					// wait until one of the copy routines exit then
					// let the fd close
					err = <-copyErr
					if err != nil {
						log.Log.Object(vmi).Infof("closing notify pipe connection for vmi with error: %v", err)
					} else {
						log.Log.Object(vmi).Infof("gracefully closed notify pipe connection for vmi")
					}

				}(vmi)
			}
		}
	}(vmi, fdChan, domainPipeStopChan)
}

func (d *VirtualMachineController) startDomainNotifyPipe(domainPipeStopChan chan struct{}, vmi *v1.VirtualMachineInstance) error {

	res, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod when setting up notify pipe: %v", err)
	}

	// inject the domain-notify.sock into the VMI pod.
	socketPath := filepath.Join(res.MountRoot(), d.virtShareDir, "domain-notify-pipe.sock")

	os.RemoveAll(socketPath)
	err = util.MkdirAllWithNosec(filepath.Dir(socketPath))
	if err != nil {
		log.Log.Reason(err).Error("unable to create directory for unix socket")
		return err
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}

	if util.IsNonRootVMI(vmi) {
		err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			log.Log.Reason(err).Error("unable to change ownership for domain notify")
			return err
		}
	}

	handleDomainNotifyPipe(domainPipeStopChan, listener, d.virtShareDir, vmi)

	return nil
}

// Determines if a domain's grace period has expired during shutdown.
// If the grace period has started but not expired, timeLeft represents
// the time in seconds left until the period expires.
// If the grace period has not started, timeLeft will be set to -1.
func (d *VirtualMachineController) hasGracePeriodExpired(dom *api.Domain) (hasExpired bool, timeLeft int64) {

	hasExpired = false
	timeLeft = 0

	if dom == nil {
		hasExpired = true
		return
	}

	startTime := int64(0)
	if dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp != nil {
		startTime = dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp.UTC().Unix()
	}
	gracePeriod := dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds

	// If gracePeriod == 0, then there will be no startTime set, deletion
	// should occur immediately during shutdown.
	if gracePeriod == 0 {
		hasExpired = true
		return
	} else if startTime == 0 {
		// If gracePeriod > 0, then the shutdown signal needs to be sent
		// and the gracePeriod start time needs to be set.
		timeLeft = -1
		return
	}

	now := time.Now().UTC().Unix()
	diff := now - startTime

	if diff >= gracePeriod {
		hasExpired = true
		return
	}

	timeLeft = int64(gracePeriod - diff)
	if timeLeft < 1 {
		timeLeft = 1
	}
	return
}

func (d *VirtualMachineController) hasTargetDetectedDomain(vmi *v1.VirtualMachineInstance) (bool, int64) {
	// give the target node 60 seconds to discover the libvirt domain via the domain informer
	// before allowing the VMI to be processed. This closes the gap between the
	// VMI's status getting updated to reflect the new source node, and the domain
	// informer firing the event to alert the source node of the new domain.
	migrationTargetDelayTimeout := 60

	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetNodeDomainDetected {

		return true, 0
	}

	nowUnix := time.Now().UTC().Unix()
	migrationEndUnix := vmi.Status.MigrationState.EndTimestamp.Time.UTC().Unix()

	diff := nowUnix - migrationEndUnix

	if diff > int64(migrationTargetDelayTimeout) {
		return false, 0
	}

	timeLeft := int64(migrationTargetDelayTimeout) - diff

	enqueueTime := timeLeft
	if enqueueTime < 5 {
		enqueueTime = 5
	}

	// re-enqueue the key to ensure it gets processed again within the right time.
	d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Duration(enqueueTime)*time.Second)

	return false, timeLeft
}

func (d *VirtualMachineController) clearPodNetworkPhase1(vmi *v1.VirtualMachineInstance) {
	// no need to cleanup with empty uid
	if string(vmi.UID) == "" {
		return
	}
	d.phase1NetworkSetupCache.Delete(vmi.UID)

	// Clean Pod interface cache from map and files
	d.podInterfaceCache.DeleteAllForVMI(vmi.UID)

	err := d.networkCacheStoreFactory.CacheForVMI(vmi).Remove()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to delete VMI Network cache files: %s", err.Error())
	}
}

// Reaching into the network namespace of the VMI's pod is expensive because
// it results in killing/spawning a posix thread. Only do this if it
// is absolutely necessary. The cache informs us if this action has
// already taken place or not for a VMI
func (d *VirtualMachineController) setPodNetworkPhase1(vmi *v1.VirtualMachineInstance) error {

	// configure network
	res, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod: %v", err)
	}

	// check to see if we've already completed phase1 for this vmi
	if _, exists := d.phase1NetworkSetupCache.Load(vmi.UID); exists {
		return nil
	}

	if virtutil.IsNonRootVMI(vmi) && virtutil.WantVirtioNetDevice(vmi) {
		rootMount := res.MountRoot()
		err := d.claimDeviceOwnership(rootMount, "vhost-net")
		if err != nil {
			return neterrors.CreateCriticalNetworkError(fmt.Errorf("failed to set up vhost-net device, %s", err))
		}
	}

	pid := res.Pid()
	err = res.DoNetNS(func() error {
		return netsetup.NewVMNetworkConfigurator(vmi, d.networkCacheStoreFactory).SetupPodNetworkPhase1(pid)
	})
	if err != nil {
		return err
	}

	// cache that phase 1 has completed for this vmi.
	d.phase1NetworkSetupCache.Store(vmi.UID, 0)

	return nil
}

func domainMigrated(domain *api.Domain) bool {
	if domain != nil && domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonMigrated {
		return true
	}
	return false
}

func (d *VirtualMachineController) getPodInterfacefromFileCache(vmi *v1.VirtualMachineInstance, ifaceName string) (*netcache.PodCacheInterface, error) {
	// Once the Interface files are set on the handler, they don't change
	// If already present in the map, don't read again
	podInterface, exists := d.podInterfaceCache.Load(vmi.UID, ifaceName)

	if exists {
		return podInterface, nil
	}

	//FIXME error handling?
	podInterface, _ = d.networkCacheStoreFactory.CacheForVMI(vmi).Read(ifaceName)

	d.podInterfaceCache.Store(vmi.UID, ifaceName, podInterface)

	return podInterface, nil
}

func canUpdateToMounted(currentPhase v1.VolumePhase) bool {
	return currentPhase == v1.VolumeBound || currentPhase == v1.VolumePending || currentPhase == v1.HotplugVolumeAttachedToNode
}

func canUpdateToUnmounted(currentPhase v1.VolumePhase) bool {
	return currentPhase == v1.VolumeReady || currentPhase == v1.HotplugVolumeMounted || currentPhase == v1.HotplugVolumeAttachedToNode
}

func (d *VirtualMachineController) setMigrationProgressStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil ||
		domain.Spec.Metadata.KubeVirt.Migration == nil ||
		vmi.Status.MigrationState == nil ||
		!d.isMigrationSource(vmi) {
		return
	}

	migrationMetadata := domain.Spec.Metadata.KubeVirt.Migration
	if migrationMetadata.UID != vmi.Status.MigrationState.MigrationUID {
		return
	}

	if vmi.Status.MigrationState.EndTimestamp == nil && migrationMetadata.EndTimestamp != nil {
		if migrationMetadata.Failed {
			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("VirtualMachineInstance migration uid %s failed. reason:%s", string(migrationMetadata.UID), migrationMetadata.FailureReason))
		}
	}

	if vmi.Status.MigrationState.StartTimestamp == nil {
		vmi.Status.MigrationState.StartTimestamp = migrationMetadata.StartTimestamp
	}
	if vmi.Status.MigrationState.EndTimestamp == nil {
		vmi.Status.MigrationState.EndTimestamp = migrationMetadata.EndTimestamp
	}
	vmi.Status.MigrationState.AbortStatus = v1.MigrationAbortStatus(migrationMetadata.AbortStatus)
	vmi.Status.MigrationState.Completed = migrationMetadata.Completed
	vmi.Status.MigrationState.Failed = migrationMetadata.Failed
	vmi.Status.MigrationState.Mode = migrationMetadata.Mode
}

func (d *VirtualMachineController) migrationSourceUpdateVMIStatus(origVMI *v1.VirtualMachineInstance, domain *api.Domain) error {

	vmi := origVMI.DeepCopy()
	oldStatus := vmi.DeepCopy().Status

	// if a migration happens very quickly, it's possible parts of the in
	// progress status wasn't set. We need to make sure we set this even
	// if the migration has completed
	d.setMigrationProgressStatus(vmi, domain)

	// handle migrations differently than normal status updates.
	//
	// When a successful migration is detected, we must transfer ownership of the VMI
	// from the source node (this node) to the target node (node the domain was migrated to).
	//
	// Transfer owership by...
	// 1. Marking vmi.Status.MigationState as completed
	// 2. Update the vmi.Status.NodeName to reflect the target node's name
	// 3. Update the VMI's NodeNameLabel annotation to reflect the target node's name
	// 4. Clear the LauncherContainerImageVersion which virt-controller will detect
	//    and accurately based on the version used on the target pod
	//
	// After a migration, the VMI's phase is no longer owned by this node. Only the
	// MigrationState status field is elgible to be mutated.
	migrationHost := ""
	if vmi.Status.MigrationState != nil {
		migrationHost = vmi.Status.MigrationState.TargetNode
	}

	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.EndTimestamp == nil {
		now := metav1.NewTime(time.Now())
		vmi.Status.MigrationState.EndTimestamp = &now
	}

	targetNodeDetectedDomain, timeLeft := d.hasTargetDetectedDomain(vmi)
	// If we can't detect where the migration went to, then we have no
	// way of transferring ownership. The only option here is to move the
	// vmi to failed.  The cluster vmi controller will then tear down the
	// resulting pods.
	if migrationHost == "" {
		// migrated to unknown host.
		vmi.Status.Phase = v1.Failed
		vmi.Status.MigrationState.Completed = true
		vmi.Status.MigrationState.Failed = true

		d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to unknown host."))
	} else if !targetNodeDetectedDomain {
		if timeLeft <= 0 {
			vmi.Status.Phase = v1.Failed
			vmi.Status.MigrationState.Completed = true
			vmi.Status.MigrationState.Failed = true

			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance's domain was never observed on the target after the migration completed within the timeout period."))
		} else {
			log.Log.Object(vmi).Info("Waiting on the target node to observe the migrated domain before performing the handoff")
		}
	} else if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetNodeDomainDetected {
		// this is the migration ACK.
		// At this point we know that the migration has completed and that
		// the target node has seen the domain event.
		vmi.Labels[v1.NodeNameLabel] = migrationHost
		delete(vmi.Labels, v1.OutdatedLauncherImageLabel)
		vmi.Status.LauncherContainerImageVersion = ""
		vmi.Status.NodeName = migrationHost
		// clean the evacuation node name since have already migrated to a new node
		vmi.Status.EvacuationNodeName = ""
		vmi.Status.MigrationState.Completed = true
		// update the vmi migrationTransport to indicate that next migration should use unix URI
		// new workloads will set the migrationTransport on their creation, however, legacy workloads
		// can make the switch only after the first migration
		vmi.Status.MigrationTransport = v1.MigrationTransportUnix
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to node %s.", migrationHost))
		log.Log.Object(vmi).Infof("migration completed to node %s", migrationHost)
	}

	if !reflect.DeepEqual(oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		d.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
		if err != nil {
			d.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}
	return nil
}

func (d *VirtualMachineController) migrationTargetUpdateVMIStatus(vmi *v1.VirtualMachineInstance, domainExists bool) error {

	vmiCopy := vmi.DeepCopy()

	if migrations.MigrationFailed(vmi) {
		// nothing left to report on the target node if the migration failed
		return nil
	}

	// Handle post migration
	if domainExists && vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.TargetNodeDomainDetected {
		// record that we've see the domain populated on the target's node
		log.Log.Object(vmi).Info("The target node received the migrated domain")
		vmiCopy.Status.MigrationState.TargetNodeDomainDetected = true
		d.finalizeMigration(vmi)
	}

	if !migrations.IsMigrating(vmi) {
		destSrcPortsMap := d.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
		if len(destSrcPortsMap) == 0 {
			msg := "target migration listener is not up for this vmi"
			log.Log.Object(vmi).Error(msg)
			return fmt.Errorf(msg)
		}

		hostAddress := ""

		// advertise the listener address to the source node
		if vmi.Status.MigrationState != nil {
			hostAddress = vmi.Status.MigrationState.TargetNodeAddress
		}
		if hostAddress != d.ipAddress {
			portsList := make([]string, 0, len(destSrcPortsMap))

			for k := range destSrcPortsMap {
				portsList = append(portsList, k)
			}
			portsStrList := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(portsList)), ","), "[]")
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), fmt.Sprintf("Migration Target is listening at %s, on ports: %s", d.ipAddress, portsStrList))
			vmiCopy.Status.MigrationState.TargetNodeAddress = d.ipAddress
			vmiCopy.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPortsMap
		}
	}

	// update the VMI if necessary
	if !reflect.DeepEqual(vmi.Status, vmiCopy.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		d.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmiCopy)
		if err != nil {
			d.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	return nil
}

func (d *VirtualMachineController) generateEventsForVolumeStatusChange(vmi *v1.VirtualMachineInstance, newStatusMap map[string]v1.VolumeStatus) {
	newStatusMapCopy := make(map[string]v1.VolumeStatus)
	for k, v := range newStatusMap {
		newStatusMapCopy[k] = v
	}
	for _, oldStatus := range vmi.Status.VolumeStatus {
		newStatus, ok := newStatusMap[oldStatus.Name]
		if !ok {
			// status got removed
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, VolumeUnplugged, fmt.Sprintf("Volume %s has been unplugged", oldStatus.Name))
			continue
		}
		if newStatus.Phase != oldStatus.Phase {
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, newStatus.Reason, newStatus.Message)
		}
		delete(newStatusMapCopy, newStatus.Name)
	}
	// Send events for any new statuses.
	for _, v := range newStatusMapCopy {
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v.Reason, v.Message)
	}
}

func (d *VirtualMachineController) updateHotplugVolumeStatus(vmi *v1.VirtualMachineInstance, volumeStatus v1.VolumeStatus, specVolumeMap map[string]v1.Volume) (v1.VolumeStatus, bool) {
	needsRefresh := false
	if volumeStatus.Target == "" {
		needsRefresh = true
		if mounted, _ := d.hotplugVolumeMounter.IsMounted(vmi, volumeStatus.Name, volumeStatus.HotplugVolume.AttachPodUID); mounted {
			if _, ok := specVolumeMap[volumeStatus.Name]; ok && canUpdateToMounted(volumeStatus.Phase) {
				log.DefaultLogger().Infof("Marking volume %s as mounted in pod, it can now be attached", volumeStatus.Name)
				// mounted, and still in spec, and in phase we can change, update status to mounted.
				volumeStatus.Phase = v1.HotplugVolumeMounted
				volumeStatus.Message = fmt.Sprintf("Volume %s has been mounted in virt-launcher pod", volumeStatus.Name)
				volumeStatus.Reason = VolumeMountedToPodReason
			}
		} else {
			// Not mounted, check if the volume is in the spec, if not update status
			if _, ok := specVolumeMap[volumeStatus.Name]; !ok && canUpdateToUnmounted(volumeStatus.Phase) {
				log.DefaultLogger().Infof("Marking volume %s as unmounted from pod, it can now be detached", volumeStatus.Name)
				// Not mounted.
				volumeStatus.Phase = v1.HotplugVolumeUnMounted
				volumeStatus.Message = fmt.Sprintf("Volume %s has been unmounted from virt-launcher pod", volumeStatus.Name)
				volumeStatus.Reason = VolumeUnMountedFromPodReason
			}
		}
	} else {
		// Successfully attached to VM.
		volumeStatus.Phase = v1.VolumeReady
		volumeStatus.Message = fmt.Sprintf("Successfully attach hotplugged volume %s to VM", volumeStatus.Name)
		volumeStatus.Reason = VolumeReadyReason
	}
	return volumeStatus, needsRefresh
}

func (d *VirtualMachineController) updateVolumeStatusesFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	hasHotplug := false

	if domain == nil {
		return hasHotplug
	}

	if len(vmi.Status.VolumeStatus) > 0 {
		diskDeviceMap := make(map[string]string)
		for _, disk := range domain.Spec.Devices.Disks {
			diskDeviceMap[disk.Alias.GetName()] = disk.Target.Device
		}
		specVolumeMap := make(map[string]v1.Volume)
		for _, volume := range vmi.Spec.Volumes {
			specVolumeMap[volume.Name] = volume
		}
		newStatusMap := make(map[string]v1.VolumeStatus)
		newStatuses := make([]v1.VolumeStatus, 0)
		needsRefresh := false
		for _, volumeStatus := range vmi.Status.VolumeStatus {
			if _, ok := diskDeviceMap[volumeStatus.Name]; ok {
				volumeStatus.Target = diskDeviceMap[volumeStatus.Name]
			}
			if volumeStatus.HotplugVolume != nil {
				hasHotplug = true
				volumeStatus, needsRefresh = d.updateHotplugVolumeStatus(vmi, volumeStatus, specVolumeMap)
			}
			newStatuses = append(newStatuses, volumeStatus)
			newStatusMap[volumeStatus.Name] = volumeStatus
		}
		sort.SliceStable(newStatuses, func(i, j int) bool {
			return strings.Compare(newStatuses[i].Name, newStatuses[j].Name) == -1
		})
		if needsRefresh {
			d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second)
		}
		d.generateEventsForVolumeStatusChange(vmi, newStatusMap)
		vmi.Status.VolumeStatus = newStatuses
	}
	return hasHotplug
}

func (d *VirtualMachineController) updateGuestInfoFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil {
		return
	}

	if vmi.Status.GuestOSInfo.Name != domain.Status.OSInfo.Name {
		vmi.Status.GuestOSInfo.Name = domain.Status.OSInfo.Name
		vmi.Status.GuestOSInfo.Version = domain.Status.OSInfo.VersionId
		vmi.Status.GuestOSInfo.KernelRelease = domain.Status.OSInfo.KernelRelease
		vmi.Status.GuestOSInfo.PrettyName = domain.Status.OSInfo.PrettyName
		vmi.Status.GuestOSInfo.VersionID = domain.Status.OSInfo.VersionId
		vmi.Status.GuestOSInfo.KernelVersion = domain.Status.OSInfo.KernelVersion
		vmi.Status.GuestOSInfo.ID = domain.Status.OSInfo.Id
	}
}

func (d *VirtualMachineController) updateInterfacesFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	if domain == nil {
		return nil
	}

	// This is needed to be backwards compatible with vmi's which have status interfaces
	// with the name not being set
	if len(domain.Spec.Devices.Interfaces) == 0 && len(vmi.Status.Interfaces) == 1 && vmi.Status.Interfaces[0].Name == "" {
		for _, network := range vmi.Spec.Networks {
			if network.NetworkSource.Pod != nil {
				vmi.Status.Interfaces[0].Name = network.Name
			}
		}
	}

	if len(vmi.Status.Interfaces) == 0 {
		// Set Pod Interface
		interfaces := make([]v1.VirtualMachineInstanceNetworkInterface, 0)
		for _, network := range vmi.Spec.Networks {
			podIface, err := d.getPodInterfacefromFileCache(vmi, network.Name)
			if err != nil {
				return err
			}

			if podIface != nil {
				ifc := v1.VirtualMachineInstanceNetworkInterface{
					Name: network.Name,
					IP:   podIface.PodIP,
					IPs:  podIface.PodIPs,
				}
				interfaces = append(interfaces, ifc)
			}
		}
		vmi.Status.Interfaces = interfaces
	}

	if len(domain.Spec.Devices.Interfaces) > 0 || len(domain.Status.Interfaces) > 0 {
		// This calculates the vmi.Status.Interfaces based on the following data sets:
		// - vmi.Status.Interfaces - previously calculated interfaces, this can contain data (pod IP)
		//   set in the previous loops (when there are no interfaces), which can not be deleted,
		//   unless overridden by Qemu agent
		// - domain.Spec - interfaces form the Spec
		// - domain.Status.Interfaces - interfaces reported by guest agent (empty if Qemu agent not running)
		newInterfaces := []v1.VirtualMachineInstanceNetworkInterface{}

		existingInterfaceStatusByName := map[string]v1.VirtualMachineInstanceNetworkInterface{}
		for _, existingInterfaceStatus := range vmi.Status.Interfaces {
			if existingInterfaceStatus.Name != "" {
				existingInterfaceStatusByName[existingInterfaceStatus.Name] = existingInterfaceStatus
			}
		}

		domainInterfaceStatusByMac := map[string]api.InterfaceStatus{}
		for _, domainInterfaceStatus := range domain.Status.Interfaces {
			domainInterfaceStatusByMac[domainInterfaceStatus.Mac] = domainInterfaceStatus
		}

		existingInterfacesSpecByName := map[string]v1.Interface{}
		for _, existingInterfaceSpec := range vmi.Spec.Domain.Devices.Interfaces {
			existingInterfacesSpecByName[existingInterfaceSpec.Name] = existingInterfaceSpec
		}
		existingNetworksByName := map[string]v1.Network{}
		for _, existingNetwork := range vmi.Spec.Networks {
			existingNetworksByName[existingNetwork.Name] = existingNetwork
		}

		// Iterate through all domain.Spec interfaces
		for _, domainInterface := range domain.Spec.Devices.Interfaces {
			interfaceMAC := domainInterface.MAC.MAC
			var newInterface v1.VirtualMachineInstanceNetworkInterface
			var isForwardingBindingInterface = false

			if existingInterfacesSpecByName[domainInterface.Alias.GetName()].Masquerade != nil || existingInterfacesSpecByName[domainInterface.Alias.GetName()].Slirp != nil {
				isForwardingBindingInterface = true
			}

			if existingInterface, exists := existingInterfaceStatusByName[domainInterface.Alias.GetName()]; exists {
				// Reuse previously calculated interface from vmi.Status.Interfaces, updating the MAC from domain.Spec
				// Only interfaces defined in domain.Spec are handled here
				newInterface = existingInterface
				newInterface.MAC = interfaceMAC

				// If it is a Combination of Masquerade+Pod network, check IP from file cache
				if existingInterfacesSpecByName[domainInterface.Alias.GetName()].Masquerade != nil && existingNetworksByName[domainInterface.Alias.GetName()].NetworkSource.Pod != nil {
					iface, err := d.getPodInterfacefromFileCache(vmi, domainInterface.Alias.GetName())
					if err != nil {
						return err
					}

					if !reflect.DeepEqual(iface.PodIPs, existingInterfaceStatusByName[domainInterface.Alias.GetName()].IPs) {
						newInterface.Name = domainInterface.Alias.GetName()
						newInterface.IP = iface.PodIP
						newInterface.IPs = iface.PodIPs
					}
				}
			} else {
				// If not present in vmi.Status.Interfaces, create a new one based on domain.Spec
				newInterface = v1.VirtualMachineInstanceNetworkInterface{
					MAC:  interfaceMAC,
					Name: domainInterface.Alias.GetName(),
				}
			}

			// Update IP info based on information from domain.Status.Interfaces (Qemu guest)
			// Remove the interface from domainInterfaceStatusByMac to mark it as handled
			if interfaceStatus, exists := domainInterfaceStatusByMac[interfaceMAC]; exists {
				newInterface.InterfaceName = interfaceStatus.InterfaceName
				// Do not update if interface has Masquerede binding
				// virt-controller should update VMI status interface with Pod IP instead
				if !isForwardingBindingInterface {
					newInterface.IP = interfaceStatus.Ip
					newInterface.IPs = interfaceStatus.IPs
				}
				delete(domainInterfaceStatusByMac, interfaceMAC)
			}
			newInterfaces = append(newInterfaces, newInterface)
		}

		// If any of domain.Status.Interfaces were not handled above, it means that the vm contains additional
		// interfaces not defined in domain.Spec.Devices.Interfaces (most likely added by user on VM or a SRIOV interface)
		// Add them to vmi.Status.Interfaces
		setMissingSRIOVInterfacesNames(existingInterfacesSpecByName, domainInterfaceStatusByMac)
		for interfaceMAC, domainInterfaceStatus := range domainInterfaceStatusByMac {
			newInterface := v1.VirtualMachineInstanceNetworkInterface{
				Name:          domainInterfaceStatus.Name,
				MAC:           interfaceMAC,
				IP:            domainInterfaceStatus.Ip,
				IPs:           domainInterfaceStatus.IPs,
				InterfaceName: domainInterfaceStatus.InterfaceName,
			}
			newInterfaces = append(newInterfaces, newInterface)
		}
		vmi.Status.Interfaces = newInterfaces
	}
	return nil
}

func (d *VirtualMachineController) updateAccessCredentialConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {

	if domain == nil || domain.Spec.Metadata.KubeVirt.AccessCredential == nil {
		return
	}

	message := domain.Spec.Metadata.KubeVirt.AccessCredential.Message
	status := k8sv1.ConditionFalse
	if domain.Spec.Metadata.KubeVirt.AccessCredential.Succeeded {
		status = k8sv1.ConditionTrue
	}

	add := false
	condition := condManager.GetCondition(vmi, v1.VirtualMachineInstanceAccessCredentialsSynchronized)
	if condition == nil {
		add = true
	} else if condition.Status != status || condition.Message != message {
		// if not as expected, remove, then add.
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceAccessCredentialsSynchronized)
		add = true
	}
	if add {
		newCondition := v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstanceAccessCredentialsSynchronized,
			LastTransitionTime: metav1.Now(),
			Status:             status,
			Message:            message,
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, newCondition)
		if status == k8sv1.ConditionTrue {
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.AccessCredentialsSyncSuccess.String(), message)
		} else {
			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.AccessCredentialsSyncFailed.String(), message)
		}
	}
}

func (d *VirtualMachineController) updateLiveMigrationConditions(vmi *v1.VirtualMachineInstance, condManager *controller.VirtualMachineInstanceConditionManager) {

	// Cacluate whether the VM is migratable
	liveMigrationCondition, isBlockMigration := d.calculateLiveMigrationCondition(vmi)
	if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceIsMigratable) {
		vmi.Status.Conditions = append(vmi.Status.Conditions, *liveMigrationCondition)
		// Set VMI Migration Method
		if isBlockMigration {
			vmi.Status.MigrationMethod = v1.BlockMigration
		} else {
			vmi.Status.MigrationMethod = v1.LiveMigration
		}
	} else {
		cond := condManager.GetCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
		if !reflect.DeepEqual(cond, liveMigrationCondition) {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
			vmi.Status.Conditions = append(vmi.Status.Conditions, *liveMigrationCondition)
		}
	}
	if vmi.IsEvictable() && liveMigrationCondition.Status == k8sv1.ConditionFalse {
		d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), "EvictionStrategy is set but vmi is not migratable")
	}
}

func (d *VirtualMachineController) updateGuestAgentConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) error {

	// Update the condition when GA is connected
	channelConnected := false
	if domain != nil {
		for _, channel := range domain.Spec.Devices.Channels {
			if channel.Target != nil {
				log.Log.V(4).Infof("Channel: %s, %s", channel.Target.Name, channel.Target.State)
				if channel.Target.Name == "org.qemu.guest_agent.0" {
					if channel.Target.State == "connected" {
						channelConnected = true
					}
				}

			}
		}
	}

	switch {
	case channelConnected && !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected):
		agentCondition := v1.VirtualMachineInstanceCondition{
			Type:          v1.VirtualMachineInstanceAgentConnected,
			LastProbeTime: metav1.Now(),
			Status:        k8sv1.ConditionTrue,
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
	case !channelConnected:
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceAgentConnected)
	}

	if condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
		client, err := d.getLauncherClient(vmi)
		if err != nil {
			return err
		}

		guestInfo, err := client.GetGuestInfo()
		if err != nil {
			return err
		}

		var supported = false

		// For current versions, virt-launcher's supported commands will always contain data.
		// For backwards compatibility: during upgrade from a previous version of KubeVirt,
		// virt-launcher might not provide any supported commands. If the list of supported
		// commands is empty, fall back to previous behavior.
		if len(guestInfo.SupportedCommands) > 0 {
			supported = isGuestAgentSupported(vmi, guestInfo.SupportedCommands)
		} else {
			for _, version := range d.clusterConfig.GetSupportedAgentVersions() {
				supported = supported || regexp.MustCompile(version).MatchString(guestInfo.GAVersion)
			}
		}

		if !supported {
			if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent) {
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceUnsupportedAgent,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
			}
		} else {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent)
		}

	}
	return nil
}

func (d *VirtualMachineController) updatePausedConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {

	// Update paused condition in case VMI was paused / unpaused
	if domain != nil && domain.Status.Status == api.Paused {
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			calculatePausedCondition(vmi, domain.Status.Reason)
		}
	} else if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		log.Log.Object(vmi).V(3).Info("Removing paused condition")
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstancePaused)
	}
}

func (d *VirtualMachineController) updateFSFreezeStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil || domain.Status.FSFreezeStatus.Status == "" {
		return
	}

	if domain.Status.FSFreezeStatus.Status == api.FSThawed {
		vmi.Status.FSFreezeStatus = ""
	} else {
		vmi.Status.FSFreezeStatus = domain.Status.FSFreezeStatus.Status
	}

}

func IsoGuestVolumePath(vmi *v1.VirtualMachineInstance, volume *v1.Volume) (string, bool) {
	var volPath string

	basepath := "/var/run"
	if volume.CloudInitNoCloud != nil {
		volPath = filepath.Join(basepath, "kubevirt-ephemeral-disks", "cloud-init-data", vmi.Namespace, vmi.Name, "noCloud.iso")
	} else if volume.CloudInitConfigDrive != nil {
		volPath = filepath.Join(basepath, "kubevirt-ephemeral-disks", "cloud-init-data", vmi.Namespace, vmi.Name, "configdrive.iso")
	} else if volume.ConfigMap != nil {
		volPath = filepath.Join(basepath, "kubevirt-private", path.Base(config.ConfigMapDisksDir), volume.Name+".iso")
	} else if volume.DownwardAPI != nil {
		volPath = filepath.Join(basepath, "kubevirt-private", path.Base(config.DownwardAPIDisksDir), volume.Name+".iso")
	} else if volume.Secret != nil {
		volPath = filepath.Join(basepath, "kubevirt-private", path.Base(config.SecretDisksDir), volume.Name+".iso")
	} else if volume.ServiceAccount != nil {
		volPath = filepath.Join(basepath, "kubevirt-private", path.Base(config.ServiceAccountDiskDir), config.ServiceAccountDiskName)
	} else if volume.Sysprep != nil {
		volPath = filepath.Join(basepath, "kubevirt-private", path.Base(config.SysprepDisksDir), volume.Name+".iso")
	} else {
		return "", false
	}

	return volPath, true
}

func (d *VirtualMachineController) updateIsoSizeStatus(vmi *v1.VirtualMachineInstance) {
	var podUID string
	if vmi.Status.Phase != v1.Running {
		return
	}

	for k, v := range vmi.Status.ActivePods {
		if v == vmi.Status.NodeName {
			podUID = string(k)
			break
		}
	}
	if podUID == "" {
		log.DefaultLogger().V(2).Warningf("failed to find pod UID for VMI %s", vmi.Name)
		return
	}

	for _, volume := range vmi.Spec.Volumes {
		volPath, found := IsoGuestVolumePath(vmi, &volume)
		if !found {
			continue
		}
		res, err := d.podIsolationDetector.Detect(vmi)
		if err != nil {
			log.DefaultLogger().V(2).Warningf("failed to detect VMI %s", vmi.Name)
			continue
		}
		size, err := isolation.GetFileSize(volPath, res)
		if err != nil {
			log.DefaultLogger().V(2).Warningf("failed to determine file size for volume %s", volPath)
			continue
		}

		for i, _ := range vmi.Status.VolumeStatus {
			if vmi.Status.VolumeStatus[i].Name == volume.Name {
				vmi.Status.VolumeStatus[i].Size = size
				continue
			}
		}
	}
}

func (d *VirtualMachineController) updateVMIStatus(origVMI *v1.VirtualMachineInstance, domain *api.Domain, syncError error) (err error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()

	// Don't update the VirtualMachineInstance if it is already in a final state
	if origVMI.IsFinal() {
		return nil
	} else if origVMI.Status.NodeName != "" && origVMI.Status.NodeName != d.host {
		// Only update the VMI's phase if this node owns the VMI.
		// not owned by this host, likely the result of a migration
		return nil
	} else if domainMigrated(domain) {
		return d.migrationSourceUpdateVMIStatus(origVMI, domain)
	}

	vmi := origVMI.DeepCopy()
	oldStatus := *vmi.Status.DeepCopy()

	// Update VMI status fields based on what is reported on the domain
	d.updateIsoSizeStatus(vmi)
	d.setMigrationProgressStatus(vmi, domain)
	d.updateGuestInfoFromDomain(vmi, domain)
	d.updateVolumeStatusesFromDomain(vmi, domain)
	d.updateFSFreezeStatus(vmi, domain)
	err = d.updateInterfacesFromDomain(vmi, domain)
	if err != nil {
		return err
	}

	// Calculate the new VirtualMachineInstance state based on what libvirt reported
	err = d.setVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}

	// Update conditions on VMI Status
	d.updateAccessCredentialConditions(vmi, domain, condManager)
	d.updateLiveMigrationConditions(vmi, condManager)
	err = d.updateGuestAgentConditions(vmi, domain, condManager)
	if err != nil {
		return err
	}
	d.updatePausedConditions(vmi, domain, condManager)

	// Handle sync error
	var criticalNetErr *neterrors.CriticalNetworkError
	if goerror.As(syncError, &criticalNetErr) {
		log.Log.Errorf("virt-launcher crashed due to a network error. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	if _, ok := syncError.(*virtLauncherCriticalSecurebootError); ok {
		log.Log.Errorf("virt-launcher does not support the Secure Boot setting. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	condManager.CheckFailure(vmi, syncError, "Synchronizing with the Domain failed.")

	controller.SetVMIPhaseTransitionTimestamp(origVMI, vmi)

	// Only issue vmi update if status has changed
	if !reflect.DeepEqual(oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		d.vmiExpectations.SetExpectations(key, 1, 0)
		_, err = d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
		if err != nil {
			d.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	// Record an event on the VMI when the VMI's phase changes
	if oldStatus.Phase != vmi.Status.Phase {
		switch vmi.Status.Phase {
		case v1.Running:
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Started.String(), VMIStarted)
		case v1.Succeeded:
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Stopped.String(), VMIShutdown)
		case v1.Failed:
			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Stopped.String(), VMICrashed)
		}
	}

	return nil
}

func _guestAgentCommandSubsetSupported(requiredCommands []string, commands []v1.GuestAgentCommandInfo) bool {
	var found bool
	for _, cmd := range requiredCommands {
		found = false
		for _, foundCmd := range commands {
			if cmd == foundCmd.Name {
				if foundCmd.Enabled {
					found = true
				}
				break
			}
		}
		if found == false {
			return false
		}
	}
	return true

}

func isGuestAgentSupported(vmi *v1.VirtualMachineInstance, commands []v1.GuestAgentCommandInfo) bool {
	log.Log.V(3).Object(vmi).Infof("checking guest agent: %v", commands)
	if !_guestAgentCommandSubsetSupported(RequiredGuestAgentCommands, commands) {
		log.Log.V(3).Object(vmi).Info("This guest agent doesn't support required basic commands")
		return false
	}

	checkSSH := false
	checkPasswd := false

	if vmi != nil && vmi.Spec.AccessCredentials != nil {
		for _, accessCredential := range vmi.Spec.AccessCredentials {
			if accessCredential.SSHPublicKey != nil && accessCredential.SSHPublicKey.PropagationMethod.QemuGuestAgent != nil {
				// defer checking the command list so we only do that once
				checkSSH = true
			}
			if accessCredential.UserPassword != nil && accessCredential.UserPassword.PropagationMethod.QemuGuestAgent != nil {
				// defer checking the command list so we only do that once
				checkPasswd = true
			}

		}
	}

	if checkSSH && !_guestAgentCommandSubsetSupported(SSHRelatedGuestAgentCommands, commands) {
		log.Log.V(3).Object(vmi).Info("This guest agent doesn't support required public key commands")
		return false
	}

	if checkPasswd && !_guestAgentCommandSubsetSupported(PasswordRelatedGuestAgentCommands, commands) {
		log.Log.V(3).Object(vmi).Info("This guest agent doesn't support required password commands")
		return false
	}

	log.Log.V(3).Object(vmi).Info("This guest agent is supported")
	return true
}

func calculatePausedCondition(vmi *v1.VirtualMachineInstance, reason api.StateChangeReason) {
	switch reason {
	case api.ReasonPausedUser:
		log.Log.Object(vmi).V(3).Info("Adding paused condition")
		now := metav1.NewTime(time.Now())
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedByUser",
			Message:            "VMI was paused by user",
		})
	case api.ReasonPausedIOError:
		log.Log.Object(vmi).V(3).Info("Adding paused condition")
		now := metav1.NewTime(time.Now())
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedIOError",
			Message:            "VMI was paused, IO error",
		})
	default:
		log.Log.Object(vmi).V(3).Infof("Domain is paused for unknown reason, %s", reason)
	}
}

func newNonMigratableCondition(msg string, reason string) *v1.VirtualMachineInstanceCondition {
	return &v1.VirtualMachineInstanceCondition{
		Type:    v1.VirtualMachineInstanceIsMigratable,
		Status:  k8sv1.ConditionFalse,
		Message: msg,
		Reason:  reason,
	}
}

func (d *VirtualMachineController) calculateLiveMigrationCondition(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstanceCondition, bool) {
	isBlockMigration, err := d.checkVolumesForMigration(vmi)
	if err != nil {
		return newNonMigratableCondition(err.Error(), v1.VirtualMachineInstanceReasonDisksNotMigratable), isBlockMigration
	}

	err = d.checkNetworkInterfacesForMigration(vmi)
	if err != nil {
		return newNonMigratableCondition(err.Error(), v1.VirtualMachineInstanceReasonInterfaceNotMigratable), isBlockMigration
	}

	if err := d.isHostModelMigratable(vmi); err != nil {
		return newNonMigratableCondition(err.Error(), v1.VirtualMachineInstanceReasonCPUModeNotMigratable), isBlockMigration
	}

	if util.IsVMIVirtiofsEnabled(vmi) {
		return newNonMigratableCondition("VMI uses virtiofs", v1.VirtualMachineInstanceReasonVirtIOFSNotMigratable), isBlockMigration
	}

	return &v1.VirtualMachineInstanceCondition{
		Type:   v1.VirtualMachineInstanceIsMigratable,
		Status: k8sv1.ConditionTrue,
	}, isBlockMigration
}

func (c *VirtualMachineController) Run(threadiness int, stopCh chan struct{}) {
	defer c.Queue.ShutDown()
	log.Log.Info("Starting virt-handler controller.")

	go c.deviceManagerController.Run(stopCh)

	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced, c.vmiSourceInformer.HasSynced, c.vmiTargetInformer.HasSynced, c.gracefulShutdownInformer.HasSynced)

	// Queue keys for previous Domains on the host that no longer exist
	// in the cache. This ensures we perform local cleanup of deleted VMs.
	for _, domain := range c.domainInformer.GetStore().List() {
		d := domain.(*api.Domain)
		vmiRef := v1.NewVMIReferenceWithUUID(
			d.ObjectMeta.Namespace,
			d.ObjectMeta.Name,
			d.Spec.Metadata.KubeVirt.UID)

		key := controller.VirtualMachineInstanceKey(vmiRef)

		_, exists, _ := c.vmiSourceInformer.GetStore().GetByKey(key)
		if !exists {
			c.Queue.Add(key)
		}
	}

	go c.heartBeat.Run(c.heartBeatInterval, stopCh)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping virt-handler controller.")
}

func (c *VirtualMachineController) runWorker() {
	for c.Execute() {
	}
}

func (c *VirtualMachineController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (d *VirtualMachineController) getVMIFromCache(key string) (vmi *v1.VirtualMachineInstance, exists bool, err error) {

	// Fetch the latest Vm state from cache
	obj, exists, err := d.vmiSourceInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		obj, exists, err = d.vmiTargetInformer.GetStore().GetByKey(key)
		if err != nil {
			return nil, false, err
		}
	}

	// Retrieve the VirtualMachineInstance
	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			// TODO log and don't retry
			return nil, false, err
		}
		vmi = v1.NewVMIReferenceFromNameWithNS(namespace, name)
	} else {
		vmi = obj.(*v1.VirtualMachineInstance)
	}
	return vmi, exists, nil
}

func (d *VirtualMachineController) getDomainFromCache(key string) (domain *api.Domain, exists bool, cachedUID types.UID, err error) {

	obj, exists, err := d.domainInformer.GetStore().GetByKey(key)

	if err != nil {
		return nil, false, "", err
	}

	if exists {
		domain = obj.(*api.Domain)
		cachedUID = domain.Spec.Metadata.KubeVirt.UID

		// We're using the DeletionTimestamp to signify that the
		// Domain is deleted rather than sending the DELETE watch event.
		if domain.ObjectMeta.DeletionTimestamp != nil {
			exists = false
			domain = nil
		}
	}
	return domain, exists, cachedUID, nil
}

func (d *VirtualMachineController) migrationOrphanedSourceNodeExecute(vmi *v1.VirtualMachineInstance, domainExists bool) error {

	if domainExists {
		err := d.processVmDelete(vmi)
		if err != nil {
			return err
		}
		// we can perform the cleanup immediately after
		// the successful delete here because we don't have
		// to report the deletion results on the VMI status
		// in this case.
		err = d.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	} else {
		err := d.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *VirtualMachineController) migrationTargetExecute(vmi *v1.VirtualMachineInstance, vmiExists bool, domainExists bool) error {

	// set to true when preparation of migration target should be aborted.
	shouldAbort := false
	// set to true when VirtualMachineInstance migration target needs to be prepared
	shouldUpdate := false
	// set true when the current migration target has exitted and needs to be cleaned up.
	shouldCleanUp := false

	if vmiExists && vmi.IsRunning() {
		shouldUpdate = true
	}

	if !vmiExists || vmi.DeletionTimestamp != nil {
		shouldAbort = true
	} else if vmi.IsFinal() {
		shouldAbort = true
	} else if d.hasStaleClientConnections(vmi) {
		// if stale client exists, force cleanup.
		// This can happen as a result of a previously
		// failed attempt to migrate the vmi to this node.
		shouldCleanUp = true
	}

	if shouldAbort {
		if domainExists {
			err := d.processVmDelete(vmi)
			if err != nil {
				return err
			}
		}

		err := d.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	} else if shouldCleanUp {
		log.Log.Object(vmi).Infof("Stale client for migration target found. Cleaning up.")

		err := d.processVmCleanup(vmi)
		if err != nil {
			return err
		}

		// if we're still the migration target, we need to keep trying until the migration fails.
		// it's possible we're simply waiting for another target pod to come online.
		d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)

	} else if shouldUpdate {
		log.Log.Object(vmi).Info("Processing vmi migration target update")

		// prepare the POD for the migration
		err := d.processVmUpdate(vmi, domainExists)
		if err != nil {
			return err
		}

		err = d.migrationTargetUpdateVMIStatus(vmi, domainExists)
		if err != nil {
			return err
		}
	}

	return nil
}

// Legacy, remove once we're certain we are no longer supporting
// VMIs running with the old graceful shutdown trigger logic
func gracefulShutdownTriggerFromNamespaceName(baseDir string, namespace string, name string) string {
	triggerFile := namespace + "_" + name
	return filepath.Join(baseDir, "graceful-shutdown-trigger", triggerFile)
}

// Legacy, remove once we're certain we are no longer supporting
// VMIs running with the old graceful shutdown trigger logic
func vmGracefulShutdownTriggerClear(baseDir string, vmi *v1.VirtualMachineInstance) error {
	triggerFile := gracefulShutdownTriggerFromNamespaceName(baseDir, vmi.Namespace, vmi.Name)
	return diskutils.RemoveFilesIfExist(triggerFile)
}

// Legacy, remove once we're certain we are no longer supporting
// VMIs running with the old graceful shutdown trigger logic
func vmHasGracefulShutdownTrigger(baseDir string, vmi *v1.VirtualMachineInstance) (bool, error) {
	triggerFile := gracefulShutdownTriggerFromNamespaceName(baseDir, vmi.Namespace, vmi.Name)
	return diskutils.FileExists(triggerFile)
}

// Determine if gracefulShutdown has been triggered by virt-launcher
func (d *VirtualMachineController) hasGracefulShutdownTrigger(vmi *v1.VirtualMachineInstance, domain *api.Domain) (bool, error) {

	// This is the new way of reporting GracefulShutdown, via domain metadata.
	if domain != nil &&
		domain.Spec.Metadata.KubeVirt.GracePeriod != nil &&
		domain.Spec.Metadata.KubeVirt.GracePeriod.MarkedForGracefulShutdown != nil &&
		*domain.Spec.Metadata.KubeVirt.GracePeriod.MarkedForGracefulShutdown == true {
		return true, nil
	}

	// Fallback to detecting the old way of reporting gracefulshutdown, via file.
	// We keep this around in order to ensure backwards compatibility
	return vmHasGracefulShutdownTrigger(d.virtShareDir, vmi)
}

func (d *VirtualMachineController) defaultExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	// set to true when domain needs to be shutdown.
	shouldShutdown := false
	// set to true when domain needs to be removed from libvirt.
	shouldDelete := false
	// optimization. set to true when processing already deleted domain.
	shouldCleanUp := false
	// set to true when VirtualMachineInstance is active or about to become active.
	shouldUpdate := false
	// set true to ensure that no updates to the current VirtualMachineInstance state will occur
	forceIgnoreSync := false

	log.Log.V(3).Infof("Processing event %v", key)

	if vmiExists && domainExists {
		log.Log.Object(vmi).Infof("VMI is in phase: %v | Domain status: %v, reason: %v", vmi.Status.Phase, domain.Status.Status, domain.Status.Reason)
	} else if vmiExists {
		log.Log.Object(vmi).Infof("VMI is in phase: %v | Domain does not exist", vmi.Status.Phase)
	} else if domainExists {
		vmiRef := v1.NewVMIReferenceWithUUID(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name, domain.Spec.Metadata.KubeVirt.UID)
		log.Log.Object(vmiRef).Infof("VMI does not exist | Domain status: %v, reason: %v", domain.Status.Status, domain.Status.Reason)
	} else {
		log.Log.Info("VMI does not exist | Domain does not exist")
	}

	domainAlive := domainExists &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	domainMigrated := domainExists && domainMigrated(domain)

	gracefulShutdown, err := d.hasGracefulShutdownTrigger(vmi, domain)
	if err != nil {
		return err
	} else if gracefulShutdown && vmi.IsRunning() {
		if domainAlive {
			log.Log.Object(vmi).V(3).Info("Shutting down due to graceful shutdown signal.")
			shouldShutdown = true
		} else {
			shouldDelete = true
		}
	}

	// Determine removal of VirtualMachineInstance from cache should result in deletion.
	if !vmiExists {
		switch {
		case domainAlive:
			// The VirtualMachineInstance is deleted on the cluster, and domain is alive,
			// then shut down the domain.
			log.Log.Object(vmi).V(3).Info("Shutting down domain for deleted VirtualMachineInstance object.")
			shouldShutdown = true
		case domainExists:
			// The VirtualMachineInstance is deleted on the cluster, and domain is not alive
			// then delete the domain.
			log.Log.Object(vmi).V(3).Info("Deleting domain for deleted VirtualMachineInstance object.")
			shouldDelete = true
		default:
			// If neither the domain nor the vmi object exist locally,
			// then ensure any remaining local ephemeral data is cleaned up.
			shouldCleanUp = true
		}
	}

	// Determine if VirtualMachineInstance is being deleted.
	if vmiExists && vmi.ObjectMeta.DeletionTimestamp != nil {
		switch {
		case domainAlive:
			log.Log.Object(vmi).V(3).Info("Shutting down domain for VirtualMachineInstance with deletion timestamp.")
			shouldShutdown = true
		case domainExists:
			log.Log.Object(vmi).V(3).Info("Deleting domain for VirtualMachineInstance with deletion timestamp.")
			shouldDelete = true
		default:
			if vmi.IsFinal() {
				shouldCleanUp = true
			}
		}
	}

	// Determine if domain needs to be deleted as a result of VirtualMachineInstance
	// shutting down naturally (guest internal invoked shutdown)
	if domainExists && vmiExists && vmi.IsFinal() {
		log.Log.Object(vmi).V(3).Info("Removing domain and ephemeral data for finalized vmi.")
		shouldDelete = true
	} else if !domainExists && vmiExists && vmi.IsFinal() {
		log.Log.Object(vmi).V(3).Info("Cleaning up local data for finalized vmi.")
		shouldCleanUp = true
	}

	// Determine if an active (or about to be active) VirtualMachineInstance should be updated.
	if vmiExists && !vmi.IsFinal() {
		// requiring the phase of the domain and VirtualMachineInstance to be in sync is an
		// optimization that prevents unnecessary re-processing VMIs during the start flow.
		phase, err := d.calculateVmPhaseForStatusReason(domain, vmi)
		if err != nil {
			return err
		}
		if vmi.Status.Phase == phase {
			shouldUpdate = true
		}
	}

	// NOTE: This must be the last check that occurs before checking the sync booleans!
	//
	// Special logic for domains migrated from a source node.
	// Don't delete/destroy domain until the handoff occurs.
	if domainMigrated {
		// only allow the sync to occur on the domain once we've done
		// the node handoff. Otherwise we potentially lose the fact that
		// the domain migrated because we'll attempt to delete the locally
		// shut off domain during the sync.
		if vmiExists &&
			!vmi.IsFinal() &&
			vmi.DeletionTimestamp == nil &&
			vmi.Status.NodeName != "" &&
			vmi.Status.NodeName == d.host {

			// If the domain migrated but the VMI still thinks this node
			// is the host, force ignore the sync until the VMI's status
			// is updated to reflect the node the domain migrated to.
			forceIgnoreSync = true
		}
	}

	var syncErr error

	// Process the VirtualMachineInstance update in this order.
	// * Shutdown and Deletion due to VirtualMachineInstance deletion, process stopping, graceful shutdown trigger, etc...
	// * Cleanup of already shutdown and Deleted VMIs
	// * Update due to spec change and initial start flow.
	switch {
	case forceIgnoreSync:
		log.Log.Object(vmi).V(3).Info("No update processing required: forced ignore")
	case shouldShutdown:
		log.Log.Object(vmi).V(3).Info("Processing shutdown.")
		syncErr = d.processVmShutdown(vmi, domain)
	case shouldDelete:
		log.Log.Object(vmi).V(3).Info("Processing deletion.")
		syncErr = d.processVmDelete(vmi)
	case shouldCleanUp:
		log.Log.Object(vmi).V(3).Info("Processing local ephemeral data cleanup for shutdown domain.")
		syncErr = d.processVmCleanup(vmi)
	case shouldUpdate:
		log.Log.Object(vmi).V(3).Info("Processing vmi update")
		syncErr = d.processVmUpdate(vmi, domainExists)
	default:
		log.Log.Object(vmi).V(3).Info("No update processing required")
	}

	if syncErr != nil && !vmi.IsFinal() {
		d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())

		// `syncErr` will be propagated anyway, and it will be logged in `re-enqueueing`
		// so there is no need to log it twice in hot path without increased verbosity.
		log.Log.Object(vmi).V(3).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	// Update the VirtualMachineInstance status, if the VirtualMachineInstance exists
	if vmiExists {
		err = d.updateVMIStatus(vmi, domain, syncErr)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Updating the VirtualMachineInstance status failed.")
			return err
		}
	}

	if syncErr != nil {
		return syncErr
	}

	log.Log.Object(vmi).V(3).Info("Synchronization loop succeeded.")
	return nil

}

func (d *VirtualMachineController) execute(key string) error {
	vmi, vmiExists, err := d.getVMIFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists {
		d.vmiExpectations.DeleteExpectations(key)
	} else if !d.vmiExpectations.SatisfiedExpectations(key) {
		return nil
	}

	domain, domainExists, domainCachedUID, err := d.getDomainFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists && string(domainCachedUID) != "" {
		// it's possible to discover the UID from cache even if the domain
		// doesn't technically exist anymore
		vmi.UID = domainCachedUID
		log.Log.Object(vmi).Infof("Using cached UID for vmi found in domain cache")
	}

	// As a last effort, if the UID still can't be determined attempt
	// to retrieve it from the ghost record
	if string(vmi.UID) == "" {
		uid := virtcache.LastKnownUIDFromGhostRecordCache(key)
		if uid != "" {
			log.Log.Object(vmi).V(3).Infof("ghost record cache provided %s as UID", uid)
			vmi.UID = uid
		} else {
			// legacy support, attempt to find UID from watchdog file it exists.
			uid := watchdog.WatchdogFileGetUID(d.virtShareDir, vmi)
			if uid != "" {
				log.Log.Object(vmi).V(3).Infof("watchdog file provided %s as UID", uid)
				vmi.UID = types.UID(uid)
			}
		}
	}

	if vmiExists && domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		oldVMI := v1.NewVMIReferenceFromNameWithNS(vmi.Namespace, vmi.Name)
		oldVMI.UID = domain.Spec.Metadata.KubeVirt.UID
		expired, initialized, err := d.isLauncherClientUnresponsive(oldVMI)
		if err != nil {
			return err
		}
		// If we found an outdated domain which is also not alive anymore, clean up
		if !initialized {
			d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
			return nil
		} else if expired {
			log.Log.Object(oldVMI).Infof("Detected stale vmi %s that still needs cleanup before new vmi %s with identical name/namespace can be processed", oldVMI.UID, vmi.UID)
			err = d.processVmCleanup(oldVMI)
			if err != nil {
				return err
			}
			// Make sure we re-enqueue the key to ensure this new VMI is processed
			// after the stale domain is removed
			d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*5)
		}

		return nil
	}

	// Take different execution paths depending on the state of the migration and the
	// node this is executed on.

	if vmiExists && d.isPreMigrationTarget(vmi) {
		// 1. PRE-MIGRATION TARGET PREPARATION PATH
		//
		// If this node is the target of the vmi's migration, take
		// a different execute path. The target execute path prepares
		// the local environment for the migration, but does not
		// start the VMI
		return d.migrationTargetExecute(vmi, vmiExists, domainExists)
	} else if vmiExists && d.isOrphanedMigrationSource(vmi) {
		// 3. POST-MIGRATION SOURCE CLEANUP
		//
		// After a migration, the migrated domain still exists in the old
		// source's domain cache. Ensure that any node that isn't currently
		// the target or owner of the VMI handles deleting the domain locally.
		return d.migrationOrphanedSourceNodeExecute(vmi, domainExists)
	}
	return d.defaultExecute(key,
		vmi,
		vmiExists,
		domain,
		domainExists)

}

func (d *VirtualMachineController) processVmCleanup(vmi *v1.VirtualMachineInstance) error {

	vmiId := string(vmi.UID)

	log.Log.Object(vmi).Infof("Performing final local cleanup for vmi with uid %s", vmiId)
	// If the VMI is using the old graceful shutdown trigger on
	// a hostmount, make sure to clear that file still.
	err := vmGracefulShutdownTriggerClear(d.virtShareDir, vmi)
	if err != nil {
		return err
	}

	d.migrationProxy.StopTargetListener(vmiId)
	d.migrationProxy.StopSourceListener(vmiId)

	// Unmount container disks and clean up remaining files
	if err := d.containerDiskMounter.UnmountKernelArtifacts(vmi); err != nil {
		return err
	}
	if err := d.containerDiskMounter.Unmount(vmi); err != nil {
		return err
	}
	if err := d.hotplugVolumeMounter.UnmountAll(vmi); err != nil {
		return err
	}

	d.clearPodNetworkPhase1(vmi)

	// Watch dog file and command client must be the last things removed here
	err = d.closeLauncherClient(vmi)
	if err != nil {
		return err
	}

	// Remove the domain from cache in the event that we're performing
	// a final cleanup and never received the "DELETE" event. This is
	// possible if the VMI pod goes away before we receive the final domain
	// "DELETE"
	domain := api.NewDomainReferenceFromName(vmi.Namespace, vmi.Name)
	log.Log.Object(domain).Infof("Removing domain from cache during final cleanup")
	return d.domainInformer.GetStore().Delete(domain)
}

func (d *VirtualMachineController) closeLauncherClient(vmi *v1.VirtualMachineInstance) error {

	// UID is required in order to close socket
	if string(vmi.GetUID()) == "" {
		return nil
	}

	clientInfo, exists := d.launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		clientInfo.Client.Close()
		close(clientInfo.DomainPipeStopChan)

		// With legacy sockets on hostpaths, we have to cleanup the sockets ourselves.
		if cmdclient.IsLegacySocket(clientInfo.SocketFile) {
			err := os.RemoveAll(clientInfo.SocketFile)
			if err != nil {
				return err
			}
		}
	}

	// for legacy support, ensure watchdog is removed when client is removed
	// in the event that watchdog VMIs are still in use
	err := watchdog.WatchdogFileRemove(d.virtShareDir, vmi)
	if err != nil {
		return err
	}

	virtcache.DeleteGhostRecord(vmi.Namespace, vmi.Name)
	d.launcherClients.Delete(vmi.UID)
	return nil
}

// used by unit tests to add mock clients
func (d *VirtualMachineController) addLauncherClient(vmUID types.UID, info *virtcache.LauncherClientInfo) error {
	d.launcherClients.Store(vmUID, info)
	return nil
}

func (d *VirtualMachineController) isLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error) {
	var socketFile string

	clientInfo, exists := d.launcherClients.Load(vmi.UID)
	if exists {
		if clientInfo.Ready == true {
			// use cached socket if we previously established a connection
			socketFile = clientInfo.SocketFile
		} else {
			socketFile, err = cmdclient.FindSocketOnHost(vmi)
			if err != nil {
				// socket does not exist, but let's see if the pod is still there
				if _, err = cmdclient.FindPodDirOnHost(vmi); err != nil {
					// no pod meanst that waiting for it to initialize makes no sense
					return true, true, nil
				}
				// pod is still there, if there is no socket let's wait for it to become ready
				if clientInfo.NotInitializedSince.Before(time.Now().Add(-3 * time.Minute)) {
					return true, true, nil
				}
				return false, false, nil
			}
			clientInfo.Ready = true
			clientInfo.SocketFile = socketFile
		}
	} else {
		clientInfo := &virtcache.LauncherClientInfo{
			NotInitializedSince: time.Now(),
			Ready:               false,
		}
		d.launcherClients.Store(vmi.UID, clientInfo)
		// attempt to find the socket if the established connection doesn't currently exist.
		socketFile, err = cmdclient.FindSocketOnHost(vmi)
		// no socket file, no VMI, so it's unresponsive
		if err != nil {
			// socket does not exist, but let's see if the pod is still there
			if _, err = cmdclient.FindPodDirOnHost(vmi); err != nil {
				// no pod meanst that waiting for it to initialize makes no sense
				return true, true, nil
			}
			return false, false, nil
		}
		clientInfo.Ready = true
		clientInfo.SocketFile = socketFile
	}
	// The new way of detecting unresponsive VMIs monitors the
	// cmd socket. This requires an updated VMI image. Old VMIs
	// still use the watchdog method.
	watchDogExists, _ := watchdog.WatchdogFileExists(d.virtShareDir, vmi)
	if cmdclient.SocketMonitoringEnabled(socketFile) && !watchDogExists {
		isUnresponsive := cmdclient.IsSocketUnresponsive(socketFile)
		return isUnresponsive, true, nil
	}

	// fall back to legacy watchdog support for backwards compatibility
	isUnresponsive, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, vmi)
	return isUnresponsive, true, err
}

func (d *VirtualMachineController) getLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	var err error

	clientInfo, exists := d.launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		return clientInfo.Client, nil
	}

	socketFile, err := cmdclient.FindSocketOnHost(vmi)
	if err != nil {
		return nil, err
	}

	err = virtcache.AddGhostRecord(vmi.Namespace, vmi.Name, socketFile, vmi.UID)
	if err != nil {
		return nil, err
	}

	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, err
	}

	domainPipeStopChan := make(chan struct{})
	// if this isn't a legacy socket, we need to
	// pipe in the domain socket into the VMI's filesystem
	if !cmdclient.IsLegacySocket(socketFile) {
		err = d.startDomainNotifyPipe(domainPipeStopChan, vmi)
		if err != nil {
			client.Close()
			close(domainPipeStopChan)
			return nil, err
		}
	}

	d.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
		Client:              client,
		SocketFile:          socketFile,
		DomainPipeStopChan:  domainPipeStopChan,
		NotInitializedSince: time.Now(),
		Ready:               true,
	})

	return client, nil
}

func (d *VirtualMachineController) processVmShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := d.getVerifiedLauncherClient(vmi)
	if err != nil {
		return err
	}

	// Only attempt to gracefully shutdown if the domain has the ACPI feature enabled
	if isACPIEnabled(vmi, domain) {
		if expired, timeLeft := d.hasGracePeriodExpired(domain); !expired {
			return d.handleVMIShutdown(vmi, domain, client, timeLeft)
		}
		log.Log.Object(vmi).Infof("Grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
	} else {
		log.Log.Object(vmi).Infof("ACPI feature not available, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
	}

	err = client.KillVirtualMachine(vmi)
	if err != nil && !cmdclient.IsDisconnected(err) {
		// Only report err if it wasn't the result of a disconnect.
		//
		// Both virt-launcher and virt-handler are trying to destroy
		// the VirtualMachineInstance at the same time. It's possible the client may get
		// disconnected during the kill request, which shouldn't be
		// considered an error.
		return err
	}

	d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), VMIStopping)

	return nil
}

func (d *VirtualMachineController) handleVMIShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain, client cmdclient.LauncherClient, timeLeft int64) error {
	if domain.Status.Status != api.Shutdown {
		return d.shutdownVMI(vmi, client, timeLeft)
	}
	log.Log.V(4).Object(vmi).Infof("%s is already shutting down.", vmi.GetObjectMeta().GetName())
	return nil
}

func (d *VirtualMachineController) shutdownVMI(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient, timeLeft int64) error {
	err := client.ShutdownVirtualMachine(vmi)
	if err != nil && !cmdclient.IsDisconnected(err) {
		// Only report err if it wasn't the result of a disconnect.
		//
		// Both virt-launcher and virt-handler are trying to destroy
		// the VirtualMachineInstance at the same time. It's possible the client may get
		// disconnected during the kill request, which shouldn't be
		// considered an error.
		return err
	}

	log.Log.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())

	// Make sure that we don't hot-loop in case we send the first domain notification
	if timeLeft == -1 {
		timeLeft = 5
		if vmi.Spec.TerminationGracePeriodSeconds != nil && *vmi.Spec.TerminationGracePeriodSeconds < timeLeft {
			timeLeft = *vmi.Spec.TerminationGracePeriodSeconds
		}
	}
	// In case we have a long grace period, we want to resend the graceful shutdown every 5 seconds
	// That's important since a booting OS can miss ACPI signals
	if timeLeft > 5 {
		timeLeft = 5
	}

	// pending graceful shutdown.
	d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Duration(timeLeft)*time.Second)
	d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.ShuttingDown.String(), VMIGracefulShutdown)
	return nil
}

func (d *VirtualMachineController) processVmDelete(vmi *v1.VirtualMachineInstance) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := d.getVerifiedLauncherClient(vmi)

	// If the pod has been torn down, we know the VirtualMachineInstance is down.
	if err == nil {

		log.Log.Object(vmi).Infof("Signaled deletion for %s", vmi.GetObjectMeta().GetName())

		// pending deletion.
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), VMISignalDeletion)

		err = client.DeleteDomain(vmi)
		if err != nil && !cmdclient.IsDisconnected(err) {
			// Only report err if it wasn't the result of a disconnect.
			//
			// Both virt-launcher and virt-handler are trying to destroy
			// the VirtualMachineInstance at the same time. It's possible the client may get
			// disconnected during the kill request, which shouldn't be
			// considered an error.
			return err
		}
	}

	return nil

}

func (d *VirtualMachineController) hasStaleClientConnections(vmi *v1.VirtualMachineInstance) bool {
	_, err := d.getVerifiedLauncherClient(vmi)
	if err == nil {
		// current client connection is good.
		return false
	}

	// no connection, but ghost file exists.
	if virtcache.HasGhostRecord(vmi.Namespace, vmi.Name) {
		return true
	}

	return false

}

func (d *VirtualMachineController) getVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error) {
	client, err = d.getLauncherClient(vmi)
	if err != nil {
		return
	}

	// Verify connectivity.
	// It's possible the pod has already been torn down along with the VirtualMachineInstance.
	err = client.Ping()
	return
}

func (d *VirtualMachineController) isOrphanedMigrationSource(vmi *v1.VirtualMachineInstance) bool {
	nodeName, ok := vmi.Labels[v1.NodeNameLabel]

	if ok && nodeName != "" && nodeName != d.host {
		return true
	}

	return false
}

func (d *VirtualMachineController) isPreMigrationTarget(vmi *v1.VirtualMachineInstance) bool {

	migrationTargetNodeName, ok := vmi.Labels[v1.MigrationTargetNodeNameLabel]

	if ok &&
		migrationTargetNodeName != "" &&
		migrationTargetNodeName != vmi.Status.NodeName &&
		migrationTargetNodeName == d.host {
		return true
	}

	return false
}

func (d *VirtualMachineController) checkNetworkInterfacesForMigration(vmi *v1.VirtualMachineInstance) error {
	err := validatePodNetworkInterfaceUsesMasqueradeBinding(vmi)
	if err != nil {
		return err
	}

	err = d.validateSRIOVInterfacesForMigration(vmi)
	if err != nil {
		return err
	}

	return nil
}

func validatePodNetworkInterfaceUsesMasqueradeBinding(vmi *v1.VirtualMachineInstance) error {
	interfacesByName := map[string]v1.Interface{}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		interfacesByName[iface.Name] = iface
	}

	vmiPodNetworkName := lookupVMIPodNetworkName(vmi.Spec.Networks)
	if vmiPodNetworkName != "" && interfacesByName[vmiPodNetworkName].Masquerade == nil {
		return fmt.Errorf("cannot migrate VMI which does not use masquerade to connect to the pod network")
	}

	return nil
}

func lookupVMIPodNetworkName(networks []v1.Network) string {
	for _, network := range networks {
		if network.Pod != nil {
			return network.Name
		}
	}

	return ""
}

func (d *VirtualMachineController) validateSRIOVInterfacesForMigration(vmi *v1.VirtualMachineInstance) error {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil && !d.clusterConfig.SRIOVLiveMigrationEnabled() {
			return fmt.Errorf("SRIOVLiveMigration feature-gate is closed, can't migrate VMI with SRIOV interfaces")
		}
	}

	return nil
}

func (d *VirtualMachineController) checkVolumesForMigration(vmi *v1.VirtualMachineInstance) (blockMigrate bool, err error) {

	volumeStatusMap := make(map[string]v1.VolumeStatus)

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	// Check if all VMI volumes can be shared between the source and the destination
	// of a live migration. blockMigrate will be returned as false, only if all volumes
	// are shared and the VMI has no local disks
	// Some combinations of disks makes the VMI no suitable for live migration.
	// A relevant error will be returned in this case.
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		if volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil {

			var claimName string
			if volSrc.PersistentVolumeClaim != nil {
				claimName = volSrc.PersistentVolumeClaim.ClaimName
			} else {
				claimName = volSrc.DataVolume.Name
			}

			volumeStatus, ok := volumeStatusMap[volume.Name]

			if !ok || volumeStatus.PersistentVolumeClaimInfo == nil {
				return true, fmt.Errorf("cannot migrate VMI: Unable to determine if PVC %v is shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)", claimName)
			} else if !pvctypes.HasSharedAccessMode(volumeStatus.PersistentVolumeClaimInfo.AccessModes) {
				return true, fmt.Errorf("cannot migrate VMI: PVC %v is not shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)", claimName)
			}

		} else if volSrc.HostDisk != nil {
			shared := volSrc.HostDisk.Shared != nil && *volSrc.HostDisk.Shared
			if !shared {
				return true, fmt.Errorf("cannot migrate VMI with non-shared HostDisk")
			}
		} else {
			blockMigrate = true
		}
	}
	return
}

func (d *VirtualMachineController) isMigrationSource(vmi *v1.VirtualMachineInstance) bool {

	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceNode == d.host &&
		vmi.Status.MigrationState.TargetNodeAddress != "" &&
		!vmi.Status.MigrationState.Completed {

		return true
	}
	return false

}

func (d *VirtualMachineController) handleTargetMigrationProxy(vmi *v1.VirtualMachineInstance) error {
	// handle starting/stopping target migration proxy
	migrationTargetSockets := []string{}
	res, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	// Get the libvirt connection socket file on the destination pod.
	socketFile := fmt.Sprintf(filepath.Join(d.virtLauncherFSRunDirPattern, "libvirt/libvirt-sock"), res.Pid())
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(d.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	migrationTargetSockets = append(migrationTargetSockets, socketFile)

	isBlockMigration := vmi.Status.MigrationMethod == v1.BlockMigration
	migrationPortsRange := migrationproxy.GetMigrationPortsList(isBlockMigration)
	for _, port := range migrationPortsRange {
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), port)
		// a proxy between the target direct qemu channel and the connector in the destination pod
		destSocketFile := migrationproxy.SourceUnixFile(baseDir, key)
		migrationTargetSockets = append(migrationTargetSockets, destSocketFile)
	}
	err = d.migrationProxy.StartTargetListener(string(vmi.UID), migrationTargetSockets)
	if err != nil {
		return err
	}
	return nil
}

func (d *VirtualMachineController) handlePostMigrationProxyCleanup(vmi *v1.VirtualMachineInstance) error {

	if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.Completed || vmi.Status.MigrationState.Failed {
		d.migrationProxy.StopTargetListener(string(vmi.UID))
		d.migrationProxy.StopSourceListener(string(vmi.UID))
	}
	return nil
}

func (d *VirtualMachineController) handleSourceMigrationProxy(vmi *v1.VirtualMachineInstance) error {

	res, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(d.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	d.migrationProxy.StopTargetListener(string(vmi.UID))
	if vmi.Status.MigrationState.TargetDirectMigrationNodePorts == nil {
		msg := "No migration proxy has been created for this vmi"
		return fmt.Errorf("%s", msg)
	}
	err = d.migrationProxy.StartSourceListener(
		string(vmi.UID),
		vmi.Status.MigrationState.TargetNodeAddress,
		vmi.Status.MigrationState.TargetDirectMigrationNodePorts,
		baseDir,
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *VirtualMachineController) getLauncherClientInfo(vmi *v1.VirtualMachineInstance) *virtcache.LauncherClientInfo {
	launcherInfo, exists := d.launcherClients.Load(vmi.UID)
	if !exists {
		return nil
	}
	return launcherInfo
}

func (d *VirtualMachineController) vmUpdateHelperMigrationSource(origVMI *v1.VirtualMachineInstance) error {
	client, err := d.getLauncherClient(origVMI)
	if err != nil {
		return fmt.Errorf("unable to create virt-launcher client connection: %v", err)
	}

	vmi := origVMI.DeepCopy()

	err = hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	err = d.handleSourceMigrationProxy(vmi)
	if err != nil {
		return fmt.Errorf("failed to handle migration proxy: %v", err)
	}

	if vmi.Status.MigrationState.AbortRequested {
		if vmi.Status.MigrationState.AbortStatus != v1.MigrationAbortInProgress {
			err = client.CancelVirtualMachineMigration(vmi)
			if err != nil {
				return err
			}
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), "VirtualMachineInstance is aborting migration.")
		}
	} else {
		migrationConfiguration := d.clusterConfig.GetMigrationConfiguration()

		options := &cmdclient.MigrationOptions{
			Bandwidth:               *migrationConfiguration.BandwidthPerMigration,
			ProgressTimeout:         *migrationConfiguration.ProgressTimeout,
			CompletionTimeoutPerGiB: *migrationConfiguration.CompletionTimeoutPerGiB,
			UnsafeMigration:         *migrationConfiguration.UnsafeMigrationOverride,
			AllowAutoConverge:       *migrationConfiguration.AllowAutoConverge,
			AllowPostCopy:           *migrationConfiguration.AllowPostCopy,
		}

		err = client.MigrateVirtualMachine(vmi, options)
		if err != nil {
			return err
		}
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), "VirtualMachineInstance is migrating.")
	}
	return nil
}

func (d *VirtualMachineController) vmUpdateHelperMigrationTarget(origVMI *v1.VirtualMachineInstance) error {
	client, err := d.getLauncherClient(origVMI)
	if err != nil {
		return fmt.Errorf("unable to create virt-launcher client connection: %v", err)
	}

	vmi := origVMI.DeepCopy()

	if migrations.MigrationFailed(vmi) {
		// if the migration failed, signal the target pod it's okay to exit
		err = client.SignalTargetPodCleanup(vmi)
		if err != nil {
			return err
		}
		log.Log.Object(vmi).Infof("Signaled target pod for failed migration to clean up")
		// nothing left to do here if the migration failed.
		return nil
	} else if migrations.IsMigrating(vmi) {
		// If the migration has already started,
		// then there's nothing left to prepare on the target side
		return nil
	}

	err = hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	// give containerDisks some time to become ready before throwing errors on retries
	info := d.getLauncherClientInfo(vmi)
	if ready, err := d.containerDiskMounter.ContainerDisksReady(vmi, info.NotInitializedSince); !ready {
		if err != nil {
			return err
		}
		d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	}

	// Mount container disks
	disksInfo, err := d.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return err
	}

	if err := d.containerDiskMounter.MountKernelArtifacts(vmi, false); err != nil {
		return fmt.Errorf("failed to mount kernel artifacts: %v", err)
	}

	// Mount hotplug disks
	if attachmentPodUID := vmi.Status.MigrationState.TargetAttachmentPodUID; attachmentPodUID != types.UID("") {
		if err := d.hotplugVolumeMounter.MountFromPod(vmi, attachmentPodUID); err != nil {
			return fmt.Errorf("failed to mount hotplug volumes: %v", err)
		}
	}

	// configure network inside virt-launcher compute container
	if err := d.setPodNetworkPhase1(vmi); err != nil {
		return fmt.Errorf("failed to configure vmi network for migration target: %w", err)
	}

	isolationRes, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod: %v", err)
	}
	virtLauncherRootMount := isolationRes.MountRoot()

	err = d.claimDeviceOwnership(virtLauncherRootMount, "kvm")
	if err != nil {
		return fmt.Errorf("failed to set up file ownership for /dev/kvm: %v", err)
	}

	lessPVCSpaceToleration := d.clusterConfig.GetLessPVCSpaceToleration()
	minimumPVCReserveBytes := d.clusterConfig.GetMinimumReservePVCBytes()

	// initialize disks images for empty PVC
	hostDiskCreator := hostdisk.NewHostDiskCreator(d.recorder, lessPVCSpaceToleration, minimumPVCReserveBytes, virtLauncherRootMount)
	err = hostDiskCreator.Create(vmi)
	if err != nil {
		return fmt.Errorf("preparing host-disks failed: %v", err)
	}

	if virtutil.IsNonRootVMI(vmi) {
		if err := d.nonRootSetup(origVMI, vmi); err != nil {
			return err
		}
	}

	options := virtualMachineOptions(nil, 0, nil, d.capabilities, disksInfo, d.clusterConfig.ExpandDisksEnabled())
	if err := client.SyncMigrationTarget(vmi, options); err != nil {
		return fmt.Errorf("syncing migration target failed: %v", err)
	}
	d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), "VirtualMachineInstance Migration Target Prepared.")

	err = d.handleTargetMigrationProxy(vmi)
	if err != nil {
		return fmt.Errorf("failed to handle post sync migration proxy: %v", err)
	}
	return nil
}
func (d *VirtualMachineController) vmUpdateHelperDefault(origVMI *v1.VirtualMachineInstance, domainExists bool) error {
	client, err := d.getLauncherClient(origVMI)
	if err != nil {
		return fmt.Errorf("unable to create virt-launcher client connection: %v", err)
	}

	vmi := origVMI.DeepCopy()
	// Find preallocated volumes
	var preallocatedVolumes []string
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.PersistentVolumeClaimInfo != nil && volumeStatus.PersistentVolumeClaimInfo.Preallocated {
			preallocatedVolumes = append(preallocatedVolumes, volumeStatus.Name)
		}
	}

	err = hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	disksInfo := map[string]*containerdisk.DiskInfo{}
	if !vmi.IsRunning() && !vmi.IsFinal() {
		// give containerDisks some time to become ready before throwing errors on retries
		info := d.getLauncherClientInfo(vmi)
		if ready, err := d.containerDiskMounter.ContainerDisksReady(vmi, info.NotInitializedSince); !ready {
			if err != nil {
				return err
			}
			d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
			return nil
		}

		disksInfo, err = d.containerDiskMounter.MountAndVerify(vmi)
		if err != nil {
			return err
		}

		if err := d.containerDiskMounter.MountKernelArtifacts(vmi, true); err != nil {
			return fmt.Errorf("failed to mount kernel artifacts: %v", err)
		}

		// Try to mount hotplug volume if there is any during startup.
		if err := d.hotplugVolumeMounter.Mount(vmi); err != nil {
			return err
		}

		if err := d.setPodNetworkPhase1(vmi); err != nil {
			return fmt.Errorf("failed to configure vmi network: %w", err)
		}

		isolationRes, err := d.podIsolationDetector.Detect(vmi)
		if err != nil {
			return fmt.Errorf("failed to detect isolation for launcher pod: %v", err)
		}
		virtLauncherRootMount := isolationRes.MountRoot()

		err = d.claimDeviceOwnership(virtLauncherRootMount, "kvm")
		if err != nil {
			return fmt.Errorf("failed to set up file ownership for /dev/kvm: %v", err)
		}

		lessPVCSpaceToleration := d.clusterConfig.GetLessPVCSpaceToleration()
		minimumPVCReserveBytes := d.clusterConfig.GetMinimumReservePVCBytes()

		// initialize disks images for empty PVC
		hostDiskCreator := hostdisk.NewHostDiskCreator(d.recorder, lessPVCSpaceToleration, minimumPVCReserveBytes, virtLauncherRootMount)
		err = hostDiskCreator.Create(vmi)
		if err != nil {
			return fmt.Errorf("preparing host-disks failed: %v", err)
		}

		if virtutil.IsNonRootVMI(vmi) {
			if err := d.nonRootSetup(origVMI, vmi); err != nil {
				return err
			}
		}

		// set runtime limits as needed
		err = d.podIsolationDetector.AdjustResources(vmi)
		if err != nil {
			return fmt.Errorf("failed to adjust resources: %v", err)
		}
	} else if vmi.IsRunning() {
		if err := d.hotplugVolumeMounter.Mount(vmi); err != nil {
			return err
		}
	}

	smbios := d.clusterConfig.GetSMBIOS()
	period := d.clusterConfig.GetMemBalloonStatsPeriod()

	options := virtualMachineOptions(smbios, period, preallocatedVolumes, d.capabilities, disksInfo, d.clusterConfig.ExpandDisksEnabled())

	err = client.SyncVirtualMachine(vmi, options)
	if err != nil {
		isSecbootError := strings.Contains(err.Error(), "EFI OVMF rom missing")
		if isSecbootError {
			return &virtLauncherCriticalSecurebootError{fmt.Sprintf("mismatch of Secure Boot setting and bootloaders: %v", err)}
		}
		return err
	}

	if !domainExists {
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Created.String(), VMIDefined)
	}

	if vmi.IsRunning() {
		// Umount any disks no longer mounted
		if err := d.hotplugVolumeMounter.Unmount(vmi); err != nil {
			return err
		}
	}
	return nil
}

func (d *VirtualMachineController) processVmUpdate(vmi *v1.VirtualMachineInstance, domainExists bool) error {

	isUnresponsive, isInitialized, err := d.isLauncherClientUnresponsive(vmi)
	if err != nil {
		return err
	}
	if !isInitialized {
		d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	} else if isUnresponsive {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with unresponsive command server."))
	}

	d.handlePostMigrationProxyCleanup(vmi)

	if d.isPreMigrationTarget(vmi) {
		return d.vmUpdateHelperMigrationTarget(vmi)
	} else if d.isMigrationSource(vmi) {
		return d.vmUpdateHelperMigrationSource(vmi)
	} else {
		return d.vmUpdateHelperDefault(vmi, domainExists)
	}
}

func (d *VirtualMachineController) setVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) error {
	phase, err := d.calculateVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}
	vmi.Status.Phase = phase
	return nil
}
func (d *VirtualMachineController) calculateVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) (v1.VirtualMachineInstancePhase, error) {

	if domain == nil {
		switch {
		case vmi.IsScheduled():
			isUnresponsive, isInitialized, err := d.isLauncherClientUnresponsive(vmi)

			if err != nil {
				return vmi.Status.Phase, err
			}
			if !isInitialized {
				d.Queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
				return vmi.Status.Phase, err
			} else if isUnresponsive {
				// virt-launcher is gone and VirtualMachineInstance never transitioned
				// from scheduled to Running.
				return v1.Failed, nil
			}
			return v1.Scheduled, nil
		case !vmi.IsRunning() && !vmi.IsFinal():
			return v1.Scheduled, nil
		case !vmi.IsFinal():
			// That is unexpected. We should not be able to delete a VirtualMachineInstance before we stop it.
			// However, if someone directly interacts with libvirt it is possible
			return v1.Failed, nil
		}
	} else {

		switch domain.Status.Status {
		case api.Shutoff, api.Crashed:
			switch domain.Status.Reason {
			case api.ReasonCrashed, api.ReasonPanicked:
				return v1.Failed, nil
			case api.ReasonDestroyed:
				// When ACPI is available, the domain was tried to be shutdown,
				// and destroyed means that the domain was destroyed after the graceperiod expired.
				// Without ACPI a destroyed domain is ok.
				if isACPIEnabled(vmi, domain) {
					return v1.Failed, nil
				}
				return v1.Succeeded, nil
			case api.ReasonShutdown, api.ReasonSaved, api.ReasonFromSnapshot:
				return v1.Succeeded, nil
			case api.ReasonMigrated:
				// if the domain migrated, we no longer know the phase.
				return vmi.Status.Phase, nil
			}
		case api.Running, api.Paused, api.Blocked, api.PMSuspended:
			return v1.Running, nil
		}
	}
	return vmi.Status.Phase, nil
}

func (d *VirtualMachineController) addFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.vmiExpectations.LowerExpectations(key, 1, 0)
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) deleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.vmiExpectations.LowerExpectations(key, 1, 0)
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) updateFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		d.vmiExpectations.LowerExpectations(key, 1, 0)
		d.Queue.Add(key)
	}
}

func (d *VirtualMachineController) addDomainFunc(obj interface{}) {
	domain := obj.(*api.Domain)
	log.Log.Object(domain).Infof("Domain is in state %s reason %s", domain.Status.Status, domain.Status.Reason)
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) deleteDomainFunc(obj interface{}) {
	domain, ok := obj.(*api.Domain)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		domain, ok = tombstone.Obj.(*api.Domain)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a domain %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	log.Log.Object(domain).Info("Domain deleted")
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) updateDomainFunc(old, new interface{}) {
	newDomain := new.(*api.Domain)
	oldDomain := old.(*api.Domain)
	if oldDomain.Status.Status != newDomain.Status.Status || oldDomain.Status.Reason != newDomain.Status.Reason {
		log.Log.Object(newDomain).Infof("Domain is in state %s reason %s", newDomain.Status.Status, newDomain.Status.Reason)
	}

	if newDomain.ObjectMeta.DeletionTimestamp != nil {
		log.Log.Object(newDomain).Info("Domain is marked for deletion")
	}

	key, err := controller.KeyFunc(new)
	if err == nil {
		d.Queue.Add(key)
	}
}

func (d *VirtualMachineController) finalizeMigration(vmi *v1.VirtualMachineInstance) error {
	const errorMessage = "failed to finalize migration"
	client, err := d.getVerifiedLauncherClient(vmi)
	if err != nil {
		return err
	}

	if err := isolation.AdjustQemuProcessMemoryLimits(d.podIsolationDetector, vmi); err != nil {
		d.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), errorMessage)
	}

	if err := client.FinalizeVirtualMachineMigration(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		return err
	}

	return nil
}

func vmiHasTerminationGracePeriod(vmi *v1.VirtualMachineInstance) bool {
	// if not set we use the default graceperiod
	return vmi.Spec.TerminationGracePeriodSeconds == nil ||
		(vmi.Spec.TerminationGracePeriodSeconds != nil && *vmi.Spec.TerminationGracePeriodSeconds != 0)
}

func domainHasGracePeriod(domain *api.Domain) bool {
	return domain != nil &&
		domain.Spec.Metadata.KubeVirt.GracePeriod != nil &&
		domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds != 0
}

func isACPIEnabled(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	return (vmiHasTerminationGracePeriod(vmi) || (vmi.Spec.TerminationGracePeriodSeconds == nil && domainHasGracePeriod(domain))) &&
		domain != nil &&
		domain.Spec.Features != nil &&
		domain.Spec.Features.ACPI != nil
}

func setMissingSRIOVInterfacesNames(interfacesSpecByName map[string]v1.Interface, interfacesStatusByMac map[string]api.InterfaceStatus) {
	for name, ifaceSpec := range interfacesSpecByName {
		if ifaceSpec.SRIOV == nil || ifaceSpec.MacAddress == "" {
			continue
		}
		if domainIfaceStatus, exists := interfacesStatusByMac[ifaceSpec.MacAddress]; exists {
			domainIfaceStatus.Name = name
			interfacesStatusByMac[ifaceSpec.MacAddress] = domainIfaceStatus
		}
	}
}

func (d *VirtualMachineController) isHostModelMigratable(vmi *v1.VirtualMachineInstance) error {
	if cpu := vmi.Spec.Domain.CPU; cpu != nil && cpu.Model == v1.CPUModeHostModel {
		node, err := d.clientset.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if !nodeHasHostModelLabel(node) {
			err = fmt.Errorf("the node \"%s\" has no (%s/...) label to allow migration with host-model", node.Name, v1.HostModelCPULabel)
			log.Log.Object(vmi).Errorf(err.Error())
			return err
		}
	}

	return nil
}

func (d *VirtualMachineController) claimDeviceOwnership(virtLauncherRootMount, deviceName string) error {
	kvmPath := filepath.Join(virtLauncherRootMount, "dev", deviceName)

	softwareEmulation, err := util.UseSoftwareEmulationForDevice(kvmPath, d.clusterConfig.AllowEmulation())
	if err != nil || softwareEmulation {
		return err
	}

	return diskutils.DefaultOwnershipManager.SetFileOwnership(kvmPath)
}

func nodeHasHostModelLabel(node *k8sv1.Node) bool {
	for key, _ := range node.Labels {
		if strings.HasPrefix(key, v1.HostModelCPULabel) {
			return true
		}
	}
	return false
}
