// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2019 Red Hat, Inc.

//
// This module reads and parses any configuration data provided
// to a container by the host. This module manages the
// file operations and the mapping between the data format
// of the provided configuration data and the data format used
// by app-netutil.
//
// Currently, configuration data can be passed to the container
// thru Environmental Variables, Annotations, or shared files.
//

package apputil

import (
	"bufio"
	"os"
	"strings"

	"github.com/golang/glog"

	"github.com/openshift/app-netutil/pkg/types"
	"github.com/openshift/app-netutil/pkg/multus"
	"github.com/openshift/app-netutil/pkg/userspace"
)


const (
	filePathAnnotation = "/etc/podnetinfo/annotations"
	filePathLabel = "/etc/podnetinfo/labels"
)

//
// Types
//

//
// API Functions
//
func GetInterfaces() (*types.InterfaceResponse, error) {
	glog.Infof("GetInterfaces: ENTER")

	response := &types.InterfaceResponse{}

	// Open Annotations File
	glog.Infof("GetInterfaces: Open %s", filePathAnnotation)
	file, err := os.Open(filePathAnnotation)
	if err != nil {
		glog.Errorf("GetInterfaces: Error opening \"annotations\" file: %v", err)
		return response, err
	}
	defer file.Close()

	// Buffers to store unmarshalled data (from annotations
	// or files) used by app-netutil
	multusData := &multusplugin.MultusPlugin{}
	usrspData := &userplugin.UserspacePlugin{}

	//
	// Parse the file into individual annotations
	//
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		status := strings.Split(string(line), "\n")

		// Loop through each annotation
		for _, s := range status {
			glog.Infof("  s-%v", s)
			parts := strings.Split(string(s), "=")

			// DEBUG
			glog.Infof("  PartsLen-%d", len(parts))
			if len(parts) >= 1 {
				glog.Infof("  parts[0]-%s", parts[0])
			}

			if len(parts) == 2 {

				// Remove the Indent from the original marshalling
				parts[1] = strings.Replace(string(parts[1]), "\\n", "", -1)
				parts[1] = strings.Replace(string(parts[1]), "\\", "", -1)
				parts[1] = strings.Replace(string(parts[1]), " ", "", -1)
				parts[1] = string(parts[1][1:len(parts[1])-1])

				// Parse any Mults Annotations. Values will be
				// saved in multusData structure for later.
				multusplugin.ParseAnnotations(parts[0], parts[1], multusData)

				// Parse any Userspace Annotations. Values will be
				// saved in usrspData structure for later.
				userplugin.ParseAnnotations(parts[0], parts[1], usrspData)
			}
		}
	}

	// Append any NetworkStatus collected data to the list
	// of interfaces.
	multusplugin.AppendInterfaceData(multusData, response)

	// Append any Userspace collected data to the list
	// of interfaces.
	userplugin.AppendInterfaceData(usrspData, response)

	// PCI Address for SR-IOV Interfaces are found in
	// Environmental Variables. Search through them to
	// see if any can be found.
	glog.Infof("PROCESS ENV:")
	envResponse, err := getEnv()
	if err != nil {
		glog.Errorf("GetInterfaces: Error calling getEnv: %v", err)
		return nil, err
	}
	pciAddressSlice := []string{}
	for k, v := range envResponse.Envs {
		if strings.HasPrefix(k, "PCIDEVICE") {
			glog.Infof("  k:%v v:%v", k, v)
			valueParts := strings.Split(string(v), ",")
			for _, id := range valueParts {
				glog.Infof("     Adding ID:%v", id)
				pciAddressSlice = append(pciAddressSlice, id)
			}
		}
	}

	var pciIndex int
	for _, interfaceData := range response.Interface {
		if interfaceData.Type == types.INTERFACE_TYPE_UNKNOWN {
			if interfaceData.Network.Default {
				glog.Infof(" Set Interface to kernel: %s", interfaceData.IfName)
				interfaceData.Type = types.INTERFACE_TYPE_KERNEL
			} else {
				// TBD: Need a better way to determine if SR_IOV or not.
				glog.Infof(" Set Interface to SR-IOV: %s  PCIIndex=%d Len=%d", interfaceData.IfName, pciIndex, len(pciAddressSlice))
				interfaceData.Type = types.INTERFACE_TYPE_SRIOV
				if pciIndex < len(pciAddressSlice) {
					interfaceData.Sriov = &types.SriovData{
						PciAddress: pciAddressSlice[pciIndex],
					}
					pciIndex++
					glog.Infof(" Added PCI Address: %s", interfaceData.Sriov.PciAddress)
				} else {
					glog.Warningf("More SR-IOV interfaces detected than PCI addresses - %s", interfaceData.IfName)
				}
			}
		}
	}

	glog.Infof("RESPONSE:")
	for _, interfaceData := range response.Interface {
		glog.Infof("%v", interfaceData)
	}

	return response, err
}
