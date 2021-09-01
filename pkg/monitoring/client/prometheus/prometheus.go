/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package prometheus creates and registers prometheus metrics with
// rest clients. To use this package, you just have to import it.
package prometheus

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/metrics"

	"kubevirt.io/client-go/kubecli"
)

var (
	// requestLatency is a Prometheus Summary metric type partitioned by
	// "verb" and "url" labels. It is used for the rest client latency metrics.
	requestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rest_client_request_latency_seconds",
			Help:    "Request latency in seconds. Broken down by verb and URL.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"verb", "url"},
	)

	rateLimiterLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rest_client_rate_limiter_duration_seconds",
			Help:    "Client side rate limiter latency in seconds. Broken down by verb and URL.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"verb", "url"},
	)

	requestResult = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rest_client_requests_total",
			Help: "Number of HTTP requests, partitioned by status code, method, and host.",
		},
		[]string{"code", "method", "host", "resource", "verb"},
	)

	resourceParsingRegexs = []*regexp.Regexp{}
)

func init() {

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

	kubecli.RegisterRestConfigHook(addHTTPRoundTripClientMonitoring)

	prometheus.MustRegister(requestLatency)
	prometheus.MustRegister(requestResult)
	prometheus.MustRegister(rateLimiterLatency)
	metrics.Register(metrics.RegisterOpts{
		RequestLatency:     &latencyAdapter{requestLatency},
		RateLimiterLatency: &latencyAdapter{rateLimiterLatency},
	})
}

type latencyAdapter struct {
	m *prometheus.HistogramVec
}

func (l *latencyAdapter) Observe(verb string, u url.URL, latency time.Duration) {
	l.m.WithLabelValues(verb, u.String()).Observe(latency.Seconds())
}

type rtWrapper struct {
	origRoundTripper http.RoundTripper
}

func parseURLResourceOperation(request *http.Request) (resource string, verb string) {
	method := request.Method

	resource = ""
	verb = ""

	if request.URL.Path == "" || method == "" {
		return
	}

	for _, r := range resourceParsingRegexs {
		if resource != "" {
			break
		}
		match := r.FindStringSubmatch(request.URL.Path)
		if len(match) > 1 {
			resource = match[1]
		}
	}

	if resource == "" {
		return
	}

	switch method {
	case "GET":
		verb = "GET"
		if strings.Contains(request.URL.Path, "/watch/") {
			verb = "WATCH"
		} else if strings.HasSuffix(request.URL.Path, resource) {
			// If the resource is the last element in the url, then
			// we're asking to list all resources of that type instead
			// of getting an individual resource
			verb = "LIST"
		}
	case "PUT":
		verb = "UPDATE"
	case "PATCH":
		verb = "PATCH"
	case "POST":
		verb = "CREATE"
	case "DELETE":
		verb = "DELETE"
	}

	return resource, verb
}

func (r *rtWrapper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	var status string
	var resource string
	var verb string
	var host string

	response, err = r.origRoundTripper.RoundTrip(request)
	if err != nil {
		status = "<error>"
	} else {
		status = strconv.Itoa(response.StatusCode)
	}

	host = "none"
	if request.URL != nil {
		host = request.URL.Host
	}

	resource, verb = parseURLResourceOperation(request)
	if verb == "" {
		verb = "none"
	}
	if resource == "" {
		resource = "none"
	}
	requestResult.WithLabelValues(status, request.Method, host, resource, verb).Add(1)

	return response, err
}

func addHTTPRoundTripClientMonitoring(config *rest.Config) {
	fn := func(rt http.RoundTripper) http.RoundTripper {
		return &rtWrapper{
			origRoundTripper: rt,
		}
	}
	config.Wrap(fn)
}
