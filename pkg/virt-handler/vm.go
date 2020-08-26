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
	"crypto/tls"
	"encoding/json"
	goerror "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	container_disk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	clusterutils "kubevirt.io/kubevirt/pkg/util/cluster"
	pvcutils "kubevirt.io/kubevirt/pkg/util/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

type launcherClientInfo struct {
	client              cmdclient.LauncherClient
	socketFile          string
	domainPipeStopChan  chan struct{}
	notInitializedSince time.Time
	ready               bool
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
	hostDevConfigMapInformer cache.SharedIndexInformer,
	watchdogTimeoutSeconds int,
	maxDevices int,
	clusterConfig *virtconfig.ClusterConfig,
	serverTLSConfig *tls.Config,
	clientTLSConfig *tls.Config,
	podIsolationDetector isolation.PodIsolationDetector,
) *VirtualMachineController {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	c := &VirtualMachineController{
		Queue:                    queue,
		recorder:                 recorder,
		clientset:                clientset,
		host:                     host,
		ipAddress:                ipAddress,
		virtShareDir:             virtShareDir,
		vmiSourceInformer:        vmiSourceInformer,
		vmiTargetInformer:        vmiTargetInformer,
		domainInformer:           domainInformer,
		gracefulShutdownInformer: gracefulShutdownInformer,
		hostDevConfigMapInformer: hostDevConfigMapInformer,
		heartBeatInterval:        1 * time.Minute,
		watchdogTimeoutSeconds:   watchdogTimeoutSeconds,
		migrationProxy:           migrationproxy.NewMigrationProxyManager(serverTLSConfig, clientTLSConfig),
		podIsolationDetector:     podIsolationDetector,
		containerDiskMounter:     container_disk.NewMounter(podIsolationDetector, virtPrivateDir+"/container-disk-mount-state"),
		clusterConfig:            clusterConfig,
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

	c.launcherClients = make(map[types.UID]*launcherClientInfo)
	c.phase1NetworkSetupCache = make(map[types.UID]int)
	c.podInterfaceCache = make(map[string]*network.PodCacheInterface)

	c.domainNotifyPipes = make(map[string]string)

	c.deviceManagerController = device_manager.NewDeviceController(c.host, maxDevices, clusterConfig, hostDevConfigMapInformer)

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
	hostDevConfigMapInformer cache.SharedIndexInformer
	launcherClients          map[types.UID]*launcherClientInfo
	launcherClientLock       sync.Mutex
	heartBeatInterval        time.Duration
	watchdogTimeoutSeconds   int
	deviceManagerController  *device_manager.DeviceController
	migrationProxy           migrationproxy.ProxyManager
	podIsolationDetector     isolation.PodIsolationDetector
	containerDiskMounter     container_disk.Mounter
	clusterConfig            *virtconfig.ClusterConfig

	// records if pod network phase1 has completed
	// phase1 involves cycling an entire posix thread
	// so for performance, knowing phase1 is complete
	// prevents cycling an unncessary posix thread.
	phase1NetworkSetupCache     map[types.UID]int
	phase1NetworkSetupCacheLock sync.Mutex

	// key is the file path, value is the contents.
	// if key exists, then don't read directly from file.
	podInterfaceCache     map[string]*network.PodCacheInterface
	podInterfaceCacheLock sync.Mutex

	domainNotifyPipes map[string]string
}

type virtLauncherCriticalNetworkError struct {
	msg string
}

func (e *virtLauncherCriticalNetworkError) Error() string { return e.msg }

func handleDomainNotifyPipe(domainPipeStopChan chan struct{}, ln net.Listener, virtShareDir string, vmi *v1.VirtualMachineInstance) {

	fdChan := make(chan net.Conn, 100)

	// Listen for new connections,
	// Close listener and exit when stop encountered
	go func(vmi *v1.VirtualMachineInstance, ln net.Listener, domainPipeStopChan chan struct{}) {
		for {
			select {
			case <-domainPipeStopChan:
				log.Log.Object(vmi).Infof("closing notify pipe listener for vmi")
				ln.Close()
				return
			default:
				fd, err := ln.Accept()
				if err != nil {
					log.Log.Reason(err).Error("Domain pipe accept error encountered.")
					// keep listening until stop invoked
					time.Sleep(1)
				} else {
					fdChan <- fd
				}
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
	err = os.MkdirAll(filepath.Dir(socketPath), 0755)
	if err != nil {
		log.Log.Reason(err).Error("unable to create directory for unix socket")
		return err
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
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
	d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Duration(enqueueTime)*time.Second)

	return false, timeLeft
}

func (d *VirtualMachineController) clearPodNetworkPhase1(uid types.UID) {
	// no need to cleanup with empty uid
	if string(uid) == "" {
		return
	}
	d.phase1NetworkSetupCacheLock.Lock()
	delete(d.phase1NetworkSetupCache, uid)
	d.phase1NetworkSetupCacheLock.Unlock()

	// Clean Pod interface cache from map and files
	d.podInterfaceCacheLock.Lock()
	for key, _ := range d.podInterfaceCache {
		if strings.Contains(key, string(uid)) {
			delete(d.podInterfaceCache, key)
		}
	}
	d.podInterfaceCacheLock.Unlock()

	vmiIfaceDir := fmt.Sprintf(virtutil.VMIInterfaceDir, uid)
	err := os.RemoveAll(vmiIfaceDir)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to delete VMI Network cache files: %s", err.Error())
	}
}

// Reaching into the network namespace of the VMI's pod is expensive because
// it results in killing/spawning a posix thread. Only do this if it
// is absolutely neccessary. The cache informs us if this action has
// already taken place or not for a VMI
func (d *VirtualMachineController) setPodNetworkPhase1(vmi *v1.VirtualMachineInstance) (bool, error) {

	// configure network
	res, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return false, fmt.Errorf("failed to detect isolation for launcher pod: %v", err)
	}

	pid := res.Pid()

	// check to see if we've already completed phase1 for this vmi
	d.phase1NetworkSetupCacheLock.Lock()
	cachedPid, ok := d.phase1NetworkSetupCache[vmi.UID]
	d.phase1NetworkSetupCacheLock.Unlock()

	if ok && cachedPid == pid {
		// already completed phase1
		return false, nil
	}

	err = res.DoNetNS(func() error { return network.SetupPodNetworkPhase1(vmi, pid) })
	if err != nil {
		_, critical := err.(*network.CriticalNetworkError)
		if critical {
			return true, err
		} else {
			return false, err
		}

	}

	// cache that phase 1 has completed for this vmi.
	d.phase1NetworkSetupCacheLock.Lock()
	d.phase1NetworkSetupCache[vmi.UID] = pid
	d.phase1NetworkSetupCacheLock.Unlock()

	return false, nil
}

func domainMigrated(domain *api.Domain) bool {
	if domain != nil && domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonMigrated {
		return true
	}
	return false
}

func (d *VirtualMachineController) getPodInterfacefromFileCache(uid types.UID, ifaceName string) (*network.PodCacheInterface, error) {
	ifacepath := fmt.Sprintf(virtutil.VMIInterfacepath, uid, ifaceName)

	// Once the Interface files are set on the handler, they don't change
	// If already present in the map, don't read again
	d.podInterfaceCacheLock.Lock()
	result, exists := d.podInterfaceCache[ifacepath]
	d.podInterfaceCacheLock.Unlock()

	if exists {
		return result, nil
	}

	content, err := ioutil.ReadFile(ifacepath)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to read from cache file: %s", err.Error())
		return nil, err
	}
	err = json.Unmarshal(content, &result)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to unmarshal interface content: %s", err.Error())
		return nil, err
	}
	d.podInterfaceCacheLock.Lock()
	d.podInterfaceCache[ifacepath] = result
	d.podInterfaceCacheLock.Unlock()

	return result, nil
}

