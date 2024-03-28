package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/machadovilaca/operator-observability/pkg/docs"
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"

	domainstats "kubevirt.io/kubevirt/pkg/monitoring/domainstats/prometheus" // import for prometheus metrics
	virt_api "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	virt_controller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	virt_operator "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-operator"
	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	_ "kubevirt.io/kubevirt/pkg/virt-controller/watch"
)

// constant parts of the file
const tpl = `# Kubevirt metrics

### kubevirt_info
Version information.

{{- range . }}

{{ $deprecatedVersion := "" -}}
{{- with index .ExtraFields "DeprecatedVersion" -}}
    {{- $deprecatedVersion = printf " in %s" . -}}
{{- end -}}

{{- $stabilityLevel := "" -}}
{{- if and (.ExtraFields.StabilityLevel) (ne .ExtraFields.StabilityLevel "STABLE") -}}
	{{- $stabilityLevel = printf "[%s%s] " .ExtraFields.StabilityLevel $deprecatedVersion -}}
{{- end -}}

### {{ .Name }}
{{ print $stabilityLevel }}{{ .Help }} Type: {{ .Type -}}.

{{- end }}

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
`

func main() {
	handler := domainstats.Handler(1)
	RegisterFakeDomainCollector()

	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	checkError(err)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	metrics := getMetricsNotIncludeInEndpointByDefault()

	if status := recorder.Code; status == http.StatusOK {
		err := parseVirtMetrics(recorder.Body, &metrics)
		checkError(err)

	} else {
		panic(fmt.Errorf("got HTTP status code of %d from /metrics", recorder.Code))
	}

	metricsList := getMetrics()
	rulesList := getRulesWithCustomMetrics(metrics)

	docsString := docs.BuildMetricsDocsWithCustomTemplate(metricsList, rulesList, tpl)
	fmt.Print(docsString)
}

func getMetrics() []operatormetrics.Metric {
	err := virt_controller.SetupMetrics(nil, nil, nil, nil, nil, nil, nil, nil)
	checkError(err)

	err = virt_api.SetupMetrics()
	checkError(err)

	err = virt_operator.SetupMetrics()
	checkError(err)

	return virt_controller.ListMetrics()
}

func getRulesWithCustomMetrics(metrics metricList) []operatorrules.RecordingRule {
	err := rules.SetupRules("")
	checkError(err)
	rulesList := rules.ListRecordingRules()

	for _, cm := range metrics {
		customMetric := operatorrules.RecordingRule{
			MetricsOpts: operatormetrics.MetricOpts{
				Name: cm.name,
				Help: cm.description,
				// Populate other necessary fields as needed
			},
			MetricType: operatormetrics.MetricType(cm.mType),
		}
		rulesList = append(rulesList, customMetric)
	}
	return rulesList
}
