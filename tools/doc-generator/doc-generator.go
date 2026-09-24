package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	"kubevirt.io/kubevirt/tests/libmonitoring"

	"github.com/rhobs/operator-observability-toolkit/pkg/docs"
)

const title = `KubeVirt metrics`

func main() {
	if err := libmonitoring.RegisterAllMetrics(); err != nil {
		panic(err)
	}

	metricsList := operatormetrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocs(title, metricsList, rulesList)
	fmt.Print(docsString)
}