func (d *VirtualMachineController) updateVMIStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain, syncError error) (err error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()

	// Don't update the VirtualMachineInstance if it is already in a final state
	if vmi.IsFinal() {
		return nil
	} else if vmi.Status.NodeName != "" && vmi.Status.NodeName != d.host {
		// Only update the VMI's phase if this node owns the VMI.
		// not owned by this host, likely the result of a migration
		return nil
	}

	oldStatus := vmi.DeepCopy().Status

	if domain != nil {
		if vmi.Status.GuestOSInfo.Name != domain.Status.OSInfo.Name {
			vmi.Status.GuestOSInfo.Name = domain.Status.OSInfo.Name
			vmi.Status.GuestOSInfo.Version = domain.Status.OSInfo.VersionId
			vmi.Status.GuestOSInfo.KernelRelease = domain.Status.OSInfo.KernelRelease
			vmi.Status.GuestOSInfo.PrettyName = domain.Status.OSInfo.PrettyName
			vmi.Status.GuestOSInfo.VersionID = domain.Status.OSInfo.VersionId
			vmi.Status.GuestOSInfo.KernelVersion = domain.Status.OSInfo.KernelVersion
			vmi.Status.GuestOSInfo.ID = domain.Status.OSInfo.Id
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
				if network.NetworkSource.Pod != nil {
					podIface, err := d.getPodInterfacefromFileCache(vmi.UID, network.Name)
					if err != nil {
						return err
					}
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
			// - domain.Status.Interfaces - interfaces reported by guest agent (emtpy if Qemu agent not running)
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

				if existingInterfacesSpecByName[domainInterface.Alias.Name].Masquerade != nil || existingInterfacesSpecByName[domainInterface.Alias.Name].Slirp != nil {
					isForwardingBindingInterface = true
				}

				if existingInterface, exists := existingInterfaceStatusByName[domainInterface.Alias.Name]; exists {
					// Reuse previously calculated interface from vmi.Status.Interfaces, updating the MAC from domain.Spec
					// Only interfaces defined in domain.Spec are handled here
					newInterface = existingInterface
					newInterface.MAC = interfaceMAC

					// If it is a Combination of Masquerade+Pod network, check IP from file cache
					if existingInterfacesSpecByName[domainInterface.Alias.Name].Masquerade != nil && existingNetworksByName[domainInterface.Alias.Name].NetworkSource.Pod != nil {
						iface, err := d.getPodInterfacefromFileCache(vmi.UID, domainInterface.Alias.Name)
						if err != nil {
							return err
						}

						if !reflect.DeepEqual(iface.PodIPs, existingInterfaceStatusByName[domainInterface.Alias.Name].IPs) {
							newInterface.Name = domainInterface.Alias.Name
							newInterface.IP = iface.PodIP
							newInterface.IPs = iface.PodIPs
						}
					}
				} else {
					// If not present in vmi.Status.Interfaces, create a new one based on domain.Spec
					newInterface = v1.VirtualMachineInstanceNetworkInterface{
						MAC:  interfaceMAC,
						Name: domainInterface.Alias.Name,
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
			// interfaces not defined in domain.Spec (most likely added by user on VM). Add them to vmi.Status.Interfaces
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
	}

	// Update migration progress if domain reports anything in the migration metadata.
	if domain != nil && domain.Spec.Metadata.KubeVirt.Migration != nil && vmi.Status.MigrationState != nil && d.isMigrationSource(vmi) {
		migrationMetadata := domain.Spec.Metadata.KubeVirt.Migration
		if migrationMetadata.UID == vmi.Status.MigrationState.MigrationUID {

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
	}

	// handle migrations differently than normal status updates.
	//
	// When a successful migration is detected, we must transfer ownership of the VMI
	// from the source node (this node) to the target node (node the domain was migrated to).
	//
	// Transfer owership by...
	// 1. Marking vmi.Status.MigationState as completed
	// 2. Update the vmi.Status.NodeName to reflect the target node's name
	// 3. Update the VMI's NodeNameLabel annotation to reflect the target node's name
	//
	// After a migration, the VMI's phase is no longer owned by this node. Only the
	// MigrationState status field is elgible to be mutated.
	if domainMigrated(domain) {
		migrationHost := ""
		if vmi.Status.MigrationState != nil {
			migrationHost = vmi.Status.MigrationState.TargetNode
		}

		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.EndTimestamp == nil {
			now := v12.NewTime(time.Now())
			vmi.Status.MigrationState.EndTimestamp = &now
		}

		targetNodeDetectedDomain, timeLeft := d.hasTargetDetectedDomain(vmi)
		// If we can't detect where the migration went to, then we have no
		// way of transfering ownership. The only option here is to move the
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
			vmi.Status.NodeName = migrationHost
			vmi.Status.MigrationState.Completed = true
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to node %s.", migrationHost))
		}

		if !reflect.DeepEqual(oldStatus, vmi.Status) {
			_, err = d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Calculate the new VirtualMachineInstance state based on what libvirt reported
	err = d.setVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}

	// Cacluate whether the VM is migratable
	if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceIsMigratable) {
		isBlockMigration, err := d.checkVolumesForMigration(vmi)
		liveMigrationCondition := v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceIsMigratable,
			Status: k8sv1.ConditionTrue,
		}
		if err != nil {
			liveMigrationCondition.Status = k8sv1.ConditionFalse
			liveMigrationCondition.Message = err.Error()
			liveMigrationCondition.Reason = v1.VirtualMachineInstanceReasonDisksNotMigratable
			vmi.Status.Conditions = append(vmi.Status.Conditions, liveMigrationCondition)
		}
		err = d.checkNetworkInterfacesForMigration(vmi)
		if err != nil {
			liveMigrationCondition = v1.VirtualMachineInstanceCondition{
				Type:    v1.VirtualMachineInstanceIsMigratable,
				Status:  k8sv1.ConditionFalse,
				Message: err.Error(),
				Reason:  v1.VirtualMachineInstanceReasonInterfaceNotMigratable,
			}
			vmi.Status.Conditions = append(vmi.Status.Conditions, liveMigrationCondition)
		}
		if liveMigrationCondition.Status == k8sv1.ConditionTrue {
			vmi.Status.Conditions = append(vmi.Status.Conditions, liveMigrationCondition)
		}

		// Set VMI Migration Method
		if isBlockMigration {
			vmi.Status.MigrationMethod = v1.BlockMigration
		} else {
			vmi.Status.MigrationMethod = v1.LiveMigration
		}
	}
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
			LastProbeTime: v12.Now(),
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

		var match = false
		for _, version := range d.clusterConfig.GetSupportedAgentVersions() {
			match = match || regexp.MustCompile(version).MatchString(guestInfo.GAVersion)
		}

		if !match {
			if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent) {
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceUnsupportedAgent,
					LastProbeTime: v12.Now(),
					Status:        k8sv1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
			}
		} else {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent)
		}

	}

	// Update paused condition in case VMI was paused / unpaused
	if domain != nil && domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedUser {
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
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
		}
	} else if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		log.Log.Object(vmi).V(3).Info("Removing paused condition")
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstancePaused)
	}

	if _, ok := syncError.(*virtLauncherCriticalNetworkError); ok {
		log.Log.Errorf("virt-launcher crashed due to a network error. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	condManager.CheckFailure(vmi, syncError, "Synchronizing with the Domain failed.")

	if !reflect.DeepEqual(oldStatus, vmi.Status) {
		_, err = d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
		if err != nil {
			return err
		}
	}

	if oldStatus.Phase != vmi.Status.Phase {
		switch vmi.Status.Phase {
		case v1.Running:
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Started.String(), "VirtualMachineInstance started.")
		case v1.Succeeded:
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Stopped.String(), "The VirtualMachineInstance was shut down.")
		case v1.Failed:
			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Stopped.String(), "The VirtualMachineInstance crashed.")
		}
	}

	return nil
}

