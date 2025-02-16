package resourcefiles

import (
	"embed"
	"fmt"
	"os"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templatesFS, "templates/*"))
}

type TemplateInfo interface {
	templateName() string
}

type VMInfo struct {
	Version   string
	Index     int
	ImageName string
}

func (VMInfo) templateName() string {
	return "vm-template.yaml"
}

type SnapshotInfo struct {
	Version string
	Name    string
	VMName  string
}

func (SnapshotInfo) templateName() string {
	return "snapshot-template.yaml"
}

type RestoreInfo struct {
	Version      string
	Name         string
	VMName       string
	SnapshotName string
}

func (RestoreInfo) templateName() string {
	return "restore-template.yaml"
}

func WriteFile(fileName string, info TemplateInfo) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %v", fileName, err)
	}

	defer file.Close()

	return templates.Lookup(info.templateName()).Execute(file, info)
}
