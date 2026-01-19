package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/config"
	"github.com/onsi/ginkgo/v2/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const (
	testAnnotationKey = "kubevirt.io/created-by-test"
)

var failOnVMLogErrors = os.Getenv("VM_LOG_FAIL_ON_ERRORS") != "false"

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

func CheckVMLogsAfterTest(specReport types.SpecReport) {
	if specReport.Failed() || specReport.State.Is(types.SpecStateSkipped) {
		return
	}

	testName := specReport.FullText()
	foundErrors := getVMLogErrors(testName)

	if len(foundErrors) > 0 {
		if failOnVMLogErrors {
			ginkgo.Fail(fmt.Sprintf("VM logs contain unexpected errors:\n%s", strings.Join(foundErrors, "\n")))
		} else {
			saveVMLogErrors(testName, foundErrors)
		}
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

func getVMLogErrors(testName string) []string {
	virtCli := kubevirt.Client()
	namespace := testsuite.NamespaceTestDefault

	vmis, err := virtCli.VirtualMachineInstance(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil
	}

	var foundErrors []string

	for _, vmi := range vmis.Items {
		if vmi.Annotations == nil {
			continue
		}
		createdBy, ok := vmi.Annotations[testAnnotationKey]
		if !ok || createdBy != testName {
			continue
		}

		labelSelector := fmt.Sprintf("%s=%s", virtv1.CreatedByLabel, string(vmi.GetUID()))
		pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil || len(pods.Items) == 0 {
			continue
		}

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			logsRaw, err := virtCli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
				Container: "compute",
			}).DoRaw(context.Background())
			if err != nil {
				continue
			}

			errors := findDisallowedErrors(string(logsRaw), vmi.Name)
			foundErrors = append(foundErrors, errors...)
		}
	}

	return foundErrors
}

func saveVMLogErrors(testName string, errors []string) {
	artifactsDir := flags.ArtifactsDir
	if artifactsDir == "" {
		return
	}

	filename := filepath.Join(artifactsDir, "vm-log-errors.log")

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to open VM log errors file: %s", filename)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "=== Test: %s ===\n", testName)
	for _, errorLine := range errors {
		fmt.Fprintln(f, errorLine)
	}
}

func findDisallowedErrors(logs string, vmiName string) []string {
	var disallowedErrors []string

	for _, line := range strings.Split(logs, "\n") {
		if !strings.Contains(line, `"level":"error"`) {
			continue
		}

		classification := ClassifyLogLine(line)
		if classification == UnexpectedError {
			disallowedErrors = append(disallowedErrors, FormatErrorLine(vmiName, line))
		}
	}

	return disallowedErrors
}