func (c *VirtualMachineController) Run(threadiness int, stopCh chan struct{}) {
	defer c.Queue.ShutDown()
	log.Log.Info("Starting virt-handler controller.")

	// Wait for the domain cache to be synced
	go c.domainInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced)

	go c.deviceManagerController.Run(stopCh)

	// Poplulate the VirtualMachineInstance store with known Domains on the host, to get deletes since the last run
	for _, domain := range c.domainInformer.GetStore().List() {
		d := domain.(*api.Domain)
		c.vmiSourceInformer.GetStore().Add(
			v1.NewVMIReferenceWithUUID(
				d.ObjectMeta.Namespace,
				d.ObjectMeta.Name,
				d.Spec.Metadata.KubeVirt.UID,
			),
		)
	}

	go c.vmiSourceInformer.Run(stopCh)
	go c.vmiTargetInformer.Run(stopCh)
	go c.gracefulShutdownInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced, c.vmiSourceInformer.HasSynced, c.vmiTargetInformer.HasSynced, c.gracefulShutdownInformer.HasSynced)

	go c.heartBeat(c.heartBeatInterval, stopCh)

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

func (d *VirtualMachineController) migrationOrphanedSourceNodeExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	if domainExists {
		err := d.processVmDelete(vmi, domain)
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

func isMigrating(vmi *v1.VirtualMachineInstance) bool {

	now := v12.Now()

	running := false
	if vmi.Status.MigrationState != nil {
		start := vmi.Status.MigrationState.StartTimestamp
		stop := vmi.Status.MigrationState.EndTimestamp
		if start != nil && (now.After(start.Time) || now.Equal(start)) {
			running = true
		}

		if stop != nil && (now.After(stop.Time) || now.Equal(stop)) {
			running = false
		}
	}
	return running
}

func (d *VirtualMachineController) migrationTargetExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

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
			err := d.processVmDelete(vmi, domain)
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
		d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*1)

	} else if shouldUpdate {
		log.Log.Object(vmi).Info("Processing vmi migration target update")
		vmiCopy := vmi.DeepCopy()

		// prepare the POD for the migration
		err := d.processVmUpdate(vmi)
		if err != nil {
			return err
		}

		// Handle post migration
		if domainExists && vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.TargetNodeDomainDetected {
			// record that we've see the domain populated on the target's node
			log.Log.Object(vmi).Info("The target node received the migrated domain")
			vmiCopy.Status.MigrationState.TargetNodeDomainDetected = true
			d.setVMIGuestTime(vmi)
		}
		if !isMigrating(vmi) {

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
			_, err := d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmiCopy)
			if err != nil {
				return err
			}
		}

		return nil
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
	return diskutils.RemoveFile(triggerFile)
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

	log.Log.Infof("Processing event %v", key)
	if vmiExists {
		log.Log.Object(vmi).Infof("VMI is in phase: %v\n", vmi.Status.Phase)
	} else {
		log.Log.Info("VMI does not exist")
	}
	if domainExists {
		log.Log.Object(domain).Infof("Domain status: %v, reason: %v\n", domain.Status.Status, domain.Status.Reason)
	} else {
		log.Log.Info("Domain does not exist")
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
		syncErr = d.processVmDelete(vmi, domain)
	case shouldCleanUp:
		log.Log.Object(vmi).V(3).Info("Processing local ephemeral data cleanup for shutdown domain.")
		syncErr = d.processVmCleanup(vmi)
	case shouldUpdate:
		log.Log.Object(vmi).V(3).Info("Processing vmi update")
		syncErr = d.processVmUpdate(vmi)
	default:
		log.Log.Object(vmi).V(3).Info("No update processing required")
	}

	if syncErr != nil && !vmi.IsFinal() {
		d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())
		log.Log.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	// Update the VirtualMachineInstance status, if the VirtualMachineInstance exists
	if vmiExists {
		err = d.updateVMIStatus(vmi.DeepCopy(), domain, syncErr)
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

	domain, domainExists, domainCachedUID, err := d.getDomainFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists && string(domainCachedUID) != "" {
		// it's possible to discover the UID from cache even if the domain
		// doesn't techincally exist anymore
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
			d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*1)
			return nil
		} else if expired {
			log.Log.Object(oldVMI).Infof("Detected stale vmi %s that still needs cleanup before new vmi %s with identical name/namespace can be processed", oldVMI.UID, vmi.UID)
			err = d.processVmCleanup(oldVMI)
			if err != nil {
				return err
			}
			// Make sure we re-enqueue the key to ensure this new VMI is processed
			// after the stale domain is removed
			d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*5)
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
		return d.migrationTargetExecute(key,
			vmi,
			vmiExists,
			domain,
			domainExists)
	} else if vmiExists && d.isOrphanedMigrationSource(vmi) {
		// 3. POST-MIGRATION SOURCE CLEANUP
		//
		// After a migration, the migrated domain still exists in the old
		// source's domain cache. Ensure that any node that isn't currently
		// the target or owner of the VMI handles deleting the domain locally.
		return d.migrationOrphanedSourceNodeExecute(key,
			vmi,
			vmiExists,
			domain,
			domainExists)
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
	err = d.containerDiskMounter.Unmount(vmi)
	if err != nil {
		return err
	}

	d.clearPodNetworkPhase1(vmi.UID)

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

	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	clientInfo, ok := d.launcherClients[vmi.UID]
	if ok && clientInfo.client != nil {
		if clientInfo.client != nil {
			clientInfo.client.Close()
			close(clientInfo.domainPipeStopChan)
		}

		// With legacy sockets on hostpaths, we have to cleanup the sockets ourselves.
		if cmdclient.IsLegacySocket(clientInfo.socketFile) {
			err := os.RemoveAll(clientInfo.socketFile)
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

	delete(d.launcherClients, vmi.UID)
	return nil
}

// used by unit tests to add mock clients
func (d *VirtualMachineController) addLauncherClient(vmUID types.UID, info *launcherClientInfo) error {
	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	d.launcherClients[vmUID] = info

	return nil
}

func (d *VirtualMachineController) isLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error) {
	var socketFile string

	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	clientInfo, ok := d.launcherClients[vmi.UID]
	if ok {
		if clientInfo.ready == true {
			// use cached socket if we previously established a connection
			socketFile = clientInfo.socketFile
		} else {
			socketFile, err = cmdclient.FindSocketOnHost(vmi)
			if err != nil {
				// socket does not exist, but let's see if the pod is still there
				if _, err = cmdclient.FindPodDirOnHost(vmi); err != nil {
					// no pod meanst that waiting for it to initialize makes no sense
					return true, true, nil
				}
				// pod is still there, if there is no socket let's wait for it to become ready
				if clientInfo.notInitializedSince.Before(time.Now().Add(-3 * time.Minute)) {
					return true, true, nil
				}
				return false, false, nil
			}
			d.launcherClients[vmi.UID].ready = true
			d.launcherClients[vmi.UID].socketFile = socketFile
		}
	} else {
		d.launcherClients[vmi.UID] = &launcherClientInfo{
			notInitializedSince: time.Now(),
			ready:               false,
		}
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
		d.launcherClients[vmi.UID].ready = true
		d.launcherClients[vmi.UID].socketFile = socketFile
	}
	// The new way of detecting unresponsive VMIs monitors the
	// cmd socket. This requires an updated VMI image. Old VMIs
	// still use the watchdog method.
	watchDogExists, _ := watchdog.WatchdogFileExists(d.virtShareDir, vmi)
	if cmdclient.SocketMonitoringEnabled(socketFile) && !watchDogExists {
		isUnresponsive := cmdclient.IsSocketUnresponsive(socketFile)
		return isUnresponsive, true, nil
	}

	// fall back to legacy watchdog support for backwards compatiblity
	isUnresponsive, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, vmi)
	return isUnresponsive, true, err
}

