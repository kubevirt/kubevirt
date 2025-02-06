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

package virthandler

import (
	"bytes"
	"context"
	"encoding/json"
	goerror "errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-ps"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"golang.org/x/sys/unix"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/controller"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/executor"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	netcache "kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/domainspec"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	pvctypes "kubevirt.io/kubevirt/pkg/storage/types"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	container_disk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	"kubevirt.io/kubevirt/pkg/virt-handler/heartbeat"
	hotplug_volume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	multipath_monitor "kubevirt.io/kubevirt/pkg/virt-handler/multipath-monitor"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

type netconf interface {
	Setup(vmi *v1.VirtualMachineInstance, networks []v1.Network, launcherPid int) error
	Teardown(vmi *v1.VirtualMachineInstance) error
}

type netstat interface {
	UpdateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error
	Teardown(vmi *v1.VirtualMachineInstance)
	PodInterfaceVolatileDataIsCached(vmi *v1.VirtualMachineInstance, ifaceName string) bool
	CachePodInterfaceVolatileData(vmi *v1.VirtualMachineInstance, ifaceName string, data *netcache.PodIfaceCacheData)
}

type downwardMetricsManager interface {
	Run(stopCh chan struct{})
	StartServer(vmi *v1.VirtualMachineInstance, pid int) error
	StopServer(vmi *v1.VirtualMachineInstance)
}

const (
	failedDetectIsolationFmt              = "failed to detect isolation for launcher pod: %v"
	unableCreateVirtLauncherConnectionFmt = "unable to create virt-launcher client connection: %v"
	// This value was determined after consulting with libvirt developers and performing extensive testing.
	parallelMultifdMigrationThreads = uint(8)
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

	// MemoryHotplugFailedReason is the reason set when the VM cannot hotplug memory
	memoryHotplugFailedReason = "Memory Hotplug Failed"
)

var getCgroupManager = func(vmi *v1.VirtualMachineInstance, host string) (cgroup.Manager, error) {
	return cgroup.NewManagerFromVM(vmi, host)
}

func NewController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	migrationIpAddress string,
	virtShareDir string,
	virtPrivateDir string,
	kubeletPodsDir string,
	vmiInformer cache.SharedIndexInformer,
	vmiSourceInformer cache.SharedIndexInformer,
	vmiTargetInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	maxDevices int,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	migrationProxy migrationproxy.ProxyManager,
	downwardMetricsManager downwardMetricsManager,
	capabilities *libvirtxml.Caps,
	hostCpuModel string,
	netConf netconf,
	netStat netstat,
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator,
) (*VirtualMachineController, error) {

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-handler-vm"},
	)

	containerDiskState := filepath.Join(virtPrivateDir, "container-disk-mount-state")
	if err := os.MkdirAll(containerDiskState, 0700); err != nil {
		return nil, err
	}

	hotplugState := filepath.Join(virtPrivateDir, "hotplug-volume-mount-state")
	if err := os.MkdirAll(hotplugState, 0700); err != nil {
		return nil, err
	}

	c := &VirtualMachineController{
		queue:                            queue,
		recorder:                         recorder,
		clientset:                        clientset,
		host:                             host,
		migrationIpAddress:               migrationIpAddress,
		virtShareDir:                     virtShareDir,
		vmiStore:                         vmiInformer.GetStore(),
		vmiSourceStore:                   vmiSourceInformer.GetStore(),
		vmiTargetStore:                   vmiTargetInformer.GetStore(),
		domainStore:                      domainInformer.GetStore(),
		heartBeatInterval:                1 * time.Minute,
		migrationProxy:                   migrationProxy,
		podIsolationDetector:             podIsolationDetector,
		containerDiskMounter:             container_disk.NewMounter(podIsolationDetector, containerDiskState, clusterConfig),
		hotplugVolumeMounter:             hotplug_volume.NewVolumeMounter(hotplugState, kubeletPodsDir, host),
		clusterConfig:                    clusterConfig,
		virtLauncherFSRunDirPattern:      "/proc/%d/root/var/run",
		capabilities:                     capabilities,
		hostCpuModel:                     hostCpuModel,
		vmiExpectations:                  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		sriovHotplugExecutorPool:         executor.NewRateLimitedExecutorPool(executor.NewExponentialLimitedBackoffCreator()),
		ioErrorRetryManager:              NewFailRetryManager("io-error-retry", 10*time.Second, 3*time.Minute, 30*time.Second),
		netConf:                          netConf,
		netStat:                          netStat,
		netBindingPluginMemoryCalculator: netBindingPluginMemoryCalculator,
		multipathSocketMonitor:           multipath_monitor.NewMultipathSocketMonitor(),
	}

	c.hasSynced = func() bool {
		return domainInformer.HasSynced() && vmiSourceInformer.HasSynced() && vmiTargetInformer.HasSynced() && vmiInformer.HasSynced()
	}

	_, err := vmiSourceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})
	if err != nil {
		return nil, err
	}

	_, err = vmiTargetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})
	if err != nil {
		return nil, err
	}

	_, err = domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDomainFunc,
		DeleteFunc: c.deleteDomainFunc,
		UpdateFunc: c.updateDomainFunc,
	})
	if err != nil {
		return nil, err
	}

	c.launcherClients = virtcache.LauncherClientInfoByVMI{}

	c.downwardMetricsManager = downwardMetricsManager

	c.domainNotifyPipes = make(map[string]string)

	permissions := "rw"
	if cgroups.IsCgroup2UnifiedMode() {
		// Need 'rwm' permissions otherwise ebpf filtering program attached by runc
		// will deny probing the device file with 'access' syscall. That in turn
		// will lead to virtqemud failure on VM startup.
		// This has been fixed upstream:
		//   https://github.com/opencontainers/runc/pull/2796
		// but the workaround is still needed to support previous versions without
		// the patch.
		permissions = "rwm"
	}

	c.deviceManagerController = device_manager.NewDeviceController(
		c.host,
		maxDevices,
		permissions,
		device_manager.PermanentHostDevicePlugins(maxDevices, permissions),
		clusterConfig,
		clientset.CoreV1())
	c.heartBeat = heartbeat.NewHeartBeat(clientset.CoreV1(), c.deviceManagerController, clusterConfig, host)

	return c, nil
}

type netBindingPluginMemoryCalculator interface {
	Calculate(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) resource.Quantity
}

type VirtualMachineController struct {
	recorder                 record.EventRecorder
	clientset                kubecli.KubevirtClient
	host                     string
	migrationIpAddress       string
	virtShareDir             string
	virtPrivateDir           string
	queue                    workqueue.TypedRateLimitingInterface[string]
	vmiStore                 cache.Store
	vmiSourceStore           cache.Store
	vmiTargetStore           cache.Store
	domainStore              cache.Store
	launcherClients          virtcache.LauncherClientInfoByVMI
	heartBeatInterval        time.Duration
	deviceManagerController  *device_manager.DeviceController
	migrationProxy           migrationproxy.ProxyManager
	podIsolationDetector     isolation.PodIsolationDetector
	containerDiskMounter     container_disk.Mounter
	hotplugVolumeMounter     hotplug_volume.VolumeMounter
	clusterConfig            *virtconfig.ClusterConfig
	sriovHotplugExecutorPool *executor.RateLimitedExecutorPool
	downwardMetricsManager   downwardMetricsManager
	multipathSocketMonitor   *multipath_monitor.MultipathSocketMonitor

	netConf                          netconf
	netStat                          netstat
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator

	domainNotifyPipes           map[string]string
	virtLauncherFSRunDirPattern string
	heartBeat                   *heartbeat.HeartBeat
	capabilities                *libvirtxml.Caps
	hostCpuModel                string
	vmiExpectations             *controller.UIDTrackingControllerExpectations
	ioErrorRetryManager         *FailRetryManager
	hasSynced                   func() bool
}

type virtLauncherCriticalSecurebootError struct {
	msg string
}

func (e *virtLauncherCriticalSecurebootError) Error() string { return e.msg }

type vmiIrrecoverableError struct {
	msg string
}

func (e *vmiIrrecoverableError) Error() string { return e.msg }

func formatIrrecoverableErrorMessage(domain *api.Domain) string {
	msg := "unknown reason"
	if domainPausedFailedPostCopy(domain) {
		msg = "VMI is irrecoverable due to failed post-copy migration"
	}
	return msg
}

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

func (c *VirtualMachineController) startDomainNotifyPipe(domainPipeStopChan chan struct{}, vmi *v1.VirtualMachineInstance) error {

	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod when setting up notify pipe: %v", err)
	}

	// inject the domain-notify.sock into the VMI pod.
	root, err := res.MountRoot()
	if err != nil {
		return err
	}
	socketDir, err := root.AppendAndResolveWithRelativeRoot(c.virtShareDir)
	if err != nil {
		return err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}
	socketPath, err := safepath.JoinNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return err
	}

	if util.IsNonRootVMI(vmi) {
		err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			log.Log.Reason(err).Error("unable to change ownership for domain notify")
			return err
		}
	}

	handleDomainNotifyPipe(domainPipeStopChan, listener, c.virtShareDir, vmi)

	return nil
}

// Determines if a domain's grace period has expired during shutdown.
// If the grace period has started but not expired, timeLeft represents
// the time in seconds left until the period expires.
// If the grace period has not started, timeLeft will be set to -1.
func (c *VirtualMachineController) hasGracePeriodExpired(dom *api.Domain) (hasExpired bool, timeLeft int64) {

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

func (c *VirtualMachineController) hasTargetDetectedReadyDomain(vmi *v1.VirtualMachineInstance) (bool, int64) {
	// give the target node 60 seconds to discover the libvirt domain via the domain informer
	// before allowing the VMI to be processed. This closes the gap between the
	// VMI's status getting updated to reflect the new source node, and the domain
	// informer firing the event to alert the source node of the new domain.
	migrationTargetDelayTimeout := 60

	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetNodeDomainDetected &&
		vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp != nil {

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
	c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Duration(enqueueTime)*time.Second)

	return false, timeLeft
}

// teardownNetwork performs network cache cleanup for a specific VMI.
func (c *VirtualMachineController) teardownNetwork(vmi *v1.VirtualMachineInstance) {
	if string(vmi.UID) == "" {
		return
	}
	if err := c.netConf.Teardown(vmi); err != nil {
		log.Log.Reason(err).Errorf("failed to delete VMI Network cache files: %s", err.Error())
	}
	c.netStat.Teardown(vmi)
}

func domainPausedFailedPostCopy(domain *api.Domain) bool {
	return domain != nil && domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedPostcopyFailed
}

func domainMigrated(domain *api.Domain) bool {
	return domain != nil && domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonMigrated
}

func canUpdateToMounted(currentPhase v1.VolumePhase) bool {
	return currentPhase == v1.VolumeBound || currentPhase == v1.VolumePending || currentPhase == v1.HotplugVolumeAttachedToNode
}

func canUpdateToUnmounted(currentPhase v1.VolumePhase) bool {
	return currentPhase == v1.VolumeReady || currentPhase == v1.HotplugVolumeMounted || currentPhase == v1.HotplugVolumeAttachedToNode
}

func wasMigrationSuccessful(migrationState *v1.VirtualMachineInstanceMigrationState) bool {
	return migrationState != nil && migrationState.EndTimestamp != nil && !migrationState.Failed
}

func (c *VirtualMachineController) setMigrationProgressStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if domain == nil ||
		domain.Spec.Metadata.KubeVirt.Migration == nil ||
		vmi.Status.MigrationState == nil ||
		!c.isMigrationSource(vmi) {
		return
	}

	migrationMetadata := domain.Spec.Metadata.KubeVirt.Migration
	if migrationMetadata.UID != vmi.Status.MigrationState.MigrationUID {
		return
	}

	if vmi.Status.MigrationState.EndTimestamp == nil && migrationMetadata.EndTimestamp != nil {
		if migrationMetadata.Failed {
			vmi.Status.MigrationState.FailureReason = migrationMetadata.FailureReason
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("VirtualMachineInstance migration uid %s failed. reason:%s", string(migrationMetadata.UID), migrationMetadata.FailureReason))
		}
	}

	if vmi.Status.MigrationState.StartTimestamp == nil {
		vmi.Status.MigrationState.StartTimestamp = migrationMetadata.StartTimestamp
	}
	if vmi.Status.MigrationState.EndTimestamp == nil {
		vmi.Status.MigrationState.EndTimestamp = migrationMetadata.EndTimestamp
	}
	vmi.Status.MigrationState.AbortStatus = v1.MigrationAbortStatus(migrationMetadata.AbortStatus)
	vmi.Status.MigrationState.Failed = migrationMetadata.Failed
	vmi.Status.MigrationState.Mode = migrationMetadata.Mode
}

