// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2019 Red Hat, Inc.

//
// This module reads and parses any Userspace-CNI configuration
// data provided to a container by the host. This module isolates
// Userspace-CNI specifics from the rest of the application.
//

package userplugin

import (
	"encoding/json"

	"github.com/golang/glog"

	usrsptypes "github.com/intel/userspace-cni-network-plugin/pkg/types"
	"github.com/intel/userspace-cni-network-plugin/pkg/annotations"

	"github.com/openshift/app-netutil/pkg/types"
)

const (
	//AnnotKeyUsrspConfigData = "userspace/configuration-data"
	//AnnotKeyUsrspMappedDir = "userspace/mapped-dir"
)

type UserspacePlugin struct {
	configDataSlice []usrsptypes.ConfigurationData
	mappedDir string
}

func ParseAnnotations(annotKey string, annotValue string, usrspData *UserspacePlugin) {
	// Parse the Configuration Data added by Userspace CNI
	if annotKey == annotations.AnnotKeyUsrspConfigData {
		if err := json.Unmarshal([]byte(annotValue), &usrspData.configDataSlice); err != nil {
			glog.Errorf("Error unmarshal \"%s\": %v", annotations.AnnotKeyUsrspConfigData, err)
		}
	}

	// Parse the Mapped Directory added by Userspace CNI
	if annotKey == annotations.AnnotKeyUsrspMappedDir {
		usrspData.mappedDir = string([]byte(annotValue))
	}

	return
}

func AppendInterfaceData(usrspData *UserspacePlugin, ifaceRsp *types.InterfaceResponse) {
	var ifaceData *types.InterfaceData

	glog.Infof("PRINT EACH Userspace MappedDir")
	glog.Infof("  usrspMappedDir:")
	glog.Infof("%v", usrspData.mappedDir)

	glog.Infof("PRINT EACH Userspace ConfigData")
	for _, configData := range usrspData.configDataSlice {
		ifaceData = nil

		glog.Infof("  configData:")
		glog.Infof("%v", configData)

		if usrspData.mappedDir == "" {
			glog.Warningf("Error: \"%s\" annotation not available but required for Userspace interfaces", annotations.AnnotKeyUsrspMappedDir)
		}

		// Loop through existing list and determine is this interface has
		// been discovered by some other means (like NetworkStatus from Multus)
		for _, interfaceData := range ifaceRsp.Interface {
			if interfaceData.IfName != "" &&
			   interfaceData.IfName == configData.IfName {

				glog.Infof("  MATCH:")
				ifaceData = interfaceData
				break
			}
		}

		// If current interface is not already in the list, then
		// create a new instance and add it to the list.
		if ifaceData == nil {
			glog.Infof("  NO MATCH: Create New Instance")

			ifaceData = &types.InterfaceData{
							IfName: configData.IfName,
							Name: configData.Name,
							Type: types.INTERFACE_TYPE_UNKNOWN,
							Network: &types.NetworkData{
								Default: false,
								DNS: configData.IPResult.DNS,
							},
						}
			// Convert the IPResult structure to the Network Struct used
			// by app-netutil (based on Multus NetworkStatus)
			for _, ipconfig := range configData.IPResult.IPs {
				if ipconfig.Version == "4" && ipconfig.Address.IP.To4() != nil {
					ifaceData.Network.IPs = append(ifaceData.Network.IPs, ipconfig.Address.IP.String())
				}
				if ipconfig.Version == "6" && ipconfig.Address.IP.To16() != nil {
					ifaceData.Network.IPs = append(ifaceData.Network.IPs, ipconfig.Address.IP.String())
				}
			}

			ifaceRsp.Interface = append(ifaceRsp.Interface, ifaceData)
		}

		if ifaceData != nil {
			if configData.Config.IfType == "vhostuser" {
				ifaceData.Type = types.INTERFACE_TYPE_VHOST
				ifaceData.Vhost = &types.VhostData{
					Mode: configData.Config.VhostConf.Mode,
					Socketpath: usrspData.mappedDir + configData.Config.VhostConf.Socketfile,
				}
			} else if configData.Config.IfType == "memif" {
				ifaceData.Type = types.INTERFACE_TYPE_MEMIF
				ifaceData.Memif = &types.MemifData{
					Role: configData.Config.MemifConf.Role,
					Mode: configData.Config.MemifConf.Mode,
					Socketpath: usrspData.mappedDir + configData.Config.MemifConf.Socketfile,
				}
			} else {
				ifaceData.Type = types.INTERFACE_TYPE_INVALID
				glog.Warningf("Invalid type found for interface %s", ifaceData.IfName)
			}
		}
	}
	return
}