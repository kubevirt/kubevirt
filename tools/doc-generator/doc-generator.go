package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	virtapi "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	virtcontroller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	virtoperator "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-operator"
	"kubevirt.io/kubevirt/pkg/monitoring/rules"

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
	if err := virtcontroller.SetupMetrics(nil, nil, nil, nil); err != nil {
		panic(err)
	}

	if err := virtcontroller.RegisterLeaderMetrics(); err != nil {
		panic(err)
	}

	if err := virtapi.SetupMetrics(); err != nil {
		panic(err)
	}

	if err := virtoperator.SetupMetrics(); err != nil {
		panic(err)
	}

	if err := virtoperator.RegisterLeaderMetrics(); err != nil {
		panic(err)
	}

	if err := virthandler.SetupMetrics("", "", 0, nil, nil); err != nil {
		panic(err)
	}

	if err := rules.SetupRules(""); err != nil {
		panic(err)
	}

	metricsList := operatormetrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocsWithCustomTemplate(metricsList, rulesList, tpl)
	fmt.Print(docsString)
}
