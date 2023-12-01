package machinetype

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	kubevirtNamespace = "kubevirt"
	machineTypeCmd    = "machine-types"
	defaultImageName  = "mass-machine-type-transition"
)

type MachineTypeCommand struct {
	clientConfig clientcmd.ClientConfig
}

// holding flag information
var (
	namespaceFlag     string
	restartNowFlag    bool
	labelSelectorFlag string
	image             string
)

// NewMachineTypeCommand generates a new "update machine-types" command
func NewMachineTypeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   machineTypeCmd,
		Short: "Perform a mass machine type transition on any VMs with a machine type matching the specified glob.",
		Long: `Create and deploy a Job that iterates through VMs, updating the machine type of any VMs that match the machine type glob specified by argument to the latest machine type.
If a VM is running, it will also update the VM Status and set MachineTypeRestartRequired to true, indicating the user will need to perform manually by default.
If --restart-now is set to true, the VM will be automatically restarted and MachineTypeRestartRequired will not be updated.
The Job will terminate once all VMs have their machine types updated, and no VMs remain with MachineTypeRestartRequired set to true.
If no namespace is specified via --namespace, the mass machine type transition will be applied across all namespaces.
The --label-selector flag can be used to further limit which VMs the machine type update will be applied to.
Note that should the Job fail, it will be restarted. Additonally, once the Job is terminated, it will not be automatically deleted.
The Job can be monitored and then deleted manually after it has been terminated using 'kubectl' commands.`,
		Example: usage(),
		Args:    templates.ExactArgs(machineTypeCmd, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := MachineTypeCommand{clientConfig: clientConfig}
			return c.Run(args)
		},
	}

	// flags for the "update machine-types" command
	cmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Namespace in which the mass machine type transition will be applied. Defaults to all namespaces.")
	cmd.Flags().BoolVar(&restartNowFlag, "restart-now", restartNowFlag, "When true, immediately restarts all VMs that have their machine types updated. Otherwise, updated VMs must be restarted manually for the machine type change to take effect. Defaults to false.")
	cmd.Flags().StringVar(&labelSelectorFlag, "label-selector", "", "Selector (label query) on which to filter VMs to be updated.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := `  # Update the machine types of all VMs with the designated machine type across all namespaces without automatically restarting running VMs:
  {{ProgramName}} update machine-types *q35-2.*

  # Update the machine types of all VMs with the designated machine type in the namespace 'default':
  {{ProgramName}} update machine-types *q35-2.* --namespace=default

  # Update the machine types of all VMs with the designated machine type and automatically restart them if they are running:
  {{ProgramName}} update machine-types *q35-2.* --restart-now=true
  
  # Update the machine types of all VMs with the designated machine type and with the label 'kubevirt.io/memory=large':
  {{ProgramName}} update machine-types *q35-2.* --label-selector=kubevirt.io/memory=large`
	return usage
}

// executing the "update machine-types" command
func (o *MachineTypeCommand) Run(args []string) error {
	machineTypeGlob := args[0]

	// get the client
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	// set the image name
	err = setImage(virtClient)
	if err != nil {
		return err
	}

	job := generateMassMachineTypeTransitionJob(machineTypeGlob)
	batch := virtClient.BatchV1()
	job, err = batch.Jobs(kubevirtNamespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating convert-machine-type job: %v", err)
	}
	fmt.Printf(`
Successfully created job %s.
This job can be monitored using 'kubectl get job %s -n kubevirt' and 'kubectl describe job %s -n kubevirt'.
Once terminated, this job can be deleted by using 'kubectl delete job %s -n kubevirt'.
`, job.Name, job.Name, job.Name, job.Name)
	return nil
}

func generateMassMachineTypeTransitionJob(machineTypeGlob string) *batchv1.Job {
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},

		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "convert-machine-type-",
			Namespace:    kubevirtNamespace,
		},

		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  machineTypeCmd,
							Image: image,
							Env: []v1.EnvVar{
								{
									Name:  "MACHINE_TYPE",
									Value: machineTypeGlob,
								},
								{
									Name:  "NAMESPACE",
									Value: namespaceFlag,
								},
								{
									Name:  "RESTART_NOW",
									Value: strconv.FormatBool(restartNowFlag),
								},
								{
									Name:  "LABEL_SELECTOR",
									Value: labelSelectorFlag,
								},
							},
							SecurityContext: &v1.SecurityContext{
								AllowPrivilegeEscalation: pointer.Bool(false),
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"ALL"},
								},
								SeccompProfile: &v1.SeccompProfile{
									Type: v1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
					SecurityContext: &v1.PodSecurityContext{
						RunAsNonRoot: pointer.Bool(true),
					},
					ServiceAccountName: "convert-machine-type",
					RestartPolicy:      v1.RestartPolicyOnFailure,
				},
			},
		},
	}
	return job
}

// setImage sets the image name based on the information retrieved by the KubeVirt server.
func setImage(virtClient kubecli.KubevirtClient) error {
	info, err := getImageInfo(virtClient)
	if err != nil {
		return fmt.Errorf("could not get guestfs image info: %v", err)
	}
	image = fmt.Sprintf("%s/%s%s%s", info.Registry, info.ImagePrefix, defaultImageName, components.AddVersionSeparatorPrefix(info.Tag))
	return nil
}

func getImageInfo(virtClient kubecli.KubevirtClient) (*kubecli.GuestfsInfo, error) {
	info, err := virtClient.GuestfsVersion().Get()
	if err != nil {
		return nil, err
	}

	return info, nil
}
