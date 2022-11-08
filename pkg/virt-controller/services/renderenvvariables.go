package services

import (
	"fmt"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type EnvVariablesRenderer struct {
	logVerbosity             uint
	requiredNetworkResources map[string]string
	labels                   map[string]string
}

func NewEnvVariablesRenderer(requiredNetworkResources map[string]string, labels map[string]string, logVerbosity uint) *EnvVariablesRenderer {
	return &EnvVariablesRenderer{
		logVerbosity:             logVerbosity,
		requiredNetworkResources: requiredNetworkResources,
		labels:                   labels,
	}
}

func (evr *EnvVariablesRenderer) Render() ([]k8sv1.EnvVar, error) {
	var environmentVariables []k8sv1.EnvVar
	for networkName, networkResource := range evr.requiredNetworkResources {
		environmentVariables = append(environmentVariables, k8sv1.EnvVar{
			Name:  kubevirtNetworkResourceName(networkName),
			Value: networkResource,
		})
	}

	clusterWideLoggingLevelForVMI, err := evr.overrideClusterWideLoggingLevelForVMI()
	if err != nil {
		return nil, err
	}
	if clusterWideLoggingLevelForVMI != nil {
		environmentVariables = append(environmentVariables, *clusterWideLoggingLevelForVMI)
	}

	if libvirtLoggingConfig := evr.configureSpecificLauncherPodLogging(
		debugLogs, ENV_VAR_LIBVIRT_DEBUG_LOGS, EXT_LOG_VERBOSITY_THRESHOLD); libvirtLoggingConfig != nil {
		environmentVariables = append(environmentVariables, *libvirtLoggingConfig)
	}

	if virtioFSDebugLogConfig := evr.configureSpecificLauncherPodLogging(
		virtiofsDebugLogs, ENV_VAR_VIRTIOFSD_DEBUG_LOGS, EXT_LOG_VERBOSITY_THRESHOLD); virtioFSDebugLogConfig != nil {
		environmentVariables = append(environmentVariables, *virtioFSDebugLogConfig)
	}

	environmentVariables = append(environmentVariables, k8sv1.EnvVar{
		Name: ENV_VAR_POD_NAME,
		ValueFrom: &k8sv1.EnvVarSource{
			FieldRef: &k8sv1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
	return environmentVariables, nil
}

func (evr *EnvVariablesRenderer) overrideClusterWideLoggingLevelForVMI() (*k8sv1.EnvVar, error) {
	if verbosity, isSet := evr.labels[logVerbosity]; isSet || evr.logVerbosity != virtconfig.DefaultVirtLauncherLogVerbosity {
		// Override the cluster wide verbosity level if a specific value has been provided for this VMI
		verbosityStr := fmt.Sprint(evr.logVerbosity)
		if isSet {
			verbosityStr = verbosity

			verbosityInt, err := strconv.Atoi(verbosity)
			if err != nil {
				return nil, fmt.Errorf("verbosity %s cannot cast to int: %v", verbosity, err)
			}

			evr.logVerbosity = uint(verbosityInt)
		}
		return &k8sv1.EnvVar{Name: ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, Value: verbosityStr}, nil
	}
	return nil, nil
}

func kubevirtNetworkResourceName(networkName string) string {
	return fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", networkName)
}

func (evr *EnvVariablesRenderer) configureSpecificLauncherPodLogging(labelName string, envVarName string, verbosityThreshold uint) *k8sv1.EnvVar {
	if labelValue, ok := evr.labels[labelName]; (ok && strings.EqualFold(labelValue, "true")) || evr.logVerbosity > verbosityThreshold {
		return &k8sv1.EnvVar{Name: envVarName, Value: "1"}
	}
	return nil
}
