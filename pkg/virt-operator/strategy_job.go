package virt_operator

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/placement"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func (c *KubeVirtController) generateInstallStrategyJob(infraPlacement *v1.ComponentConfig, config *operatorutil.KubeVirtDeploymentConfig) (*batchv1.Job, error) {

	operatorImage := config.VirtOperatorImage
	if operatorImage == "" {
		operatorImage = fmt.Sprintf("%s/%s%s%s", config.GetImageRegistry(), config.GetImagePrefix(), VirtOperator, components.AddVersionSeparatorPrefix(config.GetOperatorVersion()))
	}
	deploymentConfigJson, err := config.GetJson()
	if err != nil {
		return nil, err
	}

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},

		ObjectMeta: metav1.ObjectMeta{
			Namespace:    c.operatorNamespace,
			GenerateName: fmt.Sprintf("kubevirt-%s-job", config.GetDeploymentID()),
			Labels: map[string]string{
				v1.AppLabel:             "",
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				// Deprecated, keep it for backwards compatibility
				v1.InstallStrategyVersionAnnotation: config.GetKubeVirtVersion(),
				// Deprecated, keep it for backwards compatibility
				v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
				v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
			},
		},
		Spec: batchv1.JobSpec{
			Template: k8sv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						v1.AppLabel:                          virtOperatorJobAppLabel,
						v1.AllowAccessClusterServicesNPLabel: "true",
					},
				},
				Spec: k8sv1.PodSpec{
					ServiceAccountName: "kubevirt-operator",
					RestartPolicy:      k8sv1.RestartPolicyNever,
					ImagePullSecrets:   config.GetImagePullSecrets(),

					Containers: []k8sv1.Container{
						{
							Name:            "install-strategy-upload",
							Image:           operatorImage,
							ImagePullPolicy: config.GetImagePullPolicy(),
							Command: []string{
								VirtOperator,
								"--dump-install-strategy",
							},
							Env: []k8sv1.EnvVar{
								{
									Name:  util.VirtOperatorImageEnvName,
									Value: operatorImage,
								},
								{
									// Deprecated, keep it for backwards compatibility
									Name:  util.TargetInstallNamespace,
									Value: config.GetNamespace(),
								},
								{
									// Deprecated, keep it for backwards compatibility
									Name:  util.TargetImagePullPolicy,
									Value: string(config.GetImagePullPolicy()),
								},
								{
									Name:  util.TargetDeploymentConfig,
									Value: deploymentConfigJson,
								},
							},
							SecurityContext: &k8sv1.SecurityContext{
								AllowPrivilegeEscalation: pointer.P(false),
								Capabilities: &k8sv1.Capabilities{
									Drop: []k8sv1.Capability{"ALL"},
								},
								RunAsNonRoot: pointer.P(true),
								SeccompProfile: &k8sv1.SeccompProfile{
									Type: k8sv1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
				},
			},
		},
	}

	placement.InjectPlacementMetadata(infraPlacement, &job.Spec.Template.Spec, placement.RequireControlPlanePreferNonWorker)
	env := job.Spec.Template.Spec.Containers[0].Env
	extraEnv := util.NewEnvVarMap(config.GetExtraEnv())
	job.Spec.Template.Spec.Containers[0].Env = append(env, *extraEnv...)

	return job, nil
}
func (c *KubeVirtController) getInstallStrategyJob(config *operatorutil.KubeVirtDeploymentConfig) (*batchv1.Job, bool) {
	objs := c.stores.InstallStrategyJobCache.List()
	for _, obj := range objs {
		if job, ok := obj.(*batchv1.Job); ok {
			if job.Annotations == nil {
				continue
			}

			if idAnno, ok := job.Annotations[v1.InstallStrategyIdentifierAnnotation]; ok && idAnno == config.GetDeploymentID() {
				return job, true
			}

		}
	}
	return nil, false
}

func (c *KubeVirtController) garbageCollectInstallStrategyJobs() error {
	batch := c.k8sClientset.BatchV1()
	jobs := c.stores.InstallStrategyJobCache.List()

	for _, obj := range jobs {
		job, ok := obj.(*batchv1.Job)
		if !ok {
			continue
		}
		if job.Status.CompletionTime == nil {
			continue
		}

		propagationPolicy := metav1.DeletePropagationForeground
		err := batch.Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			return err
		}
		log.Log.Object(job).Infof("Garbage collected completed install strategy job")
	}

	return nil
}