func (c *VirtualMachineController) migrationSourceUpdateVMIStatus(origVMI *v1.VirtualMachineInstance, domain *api.Domain) error {

	vmi := origVMI.DeepCopy()
	oldStatus := vmi.DeepCopy().Status

	// if a migration happens very quickly, it's possible parts of the in
	// progress status wasn't set. We need to make sure we set this even
	// if the migration has completed
	c.setMigrationProgressStatus(vmi, domain)

	// handle migrations differently than normal status updates.
	//
	// When a successful migration is detected, we must transfer ownership of the VMI
	// from the source node (this node) to the target node (node the domain was migrated to).
	//
	// Transfer ownership by...
	// 1. Marking vmi.Status.MigrationState as completed
	// 2. Update the vmi.Status.NodeName to reflect the target node's name
	// 3. Update the VMI's NodeNameLabel annotation to reflect the target node's name
	// 4. Clear the LauncherContainerImageVersion which virt-controller will detect
	//    and accurately based on the version used on the target pod
	//
	// After a migration, the VMI's phase is no longer owned by this node. Only the
	// MigrationState status field is eligible to be mutated.
	migrationHost := ""
	if vmi.Status.MigrationState != nil {
		migrationHost = vmi.Status.MigrationState.TargetNode
	}

	targetNodeDetectedDomain, timeLeft := c.hasTargetDetectedReadyDomain(vmi)
	// If we can't detect where the migration went to, then we have no
	// way of transferring ownership. The only option here is to move the
	// vmi to failed.  The cluster vmi controller will then tear down the
	// resulting pods.
	if migrationHost == "" {
		// migrated to unknown host.
		vmi.Status.Phase = v1.Failed
		vmi.Status.MigrationState.Completed = true
		vmi.Status.MigrationState.Failed = true

		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to unknown host."))
	} else if !targetNodeDetectedDomain {
		if timeLeft <= 0 {
			vmi.Status.Phase = v1.Failed
			vmi.Status.MigrationState.Completed = true
			vmi.Status.MigrationState.Failed = true

			c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance's domain was never observed on the target after the migration completed within the timeout period."))
		} else {
			log.Log.Object(vmi).Info("Waiting on the target node to observe the migrated domain before performing the handoff")
		}
	} else if wasMigrationSuccessful(vmi.Status.MigrationState) {
		// this is the migration ACK.
		// At this point we know that the migration has completed and that
		// the target node has seen the domain event.
		vmi.Labels[v1.NodeNameLabel] = migrationHost
		delete(vmi.Labels, v1.OutdatedLauncherImageLabel)
		vmi.Status.LauncherContainerImageVersion = ""
		vmi.Status.NodeName = migrationHost
		// clean the evacuation node name since have already migrated to a new node
		vmi.Status.EvacuationNodeName = ""
		// update the vmi migrationTransport to indicate that next migration should use unix URI
		// new workloads will set the migrationTransport on their creation, however, legacy workloads
		// can make the switch only after the first migration
		vmi.Status.MigrationTransport = v1.MigrationTransportUnix
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to node %s.", migrationHost))
		log.Log.Object(vmi).Infof("migration completed to node %s", migrationHost)
	}

	if !equality.Semantic.DeepEqual(oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}
	return nil
}

func domainIsActiveOnTarget(domain *api.Domain) bool {

	if domain == nil {
		return false
	}

	// It's possible for the domain to be active on the target node if the domain is
	// 1. Running
	// 2. User initiated Paused
	if domain.Status.Status == api.Running {
		return true
	} else if domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedUser {
		return true
	}
	return false

}

func (c *VirtualMachineController) migrationTargetUpdateVMIStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	if migrations.MigrationFailed(vmi) {
		// nothing left to report on the target node if the migration failed
		return nil
	}

	domainExists := domain != nil

	// Handle post migration
	if domainExists && vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.TargetNodeDomainDetected {
		// record that we've see the domain populated on the target's node
		log.Log.Object(vmi).Info("The target node received the migrated domain")
		vmi.Status.MigrationState.TargetNodeDomainDetected = true

		// adjust QEMU process memlock limits in order to enable old virt-launcher pod's to
		// perform hotplug host-devices on post migration.
		if err := isolation.AdjustQemuProcessMemoryLimits(c.podIsolationDetector, vmi, c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio); err != nil {
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "Failed to update target node qemu memory limits during live migration")
		}

	}

	if domainExists &&
		domainIsActiveOnTarget(domain) &&
		vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp == nil {

		// record the moment we detected the domain is running.
		// This is used as a trigger to help coordinate when CNI drivers
		// fail over the IP to the new pod.
		log.Log.Object(vmi).Info("The target node received the running migrated domain")
		now := metav1.Now()
		vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp = &now
	}

	if domainExists &&
		domain.Spec.Metadata.KubeVirt.Migration != nil &&
		domain.Spec.Metadata.KubeVirt.Migration.EndTimestamp != nil &&
		vmi.Status.MigrationState.EndTimestamp == nil {
		log.Log.Object(vmi).Info("The target node detected that the migration has completed")
		vmi.Status.MigrationState.EndTimestamp = domain.Spec.Metadata.KubeVirt.Migration.EndTimestamp
	}

	if !migrations.IsMigrating(vmi) {
		destSrcPortsMap := c.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
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
		if hostAddress != c.migrationIpAddress {
			portsList := make([]string, 0, len(destSrcPortsMap))

			for k := range destSrcPortsMap {
				portsList = append(portsList, k)
			}
			portsStrList := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(portsList)), ","), "[]")
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), fmt.Sprintf("Migration Target is listening at %s, on ports: %s", c.migrationIpAddress, portsStrList))
			vmi.Status.MigrationState.TargetNodeAddress = c.migrationIpAddress
			vmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPortsMap
		}

		// If the migrated VMI requires dedicated CPUs, report the new pod CPU set to the source node
		// via the VMI migration status in order to patch the domain pre migration
		if vmi.IsCPUDedicated() {
			err := c.reportDedicatedCPUSetForMigratingVMI(vmi)
			if err != nil {
				return err
			}
			err = c.reportTargetTopologyForMigratingVMI(vmi)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *VirtualMachineController) generateEventsForVolumeStatusChange(vmi *v1.VirtualMachineInstance, newStatusMap map[string]v1.VolumeStatus) {
	newStatusMapCopy := make(map[string]v1.VolumeStatus)
	for k, v := range newStatusMap {
		newStatusMapCopy[k] = v
	}
	for _, oldStatus := range vmi.Status.VolumeStatus {
		newStatus, ok := newStatusMap[oldStatus.Name]
		if !ok {
			// status got removed
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, VolumeUnplugged, fmt.Sprintf("Volume %s has been unplugged", oldStatus.Name))
			continue
		}
		if newStatus.Phase != oldStatus.Phase {
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, newStatus.Reason, newStatus.Message)
		}
		delete(newStatusMapCopy, newStatus.Name)
	}
	// Send events for any new statuses.
	for _, v := range newStatusMapCopy {
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v.Reason, v.Message)
	}
}

func (c *VirtualMachineController) updateHotplugVolumeStatus(vmi *v1.VirtualMachineInstance, volumeStatus v1.VolumeStatus, specVolumeMap map[string]v1.Volume) (v1.VolumeStatus, bool) {
	needsRefresh := false
	if volumeStatus.Target == "" {
		needsRefresh = true
		mounted, err := c.hotplugVolumeMounter.IsMounted(vmi, volumeStatus.Name, volumeStatus.HotplugVolume.AttachPodUID)
		if err != nil {
			log.Log.Object(vmi).Errorf("error occurred while checking if volume is mounted: %v", err)
		}
		if mounted {
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

func needToComputeChecksums(vmi *v1.VirtualMachineInstance) bool {
	containerDisks := map[string]*v1.Volume{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk != nil {
			containerDisks[volume.Name] = &volume
		}
	}

	for i := range vmi.Status.VolumeStatus {
		_, isContainerDisk := containerDisks[vmi.Status.VolumeStatus[i].Name]
		if !isContainerDisk {
			continue
		}

		if vmi.Status.VolumeStatus[i].ContainerDiskVolume == nil ||
			vmi.Status.VolumeStatus[i].ContainerDiskVolume.Checksum == 0 {
			return true
		}
	}

	if util.HasKernelBootContainerImage(vmi) {
		if vmi.Status.KernelBootStatus == nil {
			return true
		}

		kernelBootContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container

		if kernelBootContainer.KernelPath != "" &&
			(vmi.Status.KernelBootStatus.KernelInfo == nil ||
				vmi.Status.KernelBootStatus.KernelInfo.Checksum == 0) {
			return true

		}

		if kernelBootContainer.InitrdPath != "" &&
			(vmi.Status.KernelBootStatus.InitrdInfo == nil ||
				vmi.Status.KernelBootStatus.InitrdInfo.Checksum == 0) {
			return true

		}
	}

	return false
}

// updateChecksumInfo is kept for compatibility with older virt-handlers
// that validate checksum calculations in vmi.status. This validation was
// removed in PR #14021, but we had to keep the checksum calculations for upgrades.
// Once we're sure old handlers won't interrupt upgrades, this can be removed.
func (c *VirtualMachineController) updateChecksumInfo(vmi *v1.VirtualMachineInstance, syncError error) error {
	// If the imageVolume feature gate is enabled, upgrade support isn't required,
	// and we can skip the checksum calculation. By the time the feature gate is GA,
	// the checksum calculation should be removed.
	if syncError != nil || vmi.DeletionTimestamp != nil || !needToComputeChecksums(vmi) || c.clusterConfig.ImageVolumeEnabled() {
		return nil
	}

	diskChecksums, err := c.containerDiskMounter.ComputeChecksums(vmi)
	if goerror.Is(err, container_disk.ErrDiskContainerGone) {
		log.Log.Errorf("cannot compute checksums as containerdisk/kernelboot containers seem to have been terminated")
		return nil
	}
	if err != nil {
		return err
	}

	// containerdisks
	for i := range vmi.Status.VolumeStatus {
		checksum, exists := diskChecksums.ContainerDiskChecksums[vmi.Status.VolumeStatus[i].Name]
		if !exists {
			// not a containerdisk
			continue
		}

		vmi.Status.VolumeStatus[i].ContainerDiskVolume = &v1.ContainerDiskInfo{
			Checksum: checksum,
		}
	}

	// kernelboot
	if util.HasKernelBootContainerImage(vmi) {
		vmi.Status.KernelBootStatus = &v1.KernelBootStatus{}

		if diskChecksums.KernelBootChecksum.Kernel != nil {
			vmi.Status.KernelBootStatus.KernelInfo = &v1.KernelInfo{
				Checksum: *diskChecksums.KernelBootChecksum.Kernel,
			}
		}

		if diskChecksums.KernelBootChecksum.Initrd != nil {
			vmi.Status.KernelBootStatus.InitrdInfo = &v1.InitrdInfo{
				Checksum: *diskChecksums.KernelBootChecksum.Initrd,
			}
		}
	}

	return nil
}

func (c *VirtualMachineController) updateVolumeStatusesFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	// used by unit test
	hasHotplug := false

	if len(vmi.Status.VolumeStatus) == 0 {
		return hasHotplug
	}

	diskDeviceMap := make(map[string]string)
	if domain != nil {
		for _, disk := range domain.Spec.Devices.Disks {
			diskDeviceMap[disk.Alias.GetName()] = disk.Target.Device
		}
	}
	specVolumeMap := make(map[string]v1.Volume)
	for _, volume := range vmi.Spec.Volumes {
		specVolumeMap[volume.Name] = volume
	}
	newStatusMap := make(map[string]v1.VolumeStatus)
	var newStatuses []v1.VolumeStatus
	needsRefresh := false
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		tmpNeedsRefresh := false
		if _, ok := diskDeviceMap[volumeStatus.Name]; ok {
			volumeStatus.Target = diskDeviceMap[volumeStatus.Name]
		}
		if volumeStatus.HotplugVolume != nil {
			hasHotplug = true
			volumeStatus, tmpNeedsRefresh = c.updateHotplugVolumeStatus(vmi, volumeStatus, specVolumeMap)
			needsRefresh = needsRefresh || tmpNeedsRefresh
		}
		if volumeStatus.MemoryDumpVolume != nil {
			volumeStatus, tmpNeedsRefresh = c.updateMemoryDumpInfo(vmi, volumeStatus, domain)
			needsRefresh = needsRefresh || tmpNeedsRefresh
		}
		newStatuses = append(newStatuses, volumeStatus)
		newStatusMap[volumeStatus.Name] = volumeStatus
	}
	sort.SliceStable(newStatuses, func(i, j int) bool {
		return strings.Compare(newStatuses[i].Name, newStatuses[j].Name) == -1
	})
	if needsRefresh {
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second)
	}
	c.generateEventsForVolumeStatusChange(vmi, newStatusMap)
	vmi.Status.VolumeStatus = newStatuses

	return hasHotplug
}

