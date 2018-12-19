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
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	libvirt "github.com/libvirt/libvirt-go"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virtcli "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

const defaultStartTimeout = 3 * time.Minute
const defaultWatchdogInterval = 5 * time.Second

func init() {
	// must registry the event impl before doing anything else.
	libvirt.EventRegisterDefaultImpl()
}

func markReady(readinessFile string) {
	f, err := os.OpenFile(readinessFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	f.Close()
	log.Log.Info("Marked as ready")
}

func startCmdServer(socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{},
	options *cmdserver.ServerOptions) chan struct{} {

	err := os.RemoveAll(socketPath)
	if err != nil {
		log.Log.Reason(err).Error("Could not clean up virt-launcher cmd socket")
		panic(err)
	}

	err = os.MkdirAll(filepath.Dir(socketPath), 0755)
	if err != nil {
		log.Log.Reason(err).Error("Could not create directory for socket.")
		panic(err)
	}

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
		client, err := cmdclient.GetClient(socketPath)
		if err != nil {
			return false, nil
		}

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

func createLibvirtConnection() virtcli.Connection {
	libvirtUri := "qemu:///system"
	domainConn, err := virtcli.NewConnection(libvirtUri, "", "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to libvirtd: %v", err))
	}

	return domainConn
}

func startDomainEventMonitoring(notifier *notifyclient.NotifyClient, virtShareDir string, domainConn virtcli.Connection, deleteNotificationSent chan watch.Event, vmiUID types.UID, qemuAgentPollerInterval *time.Duration) {
	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				log.Log.Reason(res).Error("Listening to libvirt events failed, retrying.")
				time.Sleep(time.Second)
			}
		}
	}()

	err := notifier.StartDomainNotifier(domainConn, deleteNotificationSent, vmiUID, qemuAgentPollerInterval)
	if err != nil {
		panic(err)
	}
}

func startWatchdogTicker(watchdogFile string, watchdogInterval time.Duration, stopChan chan struct{}, uid string) (done chan struct{}) {
	err := watchdog.WatchdogFileUpdate(watchdogFile, uid)
	if err != nil {
		panic(err)
	}

	log.Log.Infof("Watchdog file created at %s", watchdogFile)
	done = make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(watchdogInterval).C
		for {
			select {
			case <-stopChan:
				return
			case <-ticker:
				err := watchdog.WatchdogFileUpdate(watchdogFile, uid)
				if err != nil {
					panic(err)
				}
			}
		}
	}()
	return done
}