func (d *VirtualMachineController) getLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	var err error

	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	clientInfo, ok := d.launcherClients[vmi.UID]
	if ok && clientInfo.client != nil {
		return clientInfo.client, nil
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

	d.launcherClients[vmi.UID] = &launcherClientInfo{
		client:              client,
		socketFile:          socketFile,
		domainPipeStopChan:  domainPipeStopChan,
		notInitializedSince: time.Now(),
		ready:               true,
	}

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
		expired, timeLeft := d.hasGracePeriodExpired(domain)
		if !expired {
			if domain.Status.Status != api.Shutdown {
				err = client.ShutdownVirtualMachine(vmi)
				if err != nil && !cmdclient.IsDisconnected(err) {
					// Only report err if it wasn't the result of a disconnect.
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
				d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Duration(timeLeft)*time.Second)
				d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.ShuttingDown.String(), "Signaled Graceful Shutdown")
			} else {
				log.Log.V(4).Object(vmi).Infof("%s is already shutting down.", vmi.GetObjectMeta().GetName())
			}
			return nil
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

	d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), "VirtualMachineInstance stopping")

	return nil
}

func (d *VirtualMachineController) processVmDelete(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := d.getVerifiedLauncherClient(vmi)

	// If the pod has been torn down, we know the VirtualMachineInstance is down.
	if err == nil {

		log.Log.Object(vmi).Infof("Signaled deletion for %s", vmi.GetObjectMeta().GetName())

		// pending deletion.
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), "Signaled Deletion")

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
	networks := map[string]*v1.Network{}
	for _, network := range vmi.Spec.Networks {
		networks[network.Name] = network.DeepCopy()
	}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Masquerade == nil && networks[iface.Name].Pod != nil {
			return fmt.Errorf("cannot migrate VMI which does not use masquerade to connect to the pod network")
		}
	}
	return nil
}

