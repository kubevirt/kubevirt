package components

import (
	"fmt"
	"runtime"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	virtv1 "kubevirt.io/api/core/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	VirtHandlerName = "virt-handler"
	kubeletPodsPath = "/var/lib/kubelet/pods"
)

func NewHandlerDaemonSet(namespace string, repository string, imagePrefix string, version string, launcherVersion string, productName string, productVersion string, productComponent string, pullPolicy corev1.PullPolicy, migrationNetwork *string, verbosity string, extraEnv map[string]string) (*appsv1.DaemonSet, error) {

	deploymentName := VirtHandlerName
	imageName := fmt.Sprintf("%s%s", imagePrefix, deploymentName)
	env := operatorutil.NewEnvVarMap(extraEnv)
	podTemplateSpec, err := newPodTemplateSpec(deploymentName, imageName, repository, version, productName, productVersion, productComponent, pullPolicy, nil, env)
	if err != nil {
		return nil, err
	}

	if migrationNetwork != nil {
		if podTemplateSpec.ObjectMeta.Annotations == nil {
			podTemplateSpec.ObjectMeta.Annotations = make(map[string]string)
		}
		// Join the pod to the migration network and name the corresponding interface "migration0"
		podTemplateSpec.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/networks"] = *migrationNetwork + "@migration0"
	}

	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      VirtHandlerName,
			Labels: map[string]string{
				virtv1.AppLabel: VirtHandlerName,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io": VirtHandlerName,
				},
			},
			Template: *podTemplateSpec,
		},
	}

	if productVersion != "" {
		daemonset.ObjectMeta.Labels[virtv1.AppVersionLabel] = productVersion
	}

	if productName != "" {
		daemonset.ObjectMeta.Labels[virtv1.AppPartOfLabel] = productName
	}
	if productComponent != "" {
		daemonset.ObjectMeta.Labels[virtv1.AppComponentLabel] = productComponent
	}

	pod := &daemonset.Spec.Template.Spec
	pod.ServiceAccountName = rbac.HandlerServiceAccountName
	pod.HostPID = true

	// nodelabeller currently only support x86
	if virtconfig.IsAMD64(runtime.GOARCH) {
		launcherVersion = AddVersionSeparatorPrefix(launcherVersion)
		pod.InitContainers = []corev1.Container{
			{
				Command: []string{
					"/bin/sh",
					"-c",
				},
				Image: fmt.Sprintf("%s/%s%s%s", repository, imagePrefix, "virt-launcher", launcherVersion),
				Name:  "virt-launcher",
				Args: []string{
					"node-labeller.sh",
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: boolPtr(true),
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "node-labeller",
						MountPath: nodeLabellerVolumePath,
					},
				},
			},
		}
	}

	// give the handler grace period some padding
	// in order to ensure we have a chance to cleanly exit
	// before SIG_KILL
	podGracePeriod := int64(330)
	handlerGracePeriod := podGracePeriod - 15
	podTemplateSpec.Spec.TerminationGracePeriodSeconds = &podGracePeriod

	container := &pod.Containers[0]
	container.Command = []string{
		VirtHandlerName,
	}
	container.Args = []string{
		"--port",
		"8443",
		"--hostname-override",
		"$(NODE_NAME)",
		"--pod-ip-address",
		"$(MY_POD_IP)",
		"--max-metric-requests",
		"3",
		"--console-server-port",
		"8186",
		"--graceful-shutdown-seconds",
		fmt.Sprintf("%d", handlerGracePeriod),
		"-v",
		verbosity,
	}
	container.Ports = []corev1.ContainerPort{
		{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8443,
		},
	}
	container.SecurityContext = &corev1.SecurityContext{
		Privileged: boolPtr(true),
		SELinuxOptions: &corev1.SELinuxOptions{
			Level: "s0",
		},
	}
	containerEnv := []corev1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	}

	container.Env = append(container.Env, containerEnv...)

	container.LivenessProbe = &corev1.Probe{
		FailureThreshold: 3,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
				Path: "/healthz",
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      10,
		PeriodSeconds:       45,
	}
	container.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
				Path: "/healthz",
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      10,
		PeriodSeconds:       20,
	}

	type volume struct {
		name             string
		path             string
		mountPath        string
		mountPropagation *corev1.MountPropagationMode
	}
	attachCertificateSecret(pod, VirtHandlerCertSecretName, "/etc/virt-handler/clientcertificates")
	attachCertificateSecret(pod, VirtHandlerServerCertSecretName, "/etc/virt-handler/servercertificates")
	attachProfileVolume(pod)

	bidi := corev1.MountPropagationBidirectional
	// NOTE: the 'kubelet-pods-shortened' volume mounts the same host path as 'kubelet-pods'
	// This is because that path holds unix domain sockets. Domain sockets fail when they're over
	// ~ 100 characters, so that shortened volume path is to allow domain socket connections.
	// It's ridiculous to have to account for that, but that's the situation we're in.
	volumes := []volume{
		{"libvirt-runtimes", "/var/run/kubevirt-libvirt-runtimes", "/var/run/kubevirt-libvirt-runtimes", nil},
		{"virt-share-dir", "/var/run/kubevirt", "/var/run/kubevirt", &bidi},
		{"virt-lib-dir", "/var/lib/kubevirt", "/var/lib/kubevirt", nil},
		{"virt-private-dir", "/var/run/kubevirt-private", "/var/run/kubevirt-private", nil},
		{"device-plugin", "/var/lib/kubelet/device-plugins", "/var/lib/kubelet/device-plugins", nil},
		{"kubelet-pods-shortened", kubeletPodsPath, "/pods", nil},
		{"kubelet-pods", kubeletPodsPath, kubeletPodsPath, &bidi},
		{"node-labeller", "/var/lib/kubevirt-node-labeller", "/var/lib/kubevirt-node-labeller", nil},
	}

	for _, volume := range volumes {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:             volume.name,
			MountPath:        volume.mountPath,
			MountPropagation: volume.mountPropagation,
		})
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: volume.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volume.path,
				},
			},
		})
	}

	// Use the downward API to access the network status annotations
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      "podinfo",
		MountPath: "/etc/podinfo",
	})
	pod.Volumes = append(pod.Volumes, corev1.Volume{
		Name: "podinfo",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path: "network-status",
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: `metadata.annotations['k8s.v1.cni.cncf.io/network-status']`,
						},
					},
				},
			},
		},
	})

	container.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("10m"),
			corev1.ResourceMemory: resource.MustParse("230Mi"),
		},
	}

	return daemonset, nil

}
