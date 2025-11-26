package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	"kubevirt.io/kubevirt/tests/libmonitoring"

	"github.com/rhobs/operator-observability-toolkit/pkg/docs"
)

const tpl = "# KubeVirt metrics\n\n" +
	"| name | Type | Description |\n" +
	"|------|------|-------------|\n" +
	"{{- range . }}\n" +
	"{{ $deprecatedVersion := \"\" -}}\n" +
	"{{- with index .ExtraFields \"DeprecatedVersion\" -}}\n" +
	"    {{- $deprecatedVersion = printf \" in %s\" . -}}\n" +
	"{{- end -}}\n" +
	"{{- $stabilityLevel := \"\" -}}\n" +
	"{{- if and (.ExtraFields.StabilityLevel) (ne .ExtraFields.StabilityLevel \"STABLE\") -}}\n" +
	"	{{- $stabilityLevel = printf \"[%s%s] \" .ExtraFields.StabilityLevel $deprecatedVersion -}}\n" +
	"{{- end -}}\n" +
	"{{- $description := printf \"%s%s\" $stabilityLevel .Help -}}\n" +
	"{{- $nameWithBackticks := printf \"`%s`\" .Name -}}\n" +
	"| {{ $nameWithBackticks }} | {{ .Type }} | {{ $description }} |\n" +
	"{{- end }}\n\n" +
	"## Developing new metrics\n\n" +
	"All metrics documented here are auto-generated and reflect exactly what is being\n" +
	"exposed. After developing new metrics or changing old ones please regenerate\n" +
	"this document.\n"

func main() {
	if err := libmonitoring.RegisterAllMetrics(); err != nil {
		panic(err)
	}

	metricsList := operatormetrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocsWithCustomTemplate(metricsList, rulesList, tpl)
	fmt.Print(docsString)
}
