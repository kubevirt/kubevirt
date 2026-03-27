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
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		// chain wrapper safely
		return &DebugRoundTripper{rt: rt}
	})
}

type DebugRoundTripper struct {
	rt http.RoundTripper
}

func (d *DebugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if klog.V(9).Enabled() {
		klog.Infof("Request: %s %s", req.Method, req.URL)
	}

	resp, err := d.rt.RoundTrip(req)

	if err == nil && klog.V(9).Enabled() {
		klog.Infof("Response: %s", resp.Status)
	}

	return resp, err
}