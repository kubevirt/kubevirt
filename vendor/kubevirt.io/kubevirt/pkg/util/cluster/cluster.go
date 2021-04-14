package cluster

import (
	"context"
	"encoding/json"
	"strings"

	secv1 "github.com/openshift/api/security/v1"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const (
	OpenShift3Major = 3
	OpenShift4Major = 4
	k8sBaseForOKD   = "1.12"
)

func IsOnOpenShift(clientset kubecli.KubevirtClient) (bool, error) {
	apis, err := clientset.DiscoveryClient().ServerResources()
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

func GetKubernetesVersion(clientset kubecli.KubevirtClient) (string, error) {
	var info k8sversion.Info

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}

	response, err := virtClient.RestClient().Get().AbsPath("/version").DoRaw(context.Background())
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(response, &info); err != nil {
		return "", err
	}

	curVersion := strings.Split(info.GitVersion, "+")[0]
	curVersion = strings.Trim(curVersion, "v")

	return curVersion, nil
}

func GetOpenShiftMajorVersion(clientset kubecli.KubevirtClient) int {
	k8sVersion, err := GetKubernetesVersion(clientset)
	if err != nil {
		log.Log.Errorf("Unable to detect major OpenShift version: %v", err)
		return -1
	}

	if k8sVersion >= k8sBaseForOKD {
		return OpenShift4Major
	}
	return OpenShift3Major
}
