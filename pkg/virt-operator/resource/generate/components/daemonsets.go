package components

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/util"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	VirtHandlerName                = "virt-handler"
	kubeletPodsPath                = util.KubeletRoot + "/pods"
	runtimesPath                   = "/var/run/kubevirt-libvirt-runtimes"
	PrHelperName                   = "pr-helper"
	prVolumeName                   = "pr-helper-socket-vol"
	devDirVol                      = "dev-dir"
	SidecarShimName                = "sidecar-shim"
	etcMultipath                   = "etc-multipath"
	SupportsMigrationCNsValidation = "kubevirt.io/supports-migration-cn-types"
)

func RenderPrHelperContainer(image string, pullPolicy corev1.PullPolicy) corev1.Container {
	bidi := corev1.MountPropagationBidirectional
	return corev1.Container{
		Name:            PrHelperName,
		Image:           image,
		ImagePullPolicy: pullPolicy,
		Command:         []string{"/entrypoint.sh"},
		Args: []string{
			"-k", reservation.GetPrHelperSocketPath(),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:             prVolumeName,
				MountPath:        reservation.GetPrHelperSocketDir(),
				MountPropagation: &bidi,
			},
			{
				Name:             devDirVol,
				MountPath:        "/dev",
				MountPropagation: pointer.P(corev1.MountPropagationHostToContainer),
			},
			{
				Name:             etcMultipath,
				MountPath:        "/etc/multipath",
				MountPropagation: &bidi,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:  pointer.P(int64(util.RootUser)),
			Privileged: pointer.P(true),
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
	}
}

func NewHandlerDaemonSet(config *operatorutil.KubeVirtDeploymentConfig, productName, productVersion, productComponent string) *appsv1.DaemonSet {

	deploymentName := VirtHandlerName
	imageName := fmt.Sprintf("%s%s", config.GetImagePrefix(), deploymentName)
	image := config.VirtHandlerImage
	if image == "" {
		image = fmt.Sprintf("%s/%s%s", config.GetImageRegistry(), imageName, AddVersionSeparatorPrefix(config.GetHandlerVersion()))
	}
	env := operatorutil.NewEnvVarMap(config.GetExtraEnv())
	podTemplateSpec := newPodTemplateSpec(deploymentName, productName, productVersion, productComponent, image, config.GetImagePullPolicy(), config.GetImagePullSecrets(), nil, env)

	launcherImage := config.VirtLauncherImage
	if launcherImage == "" {
		launcherImage = fmt.Sprintf("%s/%s%s%s", config.GetImageRegistry(), config.GetImagePrefix(), "virt-launcher", AddVersionSeparatorPrefix(config.GetLauncherVersion()))
	}

	migrationNetwork := config.GetMigrationNetwork()
	if migrationNetwork != nil {
		if podTemplateSpec.ObjectMeta.Annotations == nil {
			podTemplateSpec.ObjectMeta.Annotations = make(map[string]string)
		}
		// Join the pod to the migration network and name the corresponding interface "migration0"
		podTemplateSpec.ObjectMeta.Annotations[networkv1.NetworkAttachmentAnnot] = *migrationNetwork + "@" + virtv1.MigrationInterfaceName
	}

	if podTemplateSpec.Annotations == nil {
		podTemplateSpec.Annotations = make(map[string]string)
	}
	podTemplateSpec.Annotations["openshift.io/required-scc"] = "kubevirt-handler"

	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: config.GetNamespace(),
			Name:      VirtHandlerName,
			Labels: map[string]string{
				virtv1.AppLabel:                VirtHandlerName,
				SupportsMigrationCNsValidation: "true",
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
	pod.ServiceAccountName = HandlerServiceAccountName
	pod.HostPID = true

	hypervisorNodeInfo := hypervisor.NewHypervisorNodeInformation(config.GetHypervisorName())

	// nodelabeller currently only support x86. The arch check will be done in node-labller.sh
	pod.InitContainers = []corev1.Container{
		{
			Command: []string{
				"/bin/sh",
				"-c",
			},
			Image: launcherImage,
			Name:  "virt-launcher",
			Args: []string{
				"node-labeller.sh",
			},
			Env: []corev1.EnvVar{
				{
					Name:  "PREFERRED_VIRTTYPE",
					Value: hypervisorNodeInfo.GetVirtType(),
				},
				{
					Name:  "HYPERVISOR_DEVICE",
					Value: hypervisorNodeInfo.GetHypervisorDevice(),
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: pointer.P(true),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "node-labeller",
					MountPath: nodeLabellerVolumePath,
				},
			},
			TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		},
	}

	// If there is any image pull secret added to the `virt-handler` deployment
	// it can mean that `virt-handler` is using private image. Therefore, we must
	// add `virt-launcher` container that will pre-pull and keep the (probably)
	// custom image of `virt-launcher`.
	// Note that we cannot make it an init container because the `virt-launcher`
	// image could be garbage collected by the kubelet.
	// Note that we cannot add `imagePullSecrets` to `virt-launcher` as this could
	// be a security risk - user could use this secret and abuse it.
	if len(config.GetImagePullSecrets()) > 0 {
		pod.Containers = append(pod.Containers, corev1.Container{
			Name:            "virt-launcher-image-holder",
			Image:           launcherImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/sh", "-c"},
			Args:            []string{"sleep infinity"},
			Resources: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("20Mi"),
				},
			},
			TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		})
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
		config.GetVerbosity(),
	}
	container.Ports = []corev1.ContainerPort{
		{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8443,
		},
	}
	container.SecurityContext = &corev1.SecurityContext{
		Privileged: pointer.P(true),
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
	attachCertificateSecret(pod, VirtHandlerMigrationClientCertSecretName, "/etc/virt-handler/migrationservercertificates")
	attachCertificateSecret(pod, VirtHandlerVsockClientCertSecretName, "/etc/virt-handler/vsockclientcertificates")
	attachProfileVolume(pod)

	bidi := corev1.MountPropagationBidirectional
	// NOTE: the 'kubelet-pods' volume mount exists because that path holds unix socket files.
	// Socket files fail when their path is longer than 108 characters,
	//   so that shortened volume path is to allow domain socket connections.
	// It's ridiculous to have to account for that, but that's the situation we're in.
	volumes := []volume{
		{"libvirt-runtimes", runtimesPath, runtimesPath, nil},
		{"virt-share-dir", util.VirtShareDir, util.VirtShareDir, &bidi},
		{"virt-private-dir", util.VirtPrivateDir, util.VirtPrivateDir, nil},
		{"kubelet-pods", kubeletPodsPath, "/pods", nil},
		{"kubelet", util.KubeletRoot, util.KubeletRoot, &bidi},
		{"node-labeller", nodeLabellerVolumePath, nodeLabellerVolumePath, nil},
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
	// TODO: This is not used anymore, but can't be removed because of https://github.com/kubevirt/kubevirt/issues/10632
	//   Since CR-based updates use the wrong install strategy, removing this volume and downgrading via CR will try to
	//   run the previous version of virt-handler without the volume, which will fail and CrashLoop.
	//   Please remove the volume once the above issue is fixed.
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
			corev1.ResourceMemory: resource.MustParse("325Mi"),
		},
	}
	prHelperImage := config.PrHelperImage
	if prHelperImage == "" {
		prHelperImage = fmt.Sprintf("%s/%s%s%s", config.GetImageRegistry(), config.GetImagePrefix(), PrHelperName, AddVersionSeparatorPrefix(config.GetPrHelperVersion()))
	}
	sidecarShimImage := config.SidecarShimImage
	if sidecarShimImage == "" {
		sidecarShimImage = fmt.Sprintf("%s/%s%s%s", config.GetImageRegistry(), config.GetImagePrefix(), SidecarShimName, AddVersionSeparatorPrefix(config.GetSidecarShimVersion()))
	}

	if config.PersistentReservationEnabled() {
		directoryOrCreate := corev1.HostPathDirectoryOrCreate
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: prVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: reservation.GetPrHelperSocketDir(),
					Type: &directoryOrCreate,
				},
			}}, corev1.Volume{
			Name: devDirVol,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/dev",
				},
			}}, corev1.Volume{
			Name: etcMultipath,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/multipath",
					Type: pointer.P(corev1.HostPathDirectoryOrCreate),
				},
			}})
		pod.Containers = append(pod.Containers, RenderPrHelperContainer(prHelperImage, config.GetImagePullPolicy()))
	}
	return daemonset

}

