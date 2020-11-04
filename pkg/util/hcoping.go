package util

import (
	"errors"

	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var hcoReady bool

var hcoPing healthz.Checker

func IsReady() bool {
	return hcoReady
}

func SetReady(ready bool) {
	hcoReady = ready
}

func GetHcoPing() healthz.Checker {
	return hcoPing
}

func hcoChecker(_ *http.Request) error {
	if hcoReady {
		return nil
	}
	hcoNotReady := errors.New("HCO is not ready")
	return hcoNotReady
}

func init() {
	hcoReady = true
	hcoPing = hcoChecker
}
