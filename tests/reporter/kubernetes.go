package reporter

import (
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/watch"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"os"
	"time"
)

type KubernetesReporter struct {
	startTime    time.Time
	eventWatcher watch.Interface
	log          *logging.FilteredLogger
	eventLog     *logging.FilteredLogger
	eventFile    *os.File
}

func (r *KubernetesReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {

	// Create results folder
	err := os.Mkdir("results", os.ModePerm)
	r.log = logging.Logger("reporter")
	if err != nil && !os.IsExist(err) {
		r.log.Critical().Reason(err).Msg("Failed to create test result directory")
		return
	}

	// Create subfolder for this suite instance
	err = os.Mkdir("results/"+summary.SuiteID, os.ModePerm)
	if err != nil {
		r.log.Critical().Reason(err).Msg("Failed to create test result directory")
		return
	}

	// Create event log file and event logger
	r.eventFile, err = os.Create("results/" + summary.SuiteID + "/events.log")
	if err != nil {
		r.log.Critical().Reason(err).Msg("Failed to create event log file")
	}
	r.eventLog = logging.Logger("events").SetIOWriter(r.eventFile)

	// Connect to kubernetes
	client, err := kubecli.Get()
	if err != nil {
		r.log.Critical().Reason(err).Msg("Failed to create clientset")
		return
	}

	// Create event watcher
	r.startTime = time.Now()
	r.eventWatcher, err = client.Events(v1.NamespaceAll).Watch(v1.ListOptions{})
	if err != nil {
		r.log.Critical().Reason(err).Msg("Failed to create event watcher")
		return
	}

	//Write all received events into the event log
	go func() {
		for obj := range r.eventWatcher.ResultChan() {
			event := obj.Object.(*v1.Event)
			r.eventLog.Info().
				With("time", event.LastTimestamp).
				With("namespace", event.InvolvedObject.Namespace).
				With("name", event.InvolvedObject.Name).
				With("path", event.InvolvedObject.FieldPath).
				With("kind", event.InvolvedObject.Kind).
				With("reason", event.Reason).
				Msg(event.Message)
		}
	}()
}

func (r *KubernetesReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {

}

func (r *KubernetesReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (r *KubernetesReporter) SpecDidComplete(specSummary *types.SpecSummary) {

}

func (r *KubernetesReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
	r.eventWatcher.Stop()
	r.eventFile.Close()
}

func (r *KubernetesReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {

}
