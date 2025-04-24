package vm

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const COMMAND_EVACUATE_CANCEL = "evacuate-cancel"

func NewEvacuateCancelCommand() *cobra.Command {
	c := &evacuateCancelCommand{}

	cmd := &cobra.Command{
		Use:     "evacuate-cancel (VM)",
		Short:   "Cancel evacuation of a virtual machine.",
		Example: usage(COMMAND_EVACUATE_CANCEL),
		Args:    cobra.MaximumNArgs(1),
		RunE:    c.Run,
	}

	cmd.Flags().StringVar(&c.nodeName, "node-name", "", "The name of the node where evacuation will be canceled for all virtual machines.")
	cmd.Flags().BoolVar(&c.dryRun, "dry-run", false, "If true, only print out actions that would be taken.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

type evacuateCancelCommand struct {
	nodeName string
	dryRun   bool
}

func (c evacuateCancelCommand) Run(cmd *cobra.Command, args []string) error {
	client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	var vms []types.NamespacedName

	switch {
	case c.nodeName != "":
		_, err := client.CoreV1().Nodes().Get(cmd.Context(), c.nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Error fetching node %q: %v", c.nodeName, err)
		}

		vmiList, err := client.VirtualMachineInstance("").List(cmd.Context(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("Error fetching virtual machine instance list  %v", err)
		}

		for _, vmi := range vmiList.Items {
			if vmi.Status.EvacuationNodeName == c.nodeName {
				vms = append(vms, types.NamespacedName{Namespace: vmi.Namespace, Name: vmi.Name})
			}
		}

	case len(args) > 0:
		vmName = args[0]
		vms = append(vms, types.NamespacedName{Namespace: namespace, Name: vmName})
	default:
		return fmt.Errorf("VirtualMachine name or NodeName is required")
	}

	opts := &virtv1.EvacuateCancelOptions{}
	if c.dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}

	for _, vm := range vms {
		if err := client.VirtualMachine(vm.Namespace).EvacuateCancel(cmd.Context(), vm.Name, opts); err != nil {
			return fmt.Errorf("Error canceling evacuation for %q: %v", vm.String(), err)
		}
	}

	return nil
}
