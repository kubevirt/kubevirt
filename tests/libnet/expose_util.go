package libnet

import (
	v1 "kubevirt.io/client-go/apis/core/v1"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
)

type Option func(cmdArgs []string) []string

func NewVMIExposeArgs(vmi *v1.VirtualMachineInstance, opts ...Option) []string {
	return NewExposeArgs("virtualmachineinstance", vmi.Namespace, vmi.Name, opts...)
}

func NewVMIRSExposeArgs(vmrs *v1.VirtualMachineInstanceReplicaSet, opts ...Option) []string {
	return NewExposeArgs("vmirs", vmrs.Namespace, vmrs.Name, opts...)
}

func NewVMExposeArgs(vm *v1.VirtualMachine, opts ...Option) []string {
	return NewExposeArgs("virtualmachine", vm.Namespace, vm.Name, opts...)
}

func NewExposeArgs(resource, namespace, name string, opts ...Option) []string {
	vmiExposeArgs := []string{
		expose.COMMAND_EXPOSE,
		resource, "--namespace", namespace, name,
	}

	for _, f := range opts {
		vmiExposeArgs = f(vmiExposeArgs)
	}

	return vmiExposeArgs
}

func WithServiceName(serviceName string) func(cmdArgs []string) []string {
	return func(cmdArgs []string) []string {
		return append(cmdArgs, "--name", serviceName)
	}
}

func WithPort(port string) func(cmdArgs []string) []string {
	return func(cmdArgs []string) []string {
		return append(cmdArgs, "--port", port)
	}
}

func WithTargetPort(targetPort string) func(cmdArgs []string) []string {
	return func(cmdArgs []string) []string {
		return append(cmdArgs, "--target-port", targetPort)
	}
}

func WithType(serviceType string) func(cmdArgs []string) []string {
	return func(cmdArgs []string) []string {
		return append(cmdArgs, "--type", serviceType)
	}
}

func WithProtocol(protocol string) func(cmdArgs []string) []string {
	return func(cmdArgs []string) []string {
		return append(cmdArgs, "--protocol", protocol)
	}
}
