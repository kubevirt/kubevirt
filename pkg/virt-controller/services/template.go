package services

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"io"
	"kubevirt/core/pkg/virt-controller/entities"
	"kubevirt/core/pkg/virt-controller/precond"
	"text/template"
)

type TemplateService interface {
	RenderManifest(*entities.VM, io.Writer) error
}

type manifestData struct {
	Domain         string
	DockerRegistry string
	LauncherImage  string
}

type templateService struct {
	template     *template.Template
	logger       levels.Levels
	dataTemplate manifestData
}

func (t *templateService) RenderManifest(vm *entities.VM, writer io.Writer) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(writer)
	data := t.dataTemplate
	data.Domain = precond.MustNotBeEmpty(vm.Name)
	return t.template.Execute(writer, data)
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
