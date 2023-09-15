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
	goflag "flag"
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

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/ignition"
	putil "kubevirt.io/kubevirt/pkg/util"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virtcli "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const defaultStartTimeout = 3 * time.Minute

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
	options *cmdserver.ServerOptions) chan struct{} {
	done, err := cmdserver.RunServer(socketPath, domainManager, stopChan, options)
	if err != nil {
		log.Log.Reason(err).Error("Failed to start virt-launcher cmd server")
		panic(err)
	}

	// ensure the cmdserver is responsive before continuing
	// PollImmediate breaks the poll loop when bool or err are returned OR if timeout occurs.
	//
	// Timing out causes an error to be returned
	err = utilwait.PollImmediate(1*time.Second, 15*time.Second, func() (bool, error) {
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
		user = putil.NonRootUserString
		libvirtUri = "qemu+unix:///session?socket=/var/run/libvirt/virtqemud-sock"
	}

	domainConn, err := virtcli.NewConnection(libvirtUri, user, "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to virtqemud: %v", err))
	}

	return domainConn
}

func startDomainEventMonitoring(
	notifier *notifyclient.Notifier,
	virtShareDir string,
	domainConn virtcli.Connection,
	deleteNotificationSent chan watch.Event,
	vmi *v1.VirtualMachineInstance,
	domainName string,
	agentStore *agentpoller.AsyncAgentStore,
	qemuAgentSysInterval time.Duration,
	qemuAgentFileInterval time.Duration,
	qemuAgentUserInterval time.Duration,
	qemuAgentVersionInterval time.Duration,
	qemuAgentFSFreezeStatusInterval time.Duration,
	metadataCache *metadata.Cache,
) {
	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				log.Log.Reason(res).Error("Listening to libvirt events failed, retrying.")
				time.Sleep(time.Second)
			}
		}
	}()

	err := notifier.StartDomainNotifier(domainConn, deleteNotificationSent, vmi, domainName, agentStore, qemuAgentSysInterval, qemuAgentFileInterval, qemuAgentUserInterval, qemuAgentVersionInterval, qemuAgentFSFreezeStatusInterval, metadataCache)
	if err != nil {
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

func detectDomainWithUUID(domainManager virtwrap.DomainManager) *api.Domain {
	domains, err := domainManager.ListAllDomains()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to list domains when detecting UUID")
		return nil
	}
	for _, domain := range domains {
		if domain.Spec.UUID != "" {
			return domain
		}
	}
	return nil
}

func waitForDomainUUID(timeout time.Duration, events chan watch.Event, stop chan struct{}, domainManager virtwrap.DomainManager) *api.Domain {

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	checkEarlyExit := time.NewTicker(time.Second * 2)
	defer checkEarlyExit.Stop()
	domainCheckTicker := time.NewTicker(time.Second * 10)
	defer domainCheckTicker.Stop()

	for {
		select {
		case <-ticker.C:
			panic(fmt.Errorf("timed out waiting for domain to be defined"))
		case <-domainCheckTicker.C:
			log.Log.V(3).Infof("Periodically checking for domain with UUID")
			domain := detectDomainWithUUID(domainManager)
			if domain != nil {
				return domain
			}
		case <-events:
			log.Log.V(3).Infof("Checking for domain with UUID due to incoming libvirt event")
			domain := detectDomainWithUUID(domainManager)
			if domain != nil {
				return domain
			}
		case <-stop:
			return nil
		case <-checkEarlyExit.C:
			if cmdserver.ReceivedEarlyExitSignal() {
				panic(fmt.Errorf("received early exit signal"))
			}
		}
	}
}

