package services

import (
	"strconv"
)

type VirtLauncherCommandRenderer struct {
	allowEmulation           bool
	customDebugFilters       string
	domainName               string
	isDoppelgangerPod        bool
	isNonRoot                bool
	gracePeriodSeconds       int
	numberOfHookSidecars     int
	keepAfterFailure         bool
	namespace                string
	shouldSimulateCrash      bool
	virtLauncherStaticConfig VirtLauncherStaticConfig
	vmUID                    string
}

type VirtLauncherStaticConfig struct {
	containerDiskDir string
	ephemeralDiskDir string
	launcherTimeout  int
	ovmfPath         string
	virtShareDir     string
}

type VirtLauncherCommandRendererOption func(renderer *VirtLauncherCommandRenderer)

func NewVirtLauncherCommandRenderer(
	vmUID string,
	domainName string,
	gracePeriodSeconds int,
	hookSidecarListLen int,
	vmNamespace string,
	virtLauncherStaticConfig VirtLauncherStaticConfig,
	opts ...VirtLauncherCommandRendererOption,
) *VirtLauncherCommandRenderer {

	return applyOptionsToRenderer(
		&VirtLauncherCommandRenderer{
			domainName:               domainName,
			gracePeriodSeconds:       gracePeriodSeconds,
			numberOfHookSidecars:     hookSidecarListLen,
			namespace:                vmNamespace,
			virtLauncherStaticConfig: virtLauncherStaticConfig,
			vmUID:                    vmUID,
		},
		opts,
	)
}

func applyOptionsToRenderer(virtLauncherCmdRenderer *VirtLauncherCommandRenderer, opts []VirtLauncherCommandRendererOption) *VirtLauncherCommandRenderer {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(virtLauncherCmdRenderer)
	}
	return virtLauncherCmdRenderer
}

func NewDoppelgangerPodRender(opts ...VirtLauncherCommandRendererOption) *VirtLauncherCommandRenderer {
	return applyOptionsToRenderer(
		&VirtLauncherCommandRenderer{isDoppelgangerPod: true},
		opts,
	)
}

func (vlcr *VirtLauncherCommandRenderer) Render() []string {
	var command []string
	if vlcr.isDoppelgangerPod {
		command = []string{"/bin/bash",
			"-c",
			"echo", "bound PVCs"}
	} else {
		const launcherExecBinary = "/usr/bin/virt-launcher-monitor"
		command = []string{
			launcherExecBinary,
			"--qemu-timeout", generateQemuTimeoutWithJitter(vlcr.virtLauncherStaticConfig.launcherTimeout),
			"--name", vlcr.domainName,
			"--uid", vlcr.vmUID,
			"--namespace", vlcr.namespace,
			"--kubevirt-share-dir", vlcr.virtLauncherStaticConfig.virtShareDir,
			"--ephemeral-disk-dir", vlcr.virtLauncherStaticConfig.ephemeralDiskDir,
			"--container-disk-dir", vlcr.virtLauncherStaticConfig.containerDiskDir,
			"--grace-period-seconds", strconv.Itoa(vlcr.gracePeriodSeconds),
			"--hook-sidecars", strconv.Itoa(vlcr.numberOfHookSidecars),
			"--ovmf-path", vlcr.virtLauncherStaticConfig.ovmfPath,
		}
		if vlcr.isNonRoot {
			command = append(command, "--run-as-nonroot")
		}

		if vlcr.customDebugFilters != "" {
			command = append(command, "--libvirt-log-filters", vlcr.customDebugFilters)
		}
	}

	if vlcr.allowEmulation {
		command = append(command, "--allow-emulation")
	}
	if vlcr.keepAfterFailure {
		command = append(command, "--keep-after-failure")
	}
	if vlcr.shouldSimulateCrash {
		command = append(command, "--simulate-crash")
	}
	return command
}

func WithNonRootUser() VirtLauncherCommandRendererOption {
	return func(renderer *VirtLauncherCommandRenderer) {
		renderer.isNonRoot = true
	}
}

func WithLibvirtCustomDebugFilters(debugFilters string) VirtLauncherCommandRendererOption {
	return func(renderer *VirtLauncherCommandRenderer) {
		renderer.customDebugFilters = debugFilters
	}
}

func WithEmulation() VirtLauncherCommandRendererOption {
	return func(renderer *VirtLauncherCommandRenderer) {
		renderer.allowEmulation = true
	}
}

func WithKeepAfterFailure() VirtLauncherCommandRendererOption {
	return func(renderer *VirtLauncherCommandRenderer) {
		renderer.keepAfterFailure = true
	}
}

func WithSimulatedCrash() VirtLauncherCommandRendererOption {
	return func(renderer *VirtLauncherCommandRenderer) {
		renderer.shouldSimulateCrash = true
	}
}