// NewHandlerPoolDaemonSet creates a virt-handler DaemonSet for a handler pool.
// It clones the primary handler DaemonSet and applies pool-specific overrides:
// name, image, nodeSelector, and the handler-pool label.
func NewHandlerPoolDaemonSet(primaryHandler *appsv1.DaemonSet, pool virtv1.VirtHandlerPoolConfig) *appsv1.DaemonSet {
	ds := primaryHandler.DeepCopy()

	poolName := fmt.Sprintf("%s-%s", VirtHandlerName, pool.Name)
	ds.Name = poolName
	ds.Labels[virtv1.HandlerPoolLabel] = pool.Name
	ds.Spec.Template.Labels[virtv1.HandlerPoolLabel] = pool.Name

	// Update label selector to match pool-specific pods
	ds.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"kubevirt.io":          VirtHandlerName,
			virtv1.HandlerPoolLabel: pool.Name,
		},
	}

	// Override handler image if specified
	if pool.VirtHandlerImage != "" && len(ds.Spec.Template.Spec.Containers) > 0 {
		ds.Spec.Template.Spec.Containers[0].Image = pool.VirtHandlerImage
	}

	// Override launcher image in init container and image-holder sidecar
	if pool.VirtLauncherImage != "" {
		for i := range ds.Spec.Template.Spec.InitContainers {
			if ds.Spec.Template.Spec.InitContainers[i].Name == "virt-launcher" {
				ds.Spec.Template.Spec.InitContainers[i].Image = pool.VirtLauncherImage
			}
		}
		for i := range ds.Spec.Template.Spec.Containers {
			if ds.Spec.Template.Spec.Containers[i].Name == "virt-launcher-image-holder" {
				ds.Spec.Template.Spec.Containers[i].Image = pool.VirtLauncherImage
			}
		}
	}

	// Merge pool nodeSelector with the primary handler's nodeSelector so that
	// inherited selectors (e.g., kubernetes.io/os: linux) are preserved.
	if ds.Spec.Template.Spec.NodeSelector == nil {
		ds.Spec.Template.Spec.NodeSelector = make(map[string]string)
	}
	for key, value := range pool.NodeSelector {
		ds.Spec.Template.Spec.NodeSelector[key] = value
	}

	return ds
}

