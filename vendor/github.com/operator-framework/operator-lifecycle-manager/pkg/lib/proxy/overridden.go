package proxy

import (
	corev1 "k8s.io/api/core/v1"
)

// IsOverridden returns true if the given container overrides proxy env variable(s).
// We apply the following rule:
//   If a container already defines any of the proxy env variable then it
//   overrides all of these.
func IsOverridden(envVar []corev1.EnvVar) (overrides bool) {
	for _, envVarName := range allProxyEnvVarNames {
		_, found := find(envVar, envVarName)
		if found {
			overrides = true
			return
		}
	}

	return
}

func find(proxyEnvVar []corev1.EnvVar, name string) (envVar *corev1.EnvVar, found bool) {
	for i := range proxyEnvVar {
		if name == proxyEnvVar[i].Name {
			// Environment variable names are case sensitive.
			found = true
			envVar = &proxyEnvVar[i]

			break
		}
	}

	return
}
