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
	devicePlugins []GenericDevice
	host          string
	maxDevices    int
	backoff       []time.Duration
	virtConfig    *virtconfig.ClusterConfig
	hostDevConfigMapInformer cache.SharedIndexInformer

}

func NewDeviceController(host string, maxDevices int, clusterConfig *virtconfig.ClusterConfig, hostDevConfigMapInformer cache.SharedIndexInformer) *DeviceController {
	controller := &DeviceController{
		devicePlugins: []GenericDevice{
			NewGenericDevicePlugin(KVMName, KVMPath, maxDevices, false),
			NewGenericDevicePlugin(TunName, TunPath, maxDevices, true),
			NewGenericDevicePlugin(VhostNetName, VhostNetPath, maxDevices, true),
		},
		host:       host,
		maxDevices: maxDevices,
		backoff:    []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second},
	}
	controller.virtConfig = clusterConfig
	controller.hostDevConfigMapInformer = hostDevConfigMapInformer
	hostDevConfigMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.hostDevAddFunc,
		DeleteFunc: controller.hostDevDeleteFunc,
		UpdateFunc: controller.hostDevUpdateFunc,
	})

	return controller
}

func (c *DeviceController) hostDevAddFunc(obj interface{}) {
	logger := log.DefaultLogger()
	logger.Infof("in hostDevAddFunc, obj: %v", obj)
	hostDevs := c.virtConfig.GetPermittedHostDevices()
	logger.Infof("got hostDevs: %v", hostDevs)
}

func (c *DeviceController) hostDevDeleteFunc(obj interface{}) {
	logger := log.DefaultLogger()
	logger.Infof("in hostDevDeleteFunc, obj: %v", obj)
	hostDevs := c.virtConfig.GetPermittedHostDevices()
	logger.Infof("got hostDevs: %v", hostDevs)

}

func (c *DeviceController) hostDevUpdateFunc(oldObj, newObj interface{}) {
	logger := log.DefaultLogger()
	logger.Infof("in hostDevUpdateFunc, oldObj: %v, newObj: %v", oldObj, newObj)
	hostDevs := c.virtConfig.GetPermittedHostDevices()
	logger.Infof("got hostDevs: %v", hostDevs)
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

// addPermittedHostDevicePlugins will add device pluging for permitted devices which are present on the node
func (c *DeviceController) addPermittedHostDevicePlugins() {
	if hostDevs := c.virtConfig.GetPermittedHostDevices(); hostDevs != nil {
		supportedPCIDeviceMap := make(map[string]string)
		if len(hostDevs.PciHostDevices) != 0 {
			for _, pciDev := range hostDevs.PciHostDevices {
				// do not add a device plugin for this resource if it's being provided via an external device plugin
				if !pciDev.ExternalResourceProvider {
					supportedPCIDeviceMap[pciDev.Selector] = pciDev.ResourceName
				}
			}
			pciHostDevices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
			pciDevicePlugins := []GenericDevice{}
			for pciID, pciDevices := range pciHostDevices {
				pciResourceName := supportedPCIDeviceMap[pciID]
				pciDevicePlugins = append(pciDevicePlugins, NewPCIDevicePlugin(pciDevices, pciResourceName))
			}
			c.devicePlugins = append(c.devicePlugins, pciDevicePlugins...)
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
			mdevPlugins := []GenericDevice{}
			for mdevTypeName, mdevUUIDs := range hostMdevs {
				mdevResourceName := supportedMdevsMap[mdevTypeName]
				mdevPlugins = append(mdevPlugins, NewMediatedDevicePlugin(mdevUUIDs, mdevResourceName))
			}
			c.devicePlugins = append(c.devicePlugins, mdevPlugins...)
		}
	}
}

func removeSelectorSpaces(selectorName string) string {
	// The name usually contain spaces which should be replaced with _
	// Such as GRID T4-1Q
	typeNameStr := strings.Replace(string(selectorName), " ", "_", -1)
	typeNameStr = strings.TrimSpace(typeNameStr)
	return typeNameStr

}

func (c *DeviceController) Run(stop chan struct{}) error {
	logger := log.DefaultLogger()
	// Wait for the hostDevConfigMapInformer cache to be synced
	go c.hostDevConfigMapInformer.Run(stop)
	cache.WaitForCacheSync(stop, c.hostDevConfigMapInformer.HasSynced)
	c.addPermittedHostDevicePlugins()
	for _, dev := range c.devicePlugins {
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