// ApplyPoolAntiAffinityToPrimaryHandler adds NotIn node affinity expressions
// to the primary virt-handler DaemonSet so that it does not schedule on nodes
// claimed by handler pools.
//
// Note: when a pool has multiple nodeSelector keys, the anti-affinity excludes
// nodes matching ANY individual key rather than only nodes matching ALL keys.
// This is because Kubernetes node affinity cannot express NOT(AND(...)). In
// practice pools should use a single distinguishing label in nodeSelector to
// avoid nodes being excluded from both the primary and pool handlers.
func ApplyPoolAntiAffinityToPrimaryHandler(handler *appsv1.DaemonSet, pools []virtv1.VirtHandlerPoolConfig) {
	if len(pools) == 0 {
		return
	}

	var expressions []corev1.NodeSelectorRequirement
	for _, pool := range pools {
		for key, value := range pool.NodeSelector {
			expressions = append(expressions, corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpNotIn,
				Values:   []string{value},
			})
		}
	}

	if len(expressions) == 0 {
		return
	}

	term := corev1.NodeSelectorTerm{
		MatchExpressions: expressions,
	}

	pod := &handler.Spec.Template.Spec
	if pod.Affinity == nil {
		pod.Affinity = &corev1.Affinity{}
	}
	if pod.Affinity.NodeAffinity == nil {
		pod.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}
	pod.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
		pod.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		term,
	)
}
