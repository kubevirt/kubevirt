package services

import (
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	cacheHomeEnvVarName  = "XDG_CACHE_HOME"
	configHomeEnvVarName = "XDG_CONFIG_HOME"
	runtimeDirEnvVarName = "XDG_RUNTIME_DIR"
)

type ContainerSpecRenderer struct {
	imgPullPolicy   k8sv1.PullPolicy
	isPrivileged    bool
	launcherImg     string
	name            string
	userID          int64
	volumeDevices   []k8sv1.VolumeDevice
	volumeMounts    []k8sv1.VolumeMount
	resources       k8sv1.ResourceRequirements
	liveninessProbe *k8sv1.Probe
	readinessProbe  *k8sv1.Probe
	ports           []k8sv1.ContainerPort
	capabilities    *k8sv1.Capabilities
	args            []string
}

type Option func(*ContainerSpecRenderer)

func NewContainerSpecRenderer(containerName string, launcherImg string, imgPullPolicy k8sv1.PullPolicy, opts ...Option) *ContainerSpecRenderer {
	computeContainerSpec := &ContainerSpecRenderer{
		imgPullPolicy: imgPullPolicy,
		launcherImg:   launcherImg,
		name:          containerName,
	}
	for _, opt := range opts {
		opt(computeContainerSpec)
	}
	return computeContainerSpec
}

func (csr *ContainerSpecRenderer) Render(cmd []string) k8sv1.Container {
	return k8sv1.Container{
		Name:            csr.name,
		Image:           csr.launcherImg,
		ImagePullPolicy: csr.imgPullPolicy,
		SecurityContext: securityContext(csr.userID, csr.isPrivileged, csr.capabilities),
		Command:         cmd,
		VolumeDevices:   csr.volumeDevices,
		VolumeMounts:    csr.volumeMounts,
		Resources:       csr.resources,
		Ports:           csr.ports,
		Env:             csr.envVars(),
		LivenessProbe:   csr.liveninessProbe,
		ReadinessProbe:  csr.readinessProbe,
		Args:            csr.args,
	}
}

func (csr *ContainerSpecRenderer) envVars() []k8sv1.EnvVar {
	if csr.userID == 0 {
		return nil
	}
	return xdgEnvironmentVariables()
}

func WithNonRoot(userID int64) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.userID = userID
	}
}

func WithPrivileged() Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.isPrivileged = true
	}
}

func WithCapabilities(vmi *v1.VirtualMachineInstance) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.capabilities = &k8sv1.Capabilities{
			Add:  getRequiredCapabilities(vmi),
			Drop: []k8sv1.Capability{CAP_NET_RAW},
		}
	}
}

func WithVolumeDevices(devices ...k8sv1.VolumeDevice) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.volumeDevices = devices
	}
}

func WithVolumeMounts(mounts ...k8sv1.VolumeMount) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.volumeMounts = mounts
	}
}

func WithResourceRequirements(resources k8sv1.ResourceRequirements) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.resources = resources
	}
}

func WithPorts(vmi *v1.VirtualMachineInstance) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.ports = getPortsFromVMI(vmi)
	}
}

func WithArgs(args []string) Option {
	return func(renderer *ContainerSpecRenderer) {
		renderer.args = args
	}
}

func WithLivelinessProbe(vmi *v1.VirtualMachineInstance) Option {
	return func(renderer *ContainerSpecRenderer) {
		v1.SetDefaults_Probe(vmi.Spec.LivenessProbe)
		renderer.liveninessProbe = copyProbe(vmi.Spec.LivenessProbe)
		updateLivenessProbe(vmi, renderer.liveninessProbe)
	}
}

func WithReadinessProbe(vmi *v1.VirtualMachineInstance) Option {
	return func(renderer *ContainerSpecRenderer) {
		v1.SetDefaults_Probe(vmi.Spec.ReadinessProbe)
		renderer.readinessProbe = copyProbe(vmi.Spec.ReadinessProbe)
		updateReadinessProbe(vmi, renderer.readinessProbe)
	}
}

func xdgEnvironmentVariables() []k8sv1.EnvVar {
	const varRun = "/var/run"
	return []k8sv1.EnvVar{
		{
			Name:  cacheHomeEnvVarName,
			Value: util.VirtPrivateDir,
		},
		{
			Name:  configHomeEnvVarName,
			Value: util.VirtPrivateDir,
		},
		{
			Name:  runtimeDirEnvVarName,
			Value: varRun,
		},
	}
}

func securityContext(userId int64, privileged bool, requiredCapabilities *k8sv1.Capabilities) *k8sv1.SecurityContext {
	isNonRoot := userId != 0
	context := &k8sv1.SecurityContext{
		RunAsUser:    &userId,
		RunAsNonRoot: &isNonRoot,
		Privileged:   &privileged,
		Capabilities: requiredCapabilities,
	}

	if isNonRoot {
		context.RunAsGroup = &userId
	}
	return context
}
