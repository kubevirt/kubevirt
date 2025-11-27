# KubeVirt metrics

| Name | Type | Description |
|------|------|-------------|
{{- range . }}
{{ $deprecatedVersion := "" -}}
{{- with index .ExtraFields "DeprecatedVersion" -}}
    {{- $deprecatedVersion = printf " in %s" . -}}
{{- end -}}
{{- $stabilityLevel := "" -}}
{{- if and (.ExtraFields.StabilityLevel) (ne .ExtraFields.StabilityLevel "STABLE") -}}
	{{- $stabilityLevel = printf "[%s%s] " .ExtraFields.StabilityLevel $deprecatedVersion -}}
{{- end -}}
{{- $description := printf "%s%s" $stabilityLevel .Help -}}
{{- $nameWithBackticks := printf "`%s`" .Name -}}
| {{ $nameWithBackticks }} | {{ .Type }} | {{ $description }} |
{{- end }}

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
