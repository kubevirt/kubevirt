package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

type KubernetesReporter struct {
	failureCount int
	artifactsDir string
}

func NewKubernetesReporter(artifactsDir string) *KubernetesReporter {
	return &KubernetesReporter{
		failureCount: 0,
		artifactsDir: artifactsDir,
	}
}

func (r *KubernetesReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {

}

func (r *KubernetesReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
	// clean up artifacts from previous run
	if r.artifactsDir != "" {
		os.RemoveAll(r.artifactsDir)
	}
}

func (r *KubernetesReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (r *KubernetesReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	if r.failureCount > 10 {
		return
	}
	if specSummary.HasFailureState() {
		r.failureCount++
	} else {
		return
	}

	// If we got not directory, print to stderr
	if r.artifactsDir == "" {
		return
	}

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get client: %v", err)
		return
	}

	if err := os.MkdirAll(r.artifactsDir, 0777); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory: %v", err)
		return
	}

	r.logEvents(virtCli, specSummary)
	r.logPods(virtCli, specSummary)
}
func (r *KubernetesReporter) logPods(virtCli kubecli.KubevirtClient, specSummary *types.SpecSummary) {

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, "pods.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v", err)
		return
	}
	defer f.Close()

	pods, err := virtCli.CoreV1().Pods(v1.NamespaceAll).List(v12.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pods: %v", err)
		return
	}

	fmt.Fprint(f, "===== snip =====\n")

	j, err := json.MarshalIndent(pods, "", "    ")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to marshal pods")
		return
	}
	fmt.Fprintln(f, string(j))
	fmt.Fprintln(f, "")
}

func (r *KubernetesReporter) logEvents(virtCli kubecli.KubevirtClient, specSummary *types.SpecSummary) {

	f, err := os.OpenFile(filepath.Join(r.artifactsDir, "events.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v", err)
		return
	}
	defer f.Close()

	startTime := time.Now().Add(-specSummary.RunTime).Add(-5 * time.Second)

	events, err := virtCli.CoreV1().Events(v1.NamespaceAll).List(v12.ListOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to fetch events")
		return
	}

	e := events.Items
	sort.Slice(e, func(i, j int) bool {
		return e[i].LastTimestamp.After(e[j].LastTimestamp.Time)
	})

	fmt.Fprint(f, "===== snip =====\n")

	for _, event := range e {
		if event.LastTimestamp.Time.Before(startTime) {
			continue
		}

		j, err := json.MarshalIndent(event, "", "    ")
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Failed to marshal events")
			return
		}
		fmt.Fprintln(f, string(j))
	}
	fmt.Fprintln(f, "")
}

func (r *KubernetesReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {

}

func (r *KubernetesReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {

}
