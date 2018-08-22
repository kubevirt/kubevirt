package devices

import (
	"fmt"
	"math/rand"

	"os/exec"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type HostBridge struct {
}

func (HostBridge) Setup(vmi *v1.VirtualMachineInstance, hostNamespaces *isolation.IsolationResult, podNamespaces *isolation.IsolationResult) error {
	for i, net := range vmi.Spec.Networks {
		if net.HostBridge != nil {
			c := exec.Command("virt-handler",
				"--create-bridge",
				"--pod-namespace", podNamespaces.NetNamespace(),
				"--host-namespace", hostNamespaces.NetNamespace(),
				"--node-bridge-name", net.HostBridge.NodeBridgeName,
				"--bridge-name", net.HostBridge.BridgeName,
				"--network-name", net.Name,
				fmt.Sprintf("--interface-index=%d", i))

			if output, err := c.CombinedOutput(); err != nil {
				return fmt.Errorf(string(output))
			}
		}
	}
	return nil
}

func (HostBridge) Available() error {
	return nil
}

func randString(length int) string {
	b := make([]byte, length)
	letterBytes := "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
