package cluster

import (
	secv1 "github.com/openshift/api/security/v1"
	"k8s.io/client-go/discovery"

	"kubevirt.io/client-go/kubecli"
)

func IsOnOpenShift(clientset kubecli.KubevirtClient) (bool, error) {
	_, apis, err := clientset.DiscoveryClient().ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return false, err
	}

	// In case of an error, check if security.openshift.io is the reason (unlikely).
	// If it is, we are obviously on an openshift cluster.
	// Otherwise we can do a positive check.
	if discovery.IsGroupDiscoveryFailedError(err) {
		e := err.(*discovery.ErrGroupDiscoveryFailed)
		if _, exists := e.Groups[secv1.GroupVersion]; exists {
			return true, nil
		}
	}

	for _, api := range apis {
		if api.GroupVersion == secv1.GroupVersion.String() {
			for _, resource := range api.APIResources {
				if resource.Name == "securitycontextconstraints" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
