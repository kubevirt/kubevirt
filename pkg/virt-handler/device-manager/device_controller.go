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
	"time"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	KVMPath      = "/dev/kvm"
	KVMName      = "kvm"
	TunPath      = "/dev/net/tun"
	TunName      = "tun"
	VhostNetPath = "/dev/vhost-net"
	VhostNetName = "vhost-net"
)

type DeviceController struct {
	devicePlugins            map[string]GenericDevice
	host                     string
	maxDevices               int
	backoff                  []time.Duration
	virtConfig               *virtconfig.ClusterConfig
	hostDevConfigMapInformer cache.SharedIndexInformer
	stop                     chan struct{}
}

func getPermanentHostDevicePlugins(maxDevices int) map[string]GenericDevice {
	return map[string]GenericDevice{
		KVMName:      NewGenericDevicePlugin(KVMName, KVMPath, maxDevices, false),
		TunName:      NewGenericDevicePlugin(TunName, TunPath, maxDevices, true),
		VhostNetName: NewGenericDevicePlugin(VhostNetName, VhostNetPath, maxDevices, true),
	}
}

func NewDeviceController(host string, maxDevices int, clusterConfig *virtconfig.ClusterConfig, hostDevConfigMapInformer cache.SharedIndexInformer) *DeviceController {
	controller := &DeviceController{
		devicePlugins: getPermanentHostDevicePlugins(maxDevices),
		host:          host,
		maxDevices:    maxDevices,
		backoff:       []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second},
	}
	controller.virtConfig = clusterConfig
	controller.hostDevConfigMapInformer = hostDevConfigMapInformer
	hostDevConfigMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.hostDevAddDeleteFunc,
		DeleteFunc: controller.hostDevAddDeleteFunc,
		UpdateFunc: controller.hostDevUpdateFunc,
	})

	return controller
}

func (c *DeviceController) hostDevAddDeleteFunc(obj interface{}) {
	c.refreshPermittedDevices()
}

func (c *DeviceController) hostDevUpdateFunc(oldObj, newObj interface{}) {
	c.refreshPermittedDevices()
}

func (c *DeviceController) nodeHasDevice(devicePath string) bool {
	_, err := os.Stat(devicePath)
	// Since this is a boolean question, any error means "no"
	return (err == nil)
}

func (c *DeviceController) startDevicePlugin(dev GenericDevice, stop chan struct{}) {
	logger := log.DefaultLogger()
	deviceName := dev.GetDeviceName()
	logger.Infof("Starting a device pluging for device: %s", deviceName)
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

func (c *DeviceController) stopDevicePlugin(dev GenericDevice) {
	logger := log.DefaultLogger()
	deviceName := dev.GetDeviceName()
	logger.Infof("Stopping a device pluging for device: %s", deviceName)
	err := dev.Stop()
	if err != nil {
		logger.Reason(err).Errorf("Error stopping %s device plugin", deviceName)
	}
}

// updatePermittedHostDevicePlugins will return a map of device plugings for permitted devices which are present on the node
// and a map of restricted devices that should be removed
func (c *DeviceController) updatePermittedHostDevicePlugins() (map[string]GenericDevice, map[string]GenericDevice) {
	devicePluginsToRun := make(map[string]GenericDevice)
	devicePluginsToStop := make(map[string]GenericDevice)
	if hostDevs := c.virtConfig.GetPermittedHostDevices(); hostDevs != nil {
		// generate a map of currently started device plugins
		for resourceName, hostDevDP := range c.devicePlugins {
			devicePluginsToStop[resourceName] = hostDevDP
		}
		supportedPCIDeviceMap := make(map[string]string)
		if len(hostDevs.PciHostDevices) != 0 {
			for _, pciDev := range hostDevs.PciHostDevices {
				// do not add a device plugin for this resource if it's being provided via an external device plugin
				if !pciDev.ExternalResourceProvider {
					supportedPCIDeviceMap[pciDev.Selector] = pciDev.ResourceName
				}
			}
			pciHostDevices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
			for pciID, pciDevices := range pciHostDevices {
				pciResourceName := supportedPCIDeviceMap[pciID]
				// add a device plugin only for new devices
				if _, isRunning := c.devicePlugins[pciResourceName]; !isRunning {
					devicePluginsToRun[pciResourceName] = NewPCIDevicePlugin(pciDevices, pciResourceName)
				} else {
					delete(devicePluginsToStop, pciResourceName)
				}
			}
		}
		if len(hostDevs.MediatedDevices) != 0 {
			supportedMdevsMap := make(map[string]string)
			for _, supportedMdev := range hostDevs.MediatedDevices {
				// do not add a device plugin for this resource if it's being provided via an external device plugin
				if !supportedMdev.ExternalResourceProvider {
					selector := removeSelectorSpaces(supportedMdev.Selector)
					supportedMdevsMap[selector] = supportedMdev.ResourceName
				}
			}

			hostMdevs := discoverPermittedHostMediatedDevices(supportedMdevsMap)
			for mdevTypeName, mdevUUIDs := range hostMdevs {
				mdevResourceName := supportedMdevsMap[mdevTypeName]
				// add a device plugin only for new devices
				if _, isRunning := c.devicePlugins[mdevResourceName]; !isRunning {
					devicePluginsToRun[mdevResourceName] = NewMediatedDevicePlugin(mdevUUIDs, mdevResourceName)
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
func (c *DeviceController) refreshPermittedDevices() error {
	logger := log.DefaultLogger()
	debugDevAdded := []string{}
	debugDevRemoved := []string{}
	// Wait for the hostDevConfigMapInformer cache to be synced
	enabledDevicePlugins, disabledDevicePlugins := c.updatePermittedHostDevicePlugins()

	// start device plugin for newly permitted devices
	for resourceName, dev := range enabledDevicePlugins {
		go c.startDevicePlugin(dev, c.stop)
		c.devicePlugins[resourceName] = dev
		debugDevAdded = append(debugDevAdded, resourceName)
	}
	// remove device plugin for now forbidden devices
	for resourceName, dev := range disabledDevicePlugins {
		staticDPs := getPermanentHostDevicePlugins(0)
		if _, isStaticResource := staticDPs[resourceName]; !isStaticResource {
			go c.stopDevicePlugin(dev)
			delete(c.devicePlugins, resourceName)
			debugDevRemoved = append(debugDevRemoved, resourceName)
		}
	}

	logger.Info("refreshed device plugins for permitted/forbidden host devices")
	logger.Infof("enabled device-pluings for: %v", debugDevAdded)
	logger.Infof("disabled device-pluings for: %v", debugDevRemoved)
	return nil
}

func (c *DeviceController) Run(stop chan struct{}) error {
	logger := log.DefaultLogger()
	c.stop = stop
	// Wait for the hostDevConfigMapInformer cache to be synced
	go c.hostDevConfigMapInformer.Run(stop)
	cache.WaitForCacheSync(stop, c.hostDevConfigMapInformer.HasSynced)
	enabledDevicePlugins, _ := c.updatePermittedHostDevicePlugins()
	for _, dev := range enabledDevicePlugins {
		go c.startDevicePlugin(dev, stop)
	}

	<-stop

	logger.Info("Shutting down device plugin controller")
	return nil
}

func (c *DeviceController) Initialized() bool {
	for _, dev := range c.devicePlugins {
		if !dev.GetInitialized() {
			return false
		}
	}

	return true
}