func (c *VirtualMachineController) updateGuestInfoFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil || domain.Status.OSInfo.Name == "" || vmi.Status.GuestOSInfo.Name == domain.Status.OSInfo.Name {
		return
	}

	vmi.Status.GuestOSInfo.Name = domain.Status.OSInfo.Name
	vmi.Status.GuestOSInfo.Version = domain.Status.OSInfo.Version
	vmi.Status.GuestOSInfo.KernelRelease = domain.Status.OSInfo.KernelRelease
	vmi.Status.GuestOSInfo.PrettyName = domain.Status.OSInfo.PrettyName
	vmi.Status.GuestOSInfo.VersionID = domain.Status.OSInfo.VersionId
	vmi.Status.GuestOSInfo.KernelVersion = domain.Status.OSInfo.KernelVersion
	vmi.Status.GuestOSInfo.Machine = domain.Status.OSInfo.Machine
	vmi.Status.GuestOSInfo.ID = domain.Status.OSInfo.Id
}

func (c *VirtualMachineController) updateAccessCredentialConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {

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
			eventMessage := "Access credentials sync successful."
			if message != "" {
				eventMessage = fmt.Sprintf("Access credentials sync successful: %s", message)
			}
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.AccessCredentialsSyncSuccess.String(), eventMessage)
		} else {
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.AccessCredentialsSyncFailed.String(),
				fmt.Sprintf("Access credentials sync failed: %s", message),
			)
		}
	}
}

func (c *VirtualMachineController) updateLiveMigrationConditions(vmi *v1.VirtualMachineInstance, condManager *controller.VirtualMachineInstanceConditionManager) {
	// Calculate whether the VM is migratable
	liveMigrationCondition, isBlockMigration := c.calculateLiveMigrationCondition(vmi)
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
		if !equality.Semantic.DeepEqual(cond, liveMigrationCondition) {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
			vmi.Status.Conditions = append(vmi.Status.Conditions, *liveMigrationCondition)
		}
	}
	storageLiveMigCond := c.calculateLiveStorageMigrationCondition(vmi)
	condManager.UpdateCondition(vmi, storageLiveMigCond)
	evictable := migrations.VMIMigratableOnEviction(c.clusterConfig, vmi)
	if evictable && liveMigrationCondition.Status == k8sv1.ConditionFalse {
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), "EvictionStrategy is set but vmi is not migratable; %s", liveMigrationCondition.Message)
	}
}

func (c *VirtualMachineController) updateGuestAgentConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) error {

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
		client, err := c.getLauncherClient(vmi)
		if err != nil {
			return err
		}

		guestInfo, err := client.GetGuestInfo()
		if err != nil {
			return err
		}

		var supported = false
		var reason = ""

		// For current versions, virt-launcher's supported commands will always contain data.
		// For backwards compatibility: during upgrade from a previous version of KubeVirt,
		// virt-launcher might not provide any supported commands. If the list of supported
		// commands is empty, fall back to previous behavior.
		if len(guestInfo.SupportedCommands) > 0 {
			supported, reason = isGuestAgentSupported(vmi, guestInfo.SupportedCommands)
			log.Log.V(3).Object(vmi).Info(reason)
		} else {
			for _, version := range c.clusterConfig.GetSupportedAgentVersions() {
				supported = supported || regexp.MustCompile(version).MatchString(guestInfo.GAVersion)
			}
			if !supported {
				reason = fmt.Sprintf("Guest agent version '%s' is not supported", guestInfo.GAVersion)
			}
		}

		if !supported {
			if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent) {
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceUnsupportedAgent,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
					Reason:        reason,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
			}
		} else {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent)
		}

	}
	return nil
}

func (c *VirtualMachineController) updatePausedConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {

	// Update paused condition in case VMI was paused / unpaused
	if domain != nil && domain.Status.Status == api.Paused {
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			c.calculatePausedCondition(vmi, domain.Status.Reason)
		}
	} else if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		log.Log.Object(vmi).V(3).Info("Removing paused condition")
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstancePaused)
	}
}

func dumpTargetFile(vmiName, volName string) string {
	targetFileName := fmt.Sprintf("%s-%s-%s.memory.dump", vmiName, volName, time.Now().Format("20060102-150405"))
	return targetFileName
}

func (c *VirtualMachineController) updateMemoryDumpInfo(vmi *v1.VirtualMachineInstance, volumeStatus v1.VolumeStatus, domain *api.Domain) (v1.VolumeStatus, bool) {
	needsRefresh := false
	switch volumeStatus.Phase {
	case v1.HotplugVolumeMounted:
		needsRefresh = true
		log.Log.Object(vmi).V(3).Infof("Memory dump volume %s attached, marking it in progress", volumeStatus.Name)
		volumeStatus.Phase = v1.MemoryDumpVolumeInProgress
		volumeStatus.Message = fmt.Sprintf("Memory dump Volume %s is attached, getting memory dump", volumeStatus.Name)
		volumeStatus.Reason = VolumeMountedToPodReason
		volumeStatus.MemoryDumpVolume.TargetFileName = dumpTargetFile(vmi.Name, volumeStatus.Name)
	case v1.MemoryDumpVolumeInProgress:
		var memoryDumpMetadata *api.MemoryDumpMetadata
		if domain != nil {
			memoryDumpMetadata = domain.Spec.Metadata.KubeVirt.MemoryDump
		}
		if memoryDumpMetadata == nil || memoryDumpMetadata.FileName != volumeStatus.MemoryDumpVolume.TargetFileName {
			// memory dump wasnt triggered yet
			return volumeStatus, needsRefresh
		}
		needsRefresh = true
		if memoryDumpMetadata.StartTimestamp != nil {
			volumeStatus.MemoryDumpVolume.StartTimestamp = memoryDumpMetadata.StartTimestamp
		}
		if memoryDumpMetadata.EndTimestamp != nil && memoryDumpMetadata.Failed {
			log.Log.Object(vmi).Errorf("Memory dump to pvc %s failed: %v", volumeStatus.Name, memoryDumpMetadata.FailureReason)
			volumeStatus.Message = fmt.Sprintf("Memory dump to pvc %s failed: %v", volumeStatus.Name, memoryDumpMetadata.FailureReason)
			volumeStatus.Phase = v1.MemoryDumpVolumeFailed
			volumeStatus.MemoryDumpVolume.EndTimestamp = memoryDumpMetadata.EndTimestamp
		} else if memoryDumpMetadata.Completed {
			log.Log.Object(vmi).V(3).Infof("Marking memory dump to volume %s has completed", volumeStatus.Name)
			volumeStatus.Phase = v1.MemoryDumpVolumeCompleted
			volumeStatus.Message = fmt.Sprintf("Memory dump to Volume %s has completed successfully", volumeStatus.Name)
			volumeStatus.Reason = VolumeReadyReason
			volumeStatus.MemoryDumpVolume.EndTimestamp = memoryDumpMetadata.EndTimestamp
		}
	}

	return volumeStatus, needsRefresh
}

func (c *VirtualMachineController) updateFSFreezeStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil || domain.Status.FSFreezeStatus.Status == "" {
		return
	}

	if domain.Status.FSFreezeStatus.Status == api.FSThawed {
		vmi.Status.FSFreezeStatus = ""
	} else {
		vmi.Status.FSFreezeStatus = domain.Status.FSFreezeStatus.Status
	}

}

func IsoGuestVolumePath(namespace, name string, volume *v1.Volume) string {
	const basepath = "/var/run"
	switch {
	case volume.CloudInitNoCloud != nil:
		return filepath.Join(basepath, "kubevirt-ephemeral-disks", "cloud-init-data", namespace, name, "noCloud.iso")
	case volume.CloudInitConfigDrive != nil:
		return filepath.Join(basepath, "kubevirt-ephemeral-disks", "cloud-init-data", namespace, name, "configdrive.iso")
	case volume.ConfigMap != nil:
		return config.GetConfigMapDiskPath(volume.Name)
	case volume.DownwardAPI != nil:
		return config.GetDownwardAPIDiskPath(volume.Name)
	case volume.Secret != nil:
		return config.GetSecretDiskPath(volume.Name)
	case volume.ServiceAccount != nil:
		return config.GetServiceAccountDiskPath()
	case volume.Sysprep != nil:
		return config.GetSysprepDiskPath(volume.Name)
	default:
		return ""
	}
}

func (c *VirtualMachineController) updateIsoSizeStatus(vmi *v1.VirtualMachineInstance) {
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
		log.DefaultLogger().Warningf("failed to find pod UID for VMI %s", vmi.Name)
		return
	}

	volumes := make(map[string]v1.Volume)
	for _, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume
	}

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		volume, ok := volumes[disk.Name]
		if !ok {
			log.DefaultLogger().Warningf("No matching volume with name %s found", disk.Name)
			continue
		}

		volPath := IsoGuestVolumePath(vmi.Namespace, vmi.Name, &volume)
		if volPath == "" {
			continue
		}

		res, err := c.podIsolationDetector.Detect(vmi)
		if err != nil {
			log.DefaultLogger().Reason(err).Warningf("failed to detect VMI %s", vmi.Name)
			continue
		}

		rootPath, err := res.MountRoot()
		if err != nil {
			log.DefaultLogger().Reason(err).Warningf("failed to detect VMI %s", vmi.Name)
			continue
		}

		safeVolPath, err := rootPath.AppendAndResolveWithRelativeRoot(volPath)
		if err != nil {
			log.DefaultLogger().Warningf("failed to determine file size for volume %s", volPath)
			continue
		}
		fileInfo, err := safepath.StatAtNoFollow(safeVolPath)
		if err != nil {
			log.DefaultLogger().Warningf("failed to determine file size for volume %s", volPath)
			continue
		}

		for i := range vmi.Status.VolumeStatus {
			if vmi.Status.VolumeStatus[i].Name == volume.Name {
				vmi.Status.VolumeStatus[i].Size = fileInfo.Size()
				continue
			}
		}
	}
}

func (c *VirtualMachineController) updateSELinuxContext(vmi *v1.VirtualMachineInstance) error {
	_, present, err := selinux.NewSELinux()
	if err != nil {
		return err
	}
	if present {
		context, err := selinux.GetVirtLauncherContext(vmi)
		if err != nil {
			return err
		}
		vmi.Status.SelinuxContext = context
	} else {
		vmi.Status.SelinuxContext = "none"
	}

	return nil
}

func (c *VirtualMachineController) updateVMIStatusFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	c.updateIsoSizeStatus(vmi)
	err := c.updateSELinuxContext(vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("couldn't find the SELinux context for %s", vmi.Name)
	}
	c.setMigrationProgressStatus(vmi, domain)
	c.updateGuestInfoFromDomain(vmi, domain)
	c.updateVolumeStatusesFromDomain(vmi, domain)
	c.updateFSFreezeStatus(vmi, domain)
	c.updateMachineType(vmi, domain)
	if err = c.updateMemoryInfo(vmi, domain); err != nil {
		return err
	}
	err = c.netStat.UpdateStatus(vmi, domain)
	return err
}

func (c *VirtualMachineController) updateVMIConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) error {
	c.updateAccessCredentialConditions(vmi, domain, condManager)
	c.updateLiveMigrationConditions(vmi, condManager)
	err := c.updateGuestAgentConditions(vmi, domain, condManager)
	if err != nil {
		return err
	}
	c.updatePausedConditions(vmi, domain, condManager)

	return nil
}

func (c *VirtualMachineController) updateVMIStatus(oldStatus *v1.VirtualMachineInstanceStatus, vmi *v1.VirtualMachineInstance, domain *api.Domain, syncError error) (err error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()

	// Don't update the VirtualMachineInstance if it is already in a final state
	if vmi.IsFinal() {
		return nil
	} else if vmi.Status.NodeName != "" && vmi.Status.NodeName != c.host {
		// Only update the VMI's phase if this node owns the VMI.
		// not owned by this host, likely the result of a migration
		return nil
	} else if domainMigrated(domain) {
		return c.migrationSourceUpdateVMIStatus(vmi, domain)
	}

	// Update VMI status fields based on what is reported on the domain
	err = c.updateVMIStatusFromDomain(vmi, domain)
	if err != nil {
		return err
	}

	// Calculate the new VirtualMachineInstance state based on what libvirt reported
	err = c.setVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}

	// Update conditions on VMI Status
	err = c.updateVMIConditions(vmi, domain, condManager)
	if err != nil {
		return err
	}

	// Store containerdisks and kernelboot checksums
	if err := c.updateChecksumInfo(vmi, syncError); err != nil {
		return err
	}

	// Handle sync error
	handleSyncError(vmi, condManager, syncError)

	controller.SetVMIPhaseTransitionTimestamp(oldStatus, &vmi.Status)

	// Only issue vmi update if status has changed
	if !equality.Semantic.DeepEqual(*oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err = c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	// Record an event on the VMI when the VMI's phase changes
	if oldStatus.Phase != vmi.Status.Phase {
		c.recordPhaseChangeEvent(vmi)
	}

	return nil
}

