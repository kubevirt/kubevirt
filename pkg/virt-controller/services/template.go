package services

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"io"
	"k8s.io/client-go/1.5/pkg/types"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
	"text/template"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VM, io.Writer) error
}

type manifestData struct {
	Domain         string
	VMUID          types.UID
	DockerRegistry string
	LauncherImage  string
	NodeSelector   map[string]string
	Args           []string
}

type templateService struct {
	template     *template.Template
	logger       levels.Levels
	dataTemplate manifestData
}

//Deprecated: create a nativ pod object with the kubernetes sdk
func (t *templateService) RenderLaunchManifest(vm *v1.VM, writer io.Writer) error {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(writer)
	data := t.dataTemplate
	data.Domain = precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	data.NodeSelector = vm.Spec.NodeSelector
	data.VMUID = vm.GetObjectMeta().GetUID()
	data.Args = []string{"/virt-launcher", "-qemu-timeout", "60s"}
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
