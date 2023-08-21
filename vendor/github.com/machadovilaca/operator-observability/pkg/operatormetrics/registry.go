package operatormetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type RegistryFunc func(c prometheus.Collector) error

// Register is the function used to register metrics and collectors by this package.
var Register RegistryFunc = prometheus.Register
