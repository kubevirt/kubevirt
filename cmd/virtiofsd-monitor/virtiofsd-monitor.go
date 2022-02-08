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
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	//utilNet "k8s.io/apimachinery/pkg/util/net"

	"github.com/spf13/pflag"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

	//"k8s.io/client-go/util/retry"

	"kubevirt.io/client-go/log"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

func cleanupContainerDiskDirectory(ephemeralDiskDir string) {
	// Cleanup the content of ephemeralDiskDir, to make sure that all containerDisk containers terminate
	err := RemoveContents(ephemeralDiskDir)
	if err != nil {
		log.Log.Reason(err).Errorf("could not clean up ephemeral disk directory: %s", ephemeralDiskDir)
	}
}

func createFindMeSocket(stopChan chan struct{}) error {
	fdChan := make(chan net.Conn, 1)
	listenErrChan := make(chan error, 1)
	findMeSocket := "/var/run/kubevirt/virtiofs-containers/findMe.sock"
	os.RemoveAll(findMeSocket)

	listener, err := net.Listen("unix", findMeSocket)
	if err != nil {
		log.Log.Reason(err).Error("failed to create a findMe unix socket")
		return err
	}
	if err := diskutils.DefaultOwnershipManager.SetFileOwnership(findMeSocket); err != nil {
		log.Log.Reason(err).Error("failed to change ownership on findMe unix socket")
		return err
	}

	go func(ln net.Listener, fdChan chan net.Conn, listenErr chan error, stopChan chan struct{}) {
		defer func() {
			if ln != nil {
				log.Log.Infof("stopping findMe socket listener")
				ln.Close()
			}
		}()
		for {
			fd, err := ln.Accept()
			if err != nil {
				listenErr <- err

				select {
				case <-stopChan:
					// If the stopChan is closed, then this is expected. Log at a lesser debug level
					log.Log.Reason(err).V(3).Infof("stopChan is closed. Listener exited with expected error.")
				default:
					log.Log.Reason(err).Error("proxy unix socket listener returned error.")
				}
				break
			} else {
				fdChan <- fd
			}
		}
	}(listener, fdChan, listenErrChan, stopChan)

	go func(fdChan chan net.Conn, listenErr chan error, stopChan chan struct{}) {
		for {
			select {
			case fd := <-fdChan:
				go handleConnection(fd, stopChan)
			case <-stopChan:
				return
			case <-listenErrChan:
				return
			}
		}

	}(fdChan, listenErrChan, stopChan)
	return nil
}

func handleConnection(fd net.Conn, stopChan chan struct{}) {
	defer fd.Close()

	outBoundErr := make(chan error, 1)

	//var conn net.Conn
	var err error
	go func() {
		_, err := io.Copy(io.Discard, fd)
		outBoundErr <- err
	}()

	select {
	case err = <-outBoundErr:
		if err != nil {
			log.Log.Reason(err).Errorf("error encountered copying data to outbound connection")
		}
	case <-stopChan:
		log.Log.Info("stop findMe socket listener")
	}
}

func main() {

	socketPath := pflag.String("socket-path", "", "path to the externally launched virtiofsd socket")
	volumeName := pflag.String("volume-name", "", "name of the shared volume")
	//socketPath := pflag.String("socket-path", "", "path to the externally launched virtiofsd socket")
	keepAfterFailure := pflag.Bool("keep-after-failure", false, "virt-launcher will be kept alive after failure for debugging if set to true")

	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")
	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.CommandLine.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	pflag.Parse()

	log.InitializeLogging("virtiofsd-monitor")

	// check if virt-launcher verbosity should be changed
	if verbosityStr, ok := os.LookupEnv("VIRT_LAUNCHER_LOG_VERBOSITY"); ok {
		if verbosity, err := strconv.Atoi(verbosityStr); err == nil {
			log.Log.SetVerbosityLevel(verbosity)
			log.Log.V(2).Infof("set log verbosity to %d", verbosity)
		} else {
			log.Log.Warningf("failed to set log verbosity. The value of logVerbosity label should be an integer, got %s instead.", verbosityStr)
		}
	}

	exitCode, err := RunAndMonitor(*socketPath, *volumeName)
	if *keepAfterFailure && (exitCode != 0 || err != nil) {
		log.Log.Infof("keeping virtofsd-launcher container alive since --keep-after-failure is set to true")
		<-make(chan struct{})
	}

	if err != nil {
		log.Log.Reason(err).Error("monitoring virt-launcher failed")
		os.Exit(1)
	}

	os.Exit(exitCode)
}

