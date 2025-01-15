package resourcefiles

import (
	"embed"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templatesFS, "templates/*"))
}
