// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2019 Red Hat, Inc.

//
// This module reads and parses any Multus-CNI configuration
// data provided to a container by the host. This module isolates
// Multus-CNI specifics from the rest of the application.
//

package multusplugin

import (
	"encoding/json"

	"github.com/golang/glog"

	multus "github.com/intel/multus-cni/types"

	"github.com/openshift/app-netutil/pkg/types"
)

const (
	annotKeyNetworkStatus = "k8s.v1.cni.cncf.io/networks-status"
)

type MultusPlugin struct {
	networkStatusSlice []multus.NetworkStatus
}

func ParseAnnotations(annotKey string, annotValue string, multusData *MultusPlugin) {
	// Parse the NetworkStatus added by Multus CNI
	if annotKey == annotKeyNetworkStatus {
		if err := json.Unmarshal([]byte(annotValue), &multusData.networkStatusSlice); err != nil {
			glog.Errorf("Error unmarshal \"%s\": %v", annotKeyNetworkStatus, err)
		}
	}

	return
}

func AppendInterfaceData(multusData *MultusPlugin, ifaceRsp *types.InterfaceResponse) {
	var ifaceData *types.InterfaceData

	glog.Infof("PRINT EACH NetworkStatus - len=%d", len(multusData.networkStatusSlice))
	for _, status := range multusData.networkStatusSlice {
		ifaceData = nil

		glog.Infof("  status:")
		glog.Infof("%v", status)

		// Loop through existing list and determine is this interface has
		// been discovered by some other means.
		for _, interfaceData := range ifaceRsp.Interface {
			if interfaceData.IfName != "" &&
				interfaceData.IfName == status.Interface {

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
				IfName: status.Interface,
				Name:   status.Name,
				Type:   types.INTERFACE_TYPE_UNKNOWN,
				Network: &types.NetworkData{
					IPs: status.IPs,
					Mac: status.Mac,
					DNS: status.DNS,
				},
			}

			ifaceRsp.Interface = append(ifaceRsp.Interface, ifaceData)
		}
	}

	return
}
