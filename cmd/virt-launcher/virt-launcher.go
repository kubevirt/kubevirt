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
 * Copyright 2022 Intel Corporation.
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

	utilwait "k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/client-go/log"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/ignition"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const defaultStartTimeout = 3 * time.Minute

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
}

func main() {
	pflag.Duration("qemu-timeout", defaultStartTimeout, "Amount of time to wait for qemu")
	virtShareDir := pflag.String("kubevirt-share-dir", "/var/run/kubevirt", "Shared directory between virt-handler and virt-launcher")
	ephemeralDiskDir := pflag.String("ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks", "Base directory for ephemeral disk data")
	containerDiskDir := pflag.String("container-disk-dir", "/var/run/kubevirt/container-disks", "Base directory for container disk data")
	hotplugDiskDir := pflag.String("hotplug-disk-dir", "/var/run/kubevirt/hotplug-disks", "Base directory for hotplug disk data")
	pflag.String("name", "", "Name of the VirtualMachineInstance")
	uid := pflag.String("uid", "", "UID of the VirtualMachineInstance")
	pflag.String("namespace", "", "Namespace of the VirtualMachineInstance")
	pflag.Int("grace-period-seconds", 30, "Grace period to observe before sending SIGTERM to vmi process")
	allowEmulation := pflag.Bool("allow-emulation", false, "Allow use of software emulation as fallback")
	runWithNonRoot := pflag.Bool("run-as-nonroot", false, "Run libvirtd with the 'virt' user")
	pflag.Uint("hook-sidecars", 0, "Number of requested hook sidecars, virt-launcher will wait for all of them to become available")
	ovmfPath := pflag.String("ovmf-path", "/usr/share/OVMF", "The directory that contains the EFI roms (like CLOUDHV.fd)")
	pflag.Duration("qemu-agent-sys-interval", 120*time.Second, "Interval between consecutive qemu agent calls for sys commands")
	pflag.Duration("qemu-agent-file-interval", 300*time.Second, "Interval between consecutive qemu agent calls for file command")
	pflag.Duration("qemu-agent-user-interval", 10*time.Second, "Interval between consecutive qemu agent calls for user command")
	pflag.Duration("qemu-agent-version-interval", 300*time.Second, "Interval between consecutive qemu agent calls for version command")
	pflag.Duration("qemu-fsfreeze-status-interval", 5*time.Second, "Interval between consecutive qemu agent calls for fsfreeze status command")
	simulateCrash := pflag.Bool("simulate-crash", false, "Causes virt-launcher to immediately crash. This is used by functional tests to simulate crash loop scenarios.")
	pflag.String("libvirt-log-filters", "", "Set custom log filters for libvirt")

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

	// Initialize local and shared directories
	initializeDirs(*ephemeralDiskDir, *containerDiskDir, *hotplugDiskDir, *uid)
	ephemeralDiskCreator := ephemeraldisk.NewEphemeralDiskCreator(filepath.Join(*ephemeralDiskDir, "disk-data"))
	if err := ephemeralDiskCreator.Init(); err != nil {
		panic(err)
	}

	notifier := notifyclient.NewNotifier(*virtShareDir)
	defer notifier.Close()

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

	serverStopChan := make(chan struct{})

	// Start VMM
	wrapper := util.NewCloudHvWrapper(*runWithNonRoot)
	apiSocketPath, err := wrapper.CreateCloudHvApiSocket(*virtShareDir)
	if err != nil {
		panic(err)
	}

	if err := wrapper.StartCloudHv(signalStopChan); err != nil {
		panic(err)
	}

	domainManager, err := virtwrap.NewCloudHvDomainManager(apiSocketPath, *ephemeralDiskDir, *ovmfPath, ephemeralDiskCreator)
	if err != nil {
		panic(err)
	}

	if err := notifier.StartCloudHvDomainNotifier(wrapper.EventMonitorConn(), domainManager.GetDomain()); err != nil {
		panic(err)
	}

	// Start the virt-launcher command service.
	// Clients can use this service to tell virt-launcher
	// to start/stop virtual machines
	options := cmdserver.NewServerOptions(*allowEmulation)
	cmdclient.SetLegacyBaseDir(*virtShareDir)
	cmdServerDone := startCmdServer(cmdclient.UninitializedSocketOnGuest(), domainManager, serverStopChan, options)

	// Marking Ready allows the container's readiness check to pass.
	// This informs virt-controller that virt-launcher is ready to handle
	// managing virtual machines.
	markReady()

	if err := wrapper.WaitCloudHvProcess(); err != nil {
		panic(err)
	}

	close(serverStopChan)
	<-cmdServerDone

	log.Log.Info("Exiting...")
}
