package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Launcher pod command Renderer", func() {
	Context("Doppelganger pod command", func() {
		DescribeTable("should look like", func(expectedCommand []string, options ...VirtLauncherCommandRendererOption) {
			launcherCommandRenderer := NewDoppelgangerPodRender(options...)
			Expect(launcherCommandRenderer.Render()).To(
				ConsistOf(expectedCommand))
		},
			Entry(
				"without options",
				doppelgangerPodCommand(),
			),
			Entry(
				"the non-root option is ignored for doppelgangers",
				doppelgangerPodCommand(),
				WithNonRootUser(),
			),
			Entry(
				"the custom libvirt log filters option is ignored for doppelgangers",
				doppelgangerPodCommand(),
				WithLibvirtCustomDebugFilters("debug-this!"),
			),
			Entry(
				"the simulate-crash option is ignored for doppelgangers",
				doppelgangerPodCommand(),
				WithSimulatedCrash(),
			),
			Entry(
				"the allowEmulation option is ignored for doppelgangers",
				doppelgangerPodCommand(),
				WithEmulation(),
			),
			Entry(
				"the keepAfterFailure option is ignored for doppelgangers",
				doppelgangerPodCommand(),
				WithKeepAfterFailure(),
			))
	})

	Context("Real launcher pod command", func() {
		DescribeTable("a real launcher pod command should look like", func(option VirtLauncherCommandRendererOption, expectedCommand ...interface{}) {
			const (
				domainName         = "d1"
				gracePeriodSeconds = 22
				namespace          = "ns1"
				vmUID              = "1234"
			)
			launcherCommandRenderer := NewVirtLauncherCommandRenderer(vmUID, domainName, gracePeriodSeconds, 0, namespace, dummyStaticConfig(), option)
			Expect(launcherCommandRenderer.Render()).To(
				ConsistOf(expectedCommand...))
		},
			Entry("simple pod", nil, basicRealVirtLauncherCommand()),
			Entry("WithNonRootUser option",
				WithNonRootUser(),
				append(basicRealVirtLauncherCommand(), "--run-as-nonroot")),
			Entry("WithLibvirtCustomDebugFilters option",
				WithLibvirtCustomDebugFilters("debug-this!"),
				append(
					basicRealVirtLauncherCommand(),
					"--libvirt-log-filters",
					"debug-this!")),
			Entry("WithEmulation option",
				WithEmulation(),
				append(basicRealVirtLauncherCommand(), "--allow-emulation")),
			Entry("WithKeepAfterFailure option",
				WithKeepAfterFailure(),
				append(basicRealVirtLauncherCommand(), "--keep-after-failure")),
			Entry("WithSimulatedCrash option",
				WithSimulatedCrash(),
				append(basicRealVirtLauncherCommand(), "--simulate-crash")),
		)
	})
})

func doppelgangerPodCommand() []string {
	return []string{"/bin/bash", "-c", "echo", "bound PVCs"}
}

func dummyStaticConfig() VirtLauncherStaticConfig {
	return VirtLauncherStaticConfig{
		containerDiskDir: "/cont-disk",
		ephemeralDiskDir: "/ephemeral-disk",
		launcherTimeout:  1200,
		ovmfPath:         "/over-there/behind-the-counter",
		virtShareDir:     "/next/to-the-tomato-sauce",
	}
}

func basicRealVirtLauncherCommand() []interface{} {
	return []interface{}{
		"/usr/bin/virt-launcher-monitor",
		"--qemu-timeout",
		MatchRegexp("[0-9]+s"),
		"--name",
		"d1",
		"--uid",
		"1234",
		"--namespace",
		"ns1",
		"--kubevirt-share-dir",
		"/next/to-the-tomato-sauce",
		"--ephemeral-disk-dir",
		"/ephemeral-disk",
		"--container-disk-dir",
		"/cont-disk",
		"--grace-period-seconds",
		"22",
		"--hook-sidecars",
		"0",
		"--ovmf-path",
		"/over-there/behind-the-counter"}
}
