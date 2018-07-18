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

package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type ResourceInterface struct {
	Configuration map[string]*PodInterfaceConfiguration
}

const environmentVariablePrefix = "NETWORK_INTERFACE_RESOURCES_"

type PodInterfaceConfiguration struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

type ResourceConfiguration struct {
	Name       string                      `json:"name"`
	Interfaces []PodInterfaceConfiguration `json:"interfaces"`
}

func (l *ResourceInterface) cachePodInterfacesFromEnvironment() {
	// TODO: no actual caching is happening here, so we can refresh. should e add rate limit here?
	if l.Configuration == nil {
		l.Configuration = make(map[string]*PodInterfaceConfiguration)
	}

	// go through all environment variables published by any of the device plugins
	// compliant with the "resource" network API
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if len(pair) != 2 {
			log.Log.Warningf("Failed to parse environment variable key-value pair: %s", e)
			continue
		}
		varName := pair[0]
		varValue := pair[1]
		if strings.HasPrefix(varName, environmentVariablePrefix) {
			// environment variable has the prefix expected from a compliant device plugins
			var conf ResourceConfiguration
			err := json.Unmarshal([]byte(varValue), &conf)
			if err != nil {
				log.Log.Warningf("Failed to parse configuration of environment variable %s, due to: %s", varName, err.Error())
				continue
			}

			// fetch the network name from the URL resource name
			pair = strings.Split(conf.Name, "/")
			if len(pair) != 2 {
				log.Log.Warningf("Resource name: %s is not in the form: <device-plugin>/<resource>", conf.Name)
				continue
			}
			resourceName := pair[1]

			if len(conf.Interfaces) == 0 {
				log.Log.Warningf("Environment variable %s, contains no pod interfaces", varName)
				continue
			}

			// if configuration does not exists for a resource
			// just take the first pod interface of the first device plugin which configures this network
			if l.Configuration[resourceName] == nil {
				log.Log.Infof("Add new pod interface from env var: '%s' with resource name: '%s' and pod interface name: '%s'",
					varName, resourceName, conf.Interfaces[0].Name)
				l.Configuration[resourceName] = &conf.Interfaces[0]
			}
		}
	}
}

// find the virtual machine interface definition in the list
func getInterfaceByName(ifaces []api.Interface, name string) *api.Interface {
	for _, iface := range ifaces {
		if iface.Alias.Name == name {
			return &iface
		}
	}
	return nil
}

// Unplug disconnects the device plugin device from the virtual machine
func (l *ResourceInterface) Unplug() {}

// Plug connect a device plugin device to the virtual machine
func (l *ResourceInterface) Plug(iface *v1.Interface, network *v1.Network, domain *api.Domain) error {

	precond.MustNotBeNil(domain)

	initHandler()

	l.cachePodInterfacesFromEnvironment()

	// find the required network name in an environment variable published by any of the device plugins
	if l.Configuration[network.Name] == nil {
		// no configuration exists for that resource name from any environment variable
		errMsg := fmt.Sprintf("Resource configuration was not found for network '%s'", network.Name)
		err := errors.New(errMsg)
		log.Log.Reason(err).Error(errMsg)
		return err
	}

	log.Log.Infof("Configuration for network '%s' was found", iface.Name)
	podIf := l.Configuration[network.Name]

	// check that the pod actually have the published interface
	link, err := Handler.LinkByName(podIf.Name)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to fetch pod interface: '%s'", podIf.Name)
		return err
	}

	// find the VM interface configuration as received from API
	vmInterface := getInterfaceByName(domain.Spec.Devices.Interfaces, iface.Name)
	if vmInterface == nil {
		// TODO: this should not happen, should we panic here?
		errMsg := fmt.Sprintf("Domain configuration was not found for network: '%s'", network.Name)
		err := errors.New(errMsg)
		log.Log.Reason(err).Error(errMsg)
		return err
	}

	// automatically determine binding according to device plugin configuration
	if podIf.Protocol == "Ethernet" {
		// Create a bridge connecting the pod interface with the VM
		bridge := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{Name: vmInterface.Source.Bridge},
		}

		err = Handler.LinkAdd(bridge)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to create bridge: '%s'", bridge.Name)
			return err
		}

		err = netlink.LinkSetMaster(link, bridge)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to set pod interface: '%s' as master for the bridge: '%s'", link.Attrs().Name, bridge.Name)
			netlink.LinkDel(bridge)
			return err
		}

		err = Handler.LinkSetUp(bridge)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to bring bridge: '%s'", bridge.Name)
			netlink.LinkDel(bridge)
			return err
		}
	} else if podIf.Protocol == "PCI" {
		errMsg := fmt.Sprint("PCI passthrough not supported")
		err := errors.New(errMsg)
		log.Log.Reason(err).Error(errMsg)
		return err
	} else {
		errMsg := fmt.Sprint("Unknown protocol", podIf.Protocol)
		err := errors.New(errMsg)
		log.Log.Reason(err).Error(errMsg)
		return err
	}
	// TODO: what if binding mechanism was set in the network configuration?
	return nil
}
