package docs

import (
	"bytes"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

const defaultMetricsTemplate = `# {{.Title}}

| Name | Kind | Type | Description |
|------|------|------|-------------|
{{- range .Metrics }}
{{ $deprecatedVersion := "" -}}
{{- with index .ExtraFields "DeprecatedVersion" -}}
    {{- $deprecatedVersion = printf " in %s" . -}}
{{- end -}}
{{- $stabilityLevel := "" -}}
{{- if and (.ExtraFields.StabilityLevel) (ne .ExtraFields.StabilityLevel "STABLE") -}}
	{{- $stabilityLevel = printf "[%s%s] " .ExtraFields.StabilityLevel $deprecatedVersion -}}
{{- end -}}
{{- $description := printf "%s%s" $stabilityLevel .Description -}}
| {{.Name}} | {{.Kind}} | {{.Type}} | {{ $description }} |
{{- end }}

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
`

type metricDocs struct {
	Name        string
	Kind        string
	Type        string
	Description string
	ExtraFields map[string]string
}

type templateData struct {
	Title   string
	Metrics []metricDocs
}

type docOptions interface {
	GetOpts() operatormetrics.MetricOpts
	GetType() operatormetrics.MetricType
}

// BuildMetricsDocsWithCustomTemplate returns a string with the documentation
// for the given metrics and recording rules, using the given template.
func BuildMetricsDocsWithCustomTemplate(
	title string,
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
		metricsDocs := buildMetricsDocs(metrics)
		sortMetricsDocs(metricsDocs)
		allDocs = append(allDocs, metricsDocs...)
	}

	if recordingRules != nil {
		rulesDocs := buildMetricsDocs(recordingRules)
		sortMetricsDocs(rulesDocs)
		allDocs = append(allDocs, rulesDocs...)
	}

	data := templateData{
		Title:   title,
		Metrics: allDocs,
	}

	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, data)
	if err != nil {
		log.Fatalln(err)
	}

	return buf.String()
}

// BuildMetricsDocs returns a string with the documentation for the given
// metrics and recording rules.
func BuildMetricsDocs(title string, metrics []operatormetrics.Metric, recordingRules []operatorrules.RecordingRule) string {
	return BuildMetricsDocsWithCustomTemplate(title, metrics, recordingRules, defaultMetricsTemplate)
}

func buildMetricsDocs[T docOptions](items []T) []metricDocs {
	uniqueNames := make(map[string]struct{})
	var docs []metricDocs

	kind := getKindFromType(items)

	for _, item := range items {
		itemOpts := item.GetOpts()
		if _, exists := uniqueNames[itemOpts.Name]; !exists {
			uniqueNames[itemOpts.Name] = struct{}{}
			docs = append(docs, metricDocs{
				Name:        itemOpts.Name,
				Kind:        kind,
				Type:        getAndConvertItemType(item.GetType()),
				Description: itemOpts.Help,
				ExtraFields: itemOpts.ExtraFields,
			})
		}
	}

	return docs
}

func getKindFromType[T docOptions](items []T) string {
	if len(items) == 0 {
		return "unknown"
	}

	switch any(items[0]).(type) {
	case operatormetrics.Metric:
		return "Metric"
	case operatorrules.RecordingRule:
		return "Recording rule"
	}
	return "unknown"
}

func sortMetricsDocs(docs []metricDocs) {
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Name < docs[j].Name
	})
}

func getAndConvertItemType(itemType operatormetrics.MetricType) string {
	return strings.ReplaceAll(string(itemType), "Vec", "")
}
