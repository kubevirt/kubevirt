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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package device_manager

import (
	"context"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/reservation"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var defaultBackoffTime = []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second}

type controlledDevice struct {
	devicePlugin Device
	started      bool
	stopChan     chan struct{}
	backoff      []time.Duration
}

func (c *controlledDevice) Start() {
	if c.started {
		return
	}

	stop := make(chan struct{})

	logger := log.DefaultLogger()
	dev := c.devicePlugin
	deviceName := dev.GetDeviceName()
	logger.Infof("Starting a device plugin for device: %s", deviceName)
	retries := 0

	backoff := c.backoff
	if backoff == nil {
		backoff = defaultBackoffTime
	}

	go func() {
		for {
			err := dev.Start(stop)
			if err != nil {
				logger.Reason(err).Errorf("Error starting %s device plugin", deviceName)
				retries = int(math.Min(float64(retries+1), float64(len(backoff)-1)))
			} else {
				retries = 0
			}

			select {
			case <-stop:
				// Ok we don't want to re-register
				return
			case <-time.After(backoff[retries]):
				// Wait a little and re-register
				continue
			}
		}
	}()

	c.stopChan = stop
	c.started = true
}

func (c *controlledDevice) Stop() {
	if !c.started {
		return
	}
	close(c.stopChan)

	c.stopChan = nil
	c.started = false
}

func (c *controlledDevice) GetName() string {
	return c.devicePlugin.GetDeviceName()
}

func PermanentHostDevicePlugins(maxDevices int, permissions string) []Device {
	var permanentDevicePluginPaths = map[string]string{
		"kvm":       "/dev/kvm",
		"tun":       "/dev/net/tun",
		"vhost-net": "/dev/vhost-net",
	}

	ret := make([]Device, 0, len(permanentDevicePluginPaths))
	for name, path := range permanentDevicePluginPaths {
		ret = append(ret, NewGenericDevicePlugin(name, path, maxDevices, permissions, (name != "kvm")))
	}
	return ret
}

type DeviceControllerInterface interface {
	Initialized() bool
	RefreshMediatedDeviceTypes()
}

type DeviceController struct {
	permanentPlugins    map[string]Device
	startedPlugins      map[string]controlledDevice
	startedPluginsMutex sync.Mutex
	host                string
	maxDevices          int
	permissions         string
	backoff             []time.Duration
	virtConfig          *virtconfig.ClusterConfig
	stop                chan struct{}
	mdevTypesManager    *MDEVTypesManager
	clientset           k8scli.CoreV1Interface
}

func NewDeviceController(
	host string,
	maxDevices int,
	permissions string,
	permanentPlugins []Device,
	clusterConfig *virtconfig.ClusterConfig,
	clientset k8scli.CoreV1Interface,
) *DeviceController {
	permanentPluginsMap := make(map[string]Device, len(permanentPlugins))
	for i := range permanentPlugins {
		permanentPluginsMap[permanentPlugins[i].GetDeviceName()] = permanentPlugins[i]
	}

	controller := &DeviceController{
		permanentPlugins: permanentPluginsMap,
		startedPlugins:   map[string]controlledDevice{},
		host:             host,
		maxDevices:       maxDevices,
		permissions:      permissions,
		backoff:          defaultBackoffTime,
		virtConfig:       clusterConfig,
		mdevTypesManager: NewMDEVTypesManager(),
		clientset:        clientset,
	}

	return controller
}

func (c *DeviceController) NodeHasDevice(devicePath string) bool {
	_, err := os.Stat(devicePath)
	// Since this is a boolean question, any error means "no"
	return (err == nil)
}

// updatePermittedHostDevicePlugins returns a slice of device plugins for permitted devices which are present on the node
func (c *DeviceController) updatePermittedHostDevicePlugins() []Device {
	var permittedDevices []Device

	var featureGatedDevices = []struct {
		Name      string
		Path      string
		IsAllowed func() bool
	}{
		{"sev", "/dev/sev", c.virtConfig.WorkloadEncryptionSEVEnabled},
		{"vhost-vsock", "/dev/vhost-vsock", c.virtConfig.VSOCKEnabled},
	}
	for _, dev := range featureGatedDevices {
		if dev.IsAllowed() {
			permittedDevices = append(
				permittedDevices,
				NewGenericDevicePlugin(dev.Name, dev.Path, c.maxDevices, c.permissions, true),
			)
		}
	}

	if c.virtConfig.PersistentReservationEnabled() {
		permittedDevices = append(permittedDevices, NewSocketDevicePlugin(reservation.GetPrResourceName(), reservation.GetPrHelperSocketDir(), reservation.GetPrHelperSocket(), c.maxDevices))
	}

	hostDevs := c.virtConfig.GetPermittedHostDevices()
	if hostDevs == nil {
		return permittedDevices
	}

	if len(hostDevs.PciHostDevices) != 0 {
		supportedPCIDeviceMap := make(map[string]string)
		for _, pciDev := range hostDevs.PciHostDevices {
			log.Log.V(4).Infof("Permitted PCI device in the cluster, ID: %s, resourceName: %s, externalProvider: %t",
				strings.ToLower(pciDev.PCIVendorSelector),
				pciDev.ResourceName,
				pciDev.ExternalResourceProvider)
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !pciDev.ExternalResourceProvider {
				supportedPCIDeviceMap[strings.ToLower(pciDev.PCIVendorSelector)] = pciDev.ResourceName
			}
		}
		for pciResourceName, pciDevices := range discoverPermittedHostPCIDevices(supportedPCIDeviceMap) {
			log.Log.V(4).Infof("Discovered PCIs %d devices on the node for the resource: %s", len(pciDevices), pciResourceName)
			// add a device plugin only for new devices
			permittedDevices = append(permittedDevices, NewPCIDevicePlugin(pciDevices, pciResourceName))
		}
	}
	if len(hostDevs.MediatedDevices) != 0 {
		supportedMdevsMap := make(map[string]string)
		for _, supportedMdev := range hostDevs.MediatedDevices {
			log.Log.V(4).Infof("Permitted mediated device in the cluster, ID: %s, resourceName: %s",
				supportedMdev.MDEVNameSelector,
				supportedMdev.ResourceName)
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !supportedMdev.ExternalResourceProvider {
				selector := removeSelectorSpaces(supportedMdev.MDEVNameSelector)
				supportedMdevsMap[selector] = supportedMdev.ResourceName
			}
		}
		for mdevTypeName, mdevUUIDs := range discoverPermittedHostMediatedDevices(supportedMdevsMap) {
			mdevResourceName := supportedMdevsMap[mdevTypeName]
			log.Log.V(4).Infof("Discovered mediated device on the node, type: %s, resourceName: %s", mdevTypeName, mdevResourceName)

			permittedDevices = append(permittedDevices, NewMediatedDevicePlugin(mdevUUIDs, mdevResourceName))
		}
	}

	for resourceName, pluginDevices := range discoverAllowedUSBDevices(hostDevs.USB) {
		permittedDevices = append(permittedDevices, NewUSBDevicePlugin(resourceName, pluginDevices))
	}

	return permittedDevices
}

func removeSelectorSpaces(selectorName string) string {
	// The name usually contain spaces which should be replaced with _
	// Such as GRID T4-1Q
	typeNameStr := strings.Replace(string(selectorName), " ", "_", -1)
	typeNameStr = strings.TrimSpace(typeNameStr)
	return typeNameStr

}

func (c *DeviceController) splitPermittedDevices(devices []Device) (map[string]Device, map[string]struct{}) {
	devicePluginsToRun := make(map[string]Device)
	devicePluginsToStop := make(map[string]struct{})

	// generate a map of currently started device plugins
	for resourceName := range c.startedPlugins {
		_, isPermanent := c.permanentPlugins[resourceName]
		if !isPermanent {
			devicePluginsToStop[resourceName] = struct{}{}
		}
	}

	for _, device := range devices {
		if _, isRunning := c.startedPlugins[device.GetDeviceName()]; !isRunning {
			devicePluginsToRun[device.GetDeviceName()] = device
		} else {
			delete(devicePluginsToStop, device.GetDeviceName())
		}
	}

	return devicePluginsToRun, devicePluginsToStop
}

func (c *DeviceController) RefreshMediatedDeviceTypes() {
	go func() {
		if c.refreshMediatedDeviceTypes() {
			c.refreshPermittedDevices()
		}
	}()
}

func (c *DeviceController) getExternallyProvidedMdevs() map[string]struct{} {
	externalMdevResourcesMap := make(map[string]struct{})
	if hostDevs := c.virtConfig.GetPermittedHostDevices(); hostDevs != nil {
		for _, supportedMdev := range hostDevs.MediatedDevices {
			if supportedMdev.ExternalResourceProvider {
				selector := removeSelectorSpaces(supportedMdev.MDEVNameSelector)
				externalMdevResourcesMap[selector] = struct{}{}
			}
		}
	}
	return externalMdevResourcesMap
}

func (c *DeviceController) refreshMediatedDeviceTypes() bool {
	// the handling of mediated device is disabled
	if c.virtConfig.MediatedDevicesHandlingDisabled() {
		return false
	}

	requiresDevicePluginsUpdate := false
	node, err := c.clientset.Nodes().Get(context.Background(), c.host, metav1.GetOptions{})
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure the desired mdev types, failed to get node details")
		return requiresDevicePluginsUpdate
	}
	externallyProvidedMdevMap := c.getExternallyProvidedMdevs()

	nodeDesiredMdevTypesList := c.virtConfig.GetDesiredMDEVTypes(node)
	requiresDevicePluginsUpdate, err = c.mdevTypesManager.updateMDEVTypesConfiguration(nodeDesiredMdevTypesList, externallyProvidedMdevMap)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure the desired mdev types: %s", strings.Join(nodeDesiredMdevTypesList, ", "))
	}
	return requiresDevicePluginsUpdate
}

