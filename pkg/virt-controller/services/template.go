package services

import (
	"fmt"
	"strconv"
	"strings"

	kubev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VM) (*kubev1.Pod, error)
	RenderMigrationJob(*v1.VM, *kubev1.Node, *kubev1.Node, *kubev1.Pod) (*kubev1.Pod, error)
}

type templateService struct {
	launcherImage string
}

//Deprecated: remove the service and just use a builder or contextcless helper function
func (t *templateService) RenderLaunchManifest(vm *v1.VM) (*kubev1.Pod, error) {
	precond.MustNotBeNil(vm)
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	uid := precond.MustNotBeEmpty(string(vm.GetObjectMeta().GetUID()))

	// VM target container
	container := kubev1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: kubev1.PullIfNotPresent,
		Command:         []string{"/virt-launcher", "-qemu-timeout", "60s"},
	}

	// Set up spice ports
	ports := []kubev1.ContainerPort{}
	for i, g := range vm.Spec.Domain.Devices.Graphics {
		if strings.ToLower(g.Type) == "spice" {
			ports = append(ports, kubev1.ContainerPort{
				ContainerPort: g.Port,
				Name:          "spice" + strconv.Itoa(i),
			})
		}
	}
	container.Ports = ports

	// TODO use constants for labels
	pod := kubev1.Pod{
		ObjectMeta: kubev1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-----",
			Labels: map[string]string{
				v1.AppLabel:    "virt-launcher",
				v1.DomainLabel: domain,
				v1.VMUIDLabel:  uid,
			},
		},
		Spec: kubev1.PodSpec{
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers:    []kubev1.Container{container},
			NodeSelector:  vm.Spec.NodeSelector,
		},
	}

	return &pod, nil
}

func (t *templateService) RenderMigrationJob(vm *v1.VM, sourceNode *kubev1.Node, targetNode *kubev1.Node, targetPod *kubev1.Pod) (*kubev1.Pod, error) {
	srcAddr := ""
	dstAddr := ""
	for _, addr := range sourceNode.Status.Addresses {
		if (addr.Type == kubev1.NodeInternalIP) && (srcAddr == "") {
			srcAddr = addr.Address
			break
		}
	}
	if srcAddr == "" {
		err := fmt.Errorf("migration source node is unreachable")
		logging.DefaultLogger().Error().Msg("migration target node is unreachable")
		return nil, err
	}
	srcUri := fmt.Sprintf("qemu+tcp://%s/system", srcAddr)

	for _, addr := range targetNode.Status.Addresses {
		if (addr.Type == kubev1.NodeInternalIP) && (dstAddr == "") {
			dstAddr = addr.Address
			break
		}
	}
	if dstAddr == "" {
		err := fmt.Errorf("migration target node is unreachable")
		logging.DefaultLogger().Error().Msg("migration target node is unreachable")
		return nil, err
	}
	destUri := fmt.Sprintf("qemu+tcp://%s/system", dstAddr)

	job := kubev1.Pod{
		ObjectMeta: kubev1.ObjectMeta{
			GenerateName: "virt-migration",
			Labels: map[string]string{
				v1.DomainLabel: vm.GetObjectMeta().GetName(),
				v1.AppLabel:    "migration",
			},
		},
		Spec: kubev1.PodSpec{
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers: []kubev1.Container{
				{
					Name:  "virt-migration",
					Image: "kubevirt/virt-handler:devel",
					Command: []string{
						"/migrate", vm.Spec.Domain.Name, "--source", srcUri, "--dest", destUri, "--pod-ip", targetPod.Status.PodIP,
					},
				},
			},
		},
	}

	return &job, nil
}

func NewTemplateService(launcherImage string) (TemplateService, error) {
	precond.MustNotBeEmpty(launcherImage)
	svc := templateService{
		launcherImage: launcherImage,
	}
	return &svc, nil
}
