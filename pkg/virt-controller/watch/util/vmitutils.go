package util

import (
	v1 "kubevirt.io/api/core/v1"
)

type filterFunc func(*v1.VirtualMachineInstance) bool

func Filter(vmis []*v1.VirtualMachineInstance, f filterFunc) []*v1.VirtualMachineInstance {
	filtered := []*v1.VirtualMachineInstance{}
	for _, vmi := range vmis {
		if f(vmi) {
			filtered = append(filtered, vmi)
		}
	}
	return filtered
}