func (d *VirtualMachineController) checkVolumesForMigration(vmi *v1.VirtualMachineInstance) (blockMigrate bool, err error) {
	// Check if all VMI volumes can be shared between the source and the destination
	// of a live migration. blockMigrate will be returned as false, only if all volumes
	// are shared and the VMI has no local disks
	// Some combinations of disks makes the VMI no suitable for live migration.
	// A relevant error will be returned in this case.
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		if volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil {
			var volName string
			if volSrc.PersistentVolumeClaim != nil {
				volName = volSrc.PersistentVolumeClaim.ClaimName
			} else {
				volName = volSrc.DataVolume.Name
			}
			_, shared, err := pvcutils.IsSharedPVCFromClient(d.clientset, vmi.Namespace, volName)
			if errors.IsNotFound(err) {
				return blockMigrate, fmt.Errorf("persistentvolumeclaim %v not found", volName)
			} else if err != nil {
				return blockMigrate, err
			}
			if !shared {
				return true, fmt.Errorf("cannot migrate VMI with non-shared PVCs")
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

func (d *VirtualMachineController) handlePostSyncMigrationProxy(vmi *v1.VirtualMachineInstance) error {
	// handle starting/stopping target migration proxy
	migrationTargetSockets := []string{}
	res, err := d.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	// Get the libvirt connection socket file on the destination pod.
	socketFile := fmt.Sprintf("/proc/%d/root/var/run/libvirt/libvirt-sock", res.Pid())
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf("/proc/%d/root/var/run/kubevirt", res.Pid())
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

func (d *VirtualMachineController) handleMigrationProxy(vmi *v1.VirtualMachineInstance) error {
	// handle starting/stopping source migration proxy.
	// start the source proxy once we know the target address

	if d.isMigrationSource(vmi) {
		res, err := d.podIsolationDetector.Detect(vmi)
		if err != nil {
			return err
		}
		// the migration-proxy is no longer shared via host mount, so we
		// pass in the virt-launcher's baseDir to reach the unix sockets.
		baseDir := fmt.Sprintf("/proc/%d/root/var/run/kubevirt", res.Pid())
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

	} else {
		d.migrationProxy.StopSourceListener(string(vmi.UID))
	}
	return nil
}

func (d *VirtualMachineController) getLauncherClinetInfo(vmi *v1.VirtualMachineInstance) *launcherClientInfo {
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()
	return d.launcherClients[vmi.UID]
}

func (d *VirtualMachineController) processVmUpdate(origVMI *v1.VirtualMachineInstance) error {
	vmi := origVMI.DeepCopy()

	isUnresponsive, isInitialized, err := d.isLauncherClientUnresponsive(vmi)
	if err != nil {
		return err
	}
	if !isInitialized {
		d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*1)
		return nil
	} else if isUnresponsive {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with unresponsive command server."))
	}

	err = hostdisk.ReplacePVCByHostDisk(vmi, d.clientset)
	if err != nil {
		return err
	}

	client, err := d.getLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf("unable to create virt-launcher client connection: %v", err)
	}

	// this adds, removes, and replaces migration proxy connections as needed
	err = d.handleMigrationProxy(vmi)
	if err != nil {
		return fmt.Errorf("failed to handle migration proxy: %v", err)
	}

	if d.isPreMigrationTarget(vmi) {
		if !isMigrating(vmi) {

			// give containerDisks some time to become ready before throwing errors on retries
			info := d.getLauncherClinetInfo(vmi)
			if ready, err := d.containerDiskMounter.ContainerDisksReady(vmi, info.notInitializedSince); !ready {
				if err != nil {
					return err
				}
				d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*1)
				return nil
			}

			// Mount container disks
			if err := d.containerDiskMounter.Mount(vmi, false); err != nil {
				return err
			}

			// configure network inside virt-launcher compute container
			criticalNetworkError, err := d.setPodNetworkPhase1(vmi)
			if err != nil {
				if criticalNetworkError {
					return &virtLauncherCriticalNetworkError{fmt.Sprintf("failed to configure vmi network for migration target: %v", err)}
				} else {
					return fmt.Errorf("failed to configure vmi network for migration target: %v", err)
				}

			}

			if err := client.SyncMigrationTarget(vmi); err != nil {
				return fmt.Errorf("syncing migration target failed: %v", err)

			}
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), "VirtualMachineInstance Migration Target Prepared.")

			err = d.handlePostSyncMigrationProxy(vmi)
			if err != nil {
				return fmt.Errorf("failed to handle post sync migration proxy: %v", err)
			}
		}
	} else if d.isMigrationSource(vmi) {
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
	} else {

		if !vmi.IsRunning() && !vmi.IsFinal() {

			// give containerDisks some time to become ready before throwing errors on retries
			info := d.getLauncherClinetInfo(vmi)
			if ready, err := d.containerDiskMounter.ContainerDisksReady(vmi, info.notInitializedSince); !ready {
				if err != nil {
					return err
				}
				d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*1)
				return nil
			}

			if err := d.containerDiskMounter.Mount(vmi, true); err != nil {
				return err
			}

			criticalNetworkError, err := d.setPodNetworkPhase1(vmi)
			if err != nil {
				if criticalNetworkError {
					return &virtLauncherCriticalNetworkError{fmt.Sprintf("failed to configure vmi network: %v", err)}
				} else {
					return fmt.Errorf("failed to configure vmi network: %v", err)
				}

			}

			// set runtime limits as needed
			err = d.podIsolationDetector.AdjustResources(vmi)
			if err != nil {
				return fmt.Errorf("failed to adjust resources: %v", err)
			}
		}

		smbios := d.clusterConfig.GetSMBIOS()
		period := d.clusterConfig.GetMemBalloonStatsPeriod()

		options := &cmdv1.VirtualMachineOptions{
			VirtualMachineSMBios: &cmdv1.SMBios{
				Family:       smbios.Family,
				Product:      smbios.Product,
				Manufacturer: smbios.Manufacturer,
				Sku:          smbios.Sku,
				Version:      smbios.Version,
			},
			MemBalloonStatsPeriod: period,
		}

		err = client.SyncVirtualMachine(vmi, options)
		if err != nil {
			return err
		}
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Created.String(), "VirtualMachineInstance defined.")
	}

	return err
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
				d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Second*1)
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
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) deleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) updateFunc(old, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
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
		return
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

func (d *VirtualMachineController) heartBeat(interval time.Duration, stopCh chan struct{}) {
	// This is a temporary workaround until k8s bug #66525 is resolved
	cpuManagerPath := virtutil.CPUManagerPath
	if t, err := clusterutils.IsOnOpenShift(d.clientset); err != nil {
		// in that case leave the default cpuManagerPath
		log.DefaultLogger().Reason(err).Errorf("Unable to detect cluster provider on %s, setting a default cpuManager file path %s", d.host, cpuManagerPath)
	} else if t && clusterutils.GetOpenShiftMajorVersion(d.clientset) == clusterutils.OpenShift3Major {
		cpuManagerPath = virtutil.CPUManagerOS3Path
	}

	for {
		wait.JitterUntil(func() {
			now, err := json.Marshal(v12.Now())
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine date")
				return
			}

			kubevirtSchedulable := "true"
			if !d.deviceManagerController.Initialized() {
				kubevirtSchedulable = "false"
			}

			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%s"}, "annotations": {"%s": %s}}}`, v1.NodeSchedulable, kubevirtSchedulable, v1.VirtHandlerHeartbeat, string(now)))
			_, err = d.clientset.CoreV1().Nodes().Patch(d.host, types.StrategicMergePatchType, data)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", d.host)
				return
			}
			log.DefaultLogger().V(4).Infof("Heartbeat sent")
			// Label the node if cpu manager is running on it
			// This is a temporary workaround until k8s bug #66525 is resolved
			if d.clusterConfig.CPUManagerEnabled() {
				d.updateNodeCpuManagerLabel(cpuManagerPath)
			}
		}, interval, 1.2, true, stopCh)
	}
}