// RunAndMonitor run virtiofsd process and monitor it to give qemu an extra grace period to properly terminate
// in case of crashes
func RunAndMonitor(socketPath, volumeName string) (int, error) {

	stopChan := make(chan struct{})
	defer close(stopChan)
	createFindMeSocket(stopChan)

	//args := os.Args[1:]
	socketPathArg := fmt.Sprintf("--socket-path=%s", socketPath)
	optionsArg := fmt.Sprintf("source=/%s", volumeName)
	//optionsArg := fmt.Sprintf("source=", mountVolumePath)
	args := []string{socketPathArg, "-o", optionsArg, "-o", "sandbox=chroot", "-o", "xattr", "-o", "xattrmap=:map::user.virtiofsd.:"}
	cmd := exec.Command("/usr/libexec/virtiofsd", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Log.Reason(err).Error("failed to run virtiofsd")
		return 1, err
	}
	// Wait for 10 seconds for the qemu process to disappear
	if err := utilwait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath); err != nil {
			log.Log.Reason(err).Error("failed to change ownership on virtiofsd unix socket")
			return false, nil
		}
		return true, nil
	}); err != nil {
		return 1, err
	}

	exitStatus := make(chan int, 10)
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

				log.Log.Infof("Reaped pid %d with status %d", wpid, int(wstatus))
				if wpid == cmd.Process.Pid {
					exitStatus <- wstatus.ExitStatus()
				}

			default:
				log.Log.V(3).Log("signalling virtiofsd to shut down")
				err := cmd.Process.Signal(syscall.SIGTERM)
				sig.Signal()
				if err != nil {
					log.Log.Reason(err).Errorf("received signal %s but can't signal virtiofsd to shut down", sig.String())
				}
			}
		}
	}()

	exitCode := <-exitStatus
	if exitCode != 0 {
		log.Log.Errorf("dirty virtiofsd shutdown: exit-code %d", exitCode)
	}

	// give qemu some time to shut down in case it survived virt-handler
	// Most of the time we call `qemu-system=* binaries, but qemu-system-* packages
	// are not everywhere available where libvirt and qemu are. There we usually call qemu-kvm
	// which resides in /usr/libexec/qemu-kvm
	pid, _ := findPid("virtiofsd")
	virtiofsdProcessCommandPrefix := "virtiofsd"

	if pid > 0 {
		p, err := os.FindProcess(pid)
		if err != nil {
			return 1, err
		}

		// Signal qemu to shutdown
		err = p.Signal(syscall.SIGTERM)
		if err != nil {
			return 1, err
		}

		// Wait for 10 seconds for the qemu process to disappear
		err = utilwait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			pid, _ := findPid(virtiofsdProcessCommandPrefix)
			if pid == 0 {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return 1, err
		}
	}
	return exitCode, nil
}

func RemoveContents(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sock"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func findPid(commandNamePrefix string) (int, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		// #nosec No risk for path injection. Reading specific entries under /proc
		content, err := ioutil.ReadFile(entry)
		if err != nil {
			return 0, err
		}

		if !strings.Contains(string(content), commandNamePrefix) {
			continue
		}

		//   <empty> /    proc     /    $PID   /   cmdline
		// items[0] sep items[1] sep items[2] sep  items[3]
		items := strings.Split(entry, string(os.PathSeparator))
		pid, err := strconv.Atoi(items[2])
		if err != nil {
			return 0, err
		}

		// everything matched, hooray!
		return pid, nil
	}

	return 0, fmt.Errorf("Process %s not found in /proc", commandNamePrefix)
}