func (c *DeviceController) refreshPermittedDevices() {
	logger := log.DefaultLogger()
	debugDevAdded := []string{}
	debugDevRemoved := []string{}

	// This function can be called multiple times in parallel, either because of multiple
	//   informer callbacks for the same event, or because the configmap was quickly updated
	//   multiple times in a row. To avoid starting/stopping device plugins multiple times,
	//   we need to protect c.startedPlugins, which we read from in
	//   c.updatePermittedHostDevicePlugins() and write to below.
	c.startedPluginsMutex.Lock()
	defer c.startedPluginsMutex.Unlock()

	enabledDevicePlugins, disabledDevicePlugins := c.splitPermittedDevices(
		c.updatePermittedHostDevicePlugins(),
	)

	// start device plugin for newly permitted devices
	for resourceName, dev := range enabledDevicePlugins {
		c.startDevice(resourceName, dev)
		debugDevAdded = append(debugDevAdded, resourceName)
	}
	// remove device plugin for now forbidden devices
	for resourceName := range disabledDevicePlugins {
		c.stopDevice(resourceName)
		debugDevRemoved = append(debugDevRemoved, resourceName)
	}

	logger.Info("refreshed device plugins for permitted/forbidden host devices")
	logger.Infof("enabled device-plugins for: %v", debugDevAdded)
	logger.Infof("disabled device-plugins for: %v", debugDevRemoved)
}

