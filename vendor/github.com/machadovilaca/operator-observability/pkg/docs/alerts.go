package docs

import (
	"bytes"
	"log"
	"sort"
	"text/template"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

const defaultAlertsTemplate = `# Operator Alerts

{{- range . }}

### {{.Name}}
**Summary:** {{ index .Annotations "summary" }}.

**Description:** {{ index .Annotations "description" }}.

**Severity:** {{ index .Labels "severity" }}.
{{- if .For }}

**For:** {{ .For }}.
{{- end -}}
{{- end }}

## Developing new alerts

All alerts documented here are auto-generated and reflect exactly what is being
exposed. After developing new alerts or changing old ones please regenerate
this document.
`

type alertDocs struct {
	Name        string
	Expr        string
	For         string
	Annotations map[string]string
	Labels      map[string]string
}

// BuildAlertsDocsWithCustomTemplate returns a string with the documentation
// for the given alerts, using the given template.
func BuildAlertsDocsWithCustomTemplate(
	alerts []promv1.Rule,
	tplString string,
) string {

	tpl, err := template.New("alerts").Parse(tplString)
	if err != nil {
		log.Fatalln(err)
	}

	var allDocs []alertDocs

	if alerts != nil {
		allDocs = append(allDocs, buildAlertsDocs(alerts)...)
	}

	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, allDocs)
	if err != nil {
		log.Fatalln(err)
	}

	return buf.String()
}

// BuildAlertsDocs returns a string with the documentation for the given
// metrics.
func BuildAlertsDocs(alerts []promv1.Rule) string {
	return BuildAlertsDocsWithCustomTemplate(alerts, defaultAlertsTemplate)
}

func buildAlertsDocs(alerts []promv1.Rule) []alertDocs {
	alertsDocs := make([]alertDocs, len(alerts))
	for i, alert := range alerts {
		alertsDocs[i] = alertDocs{
			Name:        alert.Alert,
			Expr:        alert.Expr.String(),
			For:         string(*alert.For),
			Annotations: alert.Annotations,
			Labels:      alert.Labels,
		}
	}
	sortAlertsDocs(alertsDocs)

	return alertsDocs
}

func sortAlertsDocs(alertsDocs []alertDocs) {
	sort.Slice(alertsDocs, func(i, j int) bool {
		return alertsDocs[i].Name < alertsDocs[j].Name
	})
}