func handleSyncError(vmi *v1.VirtualMachineInstance, condManager *controller.VirtualMachineInstanceConditionManager, syncError error) {
	var criticalNetErr *neterrors.CriticalNetworkError
	if goerror.As(syncError, &criticalNetErr) {
		log.Log.Errorf("virt-launcher crashed due to a network error. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	if _, ok := syncError.(*virtLauncherCriticalSecurebootError); ok {
		log.Log.Errorf("virt-launcher does not support the Secure Boot setting. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}

	if _, ok := syncError.(*vmiIrrecoverableError); ok {
		log.Log.Errorf("virt-launcher reached an irrecoverable error. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	condManager.CheckFailure(vmi, syncError, "Synchronizing with the Domain failed.")
}

func (c *VirtualMachineController) recordPhaseChangeEvent(vmi *v1.VirtualMachineInstance) {
	switch vmi.Status.Phase {
	case v1.Running:
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Started.String(), VMIStarted)
	case v1.Succeeded:
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Stopped.String(), VMIShutdown)
	case v1.Failed:
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Stopped.String(), VMICrashed)
	}
}

func (c *VirtualMachineController) calculatePausedCondition(vmi *v1.VirtualMachineInstance, reason api.StateChangeReason) {
	now := metav1.NewTime(time.Now())
	switch reason {
	case api.ReasonPausedMigration:
		if !isVMIPausedDuringMigration(vmi) || !c.isMigrationSource(vmi) {
			log.Log.Object(vmi).V(3).Infof("Domain is paused after migration by qemu, no condition needed")
			return
		}
		log.Log.Object(vmi).V(3).Info("Adding paused by migration monitor condition")
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedByMigrationMonitor",
			Message:            "VMI was paused by the migration monitor",
		})
	case api.ReasonPausedUser:
		log.Log.Object(vmi).V(3).Info("Adding paused condition")
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
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedIOError",
			Message:            "VMI was paused, low-level IO error detected",
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

func (c *VirtualMachineController) calculateLiveMigrationCondition(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstanceCondition, bool) {
	isBlockMigration, err := c.checkVolumesForMigration(vmi)
	if err != nil {
		return newNonMigratableCondition(err.Error(), v1.VirtualMachineInstanceReasonDisksNotMigratable), isBlockMigration
	}

	err = c.checkNetworkInterfacesForMigration(vmi)
	if err != nil {
		return newNonMigratableCondition(err.Error(), v1.VirtualMachineInstanceReasonInterfaceNotMigratable), isBlockMigration
	}

	if err := c.isHostModelMigratable(vmi); err != nil {
		return newNonMigratableCondition(err.Error(), v1.VirtualMachineInstanceReasonCPUModeNotMigratable), isBlockMigration
	}

	if vmiContainsPCIHostDevice(vmi) {
		return newNonMigratableCondition("VMI uses a PCI host devices", v1.VirtualMachineInstanceReasonHostDeviceNotMigratable), isBlockMigration
	}

	if util.IsSEVVMI(vmi) {
		return newNonMigratableCondition("VMI uses SEV", v1.VirtualMachineInstanceReasonSEVNotMigratable), isBlockMigration
	}

	if reservation.HasVMIPersistentReservation(vmi) {
		return newNonMigratableCondition("VMI uses SCSI persitent reservation", v1.VirtualMachineInstanceReasonPRNotMigratable), isBlockMigration
	}

	if tscRequirement := topology.GetTscFrequencyRequirement(vmi); !topology.AreTSCFrequencyTopologyHintsDefined(vmi) && tscRequirement.Type == topology.RequiredForMigration {
		return newNonMigratableCondition(tscRequirement.Reason, v1.VirtualMachineInstanceReasonNoTSCFrequencyMigratable), isBlockMigration
	}

	if vmiFeatures := vmi.Spec.Domain.Features; vmiFeatures != nil && vmiFeatures.HypervPassthrough != nil && *vmiFeatures.HypervPassthrough.Enabled {
		return newNonMigratableCondition("VMI uses hyperv passthrough", v1.VirtualMachineInstanceReasonHypervPassthroughNotMigratable), isBlockMigration
	}

	return &v1.VirtualMachineInstanceCondition{
		Type:   v1.VirtualMachineInstanceIsMigratable,
		Status: k8sv1.ConditionTrue,
	}, isBlockMigration
}

func vmiContainsPCIHostDevice(vmi *v1.VirtualMachineInstance) bool {
	return len(vmi.Spec.Domain.Devices.HostDevices) > 0 || len(vmi.Spec.Domain.Devices.GPUs) > 0
}

type multipleNonMigratableCondition struct {
	reasons []string
	msgs    []string
}

func newMultipleNonMigratableCondition() *multipleNonMigratableCondition {
	return &multipleNonMigratableCondition{}
}

func (cond *multipleNonMigratableCondition) addNonMigratableCondition(reason, msg string) {
	cond.reasons = append(cond.reasons, reason)
	cond.msgs = append(cond.msgs, msg)
}

func (cond *multipleNonMigratableCondition) String() string {
	var buffer bytes.Buffer
	for i, c := range cond.reasons {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(fmt.Sprintf("%s: %s", c, cond.msgs[i]))
	}
	return buffer.String()
}

func (cond *multipleNonMigratableCondition) generateStorageLiveMigrationCondition() *v1.VirtualMachineInstanceCondition {
	switch len(cond.reasons) {
	case 0:
		return &v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceIsStorageLiveMigratable,
			Status: k8sv1.ConditionTrue,
		}
	default:
		return &v1.VirtualMachineInstanceCondition{
			Type:    v1.VirtualMachineInstanceIsStorageLiveMigratable,
			Status:  k8sv1.ConditionFalse,
			Message: cond.String(),
			Reason:  v1.VirtualMachineInstanceReasonNotMigratable,
		}
	}
}

func (c *VirtualMachineController) calculateLiveStorageMigrationCondition(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceCondition {
	multiCond := newMultipleNonMigratableCondition()

	if err := c.checkNetworkInterfacesForMigration(vmi); err != nil {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonInterfaceNotMigratable, err.Error())
	}

	if err := c.isHostModelMigratable(vmi); err != nil {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonCPUModeNotMigratable, err.Error())
	}

	if vmiContainsPCIHostDevice(vmi) {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonHostDeviceNotMigratable, "VMI uses a PCI host devices")
	}

	if util.IsSEVVMI(vmi) {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonSEVNotMigratable, "VMI uses SEV")
	}

	if reservation.HasVMIPersistentReservation(vmi) {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonPRNotMigratable, "VMI uses SCSI persitent reservation")
	}

	if tscRequirement := topology.GetTscFrequencyRequirement(vmi); !topology.AreTSCFrequencyTopologyHintsDefined(vmi) && tscRequirement.Type == topology.RequiredForMigration {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonNoTSCFrequencyMigratable, tscRequirement.Reason)
	}

	if vmiFeatures := vmi.Spec.Domain.Features; vmiFeatures != nil && vmiFeatures.HypervPassthrough != nil && *vmiFeatures.HypervPassthrough.Enabled {
		multiCond.addNonMigratableCondition(v1.VirtualMachineInstanceReasonHypervPassthroughNotMigratable, "VMI uses hyperv passthrough")
	}

	return multiCond.generateStorageLiveMigrationCondition()
}

func (c *VirtualMachineController) Run(threadiness int, stopCh chan struct{}) {
	defer c.queue.ShutDown()
	log.Log.Info("Starting virt-handler controller.")

	go c.deviceManagerController.Run(stopCh)

	go c.downwardMetricsManager.Run(stopCh)

	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// queue keys for previous Domains on the host that no longer exist
	// in the cache. This ensures we perform local cleanup of deleted VMs.
	for _, domain := range c.domainStore.List() {
		d := domain.(*api.Domain)
		vmiRef := v1.NewVMIReferenceWithUUID(
			d.ObjectMeta.Namespace,
			d.ObjectMeta.Name,
			d.Spec.Metadata.KubeVirt.UID)

		key := controller.VirtualMachineInstanceKey(vmiRef)

		_, exists, _ := c.vmiSourceStore.GetByKey(key)
		if !exists {
			c.queue.Add(key)
		}
	}

	heartBeatDone := c.heartBeat.Run(c.heartBeatInterval, stopCh)

	c.multipathSocketMonitor.Run()

	go c.ioErrorRetryManager.Run(stopCh)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-heartBeatDone
	<-stopCh
	c.multipathSocketMonitor.Close()
	log.Log.Info("Stopping virt-handler controller.")
}

func (c *VirtualMachineController) runWorker() {
	for c.Execute() {
	}
}

