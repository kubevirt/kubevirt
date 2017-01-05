package services

import (
	kubev1 "k8s.io/client-go/1.5/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VM) (*kubev1.Pod, error)
}

type templateService struct {
	logger        *logging.FilteredLogger
	launcherImage string
}

//Deprecated: remove the service and just use a builder or contextcless helper function
func (t *templateService) RenderLaunchManifest(vm *v1.VM) (*kubev1.Pod, error) {
	precond.MustNotBeNil(vm)
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	uid := precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))
	True := true
	// TODO use constants for labels
	pod := kubev1.Pod{
		ObjectMeta: kubev1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-----",
			Labels: map[string]string{
				v1.AppLabel:    "virt-launcher",
				v1.DomainLabel: domain,
				v1.UIDLabel:    uid,
			},
		},
		Spec: kubev1.PodSpec{
			HostNetwork:   true,
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers: []kubev1.Container{
				{
					Name:            "compute",
					Image:           t.launcherImage,
					ImagePullPolicy: kubev1.PullIfNotPresent,
					Command:         []string{"/virt-launcher", "-qemu-timeout", "60s"},
					SecurityContext: &kubev1.SecurityContext{Privileged: &True},
				},
			},
			NodeSelector: vm.Spec.NodeSelector,
		},
	}

	return &pod, nil
}

func NewTemplateService(launcherImage string) (TemplateService, error) {
	precond.MustNotBeEmpty(launcherImage)
	svc := templateService{
		logger:        logging.DefaultLogger().With("service", "TemplateService"),
		launcherImage: launcherImage,
	}
	return &svc, nil
}
