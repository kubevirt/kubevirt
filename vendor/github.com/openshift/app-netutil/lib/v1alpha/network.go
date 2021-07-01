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
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	nritypes "github.com/intel/network-resources-injector/pkg/types"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/openshift/app-netutil/pkg/networkstatus"
	"github.com/openshift/app-netutil/pkg/types"
	"github.com/openshift/app-netutil/pkg/userspace"
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
	annotationPath := filepath.Join(nritypes.DownwardAPIMountPath, nritypes.AnnotationsPath)
	if _, err := os.Stat(annotationPath); err != nil {
		if os.IsNotExist(err) {
			glog.Infof("GetInterfaces: \"annotations\" file: %v does not exist.", annotationPath)
		}
	} else {
		file, err := os.Open(annotationPath)
		if err != nil {
			glog.Errorf("GetInterfaces: Error opening \"annotations\" file: %v ", err)
			return response, err
		}
		defer file.Close()

		// Buffers to store unmarshalled data (from annotations
		// or files) used by app-netutil
		netStatData := &networkstatus.NetStatusData{}
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
					parts[1] = string(parts[1][1 : len(parts[1])-1])

					// Parse any NetworkStatus Annotations. Values will be
					// saved in netStatData structure for later.
					networkstatus.ParseAnnotations(parts[0], parts[1], netStatData)

					// Parse any Userspace Annotations. Values will be
					// saved in usrspData structure for later.
					userplugin.ParseAnnotations(parts[0], parts[1], usrspData)
				}
			}
		}
		// Append any NetworkStatus collected data to the list
		// of interfaces.
		//
		// Because return data is based on NetworkStatus, call NetworkStatus
		// processing first. For efficiency, it assumes no interfaces have been
		// added to list, so it doesn't search existing list to make sure a given
		// interfaces has not already been added.
		networkstatus.AppendInterfaceData(netStatData, response)

		// Append any Userspace collected data to the list
		// of interfaces.
		userplugin.AppendInterfaceData(usrspData, response)
	}

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
				found := false
				// DeviceInfo in the NetworkStatus annotation also has PCI Address.
				// So skip if PCI Address already found.
				for _, ifaceData := range response.Interface {
					if ifaceData.NetworkStatus.DeviceInfo != nil {
						if ifaceData.NetworkStatus.DeviceInfo.Pci != nil &&
							strings.EqualFold(ifaceData.NetworkStatus.DeviceInfo.Pci.PciAddress, id) {
							// PCI Address in ENV matched that in DeviceInfo. Mark as SR-IOV.
							ifaceData.DeviceType = types.INTERFACE_TYPE_SRIOV
							found = true
							break
						}
						if ifaceData.NetworkStatus.DeviceInfo.Vdpa != nil &&
							strings.EqualFold(ifaceData.NetworkStatus.DeviceInfo.Vdpa.PciAddress, id) {
							// PCI Address in ENV matched that in DeviceInfo.
							// Leave the vDPA device and skip processing of the SR-IOV Interface
							found = true
							break
						}
					}
				}
				if found {
					glog.Infof("     Skip Adding ID:%v", id)
				} else {
					glog.Infof("     Adding ID:%v", id)
					pciAddressSlice = append(pciAddressSlice, id)
				}
			}
		}
	}

	// Determine how many detected interfaces with type "unknown"
	var unknownCnt int
	for _, ifaceData := range response.Interface {
		if ifaceData.DeviceType == types.INTERFACE_TYPE_UNKNOWN {
			unknownCnt++
		}
	}

	var pciIndex int
	for _, ifaceData := range response.Interface {
		if ifaceData.DeviceType == types.INTERFACE_TYPE_UNKNOWN {
			// If there are more "unknown" interface types than there are
			// PCI interfaces not in the list, then mark the "default"
			// interface as a host interface.
			if ifaceData.NetworkStatus.Default && unknownCnt > len(pciAddressSlice) {
				ifaceData.DeviceType = types.INTERFACE_TYPE_HOST
				unknownCnt--
				glog.Infof("%s is the \"default\" interface, mark as \"%s\"",
					ifaceData.NetworkStatus.Interface, ifaceData.DeviceType)
			} else if pciIndex < len(pciAddressSlice) {
				// Since type was "unknown" and there are PCI interfaces not yet
				// in the list, add the PCI interfaces one by one.
				if ifaceData.NetworkStatus.DeviceInfo == nil {
					ifaceData.DeviceType = types.INTERFACE_TYPE_SRIOV
					unknownCnt--
					ifaceData.NetworkStatus.DeviceInfo = &nettypes.DeviceInfo{
						Type:    nettypes.DeviceInfoTypePCI,
						Version: nettypes.DeviceInfoVersion,
						Pci: &nettypes.PciDevice{
							PciAddress: pciAddressSlice[pciIndex],
						},
					}
					pciIndex++
					glog.Infof("%s was \"unknown\", mark as \"%s\"",
						ifaceData.NetworkStatus.Interface, ifaceData.DeviceType)
				} else {
					glog.Warningf("%s was \"unknown\", but DeviceInfo exists with type \"%s\"",
						ifaceData.NetworkStatus.Interface, ifaceData.NetworkStatus.DeviceInfo.Type)
				}
			} else {
				// Since there are no more PCI interfaces not in the list, and the
				// type is unknown, mark this interface as "host".
				ifaceData.DeviceType = types.INTERFACE_TYPE_HOST
				unknownCnt--
				glog.Infof("%s was \"unknown\", mark as \"%s\"",
					ifaceData.NetworkStatus.Interface, ifaceData.DeviceType)
			}
		}
	}

	// PCI Address found that did not match an existing interface in the
	// NetworkStatus annotation so add to list.
	if pciIndex < len(pciAddressSlice) {
		for _, pciAddr := range pciAddressSlice[pciIndex:] {
			ifaceData := &types.InterfaceData{
				DeviceType: types.INTERFACE_TYPE_SRIOV,
				NetworkStatus: nettypes.NetworkStatus{
					DeviceInfo: &nettypes.DeviceInfo{
						Type:    nettypes.DeviceInfoTypePCI,
						Version: nettypes.DeviceInfoVersion,
						Pci: &nettypes.PciDevice{
							PciAddress: pciAddr,
						},
					},
				},
			}
			response.Interface = append(response.Interface, ifaceData)

			glog.Infof("Adding %s as new interface because no other matches, type \"%s\"",
				pciAddr, ifaceData.DeviceType)
		}
	}

	glog.Infof("RESPONSE:")
	for _, ifaceData := range response.Interface {
		glog.Infof("%v", ifaceData)
	}

	return response, err
}