func (c *VirtualMachineController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.execute(key); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VirtualMachineController) getVMIFromCache(key string) (vmi *v1.VirtualMachineInstance, exists bool, err error) {

	// Get it from the global store as during a migration
	// the VMI could disappear momentarily from both the source store
	// and the target store
	obj, exists, err := c.vmiStore.GetByKey(key)
	if err != nil {
		return nil, false, err
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

func (c *VirtualMachineController) getDomainFromCache(key string) (domain *api.Domain, exists bool, cachedUID types.UID, err error) {

	obj, exists, err := c.domainStore.GetByKey(key)

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

func (c *VirtualMachineController) migrationOrphanedSourceNodeExecute(vmi *v1.VirtualMachineInstance, domainExists bool) error {

	if domainExists {
		err := c.processVmDelete(vmi)
		if err != nil {
			return err
		}
		// we can perform the cleanup immediately after
		// the successful delete here because we don't have
		// to report the deletion results on the VMI status
		// in this case.
		err = c.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	} else {
		err := c.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *VirtualMachineController) migrationTargetExecute(vmi *v1.VirtualMachineInstance, vmiExists bool, domain *api.Domain) error {
	oldStatus := vmi.DeepCopy().Status

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
	} else if c.hasStaleClientConnections(vmi) {
		// if stale client exists, force cleanup.
		// This can happen as a result of a previously
		// failed attempt to migrate the vmi to this node.
		shouldCleanUp = true
	}

	domainExists := domain != nil
	if shouldAbort {
		if domainExists {
			err := c.processVmDelete(vmi)
			if err != nil {
				return err
			}
		}

		err := c.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	} else if shouldCleanUp {
		log.Log.Object(vmi).Infof("Stale client for migration target found. Cleaning up.")

		err := c.processVmCleanup(vmi)
		if err != nil {
			return err
		}

		// if we're still the migration target, we need to keep trying until the migration fails.
		// it's possible we're simply waiting for another target pod to come online.
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)

	} else if shouldUpdate {
		log.Log.Object(vmi).Info("Processing vmi migration target update")

		// prepare the POD for the migration
		err := c.processVmUpdate(vmi, domain)
		if err != nil {
			return err
		}

		err = c.migrationTargetUpdateVMIStatus(vmi, domain)
		if err != nil {
			return err
		}
	}

	// update the VMI if necessary
	if !equality.Semantic.DeepEqual(oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	return nil
}

// Determine if gracefulShutdown has been triggered by virt-launcher
func (c *VirtualMachineController) hasGracefulShutdownTrigger(domain *api.Domain) bool {
	if domain == nil {
		return false
	}
	gracePeriod := domain.Spec.Metadata.KubeVirt.GracePeriod

	return gracePeriod != nil &&
		gracePeriod.MarkedForGracefulShutdown != nil &&
		*gracePeriod.MarkedForGracefulShutdown
}

func (c *VirtualMachineController) defaultExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	oldStatus := vmi.DeepCopy().Status

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
	// set to true when unrecoverable domain needs to be destroyed non-gracefully.
	forceShutdownIrrecoverable := false

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
	forceShutdownIrrecoverable = domainExists && domainPausedFailedPostCopy(domain)

	gracefulShutdown := c.hasGracefulShutdownTrigger(domain)
	if gracefulShutdown && vmi.IsRunning() {
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
			shouldCleanUp = true
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
		phase, err := c.calculateVmPhaseForStatusReason(domain, vmi)
		if err != nil {
			return err
		}
		if vmi.Status.Phase == phase {
			shouldUpdate = true
		}

		if shouldDelay, delay := c.ioErrorRetryManager.ShouldDelay(string(vmi.UID), func() bool {
			return isIOError(shouldUpdate, domainExists, domain)
		}); shouldDelay {
			shouldUpdate = false
			log.Log.Object(vmi).Infof("Delay vm update for %f seconds", delay.Seconds())
			c.queue.AddAfter(key, delay)
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
			vmi.Status.NodeName == c.host {

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
		syncErr = c.processVmShutdown(vmi, domain)
	case forceShutdownIrrecoverable:
		msg := formatIrrecoverableErrorMessage(domain)
		log.Log.Object(vmi).V(3).Infof("Processing a destruction of an irrecoverable domain - %s.", msg)
		syncErr = c.processVmDestroy(vmi, domain)
		if syncErr == nil {
			syncErr = &vmiIrrecoverableError{msg}
		}
	case shouldDelete:
		log.Log.Object(vmi).V(3).Info("Processing deletion.")
		syncErr = c.processVmDelete(vmi)
	case shouldCleanUp:
		log.Log.Object(vmi).V(3).Info("Processing local ephemeral data cleanup for shutdown domain.")
		syncErr = c.processVmCleanup(vmi)
	case shouldUpdate:
		log.Log.Object(vmi).V(3).Info("Processing vmi update")
		log.Log.Object(vmi).Error("Processing vmi update")
		syncErr = c.processVmUpdate(vmi, domain)
	default:
		log.Log.Object(vmi).V(3).Info("No update processing required")
	}

	if syncErr != nil && !vmi.IsFinal() {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())

		// `syncErr` will be propagated anyway, and it will be logged in `re-enqueueing`
		// so there is no need to log it twice in hot path without increased verbosity.
		log.Log.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	// Update the VirtualMachineInstance status, if the VirtualMachineInstance exists
	if vmiExists {
		if err := c.updateVMIStatus(&oldStatus, vmi, domain, syncErr); err != nil {
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

func (c *VirtualMachineController) execute(key string) error {
	origVMI, vmiExists, err := c.getVMIFromCache(key)
	if err != nil {
		return err
	}

	vmi := origVMI.DeepCopy()

	if !vmiExists {
		c.vmiExpectations.DeleteExpectations(key)
	} else if !c.vmiExpectations.SatisfiedExpectations(key) {
		return nil
	}

	domain, domainExists, domainCachedUID, err := c.getDomainFromCache(key)
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
		uid := virtcache.GhostRecordGlobalStore.LastKnownUID(key)
		if uid != "" {
			log.Log.Object(vmi).V(3).Infof("ghost record cache provided %s as UID", uid)
			vmi.UID = uid
		}
	}

	if vmiExists && domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		oldVMI := v1.NewVMIReferenceFromNameWithNS(vmi.Namespace, vmi.Name)
		oldVMI.UID = domain.Spec.Metadata.KubeVirt.UID
		expired, initialized, err := c.isLauncherClientUnresponsive(oldVMI)
		if err != nil {
			return err
		}
		// If we found an outdated domain which is also not alive anymore, clean up
		if !initialized {
			c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
			return nil
		} else if expired {
			log.Log.Object(oldVMI).Infof("Detected stale vmi %s that still needs cleanup before new vmi %s with identical name/namespace can be processed", oldVMI.UID, vmi.UID)
			err = c.processVmCleanup(oldVMI)
			if err != nil {
				return err
			}
			// Make sure we re-enqueue the key to ensure this new VMI is processed
			// after the stale domain is removed
			c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*5)
		}

		return nil
	}

	// Take different execution paths depending on the state of the migration and the
	// node this is executed on.

	if vmiExists && c.isPreMigrationTarget(vmi) {
		// 1. PRE-MIGRATION TARGET PREPARATION PATH
		//
		// If this node is the target of the vmi's migration, take
		// a different execute path. The target execute path prepares
		// the local environment for the migration, but does not
		// start the VMI
		return c.migrationTargetExecute(vmi, vmiExists, domain)
	} else if vmiExists && c.isOrphanedMigrationSource(vmi) {
		// 3. POST-MIGRATION SOURCE CLEANUP
		//
		// After a migration, the migrated domain still exists in the old
		// source's domain cache. Ensure that any node that isn't currently
		// the target or owner of the VMI handles deleting the domain locally.
		return c.migrationOrphanedSourceNodeExecute(vmi, domainExists)
	}
	return c.defaultExecute(key,
		vmi,
		vmiExists,
		domain,
		domainExists)

}

func (c *VirtualMachineController) processVmCleanup(vmi *v1.VirtualMachineInstance) error {

	vmiId := string(vmi.UID)

	log.Log.Object(vmi).Infof("Performing final local cleanup for vmi with uid %s", vmiId)

	c.migrationProxy.StopTargetListener(vmiId)
	c.migrationProxy.StopSourceListener(vmiId)

	c.downwardMetricsManager.StopServer(vmi)

	// Unmount container disks and clean up remaining files
	if err := c.containerDiskMounter.Unmount(vmi); err != nil {
		return err
	}

	// UnmountAll does the cleanup on the "best effort" basis: it is
	// safe to pass a nil cgroupManager.
	cgroupManager, _ := getCgroupManager(vmi, c.host)
	if err := c.hotplugVolumeMounter.UnmountAll(vmi, cgroupManager); err != nil {
		return err
	}

	c.teardownNetwork(vmi)

	c.sriovHotplugExecutorPool.Delete(vmi.UID)

	// Watch dog file and command client must be the last things removed here
	if err := c.closeLauncherClient(vmi); err != nil {
		return err
	}

	// Remove the domain from cache in the event that we're performing
	// a final cleanup and never received the "DELETE" event. This is
	// possible if the VMI pod goes away before we receive the final domain
	// "DELETE"
	domain := api.NewDomainReferenceFromName(vmi.Namespace, vmi.Name)
	log.Log.Object(domain).Infof("Removing domain from cache during final cleanup")
	return c.domainStore.Delete(domain)
}

func (c *VirtualMachineController) closeLauncherClient(vmi *v1.VirtualMachineInstance) error {

	// UID is required in order to close socket
	if string(vmi.GetUID()) == "" {
		return nil
	}

	clientInfo, exists := c.launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		clientInfo.Client.Close()
		close(clientInfo.DomainPipeStopChan)
	}

	err := virtcache.GhostRecordGlobalStore.Delete(vmi.Namespace, vmi.Name)
	if err != nil {
		return err
	}

	c.launcherClients.Delete(vmi.UID)
	return nil
}

// used by unit tests to add mock clients
func (c *VirtualMachineController) addLauncherClient(vmUID types.UID, info *virtcache.LauncherClientInfo) error {
	c.launcherClients.Store(vmUID, info)
	return nil
}

func (c *VirtualMachineController) isLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error) {
	var socketFile string

	clientInfo, exists := c.launcherClients.Load(vmi.UID)
	if exists {
		if clientInfo.Ready {
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
				if c.hotplugVolumesReady(vmi) && clientInfo.NotInitializedSince.Before(time.Now().Add(-3*time.Minute)) {
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
		c.launcherClients.Store(vmi.UID, clientInfo)
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
	return cmdclient.IsSocketUnresponsive(socketFile), true, nil
}

func (c *VirtualMachineController) getLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	var err error

	clientInfo, exists := c.launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		return clientInfo.Client, nil
	}

	socketFile, err := cmdclient.FindSocketOnHost(vmi)
	if err != nil {
		return nil, err
	}

	err = virtcache.GhostRecordGlobalStore.Add(vmi.Namespace, vmi.Name, socketFile, vmi.UID)
	if err != nil {
		return nil, err
	}

	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, err
	}

	domainPipeStopChan := make(chan struct{})
	//we pipe in the domain socket into the VMI's filesystem
	err = c.startDomainNotifyPipe(domainPipeStopChan, vmi)
	if err != nil {
		client.Close()
		close(domainPipeStopChan)
		return nil, err
	}

	c.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
		Client:              client,
		SocketFile:          socketFile,
		DomainPipeStopChan:  domainPipeStopChan,
		NotInitializedSince: time.Now(),
		Ready:               true,
	})

	return client, nil
}

func (c *VirtualMachineController) processVmDestroy(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	tryGracefully := false
	return c.helperVmShutdown(vmi, domain, tryGracefully)
}

func (c *VirtualMachineController) processVmShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	tryGracefully := true
	return c.helperVmShutdown(vmi, domain, tryGracefully)
}

func (c *VirtualMachineController) helperVmShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain, tryGracefully bool) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := c.getVerifiedLauncherClient(vmi)
	if err != nil {
		return err
	}

	if domainHasGracePeriod(domain) && tryGracefully {
		if expired, timeLeft := c.hasGracePeriodExpired(domain); !expired {
			return c.handleVMIShutdown(vmi, domain, client, timeLeft)
		}
		log.Log.Object(vmi).Infof("Grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
	} else {
		log.Log.Object(vmi).Infof("Graceful shutdown not set, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
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

	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), VMIStopping)

	return nil
}

func (c *VirtualMachineController) handleVMIShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain, client cmdclient.LauncherClient, timeLeft int64) error {
	if domain.Status.Status != api.Shutdown {
		return c.shutdownVMI(vmi, client, timeLeft)
	}
	log.Log.V(4).Object(vmi).Infof("%s is already shutting down.", vmi.GetObjectMeta().GetName())
	return nil
}

func (c *VirtualMachineController) shutdownVMI(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient, timeLeft int64) error {
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
	c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Duration(timeLeft)*time.Second)
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.ShuttingDown.String(), VMIGracefulShutdown)
	return nil
}

func (c *VirtualMachineController) processVmDelete(vmi *v1.VirtualMachineInstance) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := c.getVerifiedLauncherClient(vmi)

	// If the pod has been torn down, we know the VirtualMachineInstance is down.
	if err == nil {

		log.Log.Object(vmi).Infof("Signaled deletion for %s", vmi.GetObjectMeta().GetName())

		// pending deletion.
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), VMISignalDeletion)

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

func (c *VirtualMachineController) hasStaleClientConnections(vmi *v1.VirtualMachineInstance) bool {
	_, err := c.getVerifiedLauncherClient(vmi)
	if err == nil {
		// current client connection is good.
		return false
	}

	// no connection, but ghost file exists.
	if virtcache.GhostRecordGlobalStore.Exists(vmi.Namespace, vmi.Name) {
		return true
	}

	return false

}

func (c *VirtualMachineController) getVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error) {
	client, err = c.getLauncherClient(vmi)
	if err != nil {
		return
	}

	// Verify connectivity.
	// It's possible the pod has already been torn down along with the VirtualMachineInstance.
	err = client.Ping()
	return
}

func (c *VirtualMachineController) isOrphanedMigrationSource(vmi *v1.VirtualMachineInstance) bool {
	nodeName, ok := vmi.Labels[v1.NodeNameLabel]

	if ok && nodeName != "" && nodeName != c.host {
		return true
	}

	return false
}

func (c *VirtualMachineController) isPreMigrationTarget(vmi *v1.VirtualMachineInstance) bool {

	migrationTargetNodeName, ok := vmi.Labels[v1.MigrationTargetNodeNameLabel]

	if ok &&
		migrationTargetNodeName != "" &&
		migrationTargetNodeName != vmi.Status.NodeName &&
		migrationTargetNodeName == c.host {
		return true
	}

	return false
}

func (c *VirtualMachineController) checkNetworkInterfacesForMigration(vmi *v1.VirtualMachineInstance) error {
	return netvmispec.VerifyVMIMigratable(vmi, c.clusterConfig.GetNetworkBindings())
}

func (c *VirtualMachineController) checkVolumesForMigration(vmi *v1.VirtualMachineInstance) (blockMigrate bool, err error) {
	volumeStatusMap := make(map[string]v1.VolumeStatus)

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	if len(vmi.Status.MigratedVolumes) > 0 {
		blockMigrate = true
	}

	filesystems := storagetypes.GetFilesystemsFromVolumes(vmi)

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
			} else if !pvctypes.HasSharedAccessMode(volumeStatus.PersistentVolumeClaimInfo.AccessModes) && !pvctypes.IsMigratedVolume(volumeStatus.Name, vmi) {
				return true, fmt.Errorf("cannot migrate VMI: PVC %v is not shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)", claimName)
			}

		} else if volSrc.HostDisk != nil {
			shared := volSrc.HostDisk.Shared != nil && *volSrc.HostDisk.Shared
			if !shared {
				return true, fmt.Errorf("cannot migrate VMI with non-shared HostDisk")
			}
		} else {
			if _, ok := filesystems[volume.Name]; ok {
				log.Log.Object(vmi).Infof("Volume %s is shared with virtiofs, allow live migration", volume.Name)
				continue
			}

			isVolumeUsedByReadOnlyDisk := false
			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				if isReadOnlyDisk(&disk) && disk.Name == volume.Name {
					isVolumeUsedByReadOnlyDisk = true
					break
				}
			}

			if isVolumeUsedByReadOnlyDisk {
				continue
			}

			if vmi.Status.MigrationMethod == "" || vmi.Status.MigrationMethod == v1.LiveMigration {
				log.Log.Object(vmi).Infof("migration is block migration because of %s volume", volume.Name)
			}
			blockMigrate = true
		}
	}
	return
}

