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

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	kubevirtNamespace = "kubevirt"
	machineTypeCmd    = "machine-types"
)

type MachineTypeCommand struct {
	clientConfig clientcmd.ClientConfig
}

// holding flag information
var (
	namespaceFlag     string
	restartNowFlag    bool
	labelSelectorFlag string
	machineTypeFlag   string
)

// NewConvertMachineTypeCommand generates a new "convert-machine-types" command
func NewMachineTypeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   machineTypeCmd,
		Short: "Perform a mass machine type transition on any VMs with a machine type matching the specified glob.",
		Long: `Create and deploy a Job that iterates through VMs, updating the machine type of any VMs that match the specified machine type to the latest machine type. If a VM is running, it will also label the VM with 'restart-vm-required=true', indicating the user will need to perform manually by default. If --force-restart is set to true, the VM will be automatically restarted and the label will be removed. The Job will terminate once all VMs have their machine types updated, and all 'restart-vm-required' labels have been cleared.
		If no namespace is specified via --namespace, the mass machine type transition will be applied across all namespaces.
		Note that should the Job fail, it will be restarted. Additonally, once the Job is terminated, it will not be automatically deleted. The Job can be monitored and then deleted manually after it has been terminated using 'kubectl' commands.`,
		Example: usage(),
		Args:    templates.ExactArgs(machineTypeCmd, 0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := MachineTypeCommand{clientConfig: clientConfig}
			return c.Run()
		},
	}

	// flags for the "expose" command
	cmd.Flags().StringVar(&machineTypeFlag, "which-matches-glob", "", "Machine type to be updated. This flag is required.")
	cmd.MarkFlagRequired("which-matches-glob")
	cmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Namespace in which the mass machine type transition will be applied. Leave empty to apply to all namespaces.")
	cmd.Flags().BoolVar(&restartNowFlag, "restart-now", false, "When true, immediately restarts all VMs that have their machine types updated. Otherwise, updated VMs must be restarted manually for the machine type change to take effect.")
	cmd.Flags().StringVar(&labelSelectorFlag, "label-selector", "", "Selector (label query) on which to filter VMs to be updated.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := `  # Update the machine types of all VMs with the designated machine type across all namespaces without automatically restarting running VMs:
  {{ProgramName}} update machine-types --which-matches-glob=*q35-2.*

  # Update the machine types of all VMs with the designated machine type in the namespace 'default':
  {{ProgramName}} update machine-types --which-matches-glob=*q35-2.* --namespace=default

  # Update the machine types of all VMs with the designated machine type and automatically restart them if they are running:
  {{ProgramName}} update machine-types --which-matches-glob=*q35-2.* --restart-now=true
  
  # Update the machine types of all VMs with the designated machine type and with the label 'kubevirt.io/memory=large':
  {{ProgramName}} update machine-types --which-matches-glob=*q35-2.* --label-selector=kubevirt.io/memory=large`
	return usage
}

// executing the "expose" command
func (o *MachineTypeCommand) Run() error {
	// get the client
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	job := generateMassMachineTypeTransitionJob()
	batch := virtClient.BatchV1()
	_, err = batch.Jobs(kubevirtNamespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating convert-machine-type job: %v", err)
	}
	fmt.Printf(`Successfully created job %s.
This job can be monitored using 'kubectl get job %s -n kubevirt' and 'kubectl describe job %s -n kubevirt'.
Once terminated, this job can be deleted by using 'kubectl delete job %s -n kubevirt'.\n`, job.Name, job.Name, job.Name, job.Name)
	return nil
}

func generateMassMachineTypeTransitionJob() *batchv1.Job {
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
							Image: "registry:5000/kubevirt/mass-machine-type-transition:devel",
							Env: []v1.EnvVar{
								{
									Name:  "MACHINE_TYPE",
									Value: machineTypeFlag,
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
