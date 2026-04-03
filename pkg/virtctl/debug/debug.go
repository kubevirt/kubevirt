package debug

import (
	"net/http"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"kubevirt.io/client-go/kubecli"
)

func RegisterDebugHook() {
	kubecli.RegisterRestConfigHook(addDebugLogging)
}

func addDebugLogging(config *rest.Config) {
	if !klog.V(9).Enabled() {
		return
	}

	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &debugRoundTripper{rt: rt}
	})
}

type debugRoundTripper struct {
	rt http.RoundTripper
}

func (d *debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	klog.Infof("DEBUG HIT BEFORE REQUEST") // 👈 ADD THIS

	klog.Infof("Request: %s %s", req.Method, req.URL)

	resp, err := d.rt.RoundTrip(req)

	if err == nil {
		klog.Infof("Response: %s", resp.Status)
	}

	return resp, err
}
