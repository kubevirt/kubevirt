package list

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var allNamespaces bool

func ListCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List virtualmachines",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := List{clientConfig: clientConfig}
			return v.Run(cmd, args)
		},
	}
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "a", allNamespaces, "All Namespaces")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "  # List virtualmachines:\n"
	usage += "  virtctl list"
	return usage
}

type List struct {
	clientConfig clientcmd.ClientConfig
}

func (o *List) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	options := k8smetav1.ListOptions{}
	vms, err := virtClient.VirtualMachine(namespace).List(&options)
	if err != nil {
		return fmt.Errorf("Error Listing VirtualMachines: %v", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintf(w, "Name\tStatus\tIP\tSource\n")
	fmt.Fprintf(w, "----\t------\t--\t------\n")
	for _, vm := range vms.Items {
		name := vm.Name
		status := "down"
		ip := ""
		source := ""
		running := vm.Spec.Running
		if running {
			getoptions := k8smetav1.GetOptions{}
			vmi, err := virtClient.VirtualMachineInstance(namespace).Get(name, &getoptions)
			if err != nil {
				return fmt.Errorf("Error fetching VirtualMachineInstance: %v", err)
			}
			status = fmt.Sprintf("%v", vmi.Status.Phase)
			interfaces := vmi.Status.Interfaces
			if len(interfaces) > 0 {
				ip = interfaces[0].IP
			}
		}
		volumes := vm.Spec.Template.Spec.Volumes
		if len(volumes) > 0 {
			pvc := volumes[0].PersistentVolumeClaim
			if pvc != nil {
				source = fmt.Sprintf("%v", pvc.ClaimName)
			}
			registrydisk := volumes[0].RegistryDisk
			if registrydisk != nil {
				source = fmt.Sprintf("%v", registrydisk.Image)
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, status, ip, source)
	}
	w.Flush()

	return nil
}
