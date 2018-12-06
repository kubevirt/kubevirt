package util

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"kubevirt.io/containerized-data-importer/pkg/keys"
)

// StartPrometheusEndpoint starts an http server providing a prometheus endpoint using the passed
// in directory to store the self signed certificates that will be generated before starting the
// http server.
func StartPrometheusEndpoint(certsDirectory string) {
	keyFile, certFile, err := keys.GenerateSelfSignedCert(certsDirectory, "cloner_target", "pod")
	if err != nil {
		return
	}
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServeTLS(":8443", certFile, keyFile, nil); err != nil {
			return
		}
	}()
}
