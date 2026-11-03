/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/grpcserver"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/ignition"
	containerdisk "kubevirt.io/kubevirt/pkg/storage/container-disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	domainpublisher "kubevirt.io/kubevirt/pkg/virt-launcher/domain-publisher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	premigrationhookserver "kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/cpuhook"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/vgpuhook"
	"kubevirt.io/kubevirt/pkg/virt-launcher/standalone"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virtcli "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	defaultStartTimeout               = 3 * time.Minute
	earlyExitCheckInterval            = 2 * time.Second
	finalNotificationTimeout          = 15 * time.Second
	finalNotificationAfterKillTimeout = 30 * time.Second
)

func init() {
	// must registry the event impl before doing anything else.
	libvirt.EventRegisterDefaultImpl()
}

func markReady() {
	err := os.Rename(cmdclient.UninitializedSocketOnGuest(), cmdclient.SocketOnGuest())
	if err != nil {
		panic(err)
	}
	log.Log.Info("Marked as ready")
}

func startCmdServer(socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{},
	options *cmdserver.ServerOptions) <-chan struct{} {

	done, err := grpcserver.RunServer(stopChan, socketPath,
		cmdserver.RegisterCmdServer(
			cmdserver.NewLauncher(domainManager, options),
		))
	if err != nil {
		log.Log.Reason(err).Error("Failed to start virt-launcher cmd server")
		panic(err)
	}

	// ensure the cmdserver is responsive before continuing
	// PollImmediate breaks the poll loop when bool or err are returned OR if timeout occurs.
	//
	// Timing out causes an error to be returned
	err = virtwait.PollImmediately(1*time.Second, 15*time.Second, func(_ context.Context) (bool, error) {
		client, err := cmdclient.NewClient(socketPath)
		if err != nil {
			return false, nil
		}
		defer client.Close()

		err = client.Ping()
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		panic(fmt.Errorf("failed to connect to cmd server: %v", err))
	}

	return done
}

func createLibvirtConnection(runWithNonRoot bool) virtcli.Connection {
	libvirtUri := "qemu:///system"
	user := ""
	if runWithNonRoot {
		const nonRootUserString = "qemu"
		user = nonRootUserString
		libvirtUri = "qemu+unix:///session?socket=/var/run/libvirt/virtqemud-sock"
	}

	domainConn, err := virtcli.NewConnection(libvirtUri, user, "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to virtqemud: %v", err))
	}

	return domainConn
}

func startDomainEventMonitoring(ctx context.Context, domainConn virtcli.Connection, domainPublisher *domainpublisher.DomainPublisher) {
	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				log.Log.Reason(res).Error("Listening to libvirt events failed, retrying.")
				time.Sleep(time.Second)
			}
		}
	}()

	go domainPublisher.Run(ctx)
	if err := domainpublisher.RegisterLibvirtEvents(domainConn, domainPublisher); err != nil {
		panic(err)
	}
}

