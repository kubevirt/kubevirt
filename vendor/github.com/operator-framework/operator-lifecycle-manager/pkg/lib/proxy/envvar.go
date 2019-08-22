package proxy

import (
	apiconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// HTTP_PROXY is the URL of the proxy for HTTP requests.
	// Empty means unset and will not result in an env var.
	envHTTPProxyName = "HTTP_PROXY"

	// HTTPS_PROXY is the URL of the proxy for HTTPS requests.
	// Empty means unset and will not result in an env var.
	envHTTPSProxyName = "HTTPS_PROXY"

	// NO_PROXY is the list of domains for which the proxy should not be used.
	// Empty means unset and will not result in an env var.
	envNoProxyName = "NO_PROXY"
)

var (
	allProxyEnvVarNames = []string{
		envHTTPProxyName,
		envHTTPSProxyName,
		envNoProxyName,
	}
)

// ToEnvVar accepts a config Proxy object and returns an array of all three
// proxy variables with values.
//
// Please note that the function uses the status of the Proxy object to rea the
// proxy env variables. It's because OpenShift validates the proxy variables in
// spec and writes them back to status.
// As a consumer we should be reading off of proxy.status.
func ToEnvVar(proxy *apiconfigv1.Proxy) []corev1.EnvVar {
	return []corev1.EnvVar{
		corev1.EnvVar{
			Name:  envHTTPProxyName,
			Value: proxy.Status.HTTPProxy,
		},
		corev1.EnvVar{
			Name:  envHTTPSProxyName,
			Value: proxy.Status.HTTPSProxy,
		},
		corev1.EnvVar{
			Name:  envNoProxyName,
			Value: proxy.Status.NoProxy,
		},
	}
}
