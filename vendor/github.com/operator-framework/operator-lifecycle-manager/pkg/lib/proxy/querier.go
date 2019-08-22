package proxy

import (
	corev1 "k8s.io/api/core/v1"
)

// DefaultQuerier does...
func DefaultQuerier() Querier {
	return &defaultQuerier{}
}

// Querier is an interface that wraps the QueryProxyConfig method.
//
// QueryProxyConfig returns the global cluster level proxy env variable(s).
type Querier interface {
	QueryProxyConfig() (proxy []corev1.EnvVar, err error)
}

type defaultQuerier struct {
}

// QueryProxyConfig returns no env variable(s), err is set to nil to indicate
// that the cluster has no global proxy configuration.
func (*defaultQuerier) QueryProxyConfig() (proxy []corev1.EnvVar, err error) {
	return
}
