/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package downwardapi

import v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

type Interface struct {
	Network    string         `json:"network"`
	DeviceInfo *v1.DeviceInfo `json:"deviceInfo,omitempty"`
	Mac        string         `json:"mac,omitempty"`
}

type NetworkInfo struct {
	Interfaces []Interface `json:"interfaces,omitempty"`
}
