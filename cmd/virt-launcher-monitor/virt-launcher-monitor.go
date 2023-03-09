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
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/net"

	"github.com/spf13/pflag"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"kubevirt.io/client-go/log"
)

const (
	envoyMergedPrometheusTelemetryPort = 15020
	envoyHealthCheckPort               = 15021
	httpRequestTimeout                 = 2 * time.Second
)

func cleanupContainerDiskDirectory(ephemeralDiskDir string) {
	// Cleanup the content of ephemeralDiskDir, to make sure that all containerDisk containers terminate
	err := RemoveContents(ephemeralDiskDir)
	if err != nil {
		log.Log.Reason(err).Errorf("could not clean up ephemeral disk directory: %s", ephemeralDiskDir)
	}
}

func main() {

	containerDiskDir := pflag.String("container-disk-dir", "/var/run/kubevirt/container-disks", "Base directory for container disk data")
	keepAfterFailure := pflag.Bool("keep-after-failure", false, "virt-launcher will be kept alive after failure for debugging if set to true")

	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")
	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.CommandLine.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	pflag.Parse()

	log.InitializeLogging("virt-launcher-monitor")

	// check if virt-launcher verbosity should be changed
	if verbosityStr, ok := os.LookupEnv("VIRT_LAUNCHER_LOG_VERBOSITY"); ok {
		if verbosity, err := strconv.Atoi(verbosityStr); err == nil {
			log.Log.SetVerbosityLevel(verbosity)
			log.Log.Infof("set log verbosity to %d", verbosity)
		} else {
			log.Log.Warningf("failed to set log verbosity. The value of logVerbosity label should be an integer, got %s instead.", verbosityStr)
		}
	}

	exitCode, err := RunAndMonitor(*containerDiskDir)
	if *keepAfterFailure && (exitCode != 0 || err != nil) {
		log.Log.Infof("keeping virt-launcher container alive since --keep-after-failure is set to true")
		<-make(chan struct{})
	}

	if err != nil {
		log.Log.Reason(err).Error("monitoring virt-launcher failed")
		os.Exit(1)
	}

	os.Exit(exitCode)
}

// RunAndMonitor run virt-launcher process and monitor it to give qemu an extra grace period to properly terminate
// in case of crashes
func RunAndMonitor(containerDiskDir string) (int, error) {
	defer cleanupContainerDiskDirectory(containerDiskDir)
	defer terminateIstioProxy()
	args := removeArg(os.Args[1:], "--keep-after-failure")
	cmd := exec.Command("/usr/bin/virt-launcher", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Log.Reason(err).Error("failed to run virt-launcher")
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
				log.Log.V(3).Log("signalling virt-launcher to shut down")
				err := cmd.Process.Signal(syscall.SIGTERM)
				sig.Signal()
				if err != nil {
					log.Log.Reason(err).Errorf("received signal %s but can't signal virt-launcher to shut down", sig.String())
				}
			}
		}
	}()

	exitCode := <-exitStatus
	if exitCode != 0 {
		log.Log.Errorf("dirty virt-launcher shutdown: exit-code %d", exitCode)
	}

	// give qemu some time to shut down in case it survived virt-handler
	// Most of the time we call `qemu-system=* binaries, but qemu-system-* packages
	// are not everywhere available where libvirt and qemu are. There we usually call qemu-kvm
	// which resides in /usr/libexec/qemu-kvm
	pid, _ := findPid("qemu-system")
	qemuProcessCommandPrefix := "qemu-system"
	if pid <= 0 {
		pid, _ = findPid("qemu-kvm")
		qemuProcessCommandPrefix = "qemu-kvm"
	}

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
			pid, _ := findPid(qemuProcessCommandPrefix)
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

func terminateIstioProxy() {
	httpClient := &http.Client{Timeout: httpRequestTimeout}
	if istioProxyPresent(httpClient) {
		isRetriable := func(err error) bool {
			if net.IsConnectionReset(err) || net.IsConnectionRefused(err) || k8serrors.IsServiceUnavailable(err) {
				return true
			}
			return false
		}
		err := retry.OnError(retry.DefaultBackoff, isRetriable, func() error {
			resp, err := httpClient.Post(fmt.Sprintf("http://localhost:%d/quitquitquit", envoyMergedPrometheusTelemetryPort), "", nil)
			if err != nil {
				log.Log.Reason(err).Error("failed to request istio-proxy termination, retrying...")
				return err
			}

			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Log.Errorf("status code received: %d", resp.StatusCode)
				return err
			}

			return nil
		})
		if err != nil {
			log.Log.Reason(err).Error("all attempts to terminate istio-proxy failed")
		}
	}
}

func istioProxyPresent(httpClient *http.Client) bool {
	isRetriable := func(err error) bool {
		if net.IsConnectionReset(err) || net.IsConnectionRefused(err) {
			return true
		}

		return false
	}
	err := retry.OnError(retry.DefaultBackoff, isRetriable, func() error {
		resp, err := httpClient.Get(fmt.Sprintf("http://localhost:%d/healthz/ready", envoyHealthCheckPort))
		if err != nil {
			log.Log.Reason(err).V(4).Info("error when checking for istio-proxy presence")
			return err
		}

		defer resp.Body.Close()
		if resp.Header.Get("server") == "envoy" {
			return nil
		}
		return fmt.Errorf("received response from non-istio health server: %s", resp.Header.Get("server"))
	})
	if err != nil {
		return false
	}

	return true
}

func findPid(commandNamePrefix string) (int, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		// #nosec No risk for path injection. Reading specific entries under /proc
		content, err := os.ReadFile(entry)
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

func removeArg(args []string, arg string) []string {
	i := 0
	for _, elem := range args {
		if elem != arg {
			args[i] = elem
			i++
		}
	}
	args = args[:i]

	return args
}
