package docs

import (
	"bytes"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
)

const defaultMetricsTemplate = `# Operator Metrics

{{- range . }}

### {{.Name}}
{{.Help}}.

Type: {{.Type}}.
{{- end }}

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
`

type metricDocs struct {
	Name        string
	Help        string
	Type        string
	ExtraFields map[string]string
}

type docOptions interface {
	GetOpts() operatormetrics.MetricOpts
	GetType() operatormetrics.MetricType
}

// BuildMetricsDocsWithCustomTemplate returns a string with the documentation
// for the given metrics, using the given template.
func BuildMetricsDocsWithCustomTemplate(
	metrics []operatormetrics.Metric,
	recordingRules []operatorrules.RecordingRule,
	tplString string,
) string {

	tpl, err := template.New("metrics").Parse(tplString)
	if err != nil {
		log.Fatalln(err)
	}

	var allDocs []metricDocs

	if metrics != nil {
		allDocs = append(allDocs, buildMetricsDocs(metrics)...)
	}

	if recordingRules != nil {
		allDocs = append(allDocs, buildMetricsDocs(recordingRules)...)
	}

	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, allDocs)
	if err != nil {
		log.Fatalln(err)
	}

	return buf.String()
}

// BuildMetricsDocs returns a string with the documentation for the given
// metrics.
func BuildMetricsDocs(metrics []operatormetrics.Metric, recordingRules []operatorrules.RecordingRule) string {
	return BuildMetricsDocsWithCustomTemplate(metrics, recordingRules, defaultMetricsTemplate)
}

func buildMetricsDocs[T docOptions](items []T) []metricDocs {
	metricsDocs := make([]metricDocs, len(items))
	for i, metric := range items {
		metricOpts := metric.GetOpts()
		metricsDocs[i] = metricDocs{
			Name:        metricOpts.Name,
			Help:        metricOpts.Help,
			Type:        getAndConvertMetricType(metric.GetType()),
			ExtraFields: metricOpts.ExtraFields,
		}
	}
	sortMetricsDocs(metricsDocs)

	return metricsDocs
}

func sortMetricsDocs(metricsDocs []metricDocs) {
	sort.Slice(metricsDocs, func(i, j int) bool {
		return metricsDocs[i].Name < metricsDocs[j].Name
	})
}

func getAndConvertMetricType(metricType operatormetrics.MetricType) string {
	return strings.ReplaceAll(string(metricType), "Vec", "")
}
