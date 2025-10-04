package operatormetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type RegistryFunc func(c prometheus.Collector) error
type UnregisterFunc func(c prometheus.Collector) bool

// Register is the function used to register metrics and collectors by this package.
var Register RegistryFunc = prometheus.Register

// Unregister is the function used to unregister metrics and collectors by this package.
var Unregister UnregisterFunc = prometheus.Unregister