func isVMIPausedDuringMigration(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.Mode == v1.MigrationPaused &&
		!vmi.Status.MigrationState.Completed
}

func (c *VirtualMachineController) isMigrationSource(vmi *v1.VirtualMachineInstance) bool {

	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceNode == c.host &&
		vmi.Status.MigrationState.TargetNodeAddress != "" &&
		!vmi.Status.MigrationState.Completed {

		return true
	}
	return false

}

func (c *VirtualMachineController) handleTargetMigrationProxy(vmi *v1.VirtualMachineInstance) error {
	// handle starting/stopping target migration proxy
	var migrationTargetSockets []string
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	// Get the libvirt connection socket file on the destination pod.
	socketFile := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "libvirt/virtqemud-sock"), res.Pid())
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	migrationTargetSockets = append(migrationTargetSockets, socketFile)

	migrationPortsRange := migrationproxy.GetMigrationPortsList(vmi.IsBlockMigration())
	for _, port := range migrationPortsRange {
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), port)
		// a proxy between the target direct qemu channel and the connector in the destination pod
		destSocketFile := migrationproxy.SourceUnixFile(baseDir, key)
		migrationTargetSockets = append(migrationTargetSockets, destSocketFile)
	}
	err = c.migrationProxy.StartTargetListener(string(vmi.UID), migrationTargetSockets)
	if err != nil {
		return err
	}
	return nil
}

func (c *VirtualMachineController) handlePostMigrationProxyCleanup(vmi *v1.VirtualMachineInstance) {
	if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.Completed || vmi.Status.MigrationState.Failed {
		c.migrationProxy.StopTargetListener(string(vmi.UID))
		c.migrationProxy.StopSourceListener(string(vmi.UID))
	}
}

func (c *VirtualMachineController) handleSourceMigrationProxy(vmi *v1.VirtualMachineInstance) error {

	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	c.migrationProxy.StopTargetListener(string(vmi.UID))
	if vmi.Status.MigrationState.TargetDirectMigrationNodePorts == nil {
		msg := "No migration proxy has been created for this vmi"
		return fmt.Errorf("%s", msg)
	}
	err = c.migrationProxy.StartSourceListener(
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

func (c *VirtualMachineController) getLauncherClientInfo(vmi *v1.VirtualMachineInstance) *virtcache.LauncherClientInfo {
	launcherInfo, exists := c.launcherClients.Load(vmi.UID)
	if !exists {
		return nil
	}
	return launcherInfo
}

func isMigrationInProgress(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	var domainMigrationMetadata *api.MigrationMetadata

	if domain == nil ||
		vmi.Status.MigrationState == nil ||
		domain.Spec.Metadata.KubeVirt.Migration == nil {
		return false
	}
	domainMigrationMetadata = domain.Spec.Metadata.KubeVirt.Migration

	if vmi.Status.MigrationState.MigrationUID == domainMigrationMetadata.UID &&
		domainMigrationMetadata.StartTimestamp != nil {
		return true
	}
	return false
}

func (c *VirtualMachineController) vmUpdateHelperMigrationSource(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	client, err := c.getLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
	}

	if vmi.Status.MigrationState.AbortRequested {
		err = c.handleMigrationAbort(vmi, client)
		if err != nil {
			return err
		}
	} else {
		if isMigrationInProgress(vmi, domain) {
			// we already started this migration, no need to rerun this
			log.DefaultLogger().Errorf("migration %s has already been started", vmi.Status.MigrationState.MigrationUID)
			return nil
		}

		err = c.handleSourceMigrationProxy(vmi)
		if err != nil {
			return fmt.Errorf("failed to handle migration proxy: %v", err)
		}

		migrationConfiguration := vmi.Status.MigrationState.MigrationConfiguration
		if migrationConfiguration == nil {
			migrationConfiguration = c.clusterConfig.GetMigrationConfiguration()
		}

		options := &cmdclient.MigrationOptions{
			Bandwidth:               *migrationConfiguration.BandwidthPerMigration,
			ProgressTimeout:         *migrationConfiguration.ProgressTimeout,
			CompletionTimeoutPerGiB: *migrationConfiguration.CompletionTimeoutPerGiB,
			UnsafeMigration:         *migrationConfiguration.UnsafeMigrationOverride,
			AllowAutoConverge:       *migrationConfiguration.AllowAutoConverge,
			AllowPostCopy:           *migrationConfiguration.AllowPostCopy,
			AllowWorkloadDisruption: *migrationConfiguration.AllowWorkloadDisruption,
		}

		configureParallelMigrationThreads(options, vmi)

		marshalledOptions, err := json.Marshal(options)
		if err != nil {
			log.Log.Object(vmi).Warning("failed to marshall matched migration options")
		} else {
			log.Log.Object(vmi).Infof("migration options matched for vmi %s: %s", vmi.Name, string(marshalledOptions))
		}

		vmiCopy := vmi.DeepCopy()
		err = hostdisk.ReplacePVCByHostDisk(vmiCopy)
		if err != nil {
			return err
		}

		err = client.MigrateVirtualMachine(vmiCopy, options)
		if err != nil {
			return err
		}
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), VMIMigrating)
	}
	return nil
}

func replaceMigratedVolumesStatus(vmi *v1.VirtualMachineInstance) {
	replaceVolsStatus := make(map[string]*v1.PersistentVolumeClaimInfo)
	for _, v := range vmi.Status.MigratedVolumes {
		replaceVolsStatus[v.SourcePVCInfo.ClaimName] = v.DestinationPVCInfo
	}
	for i, v := range vmi.Status.VolumeStatus {
		if v.PersistentVolumeClaimInfo == nil {
			continue
		}
		if status, ok := replaceVolsStatus[v.PersistentVolumeClaimInfo.ClaimName]; ok {
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo = status
		}
	}

}

func (c *VirtualMachineController) vmUpdateHelperMigrationTarget(origVMI *v1.VirtualMachineInstance) error {

	client, err := c.getLauncherClient(origVMI)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
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
		// Re-enqueue to trigger handler final cleanup
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second)
		return nil
	} else if migrations.IsMigrating(vmi) {
		// If the migration has already started,
		// then there's nothing left to prepare on the target side
		return nil
	}
	// The VolumeStatus is used to retrive additional information for the volume handling.
	// For example, for filesystem PVC, the information are used to create a right size image.
	// In the case of migrated volumes, we need to replace the original volume information with the
	// destination volume properties.
	replaceMigratedVolumesStatus(vmi)
	err = hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	// give containerDisks some time to become ready before throwing errors on retries
	info := c.getLauncherClientInfo(vmi)
	if ready, err := c.containerDiskMounter.ContainerDisksReady(vmi, info.NotInitializedSince); !ready {
		if err != nil {
			return err
		}
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	}

	// Mount container disks
	err = c.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return err
	}

	// Mount hotplug disks
	if attachmentPodUID := vmi.Status.MigrationState.TargetAttachmentPodUID; attachmentPodUID != types.UID("") {
		cgroupManager, err := getCgroupManager(vmi, c.host)
		if err != nil {
			return err
		}
		if err := c.hotplugVolumeMounter.MountFromPod(vmi, attachmentPodUID, cgroupManager); err != nil {
			return fmt.Errorf("failed to mount hotplug volumes: %v", err)
		}
	}

	isolationRes, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf(failedDetectIsolationFmt, err)
	}

	if err := c.netConf.Setup(vmi, netsetup.FilterNetsForMigrationTarget(vmi), isolationRes.Pid()); err != nil {
		return fmt.Errorf("failed to configure vmi network for migration target: %w", err)
	}

	virtLauncherRootMount, err := isolationRes.MountRoot()
	if err != nil {
		return err
	}

	err = c.claimDeviceOwnership(virtLauncherRootMount, "kvm")
	if err != nil {
		return fmt.Errorf("failed to set up file ownership for /dev/kvm: %v", err)
	}
	if virtutil.IsAutoAttachVSOCK(vmi) {
		if err := c.claimDeviceOwnership(virtLauncherRootMount, "vhost-vsock"); err != nil {
			return fmt.Errorf("failed to set up file ownership for /dev/vhost-vsock: %v", err)
		}
	}

	lessPVCSpaceToleration := c.clusterConfig.GetLessPVCSpaceToleration()
	minimumPVCReserveBytes := c.clusterConfig.GetMinimumReservePVCBytes()

	// initialize disks images for empty PVC
	hostDiskCreator := hostdisk.NewHostDiskCreator(c.recorder, lessPVCSpaceToleration, minimumPVCReserveBytes, virtLauncherRootMount)
	err = hostDiskCreator.Create(vmi)
	if err != nil {
		return fmt.Errorf("preparing host-disks failed: %v", err)
	}

	if virtutil.IsNonRootVMI(vmi) {
		if err := c.nonRootSetup(vmi); err != nil {
			return err
		}
	}

	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)
	options.InterfaceDomainAttachment = domainspec.DomainAttachmentByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())

	if err := client.SyncMigrationTarget(vmi, options); err != nil {
		return fmt.Errorf("syncing migration target failed: %v", err)
	}
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), VMIMigrationTargetPrepared)

	err = c.handleTargetMigrationProxy(vmi)
	if err != nil {
		return fmt.Errorf("failed to handle post sync migration proxy: %v", err)
	}
	return nil
}

func (c *VirtualMachineController) affinePitThread(vmi *v1.VirtualMachineInstance) error {
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	var Mask unix.CPUSet
	Mask.Zero()
	qemuprocess, err := res.GetQEMUProcess()
	if err != nil {
		return err
	}
	qemupid := qemuprocess.Pid()
	if qemupid == -1 {
		return nil
	}

	pitpid, err := res.KvmPitPid()
	if err != nil {
		return err
	}
	if pitpid == -1 {
		return nil
	}
	if vmi.IsRealtimeEnabled() {
		param := schedParam{priority: 2}
		err = schedSetScheduler(pitpid, schedFIFO, param)
		if err != nil {
			return fmt.Errorf("failed to set FIFO scheduling and priority 2 for thread %d: %w", pitpid, err)
		}
	}
	vcpus, err := getVCPUThreadIDs(qemupid)
	if err != nil {
		return err
	}
	vpid, ok := vcpus["0"]
	if ok == false {
		return nil
	}
	vcpupid, err := strconv.Atoi(vpid)
	if err != nil {
		return err
	}
	err = unix.SchedGetaffinity(vcpupid, &Mask)
	if err != nil {
		return err
	}
	return unix.SchedSetaffinity(pitpid, &Mask)
}

func (c *VirtualMachineController) configureHousekeepingCgroup(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error {
	if err := cgroupManager.CreateChildCgroup("housekeeping", "cpuset"); err != nil {
		log.Log.Reason(err).Error("CreateChildCgroup ")
		return err
	}

	key := controller.VirtualMachineInstanceKey(vmi)
	domain, domainExists, _, err := c.getDomainFromCache(key)
	if err != nil {
		return err
	}
	// bail out if domain does not exist
	if domainExists == false {
		return nil
	}

	if domain.Spec.CPUTune == nil || domain.Spec.CPUTune.EmulatorPin == nil {
		return nil
	}

	hkcpus, err := hardware.ParseCPUSetLine(domain.Spec.CPUTune.EmulatorPin.CPUSet, 100)
	if err != nil {
		return err
	}

	log.Log.V(3).Object(vmi).Infof("housekeeping cpu: %v", hkcpus)

	err = cgroupManager.SetCpuSet("housekeeping", hkcpus)
	if err != nil {
		return err
	}

	tids, err := cgroupManager.GetCgroupThreads()
	if err != nil {
		return err
	}
	hktids := make([]int, 0, 10)

	for _, tid := range tids {
		proc, err := ps.FindProcess(tid)
		if err != nil {
			log.Log.Object(vmi).Errorf("Failure to find process: %s", err.Error())
			return err
		}
		if proc == nil {
			return fmt.Errorf("failed to find process with tid: %d", tid)
		}
		comm := proc.Executable()
		if strings.Contains(comm, "CPU ") && strings.Contains(comm, "KVM") {
			continue
		}
		hktids = append(hktids, tid)
	}

	log.Log.V(3).Object(vmi).Infof("hk thread ids: %v", hktids)
	for _, tid := range hktids {
		err = cgroupManager.AttachTID("cpuset", "housekeeping", tid)
		if err != nil {
			log.Log.Object(vmi).Errorf("Error attaching tid %d: %v", tid, err.Error())
			return err
		}
	}

	return nil
}

func (c *VirtualMachineController) vmUpdateHelperDefault(origVMI *v1.VirtualMachineInstance, domainExists bool) error {
	client, err := c.getLauncherClient(origVMI)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
	}

	vmi := origVMI.DeepCopy()
	preallocatedVolumes := c.getPreallocatedVolumes(vmi)

	err = hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	cgroupManager, err := getCgroupManager(vmi, c.host)
	if err != nil {
		return err
	}

	var errorTolerantFeaturesError []error
	readyToProceed, err := c.handleVMIState(vmi, cgroupManager, &errorTolerantFeaturesError)
	if err != nil {
		return err
	}

	if !readyToProceed {
		return nil
	}

	// Synchronize the VirtualMachineInstance state
	err = c.syncVirtualMachine(client, vmi, preallocatedVolumes)
	if err != nil {
		return err
	}

	// Post-sync housekeeping
	err = c.handleHousekeeping(vmi, cgroupManager, domainExists)
	if err != nil {
		return err
	}

	return errors.NewAggregate(errorTolerantFeaturesError)
}

