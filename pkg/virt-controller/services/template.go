package services

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"html"
	"io"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/precond"
	"strings"
	"text/template"
)

type TemplateService interface {
	RenderManifest(*api.VM, []byte, io.Writer) error
}

type manifestData struct {
	Domain         string
	DockerRegistry string
	LauncherImage  string
	DomainXML      string
	NodeSelector   map[string]string
}

type templateService struct {
	template     *template.Template
	logger       levels.Levels
	dataTemplate manifestData
}

func (t *templateService) RenderManifest(vm *api.VM, domainXML []byte, writer io.Writer) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(writer)
	precond.MustNotBeNil(domainXML)
	data := t.dataTemplate
	data.Domain = precond.MustNotBeEmpty(vm.Name)
	data.DomainXML = EncodeDomainXML(string(domainXML))
	data.NodeSelector = vm.NodeSelector
	return t.template.Execute(writer, &data)
}

func NewTemplateService(logger log.Logger, templateFile string, dockerRegistry string, launcherImage string) (TemplateService, error) {
	precond.MustNotBeNil(logger)
	precond.MustNotBeEmpty(templateFile)
	precond.MustNotBeEmpty(dockerRegistry)
	precond.MustNotBeEmpty(launcherImage)
	template, err := template.New("manifest-template.yaml").ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}
	svc := templateService{
		logger:   levels.New(logger).With("component", "TemplateService"),
		template: template,
		dataTemplate: manifestData{
			DockerRegistry: dockerRegistry,
			LauncherImage:  launcherImage,
		},
	}
	return &svc, nil
}

func EncodeDomainXML(domainXML string) string {
	encodedXML := html.EscapeString(string(domainXML))
	encodedXML = strings.Replace(encodedXML, "\\", "\\\\", -1)
	encodedXML = strings.Replace(encodedXML, "\n", "\\n", -1)
	return encodedXML
}
