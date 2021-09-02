package util

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// the readiness prob always returns a valid answer
var hcoPing healthz.Checker = func(_ *http.Request) error {
	return nil
}

func GetHcoPing() healthz.Checker {
	return hcoPing
}
