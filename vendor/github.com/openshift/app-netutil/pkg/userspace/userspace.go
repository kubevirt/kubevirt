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
	"github.com/intel/userspace-cni-network-plugin/pkg/annotations"
	usrsptypes "github.com/intel/userspace-cni-network-plugin/pkg/types"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openshift/app-netutil/pkg/types"
)

const (
//AnnotKeyUsrspConfigData = "userspace/configuration-data"
//AnnotKeyUsrspMappedDir = "userspace/mapped-dir"
)

type UserspacePlugin struct {
	configDataSlice []usrsptypes.ConfigurationData
	mappedDir       string
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
}

func AppendInterfaceData(usrspData *UserspacePlugin, ifaceRsp *types.InterfaceResponse) {
	var ifaceData *types.InterfaceData

	glog.Infof("PRINT EACH Userspace MappedDir")
	glog.Infof("  usrspMappedDir:")
	glog.Infof("%v", usrspData.mappedDir)

	glog.Infof("PRINT EACH Userspace ConfigData")
	for _, configData := range usrspData.configDataSlice {
		ifaceData = nil
		foundMatch := false

		glog.Infof("  configData:")
		glog.Infof("%v", configData)

		if usrspData.mappedDir == "" {
			glog.Warningf("Error: \"%s\" annotation not available but required for Userspace interfaces", annotations.AnnotKeyUsrspMappedDir)
		}

		// Loop through existing list and determine is this interface has
		// been discovered by some other means (like using NetworkStatus annotation)
		for _, interfaceData := range ifaceRsp.Interface {
			glog.Infof("TEST: NetworkStatus.Interface=%s configData.IfName=%s", interfaceData.NetworkStatus.Interface, configData.IfName)
			if interfaceData.NetworkStatus.Interface != "" &&
				interfaceData.NetworkStatus.Interface == configData.IfName {
				ifaceData = interfaceData
				foundMatch = true
				glog.Infof("  FOUND MATCH")
				break
			}
		}

		// If current interface is not already in the list, then
		// create a new instance and add it to the list.
		if ifaceData == nil {
			glog.Infof("  NO MATCH: Create New Instance")

			ifaceData = &types.InterfaceData{
				DeviceType: types.INTERFACE_TYPE_UNKNOWN,
				NetworkStatus: nettypes.NetworkStatus{
					Name:      configData.Name,
					Interface: configData.IfName,
					Default:   false,
					DNS: nettypes.DNS{
						Nameservers: configData.IPResult.DNS.Nameservers,
						Domain:      configData.IPResult.DNS.Domain,
						Search:      configData.IPResult.DNS.Search,
						Options:     configData.IPResult.DNS.Options,
					},
				},
			}

			// Convert the IPResult structure to the NetworkStatus format
			for _, ipconfig := range configData.IPResult.IPs {
				if ipconfig.Version == "4" && ipconfig.Address.IP.To4() != nil {
					ifaceData.NetworkStatus.IPs = append(ifaceData.NetworkStatus.IPs, ipconfig.Address.IP.String())
				}
				if ipconfig.Version == "6" && ipconfig.Address.IP.To16() != nil {
					ifaceData.NetworkStatus.IPs = append(ifaceData.NetworkStatus.IPs, ipconfig.Address.IP.String())
				}
			}
		}

		// If the DeviceInfo data was not included in the NetworkStatus,
		// then map the Userspace data to the DeviceInfo.
		if ifaceData.NetworkStatus.DeviceInfo == nil {
			if configData.Config.IfType == "vhostuser" {
				ifaceData.NetworkStatus.DeviceInfo = &nettypes.DeviceInfo{
					Type:    nettypes.DeviceInfoTypeVHostUser,
					Version: nettypes.DeviceInfoVersion,
					VhostUser: &nettypes.VhostDevice{
						Mode: configData.Config.VhostConf.Mode,
						Path: usrspData.mappedDir + configData.Config.VhostConf.Socketfile,
					},
				}
			} else if configData.Config.IfType == "memif" {
				ifaceData.NetworkStatus.DeviceInfo = &nettypes.DeviceInfo{
					Type:    nettypes.DeviceInfoTypeMemif,
					Version: nettypes.DeviceInfoVersion,
					Memif: &nettypes.MemifDevice{
						Role: configData.Config.MemifConf.Role,
						Mode: configData.Config.MemifConf.Mode,
						Path: usrspData.mappedDir + configData.Config.MemifConf.Socketfile,
					},
				}
			} else {
				glog.Warningf("Invalid type found for interface %s", configData.IfName)
			}

			if ifaceData.NetworkStatus.DeviceInfo != nil {
				ifaceData.DeviceType = ifaceData.NetworkStatus.DeviceInfo.Type
				if !foundMatch {
					ifaceRsp.Interface = append(ifaceRsp.Interface, ifaceData)
				}
			}
		} else {
			// DeviceInfo found in the NetworkStatus data. Currently
			// don't try to reconcile data coming from two locations.
			glog.Warningf("Userspace interface found in NetworkStatus: %s", configData.IfName)
			glog.Infof("NetworkStatus Data: %v", ifaceData)
			glog.Infof("Userspace Data: %v", configData)
		}
	}
}
