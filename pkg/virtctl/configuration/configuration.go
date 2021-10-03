package configuration

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewListPermittedDevices(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permitted-devices",
		Short:   "List the permitted devices for vmis.",
		Example: usage(),
		Args:    templates.ExactArgs("permitted-devices", 0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := command{clientConfig: clientConfig}
			return c.run()
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "  # Print the permitted devices for VMIs:\n"
	usage += "  {{ProgramName}} permitted-devices"
	return usage
}

type command struct {
	clientConfig clientcmd.ClientConfig
}

func (c *command) run() error {

	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	kvList, err := virtClient.KubeVirt(namespace).List(&metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(kvList.Items) == 0 {
		return fmt.Errorf("No KubeVirt resource in %q namespace", namespace)
	}

	kv := kvList.Items[0]

	hostDeviceList := []string{}
	gpuDeviceList := []string{}

	if kv.Spec.Configuration.PermittedHostDevices != nil {
		for _, hd := range kv.Spec.Configuration.PermittedHostDevices.PciHostDevices {
			hostDeviceList = append(hostDeviceList, hd.ResourceName)
		}

		for _, hd := range kv.Spec.Configuration.PermittedHostDevices.MediatedDevices {
			gpuDeviceList = append(gpuDeviceList, hd.ResourceName)
		}
	}

	fmt.Printf("Permitted Devices: \nHost Devices: \n%s \nGPU Devices: \n%s\n",
		fmt.Sprint(strings.Join(hostDeviceList, ", ")),
		fmt.Sprint(strings.Join(gpuDeviceList, ", ")),
	)

	return nil
}
