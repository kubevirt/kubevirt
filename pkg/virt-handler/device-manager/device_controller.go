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
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"kubevirt.io/client-go/log"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var permanentDevicePluginPaths = map[string]string{
	"kvm":       "/dev/kvm",
	"tun":       "/dev/net/tun",
	"vhost-net": "/dev/vhost-net",
}

type DeviceControllerInterface interface {
	Initialized() bool
}

type DeviceController struct {
	devicePlugins      map[string]ControlledDevice
	devicePluginsMutex sync.Mutex
	host               string
	maxDevices         int
	backoff            []time.Duration
	virtConfig         *virtconfig.ClusterConfig
	stop               chan struct{}
	mdevTypesManager   *MDEVTypesManager
}

type ControlledDevice struct {
	devicePlugin GenericDevice
	stopChan     chan struct{}
}

func getPermanentHostDevicePlugins(maxDevices int, permissions string) map[string]ControlledDevice {
	ret := map[string]ControlledDevice{}
	for name, path := range permanentDevicePluginPaths {
		ret[name] = ControlledDevice{
			devicePlugin: NewGenericDevicePlugin(name, path, maxDevices, permissions, (name != "kvm")),
			stopChan:     make(chan struct{}),
		}
	}
	return ret
}

func NewDeviceController(host string, maxDevices int, permissions string, clusterConfig *virtconfig.ClusterConfig) *DeviceController {
	controller := &DeviceController{
		devicePlugins:    getPermanentHostDevicePlugins(maxDevices, permissions),
		host:             host,
		maxDevices:       maxDevices,
		backoff:          []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second},
		virtConfig:       clusterConfig,
		mdevTypesManager: NewMDEVTypesManager(),
	}

	return controller
}

func (c *DeviceController) NodeHasDevice(devicePath string) bool {
	_, err := os.Stat(devicePath)
	// Since this is a boolean question, any error means "no"
	return (err == nil)
}

func (c *DeviceController) startDevicePlugin(controlledDev ControlledDevice) {
	logger := log.DefaultLogger()
	dev := controlledDev.devicePlugin
	deviceName := dev.GetDeviceName()
	stop := controlledDev.stopChan
	logger.Infof("Starting a device plugin for device: %s", deviceName)
	retries := 0

	for {
		err := dev.Start(stop)
		if err != nil {
			logger.Reason(err).Errorf("Error starting %s device plugin", deviceName)
			retries = int(math.Min(float64(retries+1), float64(len(c.backoff)-1)))
		} else {
			retries = 0
		}

		select {
		case <-stop:
			// Ok we don't want to re-register
			return
		default:
			// Wait a little bit and re-register
			time.Sleep(c.backoff[retries])
		}
	}
}

