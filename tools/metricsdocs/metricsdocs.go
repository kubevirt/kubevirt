package main

import (
	"fmt"

	"github.com/machadovilaca/operator-observability/pkg/docs"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/metrics"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/rules"
)

const tpl = `# Hyperconverged Cluster Operator metrics

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
{{ print $stabilityLevel }}{{ .Help }}. Type: {{ .Type -}}.

{{- end }}

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
`

func main() {
	err := metrics.SetupMetrics()
	if err != nil {
		panic(err)
	}

	err = rules.SetupRules()
	if err != nil {
		panic(err)
	}

	metricsList := metrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocsWithCustomTemplate(metricsList, rulesList, tpl)
	fmt.Print(docsString)
}
