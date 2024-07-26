package testing

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/controller"
)

func MarkAsReady(vmi *v1.VirtualMachineInstance) {
	vmi.Status.Phase = "Running"
	controller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})
}

func MarkAsNonReady(vmi *v1.VirtualMachineInstance) {
	controller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	controller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse})
}

func VirtualMachineFromVMI(name string, vmi *v1.VirtualMachineInstance, started bool) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1", UID: "vm-uid"},
		Spec: v1.VirtualMachineSpec{
			Running: &started,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
		Status: v1.VirtualMachineStatus{
			Conditions: []v1.VirtualMachineCondition{
				{
					Type:   v1.VirtualMachineReady,
					Status: k8sv1.ConditionFalse,
					Reason: "VMINotExists",
				},
			},
		},
	}
	return vm
}

func NewRunningVirtualMachine(vmiName string, node *k8sv1.Node) *v1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI(vmiName)
	vmi.UID = types.UID(uuid.NewString())
	vmi.Status.Phase = v1.Running
	vmi.Status.NodeName = node.Name
	vmi.Labels = map[string]string{
		v1.NodeNameLabel: node.Name,
	}
	return vmi
}