func (c *DeviceController) startDevice(resourceName string, dev Device) {
	c.stopDevice(resourceName)
	controlledDev := controlledDevice{
		devicePlugin: dev,
		backoff:      c.backoff,
	}
	controlledDev.Start()
	c.startedPlugins[resourceName] = controlledDev
}

func (c *DeviceController) stopDevice(resourceName string) {
	dev, exists := c.startedPlugins[resourceName]
	if exists {
		dev.Stop()
		delete(c.startedPlugins, resourceName)
	}
}

func (c *DeviceController) Run(stop chan struct{}) error {
	logger := log.DefaultLogger()

	// start the permanent DevicePlugins
	func() {
		c.startedPluginsMutex.Lock()
		defer c.startedPluginsMutex.Unlock()
		for name, dev := range c.permanentPlugins {
			c.startDevice(name, dev)
		}
	}()

	refreshMediatedDeviceTypesFn := func() {
		c.refreshMediatedDeviceTypes()
	}
	c.virtConfig.SetConfigModifiedCallback(refreshMediatedDeviceTypesFn)
	c.virtConfig.SetConfigModifiedCallback(c.refreshPermittedDevices)
	c.refreshPermittedDevices()

	// keep running until stop
	<-stop

	// stop all device plugins
	func() {
		c.startedPluginsMutex.Lock()
		defer c.startedPluginsMutex.Unlock()
		for name := range c.startedPlugins {
			c.stopDevice(name)
		}
	}()
	logger.Info("Shutting down device plugin controller")
	return nil
}

func (c *DeviceController) Initialized() bool {
	c.startedPluginsMutex.Lock()
	defer c.startedPluginsMutex.Unlock()
	for _, dev := range c.startedPlugins {
		if !dev.devicePlugin.GetInitialized() {
			return false
		}
	}

	return true
}