// updatePermittedHostDevicePlugins will return a map of device plugins for permitted devices which are present on the node
// and a map of restricted devices that should be removed
func (c *DeviceController) updatePermittedHostDevicePlugins() (map[string]ControlledDevice, map[string]ControlledDevice) {
	devicePluginsToRun := make(map[string]ControlledDevice)
	devicePluginsToStop := make(map[string]ControlledDevice)
	// generate a map of currently started device plugins
	for resourceName, hostDevDP := range c.devicePlugins {
		_, isPermanent := permanentDevicePluginPaths[resourceName]
		if !isPermanent {
			devicePluginsToStop[resourceName] = hostDevDP
		}
	}
	if hostDevs := c.virtConfig.GetPermittedHostDevices(); hostDevs != nil {
		supportedPCIDeviceMap := make(map[string]string)
		if len(hostDevs.PciHostDevices) != 0 {
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
			pciHostDevices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
			for pciID, pciDevices := range pciHostDevices {
				pciResourceName := supportedPCIDeviceMap[pciID]
				log.Log.V(4).Infof("Discovered PCI device on the node, ID: %s, resourceName: %s", pciID, pciResourceName)
				// add a device plugin only for new devices
				if _, isRunning := c.devicePlugins[pciResourceName]; !isRunning {
					devicePluginsToRun[pciResourceName] = ControlledDevice{
						devicePlugin: NewPCIDevicePlugin(pciDevices, pciResourceName),
						stopChan:     make(chan struct{}),
					}
				} else {
					delete(devicePluginsToStop, pciResourceName)
				}
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

			hostMdevs := discoverPermittedHostMediatedDevices(supportedMdevsMap)
			for mdevTypeName, mdevUUIDs := range hostMdevs {
				mdevResourceName := supportedMdevsMap[mdevTypeName]
				log.Log.V(4).Infof("Discovered mediated device on the node, type: %s, resourceName: %s", mdevTypeName, mdevResourceName)
				// add a device plugin only for new devices
				if _, isRunning := c.devicePlugins[mdevResourceName]; !isRunning {
					devicePluginsToRun[mdevResourceName] = ControlledDevice{
						devicePlugin: NewMediatedDevicePlugin(mdevUUIDs, mdevResourceName),
						stopChan:     make(chan struct{}),
					}
				} else {
					delete(devicePluginsToStop, mdevResourceName)
				}
			}
		}
	}
	return devicePluginsToRun, devicePluginsToStop
}

func removeSelectorSpaces(selectorName string) string {
	// The name usually contain spaces which should be replaced with _
	// Such as GRID T4-1Q
	typeNameStr := strings.Replace(string(selectorName), " ", "_", -1)
	typeNameStr = strings.TrimSpace(typeNameStr)
	return typeNameStr

}

func (c *DeviceController) refreshMediatedDevicesTypes() {
	nodeDesiredMdevTypesList := c.virtConfig.GetDesiredMDEVTypes(c.host)
	err := c.mdevTypesManager.updateMDEVTypesConfiguration(nodeDesiredMdevTypesList)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure the desired mdev types: %s", strings.Join(nodeDesiredMdevTypesList, ", "))
	}
}

func (c *DeviceController) refreshPermittedDevices() {
	logger := log.DefaultLogger()
	debugDevAdded := []string{}
	debugDevRemoved := []string{}

	// This function can be called multiple times in parallel, either because of multiple
	//   informer callbacks for the same event, or because the configmap was quickly updated
	//   multiple times in a row. To avoid starting/stopping device plugins multiple times,
	//   we need to protect c.devicePlugins, which we read from in
	//   c.updatePermittedHostDevicePlugins() and write to below.
	c.devicePluginsMutex.Lock()

	enabledDevicePlugins, disabledDevicePlugins := c.updatePermittedHostDevicePlugins()

	// start device plugin for newly permitted devices
	for resourceName, dev := range enabledDevicePlugins {
		go c.startDevicePlugin(dev)
		c.devicePlugins[resourceName] = dev
		debugDevAdded = append(debugDevAdded, resourceName)
	}
	// remove device plugin for now forbidden devices
	for resourceName, dev := range disabledDevicePlugins {
		close(dev.stopChan)
		delete(c.devicePlugins, resourceName)
		debugDevRemoved = append(debugDevRemoved, resourceName)
	}

	c.devicePluginsMutex.Unlock()

	logger.Info("refreshed device plugins for permitted/forbidden host devices")
	logger.Infof("enabled device-plugins for: %v", debugDevAdded)
	logger.Infof("disabled device-plugins for: %v", debugDevRemoved)
}

func (c *DeviceController) Run(stop chan struct{}) error {
	logger := log.DefaultLogger()
	// start the permanent DevicePlugins
	for _, dev := range c.devicePlugins {
		go c.startDevicePlugin(dev)
	}
	c.virtConfig.SetConfigModifiedCallback(c.refreshMediatedDevicesTypes)
	c.virtConfig.SetConfigModifiedCallback(c.refreshPermittedDevices)
	c.refreshPermittedDevices()

	// keep running until stop
	<-stop

	// stop all device plugins
	c.devicePluginsMutex.Lock()
	for _, dev := range c.devicePlugins {
		dev.stopChan <- struct{}{}
	}
	c.devicePluginsMutex.Unlock()
	logger.Info("Shutting down device plugin controller")
	return nil
}

func (c *DeviceController) Initialized() bool {
	c.devicePluginsMutex.Lock()
	defer c.devicePluginsMutex.Unlock()
	for _, dev := range c.devicePlugins {
		if !dev.devicePlugin.GetInitialized() {
			return false
		}
	}

	return true
}