func initializeDirs(virtShareDir string,
	ephemeralDiskDir string,
	uid string) {

	err := virtlauncher.InitializeSharedDirectories(virtShareDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializePrivateDirectories(filepath.Join("/var/run/kubevirt-private", uid))
	if err != nil {
		panic(err)
	}

	err = cloudinit.SetLocalDirectory(ephemeralDiskDir + "/cloud-init-data")
	if err != nil {
		panic(err)
	}

	err = containerdisk.SetLocalDirectory(ephemeralDiskDir + "/container-disk-data")
	if err != nil {
		panic(err)
	}

	err = ephemeraldisk.SetLocalDirectory(ephemeralDiskDir + "/disk-data")
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

	err = virtlauncher.InitializeDisksDirectories(config.SecretDisksDir)
	if err != nil {
		panic(err)
	}

	err = virtlauncher.InitializeDisksDirectories(config.ServiceAccountDiskDir)
	if err != nil {
		panic(err)
	}
}

func waitForDomainUUID(timeout time.Duration, events chan watch.Event, stop chan struct{}, domainManager virtwrap.DomainManager) *api.Domain {

	ticker := time.NewTicker(timeout).C
	select {
	case <-ticker:
		panic(fmt.Errorf("timed out waiting for domain to be defined"))
	case e := <-events:
		if e.Type == watch.Deleted {
			// we are done already
			return nil
		}
		if e.Object != nil && e.Type == watch.Added {
			domain := e.Object.(*api.Domain)
			log.Log.Infof("Detected domain with UUID %s", domain.Spec.UUID)
			return domain
		}
	case <-stop:
		return nil
	}
	return nil
}

func waitForFinalNotify(deleteNotificationSent chan watch.Event,
	domainManager virtwrap.DomainManager,
	vm *v1.VirtualMachineInstance) {

	log.Log.Info("Waiting on final notifications to be sent to virt-handler.")

	// First attempt to wait for domain event to occur as a part of the normal shutdown flow.
	// If that fails, call Kill on the domain and wait for the event again.
	// If that that fails, exit. We did our best to shutdown the domain gracefully. We can't block
	// the pod forever. Virt-handler will learn of the domain's exit through the watchdog file expire.

	killTimeout := time.After(15 * time.Second)
	timedOut := false
	for timedOut == false {
		select {
		case e := <-deleteNotificationSent:
			if e.Type == watch.Deleted {
				log.Log.Info("Final Delete notification sent")
				return
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
	domainManager.KillVMI(vm)

	// We don't want to block here forever. If the delete does not occur, that could mean
	// something is wrong with libvirt. In this situation, virt-handler will detect that
	// the domain went away eventually, however the exit status will be unknown.
	finalTimeout := time.After(30 * time.Second)
	for {
		select {
		case e := <-deleteNotificationSent:
			if e.Type == watch.Deleted {
				log.Log.Info("Final Delete notification sent after calling kill.")
				return
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
	name := pflag.String("name", "", "Name of the VirtualMachineInstance")
	uid := pflag.String("uid", "", "UID of the VirtualMachineInstance")
	namespace := pflag.String("namespace", "", "Namespace of the VirtualMachineInstance")
	watchdogInterval := pflag.Duration("watchdog-update-interval", defaultWatchdogInterval, "Interval at which watchdog file should be updated")
	readinessFile := pflag.String("readiness-file", "/var/run/kubevirt-infra/healthy", "Pod looks for this file to determine when virt-launcher is initialized")
	gracePeriodSeconds := pflag.Int("grace-period-seconds", 30, "Grace period to observe before sending SIGTERM to vm process")
	useEmulation := pflag.Bool("use-emulation", false, "Use software emulation")
	hookSidecars := pflag.Uint("hook-sidecars", 0, "Number of requested hook sidecars, virt-launcher will wait for all of them to become available")
	noFork := pflag.Bool("no-fork", false, "Fork and let virt-launcher watch itself to react to crashes if set to false")
	lessPVCSpaceToleration := pflag.Int("less-pvc-space-toleration", 0, "Toleration in percent when PVs' available space is smaller than requested")
	qemuAgentPollerInterval := pflag.Duration("qemu-agent-poller-interval", 60, "Interval in seconds between consecutive qemu agent calls")
	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")

	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.Parse()

	log.InitializeLogging("virt-launcher")

	if !*noFork {
		err := ForkAndMonitor("qemu-system")
		if err != nil {
			log.Log.Reason(err).Error("monitoring virt-launcher failed")
			os.Exit(1)
		}
		return
	}

	// Block until all requested hookSidecars are ready
	hookManager := hooks.GetManager()
	err := hookManager.Collect(*hookSidecars, *qemuTimeout)
	if err != nil {
		panic(err)
	}

	vm := v1.NewVMIReferenceFromNameWithNS(*namespace, *name)

	// Initialize local and shared directories
	initializeDirs(*virtShareDir, *ephemeralDiskDir, *uid)

	// Start libvirtd, virtlogd, and establish libvirt connection
	stopChan := make(chan struct{})

	watchdogFile := watchdog.WatchdogFileFromNamespaceName(*virtShareDir,
		*namespace,
		*name)
	watchdogDone := startWatchdogTicker(watchdogFile, *watchdogInterval, stopChan, *uid)

	err = util.SetupLibvirt()
	if err != nil {
		panic(err)
	}
	util.StartLibvirt(stopChan)
	if err != nil {
		panic(err)
	}
	util.StartVirtlog(stopChan)

	domainConn := createLibvirtConnection()
	defer domainConn.Close()

	notifier, err := notifyclient.NewNotifyClient(*virtShareDir)
	if err != nil {
		panic(err)
	}

	domainManager, err := virtwrap.NewLibvirtDomainManager(domainConn, *virtShareDir, notifier, *lessPVCSpaceToleration)
	if err != nil {
		panic(err)
	}

	// Start the virt-launcher command service.
	// Clients can use this service to tell virt-launcher
	// to start/stop virtual machines
	options := cmdserver.NewServerOptions(*useEmulation)
	socketPath := cmdclient.SocketFromUID(*virtShareDir, *uid)
	cmdServerDone := startCmdServer(socketPath, domainManager, stopChan, options)

	gracefulShutdownTriggerFile := virtlauncher.GracefulShutdownTriggerFromNamespaceName(*virtShareDir,
		*namespace,
		*name)
	err = virtlauncher.GracefulShutdownTriggerClear(gracefulShutdownTriggerFile)
	if err != nil {
		log.Log.Reason(err).Errorf("Error clearing shutdown trigger file %s.", gracefulShutdownTriggerFile)
		panic(err)
	}

	shutdownCallback := func(pid int) {
		err := domainManager.KillVMI(vm)
		if err != nil {
			log.Log.Reason(err).Errorf("Unable to stop qemu with libvirt, falling back to SIGTERM")
			syscall.Kill(pid, syscall.SIGTERM)
		}
	}

	events := make(chan watch.Event, 10)
	// Send domain notifications to virt-handler
	startDomainEventMonitoring(notifier, *virtShareDir, domainConn, events, vm.UID, qemuAgentPollerInterval)

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
	markReady(*readinessFile)

	domain := waitForDomainUUID(*qemuTimeout, events, signalStopChan, domainManager)
	if domain != nil {
		mon := virtlauncher.NewProcessMonitor(domain.Spec.UUID,
			gracefulShutdownTriggerFile,
			*gracePeriodSeconds,
			shutdownCallback)

		// This is a wait loop that monitors the qemu pid. When the pid
		// exits, the wait loop breaks.
		mon.RunForever(*qemuTimeout, signalStopChan)

		// Now that the pid has exited, we wait for the final delete notification to be
		// sent back to virt-handler. This delete notification contains the reason the
		// domain exited.
		waitForFinalNotify(events, domainManager, vm)
	}

	close(stopChan)
	<-cmdServerDone
	<-watchdogDone

	log.Log.Info("Exiting...")
}

// ForkAndMonitor itself to give qemu an extra grace period to properly terminate
// in case of virt-launcher crashes
func ForkAndMonitor(qemuProcessCommandPrefix string) error {
	cmd := exec.Command(os.Args[0], append(os.Args[1:], "--no-fork", "true")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Log.Reason(err).Error("failed to fork virt-launcher")
		return err
	}

	sigs := make(chan os.Signal, 10)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGCHLD)
	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGCHLD:
				var wstatus syscall.WaitStatus
				wpid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
				if err != nil {
					log.Log.Reason(err).Errorf("Failed to reap process %d", wpid)
				}
			default:
				log.Log.V(3).Log("signalling virt-launcher to shut down")
				err := cmd.Process.Signal(syscall.SIGTERM)
				sig.Signal()
				if err != nil {
					log.Log.Reason(err).Errorf("received signal %s but can't signal virt-launcher to shut down", sig.String())
				}
			}
		}
	}()

	// wait for virt-launcher and collect the exit code
	exitCode := 0
	if err := cmd.Wait(); err != nil {
		exitCode = 1
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
		log.Log.Reason(err).Error("dirty virt-launcher shutdown")
	}
	// give qemu some time to shut down in case it survived virt-handler
	pid, _ := virtlauncher.FindPid(qemuProcessCommandPrefix)
	if pid > 0 {
		p, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		// Signal qemu to shutdown
		err = p.Signal(syscall.SIGTERM)
		if err != nil {
			return err
		}
		// Wait for 10 seconds for the qemu process to disappear
		err = utilwait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			pid, _ := virtlauncher.FindPid(qemuProcessCommandPrefix)
			if pid == 0 {
				return true, nil
			}
			return false, nil
		})
	}
	os.Exit(exitCode)
	return nil
}
