// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2019 Red Hat, Inc.

//
// This module reads and parses the NetworkStatus annotation
// provided to a container by the host. This module isolates
// Network Status specifics from the rest of the application.
//

package networkstatus

import (
	"encoding/json"

	"github.com/golang/glog"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/openshift/app-netutil/pkg/types"
)

const (
	annotKeyNetworkStatus = "k8s.v1.cni.cncf.io/network-status"
)

type NetStatusData struct {
	networkStatusSlice []nettypes.NetworkStatus
}

func ParseAnnotations(annotKey string, annotValue string, netStatData *NetStatusData) {
	// Parse the NetworkStatus annotation
	if annotKey == annotKeyNetworkStatus {
		if err := json.Unmarshal([]byte(annotValue), &netStatData.networkStatusSlice); err != nil {
			glog.Errorf("Error unmarshal \"%s\": %v", annotKeyNetworkStatus, err)
		}
	}
}

func AppendInterfaceData(netStatData *NetStatusData, ifaceRsp *types.InterfaceResponse) {
	var ifaceData *types.InterfaceData

	glog.Infof("PRINT EACH NetworkStatus - len=%d", len(netStatData.networkStatusSlice))
	for _, status := range netStatData.networkStatusSlice {
		ifaceData = nil

		glog.Infof("  status:")
		glog.Infof("%v", status)

		// For efficiency, assume no interfaces have been added to list,
		// so don't search existing list to make sure this interfaces has
		// already been added.
		ifaceData = &types.InterfaceData{
			NetworkStatus: status,
		}

		if ifaceData.NetworkStatus.DeviceInfo != nil {
			ifaceData.DeviceType = ifaceData.NetworkStatus.DeviceInfo.Type
		} else {
			ifaceData.DeviceType = types.INTERFACE_TYPE_UNKNOWN
		}

		ifaceRsp.Interface = append(ifaceRsp.Interface, ifaceData)
	}
}