func (d *VirtualMachineController) updateNodeCpuManagerLabel(cpuManagerPath string) {
	var cpuManagerOptions map[string]interface{}

	content, err := ioutil.ReadFile(cpuManagerPath)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", d.host)
		return
	}

	err = json.Unmarshal(content, &cpuManagerOptions)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", d.host)
		return
	}

	isEnabled := false
	if v, ok := cpuManagerOptions["policyName"]; ok && v == "static" {
		isEnabled = true
	}

	data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%t"}}}`, v1.CPUManager, isEnabled))
	_, err = d.clientset.CoreV1().Nodes().Patch(d.host, types.StrategicMergePatchType, data)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", d.host)
		return
	}
	log.DefaultLogger().V(4).Infof("Node has CPU Manager running")

}

func (d *VirtualMachineController) setVMIGuestTime(vmi *v1.VirtualMachineInstance) error {
	// update the vmi guest with the current time
	client, err := d.getVerifiedLauncherClient(vmi)
	if err != nil {
		return err
	}
	err = client.SetVirtualMachineGuestTime(vmi)
	if err != nil {
		log.Log.Reason(err).Error("failed to set vmi guest time to the current")
		return err
	}
	return nil
}

func isACPIEnabled(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	zero := int64(0)
	return vmi.Spec.TerminationGracePeriodSeconds != &zero &&
		domain != nil &&
		domain.Spec.Features != nil &&
		domain.Spec.Features.ACPI != nil
}
