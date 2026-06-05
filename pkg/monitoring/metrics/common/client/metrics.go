/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package client

import (
	"net/http"
	"regexp"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/metrics"
	"kubevirt.io/client-go/kubecli"
)

var resourceParsingRegexs []*regexp.Regexp

func init() { //nolint:gochecknoinits // Force KubeVirt client metrics to be registered before default k8s client metrics
	metrics.Register(metrics.RegisterOpts{
		RequestLatency:     &latencyAdapter{requestLatency},
		RateLimiterLatency: &latencyAdapter{rateLimiterLatency},
	})
}

// RegisterRestConfigHooks adds hooks to the KubeVirt client and should be executed before building its config
func RegisterRestConfigHooks() {
	setupResourcesToWatch()
	kubecli.RegisterRestConfigHook(addHTTPRoundTripClientMonitoring)
}

func SetupMetrics() error {
	return operatormetrics.RegisterMetrics(
		restMetrics,
	)
}

func setupResourcesToWatch() {
	p := `[A-Za-z0-9.\-]*`
	res := `(?P<resource>` + p + `)`

	resourceParsingRegexs = append(resourceParsingRegexs,
		// watch core k8s apis
		regexp.MustCompile(`/api/`+p+`/watch/namespaces/`+p+`/`+res),
		regexp.MustCompile(`/api/`+p+`/watch/`+res),

		// watch custom resource apis
		regexp.MustCompile(`/apis/`+p+`/`+p+`/watch/namespaces/`+p+`/`+res),
		regexp.MustCompile(`/apis/`+p+`/`+p+`/watch/`+res),

		// namespaced core k8s apis and namespaced custom apis
		regexp.MustCompile(`/api/`+p+`/namespaces/`+p+`/`+res),
		regexp.MustCompile(`/apis/`+p+`/`+p+`/namespaces/`+p+`/`+res),

		// globally scoped core k8s apis and globally scoped custom apis
		regexp.MustCompile(`/api/`+p+`/`+res),
		regexp.MustCompile(`/apis/`+p+`/`+p+`/`+res),
	)
}

func addHTTPRoundTripClientMonitoring(config *rest.Config) {
	fn := func(rt http.RoundTripper) http.RoundTripper {
		return &rtWrapper{
			origRoundTripper: rt,
		}
	}
	config.Wrap(fn)
}