// handleVMIState: Decides whether to call handleRunningVMI or handleStartingVMI based on the VMI's state.
func (c *VirtualMachineController) handleVMIState(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, errorTolerantFeaturesError *[]error) (bool, error) {
	if vmi.IsRunning() {
		return true, c.handleRunningVMI(vmi, cgroupManager, errorTolerantFeaturesError)
	} else if !vmi.IsFinal() {
		return c.handleStartingVMI(vmi, cgroupManager)
	}
	return true, nil
}

// handleRunningVMI contains the logic specifically for running VMs (hotplugging in running state, metrics, network updates)
func (c *VirtualMachineController) handleRunningVMI(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, errorTolerantFeaturesError *[]error) error {
	if wasMigrationSuccessful(vmi.Status.MigrationState) && !vmi.Status.MigrationState.Completed {
		c.finalizeMigration(vmi)
	}

	if err := c.hotplugSriovInterfaces(vmi); err != nil {
		log.Log.Object(vmi).Error(err.Error())
	}

	if err := c.hotplugVolumeMounter.Mount(vmi, cgroupManager); err != nil {
		if !goerror.Is(err, os.ErrNotExist) {
			return err
		}
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, "HotplugFailed", err.Error())
	}

	if err := c.getMemoryDump(vmi); err != nil {
		return err
	}

	isolationRes, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf(failedDetectIsolationFmt, err)
	}

	if err := c.downwardMetricsManager.StartServer(vmi, isolationRes.Pid()); err != nil {
		return err
	}

	if err := c.netConf.Setup(vmi, netsetup.FilterNetsForLiveUpdate(vmi), isolationRes.Pid()); err != nil {
		log.Log.Object(vmi).Error(err.Error())
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, "NicHotplug", err.Error())
		*errorTolerantFeaturesError = append(*errorTolerantFeaturesError, err)
	}

	return nil
}

// handleStartingVMI: Contains the logic for starting VMs (container disks, initial network setup, device ownership).
func (c *VirtualMachineController) handleStartingVMI(
	vmi *v1.VirtualMachineInstance,
	cgroupManager cgroup.Manager,
) (bool, error) {
	// give containerDisks some time to become ready before throwing errors on retries
	info := c.getLauncherClientInfo(vmi)
	if ready, err := c.containerDiskMounter.ContainerDisksReady(vmi, info.NotInitializedSince); !ready {
		if err != nil {
			return false, err
		}
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return false, nil
	}

	var err error
	err = c.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return false, err
	}

	if err := c.hotplugVolumeMounter.Mount(vmi, cgroupManager); err != nil {
		if !goerror.Is(err, os.ErrNotExist) {
			return false, err
		}
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, "HotplugFailed", err.Error())
	}

	if !c.hotplugVolumesReady(vmi) {
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return false, nil
	}

	isolationRes, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return false, fmt.Errorf(failedDetectIsolationFmt, err)
	}

	if err := c.netConf.Setup(vmi, netsetup.FilterNetsForVMStartup(vmi), isolationRes.Pid()); err != nil {
		return false, fmt.Errorf("failed to configure vmi network: %w", err)
	}

	if err := c.setupDevicesOwnerships(vmi, isolationRes); err != nil {
		return false, err
	}

	if err := c.adjustResources(vmi); err != nil {
		return false, err
	}

	if c.shouldWaitForSEVAttestation(vmi) {
		return false, nil
	}

	return true, nil
}

func (c *VirtualMachineController) adjustResources(vmi *v1.VirtualMachineInstance) error {
	err := c.podIsolationDetector.AdjustResources(vmi, c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio)
	if err != nil {
		return fmt.Errorf("failed to adjust resources: %v", err)
	}
	return nil
}

func (c *VirtualMachineController) shouldWaitForSEVAttestation(vmi *v1.VirtualMachineInstance) bool {
	if util.IsSEVAttestationRequested(vmi) {
		sev := vmi.Spec.Domain.LaunchSecurity.SEV
		// Wait for the session parameters to be provided
		return sev.Session == "" || sev.DHCert == ""
	}
	return false
}

func (c *VirtualMachineController) setupDevicesOwnerships(vmi *v1.VirtualMachineInstance, isolationRes isolation.IsolationResult) error {
	virtLauncherRootMount, err := isolationRes.MountRoot()
	if err != nil {
		return err
	}

	err = c.claimDeviceOwnership(virtLauncherRootMount, "kvm")
	if err != nil {
		return fmt.Errorf("failed to set up file ownership for /dev/kvm: %v", err)
	}

	if virtutil.IsAutoAttachVSOCK(vmi) {
		if err := c.claimDeviceOwnership(virtLauncherRootMount, "vhost-vsock"); err != nil {
			return fmt.Errorf("failed to set up file ownership for /dev/vhost-vsock: %v", err)
		}
	}

	if err := c.configureHostDisks(vmi, isolationRes, virtLauncherRootMount); err != nil {
		return err
	}

	if err := c.configureSEVDeviceOwnership(vmi, isolationRes, virtLauncherRootMount); err != nil {
		return err
	}

	if virtutil.IsNonRootVMI(vmi) {
		if err := c.nonRootSetup(vmi); err != nil {
			return err
		}
	}

	if err := c.configureVirtioFS(vmi, isolationRes); err != nil {
		return err
	}

	return nil
}

func (c *VirtualMachineController) configureHostDisks(vmi *v1.VirtualMachineInstance, isolationRes isolation.IsolationResult, virtLauncherRootMount *safepath.Path) error {
	lessPVCSpaceToleration := c.clusterConfig.GetLessPVCSpaceToleration()
	minimumPVCReserveBytes := c.clusterConfig.GetMinimumReservePVCBytes()

	hostDiskCreator := hostdisk.NewHostDiskCreator(c.recorder, lessPVCSpaceToleration, minimumPVCReserveBytes, virtLauncherRootMount)
	if err := hostDiskCreator.Create(vmi); err != nil {
		return fmt.Errorf("preparing host-disks failed: %v", err)
	}
	return nil
}

func (c *VirtualMachineController) configureSEVDeviceOwnership(vmi *v1.VirtualMachineInstance, isolationRes isolation.IsolationResult, virtLauncherRootMount *safepath.Path) error {
	if virtutil.IsSEVVMI(vmi) {
		sevDevice, err := safepath.JoinNoFollow(virtLauncherRootMount, filepath.Join("dev", "sev"))
		if err != nil {
			return err
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(sevDevice); err != nil {
			return fmt.Errorf("failed to set SEV device owner: %v", err)
		}
	}
	return nil
}

func (c *VirtualMachineController) configureVirtioFS(vmi *v1.VirtualMachineInstance, isolationRes isolation.IsolationResult) error {
	for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
		socketPath, err := isolation.SafeJoin(isolationRes, virtiofs.VirtioFSSocketPath(fs.Name))
		if err != nil {
			return err
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath); err != nil {
			return err
		}
	}
	return nil
}

func (c *VirtualMachineController) syncVirtualMachine(client cmdclient.LauncherClient, vmi *v1.VirtualMachineInstance, preallocatedVolumes []string) error {
	smbios := c.clusterConfig.GetSMBIOS()
	period := c.clusterConfig.GetMemBalloonStatsPeriod()

	options := virtualMachineOptions(smbios, period, preallocatedVolumes, c.capabilities, c.clusterConfig)
	options.InterfaceDomainAttachment = domainspec.DomainAttachmentByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())

	err := client.SyncVirtualMachine(vmi, options)
	if err != nil {
		if strings.Contains(err.Error(), "EFI OVMF rom missing") {
			return &virtLauncherCriticalSecurebootError{fmt.Sprintf("mismatch of Secure Boot setting and bootloaders: %v", err)}
		}
	}

	return err
}

func (c *VirtualMachineController) handleHousekeeping(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domainExists bool) error {

	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		err := c.configureHousekeepingCgroup(vmi, cgroupManager)
		if err != nil {
			return err
		}
	}

	// Configure vcpu scheduler for realtime workloads and affine PIT thread for dedicated CPU
	if vmi.IsRealtimeEnabled() && !vmi.IsRunning() && !vmi.IsFinal() {
		log.Log.Object(vmi).Info("Configuring vcpus for real time workloads")
		if err := c.configureVCPUScheduler(vmi); err != nil {
			return err
		}
	}
	if vmi.IsCPUDedicated() && !vmi.IsRunning() && !vmi.IsFinal() {
		log.Log.V(3).Object(vmi).Info("Affining PIT thread")
		if err := c.affinePitThread(vmi); err != nil {
			return err
		}
	}
	if !domainExists {
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Created.String(), VMIDefined)
	}

	if vmi.IsRunning() {
		// Umount any disks no longer mounted
		if err := c.hotplugVolumeMounter.Unmount(vmi, cgroupManager); err != nil {
			return err
		}
	}
	return nil
}

func (c *VirtualMachineController) getPreallocatedVolumes(vmi *v1.VirtualMachineInstance) []string {
	var preallocatedVolumes []string
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.PersistentVolumeClaimInfo != nil && volumeStatus.PersistentVolumeClaimInfo.Preallocated {
			preallocatedVolumes = append(preallocatedVolumes, volumeStatus.Name)
		}
	}
	return preallocatedVolumes
}

func (c *VirtualMachineController) hotplugSriovInterfaces(vmi *v1.VirtualMachineInstance) error {
	sriovSpecInterfaces := netvmispec.FilterSRIOVInterfaces(vmi.Spec.Domain.Devices.Interfaces)

	sriovSpecIfacesNames := netvmispec.IndexInterfaceSpecByName(sriovSpecInterfaces)
	attachedSriovStatusIfaces := netvmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, func(iface v1.VirtualMachineInstanceNetworkInterface) bool {
		_, exist := sriovSpecIfacesNames[iface.Name]
		return exist && netvmispec.ContainsInfoSource(iface.InfoSource, netvmispec.InfoSourceDomain) &&
			netvmispec.ContainsInfoSource(iface.InfoSource, netvmispec.InfoSourceMultusStatus)
	})

	desiredSriovMultusPluggedIfaces := netvmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, func(iface v1.VirtualMachineInstanceNetworkInterface) bool {
		_, exist := sriovSpecIfacesNames[iface.Name]
		return exist && netvmispec.ContainsInfoSource(iface.InfoSource, netvmispec.InfoSourceMultusStatus)
	})

	if len(desiredSriovMultusPluggedIfaces) == len(attachedSriovStatusIfaces) {
		c.sriovHotplugExecutorPool.Delete(vmi.UID)
		return nil
	}

	rateLimitedExecutor := c.sriovHotplugExecutorPool.LoadOrStore(vmi.UID)
	return rateLimitedExecutor.Exec(func() error {
		return c.hotplugSriovInterfacesCommand(vmi)
	})
}

func (c *VirtualMachineController) hotplugSriovInterfacesCommand(vmi *v1.VirtualMachineInstance) error {
	const errMsgPrefix = "failed to hot-plug SR-IOV interfaces"

	client, err := c.getVerifiedLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := isolation.AdjustQemuProcessMemoryLimits(c.podIsolationDetector, vmi, c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio); err != nil {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), err.Error())
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	log.Log.V(3).Object(vmi).Info("sending hot-plug host-devices command")
	if err := client.HotplugHostDevices(vmi); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return nil
}

func memoryDumpPath(volumeStatus v1.VolumeStatus) string {
	target := hotplugdisk.GetVolumeMountDir(volumeStatus.Name)
	dumpPath := filepath.Join(target, volumeStatus.MemoryDumpVolume.TargetFileName)
	return dumpPath
}