func waitForFinalNotify(deleteNotificationSent chan watch.Event,
	domainManager virtwrap.DomainManager,
	vmi *v1.VirtualMachineInstance) {

	log.Log.Info("Waiting on final notifications to be sent to virt-handler.")

	// First attempt to wait for domain event to occur as a part of the normal shutdown flow.
	// If that fails, call Kill on the domain and wait for the event again.
	// If that that fails, exit. We did our best to shutdown the domain gracefully. We can't block
	// the pod forever. Virt-handler will learn of the domain's exit through monitoring cmd server socket.

	killTimeout := time.After(15 * time.Second)
	timedOut := false
	for timedOut == false {
		select {
		case e := <-deleteNotificationSent:
			if e.Object != nil && e.Type == watch.Modified {
				domain, ok := e.Object.(*api.Domain)
				if ok && domain.ObjectMeta.DeletionTimestamp != nil {
					log.Log.Info("Final Delete notification sent")
					return
				}
			}
		case <-killTimeout:
			log.Log.Info("Timed out waiting for final delete notification. Attempting to kill domain")
			timedOut = true
		}
	}

	// There are many conditions that can cause the qemu pid to exit that
	// don't involve the VirtualMachineInstance's domain from being deleted from libvirt.
	//
	// KillVMI is idempotent. Making a call to KillVMI here ensures that the deletion
	// occurs regardless if the VirtualMachineInstance crashed unexpectedly or if virt-handler requested
	// a graceful shutdown.
	domainManager.KillVMI(vmi)

	// We don't want to block here forever. If the delete does not occur, that could mean
	// something is wrong with libvirt. In this situation, virt-handler will detect that
	// the domain went away eventually, however the exit status will be unknown.
	finalTimeout := time.After(30 * time.Second)
	for {
		select {
		case e := <-deleteNotificationSent:
			if e.Object != nil && e.Type == watch.Modified {
				domain, ok := e.Object.(*api.Domain)
				if ok && domain.ObjectMeta.DeletionTimestamp != nil {
					log.Log.Info("Final Delete notification sent after calling kill.")
					return
				}
			}
			return
		case <-finalTimeout:
			log.Log.Info("Timed out waiting for final delete notification after calling kill.")
			return
		}
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
	runWithNonRoot := pflag.Bool("run-as-nonroot", false, "Run virtqemud with the 'virt' user")
	hookSidecars := pflag.Uint("hook-sidecars", 0, "Number of requested hook sidecars, virt-launcher will wait for all of them to become available")
	ovmfPath := pflag.String("ovmf-path", "/usr/share/OVMF", "The directory that contains the EFI roms (like OVMF_CODE.fd)")
	qemuAgentSysInterval := pflag.Duration("qemu-agent-sys-interval", 120*time.Second, "Interval between consecutive qemu agent calls for sys commands")
	qemuAgentFileInterval := pflag.Duration("qemu-agent-file-interval", 300*time.Second, "Interval between consecutive qemu agent calls for file command")
	qemuAgentUserInterval := pflag.Duration("qemu-agent-user-interval", 10*time.Second, "Interval between consecutive qemu agent calls for user command")
	qemuAgentVersionInterval := pflag.Duration("qemu-agent-version-interval", 300*time.Second, "Interval between consecutive qemu agent calls for version command")
	qemuAgentFSFreezeStatusInterval := pflag.Duration("qemu-fsfreeze-status-interval", 5*time.Second, "Interval between consecutive qemu agent calls for fsfreeze status command")
	simulateCrash := pflag.Bool("simulate-crash", false, "Causes virt-launcher to immediately crash. This is used by functional tests to simulate crash loop scenarios.")
	libvirtLogFilters := pflag.String("libvirt-log-filters", "", "Set custom log filters for libvirt")

	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")

	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
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

	// Initialize local and shared directories
	initializeDirs(*ephemeralDiskDir, *containerDiskDir, *hotplugDiskDir, *uid)
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

	l.StartVirtquemud(stopChan)
	// only single domain should be present
	domainName := api.VMINamespaceKeyFunc(vmi)

	util.StartVirtlog(stopChan, domainName, *runWithNonRoot)

	domainConn := createLibvirtConnection(*runWithNonRoot)
	defer domainConn.Close()

	var agentStore = agentpoller.NewAsyncAgentStore()

	notifier := notifyclient.NewNotifier(*virtShareDir)
	defer notifier.Close()

	metadataCache := metadata.NewCache()

	domainManager, err := virtwrap.NewLibvirtDomainManager(domainConn, *virtShareDir, *ephemeralDiskDir, &agentStore, *ovmfPath, ephemeralDiskCreator, metadataCache)
	if err != nil {
		panic(err)
	}

	// Start the virt-launcher command service.
	// Clients can use this service to tell virt-launcher
	// to start/stop virtual machines
	options := cmdserver.NewServerOptions(*allowEmulation)
	cmdclient.SetLegacyBaseDir(*virtShareDir)
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

	events := make(chan watch.Event, 2)
	// Send domain notifications to virt-handler
	startDomainEventMonitoring(notifier, *virtShareDir, domainConn, events, vmi, domainName, &agentStore, *qemuAgentSysInterval, *qemuAgentFileInterval, *qemuAgentUserInterval, *qemuAgentVersionInterval, *qemuAgentFSFreezeStatusInterval, metadataCache)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	signalStopChan := make(chan struct{})
	go func() {
		s := <-c
		log.Log.Infof("Received signal %s", s.String())
		close(signalStopChan)
	}()

	// Marking Ready allows the container's readiness check to pass.
	// This informs virt-controller that virt-launcher is ready to handle
	// managing virtual machines.
	markReady()

	domain := waitForDomainUUID(*qemuTimeout, events, signalStopChan, domainManager)
	if domain != nil {
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

		// Now that the pid has exited, we wait for the final delete notification to be
		// sent back to virt-handler. This delete notification contains the reason the
		// domain exited.
		waitForFinalNotify(events, domainManager, vmi)
	}

	close(stopChan)
	<-cmdServerDone

	log.Log.Info("Exiting...")
}
