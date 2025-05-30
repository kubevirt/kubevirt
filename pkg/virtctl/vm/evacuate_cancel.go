package vm

import (
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
		Example: usageEvacuateCancel(),
		Args:    cobra.ExactArgs(2),
		RunE:    c.Run,
	}

	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.Flags().BoolVar(&c.MigrateCancel, "migrate-cancel", false, "Also cancel an active migration for the specified VMI")

	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usageEvacuateCancel() string {
	return `  # Cancel evacuation for a virtual machine
  {{ProgramName}} evacuate-cancel vm my-vm

  # Cancel evacuation for a virtual machine instance
  {{ProgramName}} evacuate-cancel vmi my-vmi

  # Cancel evacuation and also cancel an active migration for the VMI
  {{ProgramName}} evacuate-cancel vmi my-vmi --migrate-cancel

  # Cancel evacuation for all VMIs on a specific node
  {{ProgramName}} evacuate-cancel node node01
  `
}

type evacuateCancelCommand struct {
	MigrateCancel bool
	cmd           *cobra.Command
	virtClient    kubecli.KubevirtClient
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

	opts := &virtv1.EvacuateCancelOptions{
		DryRun: setDryRunOption(dryRun),
	}

	return handler(name, namespace, opts)
}

func (c *evacuateCancelCommand) getHandler(kind string) (func(name, namespace string, opts *virtv1.EvacuateCancelOptions) error, error) {
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
	err := c.virtClient.VirtualMachine(namespace).EvacuateCancel(c.cmd.Context(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VM %s/%s: %w", namespace, name, err)
	}
	c.cmd.Printf("VM %s/%s was canceled evacuation\n", namespace, name)
	return c.runMigrateCancel(name, namespace)
}

func (c *evacuateCancelCommand) handleVMI(name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := c.virtClient.VirtualMachineInstance(namespace).EvacuateCancel(c.cmd.Context(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VMI %s/%s: %w", namespace, name, err)
	}
	c.cmd.Printf("VMI %s/%s was canceled evacuation\n", namespace, name)
	return c.runMigrateCancel(name, namespace)
}

func (c *evacuateCancelCommand) handleNode(name, _ string, opts *virtv1.EvacuateCancelOptions) error {
	node, err := c.virtClient.CoreV1().Nodes().Get(c.cmd.Context(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting node %s: %w", name, err)
	}

	vmiList, err := c.virtClient.VirtualMachineInstance(metav1.NamespaceAll).List(c.cmd.Context(), metav1.ListOptions{})
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

func (c *evacuateCancelCommand) runMigrateCancel(vmiName, namespace string) error {
	if !c.MigrateCancel {
		return nil
	}
	c.cmd.Printf("Invoking %q for VMI %s/%s\n", COMMAND_MIGRATE_CANCEL, namespace, vmiName)
	return migrateCancel(c.cmd.Context(), c.virtClient, vmiName, namespace)
}