func (c *VirtualMachineController) getMemoryDump(vmi *v1.VirtualMachineInstance) error {
	const errMsgPrefix = "failed to getting memory dump"

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.MemoryDumpVolume == nil || volumeStatus.Phase != v1.MemoryDumpVolumeInProgress {
			continue
		}
		client, err := c.getVerifiedLauncherClient(vmi)
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}

		log.Log.V(3).Object(vmi).Info("sending memory dump command")
		err = client.VirtualMachineMemoryDump(vmi, memoryDumpPath(volumeStatus))
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}
	}

	return nil
}

func (d *VirtualMachineController) hotplugVolumesReady(vmi *v1.VirtualMachineInstance) bool {
	hasHotplugVolume := false
	for _, v := range vmi.Spec.Volumes {
		if storagetypes.IsHotplugVolume(&v) {
			hasHotplugVolume = true
			break
		}
	}
	if !hasHotplugVolume {
		return true
	}
	if len(vmi.Status.VolumeStatus) == 0 {
		return false
	}
	for _, vs := range vmi.Status.VolumeStatus {
		if vs.HotplugVolume != nil && !(vs.Phase == v1.VolumeReady || vs.Phase == v1.HotplugVolumeMounted) {
			// wait for volume to be mounted
			return false
		}
	}
	return true
}

func (c *VirtualMachineController) processVmUpdate(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	isUnresponsive, isInitialized, err := c.isLauncherClientUnresponsive(vmi)
	if err != nil {
		return err
	}
	if !isInitialized {
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	} else if isUnresponsive {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with unresponsive command server."))
	}

	c.handlePostMigrationProxyCleanup(vmi)

	if c.isPreMigrationTarget(vmi) {
		return c.vmUpdateHelperMigrationTarget(vmi)
	} else if c.isMigrationSource(vmi) {
		return c.vmUpdateHelperMigrationSource(vmi, domain)
	} else {
		return c.vmUpdateHelperDefault(vmi, domain != nil)
	}
}

func (c *VirtualMachineController) setVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) error {
	phase, err := c.calculateVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}
	vmi.Status.Phase = phase
	return nil
}
func (c *VirtualMachineController) calculateVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) (v1.VirtualMachineInstancePhase, error) {

	if domain == nil {
		switch {
		case vmi.IsScheduled():
			isUnresponsive, isInitialized, err := c.isLauncherClientUnresponsive(vmi)

			if err != nil {
				return vmi.Status.Phase, err
			}
			if !isInitialized {
				c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
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

func (c *VirtualMachineController) addFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}
func (c *VirtualMachineController) deleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}
func (c *VirtualMachineController) updateFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}

func (c *VirtualMachineController) addDomainFunc(obj interface{}) {
	domain := obj.(*api.Domain)
	log.Log.Object(domain).Infof("Domain is in state %s reason %s", domain.Status.Status, domain.Status.Reason)
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}
func (c *VirtualMachineController) deleteDomainFunc(obj interface{}) {
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
		c.queue.Add(key)
	}
}
func (c *VirtualMachineController) updateDomainFunc(old, new interface{}) {
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
		c.queue.Add(key)
	}
}

func (c *VirtualMachineController) finalizeMigration(vmi *v1.VirtualMachineInstance) error {
	const errorMessage = "failed to finalize migration"

	client, err := c.getVerifiedLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf("%s: %v", errorMessage, err)
	}

	if err := c.hotplugCPU(vmi, client); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "failed to change vCPUs")
	}

	if err := c.hotplugMemory(vmi, client); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "failed to update guest memory")
	}
	removeMigratedVolumes(vmi)

	options := &cmdv1.VirtualMachineOptions{}
	options.InterfaceMigration = domainspec.BindingMigrationByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())
	if err := client.FinalizeVirtualMachineMigration(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		return fmt.Errorf("%s: %v", errorMessage, err)
	}

	vmi.Status.MigrationState.Completed = true

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

func (c *VirtualMachineController) isHostModelMigratable(vmi *v1.VirtualMachineInstance) error {
	if cpu := vmi.Spec.Domain.CPU; cpu != nil && cpu.Model == v1.CPUModeHostModel {
		if c.hostCpuModel == "" {
			err := fmt.Errorf("the node \"%s\" does not allow migration with host-model", vmi.Status.NodeName)
			log.Log.Object(vmi).Errorf(err.Error())
			return err
		}
	}
	return nil
}

func (c *VirtualMachineController) claimDeviceOwnership(virtLauncherRootMount *safepath.Path, deviceName string) error {
	softwareEmulation := c.clusterConfig.AllowEmulation()
	devicePath, err := safepath.JoinNoFollow(virtLauncherRootMount, filepath.Join("dev", deviceName))
	if err != nil {
		if softwareEmulation && deviceName == "kvm" {
			return nil
		}
		return err
	}

	return diskutils.DefaultOwnershipManager.SetFileOwnership(devicePath)
}

func (c *VirtualMachineController) reportDedicatedCPUSetForMigratingVMI(vmi *v1.VirtualMachineInstance) error {
	cgroupManager, err := getCgroupManager(vmi, c.host)
	if err != nil {
		return err
	}

	cpusetStr, err := cgroupManager.GetCpuSet()
	if err != nil {
		return err
	}

	cpuSet, err := hardware.ParseCPUSetLine(cpusetStr, 50000)
	if err != nil {
		return fmt.Errorf("failed to parse target VMI cpuset: %v", err)
	}

	vmi.Status.MigrationState.TargetCPUSet = cpuSet

	return nil
}

func (c *VirtualMachineController) reportTargetTopologyForMigratingVMI(vmi *v1.VirtualMachineInstance) error {
	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)
	topology, err := json.Marshal(options.Topology)
	if err != nil {
		return err
	}
	vmi.Status.MigrationState.TargetNodeTopology = string(topology)
	return nil
}

func (c *VirtualMachineController) handleMigrationAbort(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient) error {
	if vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortInProgress || vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortSucceeded {
		return nil
	}

	if err := client.CancelVirtualMachineMigration(vmi); err != nil {
		if err.Error() == migrations.CancelMigrationFailedVmiNotMigratingErr {
			// If migration did not even start there is no need to cancel it
			log.Log.Object(vmi).Infof("skipping migration cancellation since vmi is not migrating")
		}
		return err
	}
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), VMIAbortingMigration)
	return nil
}

func isIOError(shouldUpdate, domainExists bool, domain *api.Domain) bool {
	return shouldUpdate && domainExists && domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedIOError
}

func (c *VirtualMachineController) updateMachineType(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if domain == nil || vmi == nil {
		return
	}
	if domain.Spec.OS.Type.Machine != "" {
		vmi.Status.Machine = &v1.Machine{Type: domain.Spec.OS.Type.Machine}
	}
}

func (c *VirtualMachineController) hotplugCPU(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient) error {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()

	removeVMIVCPUChangeConditionAndLabel := func() {
		delete(vmi.Labels, v1.VirtualMachinePodCPULimitsLabel)
		vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceVCPUChange)
	}
	defer removeVMIVCPUChangeConditionAndLabel()

	if !vmiConditions.HasCondition(vmi, v1.VirtualMachineInstanceVCPUChange) {
		return nil
	}

	if vmi.IsCPUDedicated() {
		cpuLimitStr, ok := vmi.Labels[v1.VirtualMachinePodCPULimitsLabel]
		if !ok || len(cpuLimitStr) == 0 {
			return fmt.Errorf("cannot read CPU limit from VMI annotation")
		}

		cpuLimit, err := strconv.Atoi(cpuLimitStr)
		if err != nil {
			return fmt.Errorf("cannot parse CPU limit from VMI annotation: %v", err)
		}

		vcpus := hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU)
		if vcpus > int64(cpuLimit) {
			return fmt.Errorf("number of requested VCPUS (%d) exceeds the limit (%d)", vcpus, cpuLimit)
		}
	}

	options := virtualMachineOptions(
		nil,
		0,
		nil,
		c.capabilities,
		c.clusterConfig)

	if err := client.SyncVirtualMachineCPUs(vmi, options); err != nil {
		return err
	}

	if vmi.Status.CurrentCPUTopology == nil {
		vmi.Status.CurrentCPUTopology = &v1.CPUTopology{}
	}

	vmi.Status.CurrentCPUTopology.Sockets = vmi.Spec.Domain.CPU.Sockets
	vmi.Status.CurrentCPUTopology.Cores = vmi.Spec.Domain.CPU.Cores
	vmi.Status.CurrentCPUTopology.Threads = vmi.Spec.Domain.CPU.Threads

	return nil
}

func (c *VirtualMachineController) hotplugMemory(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient) error {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()

	removeVMIMemoryChangeLabel := func() {
		delete(vmi.Labels, v1.VirtualMachinePodMemoryRequestsLabel)
		delete(vmi.Labels, v1.MemoryHotplugOverheadRatioLabel)
	}
	defer removeVMIMemoryChangeLabel()

	if !vmiConditions.HasCondition(vmi, v1.VirtualMachineInstanceMemoryChange) {
		return nil
	}

	podMemReqStr := vmi.Labels[v1.VirtualMachinePodMemoryRequestsLabel]
	podMemReq, err := resource.ParseQuantity(podMemReqStr)
	if err != nil {
		vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
		return fmt.Errorf("cannot parse Memory requests from VMI label: %v", err)
	}

	overheadRatio := vmi.Labels[v1.MemoryHotplugOverheadRatioLabel]
	requiredMemory := services.GetMemoryOverhead(vmi, runtime.GOARCH, &overheadRatio)
	requiredMemory.Add(
		c.netBindingPluginMemoryCalculator.Calculate(vmi, c.clusterConfig.GetNetworkBindings()),
	)

	requiredMemory.Add(*vmi.Spec.Domain.Resources.Requests.Memory())

	if podMemReq.Cmp(requiredMemory) < 0 {
		vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
		return fmt.Errorf("amount of requested guest memory (%s) exceeds the launcher memory request (%s)", vmi.Spec.Domain.Memory.Guest.String(), podMemReqStr)
	}

	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)

	if err := client.SyncVirtualMachineMemory(vmi, options); err != nil {
		// mark hotplug as failed
		vmiConditions.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
			Type:    v1.VirtualMachineInstanceMemoryChange,
			Status:  k8sv1.ConditionFalse,
			Reason:  memoryHotplugFailedReason,
			Message: "memory hotplug failed, the VM configuration is not supported",
		})
		return err
	}

	vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
	vmi.Status.Memory.GuestRequested = vmi.Spec.Domain.Memory.Guest
	return nil
}

func removeMigratedVolumes(vmi *v1.VirtualMachineInstance) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceVolumesChange)
	vmi.Status.MigratedVolumes = nil
}

func parseLibvirtQuantity(value int64, unit string) *resource.Quantity {
	switch unit {
	case "b", "bytes":
		return resource.NewQuantity(value, resource.BinarySI)
	case "KB":
		return resource.NewQuantity(value*1000, resource.DecimalSI)
	case "MB":
		return resource.NewQuantity(value*1000*1000, resource.DecimalSI)
	case "GB":
		return resource.NewQuantity(value*1000*1000*1000, resource.DecimalSI)
	case "TB":
		return resource.NewQuantity(value*1000*1000*1000*1000, resource.DecimalSI)
	case "k", "KiB":
		return resource.NewQuantity(value*1024, resource.BinarySI)
	case "M", "MiB":
		return resource.NewQuantity(value*1024*1024, resource.BinarySI)
	case "G", "GiB":
		return resource.NewQuantity(value*1024*1024*1024, resource.BinarySI)
	case "T", "TiB":
		return resource.NewQuantity(value*1024*1024*1024*1024, resource.BinarySI)
	}
	return nil
}

func (c *VirtualMachineController) updateMemoryInfo(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil || vmi == nil || domain.Spec.CurrentMemory == nil {
		return nil
	}
	if vmi.Status.Memory == nil {
		vmi.Status.Memory = &v1.MemoryStatus{}
	}
	currentGuest := parseLibvirtQuantity(int64(domain.Spec.CurrentMemory.Value), domain.Spec.CurrentMemory.Unit)
	vmi.Status.Memory.GuestCurrent = currentGuest
	return nil
}

func configureParallelMigrationThreads(options *cmdclient.MigrationOptions, vm *v1.VirtualMachineInstance) {
	// When the CPU is limited, there's a risk of the migration threads choking the CPU resources on the compute container.
	// For this reason, we will avoid configuring migration threads in such scenarios.
	if cpuLimit, cpuLimitExists := vm.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; cpuLimitExists && !cpuLimit.IsZero() {
		return
	}

	options.ParallelMigrationThreads = pointer.P(parallelMultifdMigrationThreads)
}

func isReadOnlyDisk(disk *v1.Disk) bool {
	isReadOnlyCDRom := disk.CDRom != nil && (disk.CDRom.ReadOnly == nil || *disk.CDRom.ReadOnly)

	return isReadOnlyCDRom
}
