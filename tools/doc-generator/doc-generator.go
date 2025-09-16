package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	"kubevirt.io/kubevirt/tests/libmonitoring"

	"github.com/rhobs/operator-observability-toolkit/pkg/docs"
)

const tpl = `# KubeVirt metrics

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
	if err := libmonitoring.RegisterAllMetrics(); err != nil {
		panic(err)
	}

	metricsList := operatormetrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocsWithCustomTemplate(metricsList, rulesList, tpl)
	fmt.Print(docsString)
}
