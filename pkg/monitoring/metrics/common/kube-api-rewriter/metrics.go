package kubeapirewriter

import (
	"sync"

	kametrics "github.com/deckhouse/kube-api-rewriter/pkg/monitoring/metrics"
	"github.com/deckhouse/kube-api-rewriter/pkg/proxy"
	"github.com/prometheus/client_golang/prometheus"
)

type registererGatherer struct {
	prometheus.Registerer
	prometheus.Gatherer
}

var setupMetricsOnce sync.Once

func SetupMetrics() error {
	setupMetricsOnce.Do(func() {
		kametrics.Registry = registererGatherer{
			Registerer: prometheus.DefaultRegisterer,
			Gatherer:   prometheus.DefaultGatherer,
		}
		proxy.RegisterMetrics()
	})

	return nil
}