func initializeDirs(ephemeralDiskDir string,
	containerDiskDir string,
	hotplugDiskDir string,
	uid string) {

	// Resolve permission mismatch when system default mask is set more restrictive than 022.
	mask := syscall.Umask(0)
	defer syscall.Umask(mask)

	err := virtlauncher.InitializePrivateDirectories(filepath.Join("/var/run/kubevirt-private", uid))
	if err != nil {
		panic(err)
	}

	err = cloudinit.SetLocalDirectory(filepath.Join(ephemeralDiskDir, "cloud-init-data"))
	if err != nil {
		panic(err)
	}

	err = ignition.SetLocalDirectory(filepath.Join(ephemeralDiskDir, "ignition-data"))
	if err != nil {
		panic(err)
	}

	err = containerdisk.SetLocalDirectory(containerDiskDir)
	if err != nil {
		panic(err)
	}

	err = hotplugdisk.SetLocalDirectory(hotplugDiskDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(filepath.Join("/var/run/kubevirt-private", "vm-disks"))
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(config.ConfigMapDisksDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(config.SysprepDisksDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(config.SecretDisksDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(config.DownwardAPIDisksDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(config.ServiceAccountDiskDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(downwardmetrics.DownwardMetricsChannelDir)
	if err != nil {
		panic(err)
	}
}

func waitForDomainDefined(timeout time.Duration, domainDefined <-chan struct{}, stop chan struct{}) (bool, error) {
	checkEarlyExit := time.NewTicker(earlyExitCheckInterval)
	defer checkEarlyExit.Stop()
	timedOut := time.After(timeout)

	for {
		select {
		case <-domainDefined:
			return true, nil
		case <-stop:
			return false, nil
		case <-timedOut:
			return false, fmt.Errorf("timed out waiting for domain to be defined")
		case <-checkEarlyExit.C:
			if cmdserver.ReceivedEarlyExitSignal() {
				return false, fmt.Errorf("received early exit signal")
			}
		}
	}
}

func waitForFinalNotify(finalNotificationSent <-chan struct{}, domainManager virtwrap.DomainManager, vmi *v1.VirtualMachineInstance) {
	log.Log.Info("Waiting for the final domain notification to be sent to virt-handler")
	// QEMU exiting leaves a persistent domain defined, so the final notification,
	// which carries the exit reason, waits for virt-handler to undefine it.
	if closedWithin(finalNotificationSent, finalNotificationTimeout) {
		return
	}

	// KillVMI only destroys the domain if libvirt still reports it running.
	log.Log.Info("Timed out waiting for the final domain notification, killing the domain")
	if err := domainManager.KillVMI(vmi); err != nil {
		log.Log.Reason(err).Error("Failed to kill the domain")
	}
	// Bounded so libvirt cannot hold the pod; virt-handler then learns of the exit
	// from the command socket closing, without the exit reason.
	if !closedWithin(finalNotificationSent, finalNotificationAfterKillTimeout) {
		log.Log.Info("Timed out waiting for the final domain notification after killing the domain")
	}
}

func closedWithin(signal <-chan struct{}, timeout time.Duration) bool {
	select {
	case <-signal:
		return true
	case <-time.After(timeout):
		return false
	}
}

func main() {
	qemuTimeout := pflag.Duration("qemu-timeout", defaultStartTimeout, "Amount of time to wait for qemu")
	virtShareDir := pflag.String("kubevirt-share-dir", "/var/run/kubevirt", "Shared directory between virt-handler and virt-launcher")
	ephemeralDiskDir := pflag.String("ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks", "Base directory for ephemeral disk data")
	containerDiskDir := pflag.String("container-disk-dir", "/var/run/kubevirt/container-disks", "Base directory for container disk data")
	hotplugDiskDir := pflag.String("hotplug-disk-dir", v1.HotplugDiskDir, "Base directory for hotplug disk data")
	name := pflag.String("name", "", "Name of the VirtualMachineInstance")
	uid := pflag.String("uid", "", "UID of the VirtualMachineInstance")
	namespace := pflag.String("namespace", "", "Namespace of the VirtualMachineInstance")
	gracePeriodSeconds := pflag.Int("grace-period-seconds", 30, "Grace period to observe before sending SIGTERM to vmi process")
	allowEmulation := pflag.Bool("allow-emulation", false, "Allow use of software emulation as fallback")
	allowCrossArchEmulation := pflag.Bool("allow-cross-arch-emulation", false, "Allow cross-architecture software emulation via QEMU TCG")
	runWithNonRoot := pflag.Bool("run-as-nonroot", false, "Run virtqemud with the 'virt' user")
	imageVolumeEnabled := pflag.Bool("image-volume", false, "Generated with ImageVolume instead of containerDisk") //remove this once ImageVolume is GAed
	vGPUDedicatedHookEnabled := pflag.Bool("vgpu-dedicated-hook", false, "Enable target mdev UUID mutation for vGPU live migration")
	vmStatsCollectorEnabled := pflag.Bool("vm-stats-collector", false, "Enable additional guest agent polling workers for VMStats monitoring data collection")
	firmwareAutoSelectionEnabled := pflag.Bool("firmware-auto-selection", false, "Use libvirt firmware auto-selection for EFI Secure Boot")
	arm64SecureBootEnabled := pflag.Bool("arm64-secure-boot", false, "Enable ARM64 UEFI Secure Boot via firmware auto-selection and uefi-vars")
	hookSidecars := pflag.Uint("hook-sidecars", 0, "Number of requested hook sidecars, virt-launcher will wait for all of them to become available")
	diskMemoryLimitBytes := pflag.Int64("disk-memory-limit", virtconfig.DefaultDiskVerificationMemoryLimitBytes, "Memory limit for disk verification")
	ovmfPath := pflag.String("ovmf-path", virtconfig.DefaultARCHOVMFPath, "The directory that contains the EFI roms (like OVMF_CODE.fd)")
	qemuAgentSysInterval := pflag.Duration("qemu-agent-sys-interval", 120*time.Second, "Interval between consecutive qemu agent calls for sys commands")
	qemuAgentFileInterval := pflag.Duration("qemu-agent-file-interval", 300*time.Second, "Interval between consecutive qemu agent calls for file command")
	qemuAgentUserInterval := pflag.Duration("qemu-agent-user-interval", 10*time.Second, "Interval between consecutive qemu agent calls for user command")
	qemuAgentVersionInterval := pflag.Duration("qemu-agent-version-interval", 300*time.Second, "Interval between consecutive qemu agent calls for version command")
	qemuAgentFSFreezeStatusInterval := pflag.Duration("qemu-fsfreeze-status-interval", 5*time.Second, "Interval between consecutive qemu agent calls for fsfreeze status command")
	simulateCrash := pflag.Bool("simulate-crash", false, "Causes virt-launcher to immediately crash. This is used by functional tests to simulate crash loop scenarios.")
	libvirtLogFilters := pflag.String("libvirt-log-filters", "", "Set custom log filters for libvirt")
	hypervisor := pflag.String("hypervisor", v1.KvmHypervisorName, "Hypervisor to be used by the VMI")

	pflag.CommandLine.AddGoFlag(log.VerbosityFlag())
	pflag.Parse()

	log.InitializeLogging("virt-launcher")

	// check if virt-launcher verbosity should be changed
	if verbosityStr, ok := os.LookupEnv("VIRT_LAUNCHER_LOG_VERBOSITY"); ok {
		if verbosity, err := strconv.Atoi(verbosityStr); err == nil {
			log.Log.SetVerbosityLevel(verbosity)
			log.Log.V(2).Infof("set log verbosity to %d", verbosity)
		} else {
			log.Log.Warningf("failed to set log verbosity. The value of logVerbosity label should be an integer, got %s instead.", verbosityStr)
		}
	}

	log.Log.V(2).Infof("Hypervisor set to %s", *hypervisor)

	// Initialize local and shared directories
	initializeDirs(*ephemeralDiskDir, *containerDiskDir, *hotplugDiskDir, *uid)

	if !*runWithNonRoot {
		err := virtlauncher.InitializeConsoleLogFile(filepath.Join("/var/run/kubevirt-private", *uid))
		if err != nil {
			panic(err)
		}
	}

	if *simulateCrash {
		panic(fmt.Errorf("Simulated virt-launcher crash"))
	}

	// Block until all requested hookSidecars are ready
	hookManager := hooks.GetManager()
	err := hookManager.Collect(*hookSidecars, *qemuTimeout)
	if err != nil {
		panic(err)
	}

	vmi := v1.NewVMIReferenceWithUUID(*namespace, *name, types.UID(*uid))

	ephemeralDiskCreator := ephemeraldisk.NewEphemeralDiskCreator(filepath.Join(*ephemeralDiskDir, "disk-data"))
	if err := ephemeralDiskCreator.Init(); err != nil {
		panic(err)
	}

	// Start virtqemud, virtlogd, and establish libvirt connection
	stopChan := make(chan struct{})

	l := util.NewLibvirtWrapper(*runWithNonRoot)
	err = l.SetupLibvirt(libvirtLogFilters)
	if err != nil {
		panic(err)
	}

	l.StartVirtqemud(stopChan)
	// only single domain should be present
	domainName := api.VMINamespaceKeyFunc(vmi)

	util.StartVirtlog(stopChan, domainName, *runWithNonRoot)

	domainConn := createLibvirtConnection(*runWithNonRoot)
	defer domainConn.Close()

	var agentStore = agentpoller.NewAsyncAgentStore()

	notifier := notifyclient.NewNotifier(*virtShareDir)
	defer notifier.Close()

	metadataCache := metadata.NewCache()

	signalStopChan := make(chan struct{})

	hookFuncs := []premigrationhookserver.HookFunc{
		cpuhook.CPUDedicatedHook,
		disk.Hook,
		network.UpgradeOrdinalNamingScheme,
	}
	if *vGPUDedicatedHookEnabled {
		hookFuncs = append(hookFuncs, vgpuhook.MDevVGPULiveMigration)
	}

	preMigrationHookServer := premigrationhookserver.NewPreMigrationHookServer(
		stopChan,
		hookFuncs...,
	)
	domainManager, err := virtwrap.NewLibvirtDomainManager(
		domainConn,
		*virtShareDir,
		*ephemeralDiskDir,
		&agentStore,
		*ovmfPath,
		ephemeralDiskCreator,
		metadataCache,
		signalStopChan,
		*diskMemoryLimitBytes,
		util.GetPodCPUSet,
		*imageVolumeEnabled,
		preMigrationHookServer,
		*hypervisor,
		domainName,
		*vmStatsCollectorEnabled,
		*firmwareAutoSelectionEnabled,
		*arm64SecureBootEnabled,
		*allowCrossArchEmulation,
		notifier)
	if err != nil {
		panic(err)
	}
	// Send domain notifications to virt-handler
	agentPoller := agentpoller.CreatePoller(
		domainConn,
		vmi.UID,
		domainName,
		&agentStore,
		*qemuAgentSysInterval,
		*qemuAgentFileInterval,
		*qemuAgentUserInterval,
		*qemuAgentVersionInterval,
		*qemuAgentFSFreezeStatusInterval,
	)
	domainPublisher := domainpublisher.NewDomainPublisher(notifier, domainConn, vmi, agentPoller, metadataCache, *runWithNonRoot)
	publisherCtx, stopPublisher := context.WithCancel(context.Background())
	startDomainEventMonitoring(publisherCtx, domainConn, domainPublisher)

	// Start the virt-launcher command service.
	// Clients can use this service to tell virt-launcher
	// to start/stop virtual machines
	options := cmdserver.NewServerOptions(*allowEmulation).WithVMStatsCollector(*vmStatsCollectorEnabled).WithNotifier(notifier).WithVMI(vmi)
	cmdclient.SetBaseDir(*virtShareDir)
	cmdServerDone := startCmdServer(cmdclient.UninitializedSocketOnGuest(), domainManager, stopChan, options)

	gracefulShutdownCallback := func() {
		domainManager.MarkGracefulShutdownVMI()
		log.Log.Object(vmi).Info("Signaled graceful shutdown")
	}

	finalShutdownCallback := func(pid int) {
		if err := domainManager.KillVMI(vmi); err != nil {
			log.Log.Reason(err).Errorf("Unable to stop qemu with libvirt")
			if pid != 0 {
				log.Log.Warning("Falling back to SIGTERM")
				if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
					log.Log.Reason(err).Errorf("Unable to kill PID %d", pid)
				}
			}
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		s := <-c
		log.Log.Infof("Received signal %s", s.String())
		close(signalStopChan)
	}()

	// Marking Ready allows the container's readiness check to pass.
	// This informs virt-controller that virt-launcher is ready to handle
	// managing virtual machines.
	markReady()

	standalone.HandleStandaloneMode(domainManager)
	defined, err := waitForDomainDefined(*qemuTimeout, domainPublisher.DomainDefined(), signalStopChan)
	if err != nil {
		panic(err)
	}
	if defined {
		var pidDir string
		if *runWithNonRoot {
			pidDir = "/run/libvirt/qemu/run"
		} else {
			pidDir = "/run/libvirt/qemu"
		}
		mon := virtlauncher.NewProcessMonitor(domainName,
			pidDir,
			*gracePeriodSeconds,
			finalShutdownCallback,
			gracefulShutdownCallback)

		// This is a wait loop that monitors the qemu pid. When the pid
		// exits, the wait loop breaks.
		mon.RunForever(*qemuTimeout, signalStopChan)

		// Allow hooks to gracefully shutdown
		hookManager.Shutdown()

		// Now that the pid has exited, we wait for the final delete notification to be
		// sent back to virt-handler. This delete notification contains the reason the
		// domain exited.
		waitForFinalNotify(domainPublisher.FinalNotificationSent(), domainManager, vmi)
	}

	stopPublisher()
	close(stopChan)
	<-cmdServerDone
	<-preMigrationHookServer.Done()

	log.Log.Info("Exiting...")
}
