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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/libvirt/libvirt-go"
	"github.com/spf13/pflag"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/log"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	virtcli "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
	watchdog "kubevirt.io/kubevirt/pkg/watchdog"
)

const defaultStartTimeout = 3 * time.Minute
const defaultWatchdogInterval = 5 * time.Second

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
	options *cmdserver.ServerOptions) {

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

	err = cmdserver.RunServer(socketPath, domainManager, stopChan, options)
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
}

func createLibvirtConnection() virtcli.Connection {
	libvirtUri := "qemu:///system"
	domainConn, err := virtcli.NewConnection(libvirtUri, "", "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to libvirtd: %v", err))
	}

	return domainConn
}

func startDomainEventMonitoring(virtShareDir string, domainConn virtcli.Connection, deleteNotificationSent chan watch.Event) {
	libvirt.EventRegisterDefaultImpl()

	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				log.Log.Reason(res).Error("Listening to libvirt events failed, retrying.")
				time.Sleep(time.Second)
			}
		}
	}()

	err := notifyclient.StartNotifier(virtShareDir, domainConn, deleteNotificationSent)
	if err != nil {
		panic(err)
	}

}

func startWatchdogTicker(watchdogFile string, watchdogInterval time.Duration, stopChan chan struct{}) {
	err := watchdog.WatchdogFileUpdate(watchdogFile)
	if err != nil {
		panic(err)
	}

	log.Log.Infof("Watchdog file created at %s", watchdogFile)

	go func() {

		ticker := time.NewTicker(watchdogInterval).C
		for {
			select {
			case <-stopChan:
				return
			case <-ticker:
				err := watchdog.WatchdogFileUpdate(watchdogFile)
				if err != nil {
					panic(err)
				}
			}
		}
	}()
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

	err = registrydisk.SetLocalDirectory(ephemeralDiskDir + "/registry-disk-data")
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
}

func waitForDomainUUID(timeout time.Duration, domainManager virtwrap.DomainManager) string {
	start := time.Now()

	for time.Since(start) < timeout {
		time.Sleep(time.Second)
		list, err := domainManager.ListAllDomains()
		if err != nil {
			log.Log.Reason(err).Error("failed to retrieve domains from libvirt")
			continue
		}

		if len(list) == 0 {
			continue
		}

		domain := list[0]
		if domain.Spec.UUID != "" {
			log.Log.Infof("Detected domain with UUID %s", domain.Spec.UUID)
			return domain.Spec.UUID
		}
	}

	panic(fmt.Errorf("timed out waiting for domain to be defined"))
}

func waitForFinalNotify(deleteNotificationSent chan watch.Event,
	domainManager virtwrap.DomainManager,
	vm *v1.VirtualMachineInstance) {
	// There are many conditions that can cause the qemu pid to exit that
	// don't involve the VirtualMachineInstance's domain from being deleted from libvirt.
	//
	// KillVMI is idempotent. Making a call to KillVMI here ensures that the deletion
	// occurs regardless if the VirtualMachineInstance crashed unexpectedly or if virt-handler requested
	// a graceful shutdown.
	domainManager.KillVMI(vm)

	log.Log.Info("Waiting on final notifications to be sent to virt-handler.")

	// We don't want to block here forever. If the delete does not occur, that could mean
	// something is wrong with libvirt. In this situation, wirt-handler will detect that
	// the domain went away eventually, however the exit status will be unknown.
	timeout := time.After(30 * time.Second)
	select {
	case <-deleteNotificationSent:
		log.Log.Info("Final Delete notification sent")
	case <-timeout:
		log.Log.Info("Timed out waiting for final delete notification.")
	}
}

func main() {
	qemuTimeout := flag.Duration("qemu-timeout", defaultStartTimeout, "Amount of time to wait for qemu")
	virtShareDir := flag.String("kubevirt-share-dir", "/var/run/kubevirt", "Shared directory between virt-handler and virt-launcher")
	ephemeralDiskDir := flag.String("ephemeral-disk-dir", "/var/run/libvirt/kubevirt-ephemeral-disk", "Base directory for ephemeral disk data")
	name := flag.String("name", "", "Name of the VirtualMachineInstance")
	uid := flag.String("uid", "", "UID of the VirtualMachineInstance")
	namespace := flag.String("namespace", "", "Namespace of the VirtualMachineInstance")
	watchdogInterval := flag.Duration("watchdog-update-interval", defaultWatchdogInterval, "Interval at which watchdog file should be updated")
	readinessFile := flag.String("readiness-file", "/tmp/health", "Pod looks for this file to determine when virt-launcher is initialized")
	gracePeriodSeconds := flag.Int("grace-period-seconds", 30, "Grace period to observe before sending SIGTERM to vm process")
	useEmulation := flag.Bool("use-emulation", false, "Use software emulation")
	hookSidecars := flag.Uint("hook-sidecars", 0, "Number of requested hook sidecars, virt-launcher will wait for all of them to become available")

	// set new default verbosity, was set to 0 by glog
	flag.Set("v", "2")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	log.InitializeLogging("virt-launcher")

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
	defer close(stopChan)

	util.StartLibvirt(stopChan)
	util.StartVirtlog(stopChan)

	domainConn := createLibvirtConnection()
	defer domainConn.Close()

	domainManager, err := virtwrap.NewLibvirtDomainManager(domainConn)
	if err != nil {
		panic(err)
	}

	// Start the virt-launcher command service.
	// Clients can use this service to tell virt-launcher
	// to start/stop virtual machines
	options := cmdserver.NewServerOptions(*useEmulation)
	socketPath := cmdclient.SocketFromUID(*virtShareDir, *uid)
	startCmdServer(socketPath, domainManager, stopChan, options)

	watchdogFile := watchdog.WatchdogFileFromNamespaceName(*virtShareDir,
		*namespace,
		*name)
	startWatchdogTicker(watchdogFile, *watchdogInterval, stopChan)

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

	deleteNotificationSent := make(chan watch.Event, 10)
	// Send domain notifications to virt-handler
	startDomainEventMonitoring(*virtShareDir, domainConn, deleteNotificationSent)

	// Marking Ready allows the container's readiness check to pass.
	// This informs virt-controller that virt-launcher is ready to handle
	// managing virtual machines.
	markReady(*readinessFile)

	domainUUID := waitForDomainUUID(*qemuTimeout, domainManager)
	mon := virtlauncher.NewProcessMonitor(domainUUID,
		gracefulShutdownTriggerFile,
		*gracePeriodSeconds,
		shutdownCallback)

	// This is a wait loop that monitors the qemu pid. When the pid
	// exits, the wait loop breaks.
	mon.RunForever(*qemuTimeout)

	// Now that the pid has exited, we wait for the final delete notification to be
	// sent back to virt-handler. This delete notification contains the reason the
	// domain exited.
	waitForFinalNotify(deleteNotificationSent, domainManager, vm)

	log.Log.Info("Exiting...")
}
