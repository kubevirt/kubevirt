package vm

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewEvacuateCancelCommand() *cobra.Command {
	c := &evacuateCancelCommand{}

	cmd := &cobra.Command{
		Use:     "evacuate-cancel (vm <vm-name> | vmi <vmi-name> | node <node-name>)",
		Short:   "Cancel evacuation for a VM, VMI, or all VMIs on a node",
		Example: usage("evacuate-cancel"),
		Args:    cobra.ExactArgs(2),
		RunE:    c.Run,
	}

	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

type evacuateCancelCommand struct {
	cmd        *cobra.Command
	virtClient kubecli.KubevirtClient
}

func (c *evacuateCancelCommand) Run(cmd *cobra.Command, args []string) error {
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	c.cmd = cmd
	c.virtClient = virtClient

	kind := args[0]
	name := args[1]

	handler, err := c.getHandler(kind)
	if err != nil {
		return err
	}

	opts := &virtv1.EvacuateCancelOptions{}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
		c.cmd.Printf("Dry Run execution")
	}

	return handler(name, namespace, opts)
}

type evacuateCancelHandler func(name, namespace string, opts *virtv1.EvacuateCancelOptions) error

func (c *evacuateCancelCommand) getHandler(kind string) (evacuateCancelHandler, error) {
	switch strings.ToLower(kind) {
	case "vm", "vms", "virtualmachine", "virtualmachines":
		return c.handleVM, nil
	case "vmi", "vmis", "virtualmachineinstance", "virtualmachineinstances":
		return c.handleVMI, nil
	case "node", "nodes", "no":
		return c.handleNode, nil
	}
	return nil, fmt.Errorf("unsupported resource type %q", kind)
}

func (c *evacuateCancelCommand) handleVM(name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := c.virtClient.VirtualMachine(namespace).EvacuateCancel(context.Background(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VM %s/%s: %w", namespace, name, err)
	}
	c.cmd.Printf("VM %s/%s was canceled evacuation\n", namespace, name)
	return nil
}

func (c *evacuateCancelCommand) handleVMI(name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := c.virtClient.VirtualMachineInstance(namespace).EvacuateCancel(context.Background(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VMI %s/%s: %w", namespace, name, err)
	}
	c.cmd.Printf("VMI %s/%s was canceled evacuation\n", namespace, name)
	return nil
}

func (c *evacuateCancelCommand) handleNode(name, _ string, opts *virtv1.EvacuateCancelOptions) error {
	node, err := c.virtClient.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting node %s: %w", name, err)
	}

	vmiList, err := c.virtClient.VirtualMachineInstance(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing VMIs on node %s: %w", name, err)
	}

	for _, vmi := range vmiList.Items {
		if vmi.Status.EvacuationNodeName == node.Name {
			err = c.handleVMI(vmi.Name, vmi.Namespace, opts)
			if err != nil {
				c.cmd.PrintErr(err)
			}
		}
	}

	return nil
}
