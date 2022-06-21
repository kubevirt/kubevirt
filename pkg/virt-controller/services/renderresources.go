package services

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ResourceRenderer struct {
	vmLimits           k8sv1.ResourceList
	vmRequests         k8sv1.ResourceList
	calculatedLimits   k8sv1.ResourceList
	calculatedRequests k8sv1.ResourceList
}

func NewResourceRenderer(vmLimits k8sv1.ResourceList, vmRequests k8sv1.ResourceList) *ResourceRenderer {
	limits := map[k8sv1.ResourceName]resource.Quantity{}
	requests := map[k8sv1.ResourceName]resource.Quantity{}
	copyResources(vmLimits, limits)
	copyResources(vmRequests, requests)
	return &ResourceRenderer{
		vmLimits:           limits,
		vmRequests:         requests,
		calculatedLimits:   map[k8sv1.ResourceName]resource.Quantity{},
		calculatedRequests: map[k8sv1.ResourceName]resource.Quantity{},
	}
}

func (rr *ResourceRenderer) Limits() k8sv1.ResourceList {
	podLimits := map[k8sv1.ResourceName]resource.Quantity{}
	copyResources(rr.calculatedLimits, podLimits)
	copyResources(rr.vmLimits, podLimits)
	return podLimits
}

func (rr *ResourceRenderer) Requests() k8sv1.ResourceList {
	podRequests := map[k8sv1.ResourceName]resource.Quantity{}
	copyResources(rr.calculatedRequests, podRequests)
	copyResources(rr.vmRequests, podRequests)
	return podRequests
}

func copyResources(srcResources, dstResources k8sv1.ResourceList) {
	for key, value := range srcResources {
		dstResources[key] = value
	}
}
