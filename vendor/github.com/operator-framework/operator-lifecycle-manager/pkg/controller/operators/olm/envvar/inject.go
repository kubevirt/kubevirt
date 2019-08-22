package envvar

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

// InjectEnvIntoDeployment injects the proxy env variables specified in
// proxyEnvVar into the container(s) of the given PodSpec.
//
// If any Container in PodSpec already defines an env variable of the same name
// as any of the proxy env variables then it
func InjectEnvIntoDeployment(podSpec *corev1.PodSpec, envVars []corev1.EnvVar) error {
	if podSpec == nil {
		return errors.New("no pod spec provided")
	}

	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		container.Env = merge(container.Env, envVars)
	}

	return nil
}

func merge(containerEnvVars []corev1.EnvVar, newEnvVars []corev1.EnvVar) (merged []corev1.EnvVar) {
	merged = containerEnvVars

	for _, newEnvVar := range newEnvVars {
		existing, found := find(containerEnvVars, newEnvVar.Name)
		if !found {
			if newEnvVar.Value != "" {
				merged = append(merged, corev1.EnvVar{
					Name:  newEnvVar.Name,
					Value: newEnvVar.Value,
				})
			}

			continue
		}

		existing.Value = newEnvVar.Value
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
