package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests"

	"kubevirt.io/client-go/kubecli"
)

type XMLReporter struct {
	artifactsDir string
	mux          sync.Mutex
}

func NewXMLReporter(artifactsDir string) *XMLReporter {
	return &XMLReporter{
		artifactsDir: artifactsDir,
	}
}

func (r *XMLReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {

}

func (r *XMLReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
	// clean up artifacts from previous run
	if r.artifactsDir != "" {
		os.RemoveAll(r.artifactsDir)
	}
}

func (r *XMLReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (r *XMLReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if specSummary.State != types.SpecStatePassed {
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

	r.logDomainXMLs(virtCli, specSummary)
}

func (r *XMLReporter) logDomainXMLs(virtCli kubecli.KubevirtClient, specSummary *types.SpecSummary) {

	testName := strings.Join(specSummary.ComponentTexts, "__")
	rex := regexp.MustCompile("\\[[^\\]]*\\]")
	testName = strings.TrimPrefix(strings.Replace(rex.ReplaceAllString(testName, ""), " ", "_", -1), "__")
	f, err := os.OpenFile(filepath.Join(r.artifactsDir, fmt.Sprintf("%s_domains.log", testName)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the file: %v", err)
		return
	}
	defer f.Close()

	vmis, err := virtCli.VirtualMachineInstance(v1.NamespaceAll).List(&v12.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch vmis: %v", err)
		return
	}

	for _, vmi := range vmis.Items {
		if vmi.IsFinal() {
			continue
		}
		domxml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtCli, &vmi)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to fetch domain XML: %v", err)
		}
		fmt.Fprintln(f, domxml)
	}
}

func (r *XMLReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {

}

func (r *XMLReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {

}
