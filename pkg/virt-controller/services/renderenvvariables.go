package services

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
)

type EnvVariablesRenderer struct {
	requiredNetworkResources map[string]string
}

func NewEnvVariablesRenderer(requiredNetworkResources map[string]string) *EnvVariablesRenderer {
	return &EnvVariablesRenderer{
		requiredNetworkResources: requiredNetworkResources,
	}
}

func (evr *EnvVariablesRenderer) Render() []k8sv1.EnvVar {
	var environmentVariables []k8sv1.EnvVar
	for networkName, networkResource := range evr.requiredNetworkResources {
		environmentVariables = append(environmentVariables, k8sv1.EnvVar{
			Name:  kubevirtNetworkResourceName(networkName),
			Value: networkResource,
		})
	}
	return environmentVariables
}

func kubevirtNetworkResourceName(networkName string) string {
	return fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", networkName)
}
