package components

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
)

// DeploymentData contains all needed data to create new deployment object
type DeploymentData struct {
	Name            string
	Namespace       string
	ImageRepository string
	PullPolicy      corev1.PullPolicy
	Verbosity       string
	OperatorVersion string
}

func getImage(name string, imageRepository string, imageTag string) string {
	return fmt.Sprintf("%s/%s:%s", imageRepository, name, imageTag)
}

// NewDeployment returns new deployment object
func NewDeployment(data *DeploymentData) *appsv1.Deployment {
	template := newPodTemplateSpec(data)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.Namespace,
			Labels: map[string]string{
				mrv1.SchemeGroupVersion.Group: data.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					mrv1.SchemeGroupVersion.Group:              data.Name,
					mrv1.SchemeGroupVersion.Group + "/version": data.OperatorVersion,
				},
			},
			Template: *template,
		},
	}
}

func newPodTemplateSpec(data *DeploymentData) *corev1.PodTemplateSpec {
	containers := newContainers(data)
	tolerations := []corev1.Toleration{
		{
			Key:    "node-role.kubernetes.io/master",
			Effect: corev1.TaintEffectNoSchedule,
		},
		{
			Key:      "CriticalAddonsOnly",
			Operator: corev1.TolerationOpExists,
		},
		{
			Key:               "node.kubernetes.io/not-ready",
			Effect:            corev1.TaintEffectNoExecute,
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: pointer.Int64Ptr(120),
		},
		{
			Key:               "node.kubernetes.io/unreachable",
			Effect:            corev1.TaintEffectNoExecute,
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: pointer.Int64Ptr(120),
		},
	}

	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				mrv1.SchemeGroupVersion.Group:              data.Name,
				mrv1.SchemeGroupVersion.Group + "/version": data.OperatorVersion,
			},
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      mrv1.SchemeGroupVersion.Group,
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{data.Name},
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
			Containers:   containers,
			NodeSelector: map[string]string{"node-role.kubernetes.io/master": ""},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: pointer.BoolPtr(true),
			},
			ServiceAccountName: data.Name,
			Tolerations:        tolerations,
		},
	}
}

func newContainers(data *DeploymentData) []corev1.Container {
	resources := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceMemory: resource.MustParse("20Mi"),
			corev1.ResourceCPU:    resource.MustParse("10m"),
		},
	}
	args := []string{
		"--logtostderr=true",
		fmt.Sprintf("--v=%s", data.Verbosity),
		fmt.Sprintf("--namespace=%s", data.Namespace),
	}

	containers := []corev1.Container{
		{
			Name:            data.Name,
			Image:           getImage(data.Name, data.ImageRepository, data.OperatorVersion),
			Command:         []string{fmt.Sprintf("/usr/bin/%s", data.Name)},
			Args:            args,
			Resources:       resources,
			ImagePullPolicy: data.PullPolicy,
			Env: []corev1.EnvVar{
				{
					Name:  EnvVarOperatorVersion,
					Value: data.OperatorVersion,
				},
			},
		},
	}
	return containers
}
