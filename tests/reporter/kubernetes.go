package reporter

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-logfmt/logfmt"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"strconv"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
)

type KubernetesReporter struct {
	startTime     time.Time
	eventWatcher  watch.Interface
	log           *logging.FilteredLogger
	eventLog      *logging.FilteredLogger
	containerLog  *log.Context
	eventFile     *os.File
	containerFile *os.File
	cli           core_v1.CoreV1Interface
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

	// Create container log file and container logger
	r.containerFile, err = os.Create("results/" + summary.SuiteID + "/container.log")
	if err != nil {
		r.log.Critical().Reason(err).Msg("Failed to create container log file")
	}
	r.containerLog = log.NewContext(log.NewLogfmtLogger(r.containerFile))

	// Connect to kubernetes
	client, err := kubecli.Get()
	if err != nil {
		r.log.Critical().Reason(err).Msg("Failed to create clientset")
		return
	}
	r.cli = client.CoreV1()

	// Create event watcher
	r.startTime = time.Now()
	r.eventWatcher, err = r.cli.Events(metav1.NamespaceAll).Watch(metav1.ListOptions{})
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

}

func (r *KubernetesReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	r.eventWatcher.Stop()
	r.eventFile.Close()

	defer r.containerFile.Close()
	pods, err := r.cli.Pods(metav1.NamespaceAll).List(metav1.ListOptions{FieldSelector: "status.phase=" + string(v1.PodRunning)})
	if err != nil {
		r.log.Critical().Reason(err).Msg("Could not fetch running pods")
		return
	}
	since := metav1.NewTime(r.startTime)
	for _, p := range pods.Items {
		r.log.Info().V(2).Object(&p).Msg("Pod fetched to collect logs")
		for _, c := range p.Spec.Containers {
			r.log.Info().V(2).Object(&p).With("container", c.Name).Msg("Fetching logs")
			logs, err := r.cli.Pods(p.GetObjectMeta().GetNamespace()).
				GetLogs(p.GetObjectMeta().GetName(), &v1.PodLogOptions{
					SinceTime: &since,
					Container: c.Name,
				}).Stream()
			if err != nil {
				r.log.Error().Reason(err).Object(&p).Msg("Failed to fetch logs")
			}
			err = r.convertLog(&p, &c, logs)
			if err != nil {
				r.log.Error().Reason(err).Object(&p).Msg("Failed to write logs")
			}
		}
	}
}

func (r *KubernetesReporter) convertLog(pod *v1.Pod, c *v1.Container, reader io.Reader) error {
	buffer := bufio.NewReader(reader)
lines:
	for {
		line, err := buffer.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				r.log.Error().Reason(err).Object(pod).With("container", c.Name).Msg("Failed to parse logs")
			}
			break
		}
		l := r.containerLog.WithPrefix("node", pod.Spec.NodeName, "pod", pod.GetObjectMeta().GetName(), "container", c.Name)
		for _, converter := range []LogConverter{LogfmtConverter{}, GLogConverter{}, LogWrapper{}} {
			fields, matches, err := converter.Convert(line)
			if !matches {
				continue
			}
			if err != nil {
				fmt.Println(line)
				fmt.Println(err)
				continue
			}
			if matches {
				l.Log(fields...)
				continue lines
			}

		}
	}
	return nil
}

var glogRegexp = regexp.MustCompile(`([WIED])([0-9]{2,2})([0-9]{2,2})[\s]+([0-9]{2,2}):([0-9]{2,2}):([0-9]{2,2}).([0-9]{6,6})[\s]+[0-9]+[\s]+([^\s]+):([0-9]+)][\s]+(.+)`)
var logfmtPrefix = regexp.MustCompile(`^[\s]*[a-z].+`)

type LogConverter interface {
	Convert(line string) ([]interface{}, bool, error)
}

type GLogConverter struct{}

func (GLogConverter) Convert(line string) ([]interface{}, bool, error) {
	findings := glogRegexp.FindStringSubmatch(line)
	if findings == nil {
		return nil, false, nil
	}
	level := "unknown"
	switch findings[1] {
	case "W":
		level = "warning"
	case "I":
		level = "info"
	case "E":
		level = "error"
	case "D":
		level = "debug"
	}
	month, _ := strconv.Atoi(findings[2])
	day, _ := strconv.Atoi(findings[3])
	hour, _ := strconv.Atoi(findings[4])
	min, _ := strconv.Atoi(findings[5])
	sec, _ := strconv.Atoi(findings[6])
	nsec, _ := strconv.Atoi(findings[7])
	date := time.Date(
		time.Now().Year(),
		time.Month(month),
		day,
		hour,
		min,
		sec,
		nsec,
		time.UTC,
	)

	// TODO extract more fields
	return []interface{}{"level", level, "timestamp", date, "pos", findings[8] + ":" + findings[9], "log", findings[10]}, true, nil
}

type LogfmtConverter struct{}

func (LogfmtConverter) Convert(line string) ([]interface{}, bool, error) {
	logPraser := logfmt.NewDecoder(strings.NewReader(line))
	if !logfmtPrefix.MatchString(line) {
		return nil, false, nil
	}
	logPraser.ScanRecord()
	fields := []interface{}{}
	for logPraser.ScanKeyval() {
		fields = append(fields, string(logPraser.Key()), string(logPraser.Value()))
	}
	return fields, logPraser.Err() == nil, logPraser.Err()
}

type LogWrapper struct{}

func (LogWrapper) Convert(line string) ([]interface{}, bool, error) {
	return []interface{}{"log", line}, true, nil
}
