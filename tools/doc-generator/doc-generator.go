package main

import (
	_ "embed"
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	"kubevirt.io/kubevirt/tests/libmonitoring"

	"github.com/rhobs/operator-observability-toolkit/pkg/docs"
)

//go:embed metrics.tpl
var metricTemplate string

func main() {
	if err := libmonitoring.RegisterAllMetrics(); err != nil {
		panic(err)
	}

	metricsList := operatormetrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocsWithCustomTemplate(metricsList, rulesList, metricTemplate)
	fmt.Print(docsString)
}
