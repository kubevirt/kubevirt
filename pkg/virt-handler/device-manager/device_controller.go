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
	"os"
	"time"

	"kubevirt.io/kubevirt/pkg/log"
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
}

func NewDeviceController(host string, maxDevices int) *DeviceController {
	return &DeviceController{
		devicePlugins: []GenericDevice{
			NewGenericDevicePlugin(KVMName, KVMPath, maxDevices),
			NewGenericDevicePlugin(TunName, TunPath, maxDevices),
			NewGenericDevicePlugin(VhostNetName, VhostNetPath, maxDevices),
		},
		host:       host,
		maxDevices: maxDevices,
	}
}

func (c *DeviceController) nodeHasDevice(devicePath string) bool {
	_, err := os.Stat(devicePath)
	// Since this is a boolean question, any error means "no"
	return (err == nil)
}

func (c *DeviceController) startDevicePlugin(dev GenericDevice, stop chan struct{}) {
	logger := log.DefaultLogger()
	deviceName := dev.GetDeviceName()

	for {
		err := dev.Start(stop)
		if err != nil {
			logger.Errorf("Error starting %s device plugin: %v", deviceName, err)
		}

		select {
		case <-stop:
			// Ok we don't want to re-register
			return
		default:
			// Wait a little bit and re-register
			time.Sleep(1 * time.Second)
		}

	}
}

func (c *DeviceController) Run(stop chan struct{}) error {
	logger := log.DefaultLogger()
	logger.Info("Starting device plugin controller")

	for _, dev := range c.devicePlugins {
		go c.startDevicePlugin(dev, stop)
	}

	<-stop

	logger.Info("Shutting down device plugin controller")
	return nil
}
