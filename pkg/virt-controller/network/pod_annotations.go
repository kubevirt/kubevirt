package network

import (
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func GeneratePodAnnotations(networks []virtv1.Network, interfaces []virtv1.Interface, multusStatusAnnotation string) map[string]string {
	newAnnotations := map[string]string{}
	if vmispec.SRIOVInterfaceExist(interfaces) {
		networkPCIMapAnnotationValue := deviceinfo.CreateNetworkPCIAnnotationValue(
			networks, interfaces, multusStatusAnnotation,
		)
		newAnnotations[deviceinfo.NetworkPCIMapAnnot] = networkPCIMapAnnotationValue
	}
	networkDeviceInfoAnnotation := deviceinfo.CreateNetworkDeviceInfoAnnotationValue(
		networks, interfaces, multusStatusAnnotation,
	)

	newAnnotations[deviceinfo.NetworkDeviceInfoMapAnnot] = networkDeviceInfoAnnotation

	return newAnnotations
}
