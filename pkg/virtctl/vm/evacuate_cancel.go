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
		Use:     "evacuate-cancel (vm/<vm-name> | vmi/<vmi-name> | node/<node-name>)",
		Short:   "Cancel evacuation for a VM, VMI, or all VMIs on a node",
		Example: usage("evacuate-cancel"),
		Args:    cobra.ExactArgs(1),
		RunE:    c.Run,
	}

	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

type evacuateCancelCommand struct{}

func (c evacuateCancelCommand) Run(cmd *cobra.Command, args []string) error {
	kind, name, err := parseTarget(args)
	if err != nil {
		return err
	}

	handler, err := c.getHandler(kind)
	if err != nil {
		return err
	}

	opts := &virtv1.EvacuateCancelOptions{}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Dry Run execution")
	}

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	err = handler(cmd, virtClient, name, namespace, opts)
	if err != nil {
		return err
	}

	return nil
}

type evacuateCancelHandler func(cmd *cobra.Command, virtClient kubecli.KubevirtClient, name, namespace string, opts *virtv1.EvacuateCancelOptions) error

func (c evacuateCancelCommand) getHandler(kind string) (evacuateCancelHandler, error) {
	switch strings.ToLower(kind) {
	case "vm", "vms", "virtualmachine", "virtualmachines":
		return c.handleVM, nil
	case "vmi", "vmis", "virtualmachineinstance", "virtualmachineinstances":
		return c.handleVMI, nil
	case "node", "nodes", "no":
		return c.handleNode, nil
	}
	return nil, fmt.Errorf("unsupported resource type '%s'", kind)
}

func (c evacuateCancelCommand) handleVM(cmd *cobra.Command, virtClient kubecli.KubevirtClient, name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := virtClient.VirtualMachine(namespace).EvacuateCancel(context.Background(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VM %s/%s: %w", namespace, name, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "VM %s/%s was canceled evacuation\n", namespace, name)
	return nil
}

func (c evacuateCancelCommand) handleVMI(cmd *cobra.Command, virtClient kubecli.KubevirtClient, name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := virtClient.VirtualMachineInstance(namespace).EvacuateCancel(context.Background(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VMI %s/%s: %w", namespace, name, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "VMI %s/%s was canceled evacuation\n", namespace, name)
	return nil
}

func (c evacuateCancelCommand) handleNode(cmd *cobra.Command, virtClient kubecli.KubevirtClient, name, _ string, opts *virtv1.EvacuateCancelOptions) error {
	node, err := virtClient.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting node %s: %w", name, err)
	}

	vmiList, err := virtClient.VirtualMachineInstance(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing VMIs on node %s: %w", name, err)
	}

	for _, vmi := range vmiList.Items {
		if vmi.Status.EvacuationNodeName == node.Name {
			err = c.handleVMI(cmd, virtClient, vmi.Name, vmi.Namespace, opts)
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
			}
		}
	}

	return nil
}

func parseTarget(args []string) (kind, name string, err error) {
	parts := strings.Split(args[0], "/")
	switch len(parts) {
	case 1:
		return "", "", fmt.Errorf("target must contain type and name separated by '/'")
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("target is not valid with more than two '/'")
	}
}
