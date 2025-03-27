package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/config"
	"github.com/onsi/ginkgo/v2/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

// NewCapturedOutputEnricher captures additional interesting cluster info and adds it to the captured output
// to enrich existing reporters, like the junit reporter, with additional data.
func NewCapturedOutputEnricher(reporters ...ginkgo.Reporter) *capturedOutputEnricher {
	return &capturedOutputEnricher{
		reporters: reporters,
	}
}

type capturedOutputEnricher struct {
	reporters        []ginkgo.Reporter
	additionalOutput interface{}
}

func (j *capturedOutputEnricher) SuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
	for _, report := range j.reporters {
		report.SuiteWillBegin(config, summary)
	}
}

func (j *capturedOutputEnricher) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
	for _, report := range j.reporters {
		report.BeforeSuiteDidRun(setupSummary)
	}
}

func (j *capturedOutputEnricher) SpecWillRun(specSummary *types.SpecSummary) {
	j.additionalOutput = ""
	for _, report := range j.reporters {
		report.SpecWillRun(specSummary)
	}
}

func (j *capturedOutputEnricher) SpecDidComplete(specSummary *types.SpecSummary) {
	if specSummary.HasFailureState() {
		if j.additionalOutput != "" {
			specSummary.CapturedOutput = fmt.Sprintf("%s\n%s", specSummary.CapturedOutput, j.additionalOutput)
		}
	}
	for _, report := range j.reporters {
		report.SpecDidComplete(specSummary)
	}
}

func (j *capturedOutputEnricher) JustAfterEach(specReport types.SpecReport) {
	if specReport.Failed() {
		j.additionalOutput = j.collect(specReport.RunTime)
	}
}

func (j *capturedOutputEnricher) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
	for _, report := range j.reporters {
		report.AfterSuiteDidRun(setupSummary)
	}
}

func (j *capturedOutputEnricher) SuiteDidEnd(summary *types.SuiteSummary) {
	for _, report := range j.reporters {
		report.SuiteDidEnd(summary)
	}
}

func (j *capturedOutputEnricher) collect(duration time.Duration) string {
	virtCli := kubevirt.Client()

	duration += 5 * time.Second
	since := time.Now().Add(-duration)

	return j.getWarningEvents(virtCli, since)
}

func (j *capturedOutputEnricher) getWarningEvents(virtCli kubecli.KubevirtClient, since time.Time) string {
	events, err := virtCli.CoreV1().Events(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to fetch events")
		return ""
	}

	e := events.Items
	sort.Slice(e, func(i, j int) bool {
		return e[i].LastTimestamp.After(e[j].LastTimestamp.Time)
	})

	eventsToPrint := v1.EventList{}
	for _, event := range e {
		if event.LastTimestamp.Time.After(since) && event.Type == v1.EventTypeWarning {
			eventsToPrint.Items = append(eventsToPrint.Items, event)
		}
	}

	rawEvents, err := json.MarshalIndent(eventsToPrint, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal events")
		return ""
	}
	return string(rawEvents)
}
