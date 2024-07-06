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
 * Copyright the KubeVirt Authors.
 */

package client

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/metrics"
	"kubevirt.io/client-go/kubecli"
)

var resourceParsingRegexs []*regexp.Regexp

// RegisterRestConfigHooks adds hooks to the KubeVirt client and should be executed before building its config
func RegisterRestConfigHooks() {
	setupResourcesToWatch()
	kubecli.RegisterRestConfigHook(addHTTPRoundTripClientMonitoring)
}

func SetupMetrics() error {
	metrics.Register(metrics.RegisterOpts{
		RequestLatency:     &latencyAdapter{requestLatency},
		RateLimiterLatency: &latencyAdapter{rateLimiterLatency},
	})

	return operatormetrics.RegisterMetrics(
		restMetrics,
	)
}

func setupResourcesToWatch() {
	resPat := `[A-Za-z0-9.\-]*`

	// watch core k8s apis
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/api/%s/watch/namespaces/%s/(?P<resource>%s)`, resPat, resPat, resPat)))
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/api/%s/watch/(?P<resource>%s)`, resPat, resPat)))

	// watch custom resource apis
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/apis/%s/%s/watch/namespaces/%s/(?P<resource>%s)`, resPat, resPat, resPat, resPat)))
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/apis/%s/%s/watch/(?P<resource>%s)`, resPat, resPat, resPat)))

	// namespaced core k8 apis and namespaced custom apis
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/api/%s/namespaces/%s/(?P<resource>%s)`, resPat, resPat, resPat)))
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/apis/%s/%s/namespaces/%s/(?P<resource>%s)`, resPat, resPat, resPat, resPat)))

	// globally scoped core k8s apis and globally scoped custom apis
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/api/%s/(?P<resource>%s)`, resPat, resPat)))
	resourceParsingRegexs = append(resourceParsingRegexs, regexp.MustCompile(fmt.Sprintf(`/apis/%s/%s/(?P<resource>%s)`, resPat, resPat, resPat)))
}

func addHTTPRoundTripClientMonitoring(config *rest.Config) {
	fn := func(rt http.RoundTripper) http.RoundTripper {
		return &rtWrapper{
			origRoundTripper: rt,
		}
	}
	config.Wrap(fn)
}
