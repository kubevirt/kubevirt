package configuration

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

func NewListPermittedDevices() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permitted-devices",
		Short:   "List the permitted devices for vmis.",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE:    run,
	}

	return cmd
}

func usage() string {
	return "# Print the permitted devices for VMIs:\n  {{ProgramName}} permitted-devices"
}

func run(cmd *cobra.Command, _ []string) error {
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	kvList, err := virtClient.KubeVirt(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(kvList.Items) == 0 {
		return fmt.Errorf("no KubeVirt resource in %q namespace", namespace)
	}

	kv := kvList.Items[0]

	var (
		hostDeviceList []string
		gpuDeviceList  []string
	)

	if kv.Spec.Configuration.PermittedHostDevices != nil {
		for _, hd := range kv.Spec.Configuration.PermittedHostDevices.PciHostDevices {
			hostDeviceList = append(hostDeviceList, hd.ResourceName)
		}

		for _, hd := range kv.Spec.Configuration.PermittedHostDevices.MediatedDevices {
			gpuDeviceList = append(gpuDeviceList, hd.ResourceName)
		}
	}

	cmd.Printf("Permitted Devices: \nHost Devices: \n%s \nGPU Devices: \n%s\n",
		strings.Join(hostDeviceList, ", "),
		strings.Join(gpuDeviceList, ", "),
	)

	return nil
}
