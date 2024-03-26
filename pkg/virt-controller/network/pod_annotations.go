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
 * Copyright the KubeVirt Authors.
 *
 */

package network

import (
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func GeneratePodAnnotations(networks []virtv1.Network, interfaces []virtv1.Interface, multusStatusAnnotation string, bindingPlugins map[string]virtv1.InterfaceBindingPlugin) map[string]string {
	newAnnotations := map[string]string{}
	if vmispec.SRIOVInterfaceExist(interfaces) {
		networkPCIMapAnnotationValue := deviceinfo.CreateNetworkPCIAnnotationValue(
			networks, interfaces, multusStatusAnnotation,
		)
		newAnnotations[deviceinfo.NetworkPCIMapAnnot] = networkPCIMapAnnotationValue
	}
	if vmispec.BindingPluginNetworkWithDeviceInfoExist(interfaces, bindingPlugins) {
		networkDeviceInfoMap, err := deviceinfo.MapBindingPluginNetworkNameToDeviceInfo(networks, interfaces, multusStatusAnnotation, bindingPlugins)
		if err != nil {
			log.Log.Warningf("failed to create network-device-info-map: %v", err)
			networkDeviceInfoMap = map[string]*networkv1.DeviceInfo{}
		}
		networkDeviceInfoAnnotation := downwardapi.CreateNetworkInfoAnnotationValue(networkDeviceInfoMap)
		newAnnotations[downwardapi.NetworkInfoAnnot] = networkDeviceInfoAnnotation
	}

	return newAnnotations
}
