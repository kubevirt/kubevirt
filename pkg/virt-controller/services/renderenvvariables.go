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

	environmentVariables = append(environmentVariables, k8sv1.EnvVar{
		Name: ENV_VAR_POD_NAME,
		ValueFrom: &k8sv1.EnvVarSource{
			FieldRef: &k8sv1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
	return environmentVariables
}

func kubevirtNetworkResourceName(networkName string) string {
	return fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", networkName)
}
