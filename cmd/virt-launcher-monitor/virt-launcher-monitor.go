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
	"bufio"
	"errors"
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

	"github.com/spf13/pflag"

	"golang.org/x/sys/unix"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

const (
	envoyMergedPrometheusTelemetryPort = 15020
	envoyHealthCheckPort               = 15021
	httpRequestTimeout                 = 2 * time.Second

	passtLogFile = "/var/run/kubevirt/passt.log" // #nosec G101
)

func cleanupContainerDiskDirectory(ephemeralDiskDir string) {
	// Cleanup the content of ephemeralDiskDir, to make sure that all containerDisk containers terminate
	err := RemoveContents(ephemeralDiskDir)
	if err != nil {
		log.Log.Reason(err).Errorf("could not clean up ephemeral disk directory: %s", ephemeralDiskDir)
	}
}

func createSerialConsoleTermFile(uid, suffix string) (bool, error) {
	// Create a file that it will be removed to quickly signal the
	// shutdown to the guest-console-log container in the case the sigterm signal got
	// missed and some client process is still connected to the serial console socket
	const serialPort = 0
	if len(uid) > 0 {
		logSigPath := fmt.Sprintf("%s/%s/virt-serial%d-log-sigTerm%s", util.VirtPrivateDir, uid, serialPort, suffix)

		if _, err := os.Stat(logSigPath); os.IsNotExist(err) {
			file, err := os.Create(logSigPath)
			if err != nil {
				log.Log.Reason(err).Errorf("could not create up serial console term file: %s", logSigPath)
				return false, err
			}
			if err = file.Close(); err != nil {
				log.Log.Reason(err).Errorf("could not create up serial console term file: %s", logSigPath)
				return false, err
			}
			log.Log.V(3).Infof("serial console term file created: %s", logSigPath)
			return true, nil
		}
	}
	return false, nil

}

func removeSerialConsoleTermFile(uid string) {
	// Delete a file (if there) to quickly signal the shutdown to the guest-console-log container in the case the sigterm signal got
	// missed and some client process is still connected to the serial console socket
	const serialPort = 0
	if len(uid) > 0 {
		logSigPath := fmt.Sprintf("%s/%s/virt-serial%d-log-sigTerm", util.VirtPrivateDir, uid, serialPort)

		if _, err := os.Stat(logSigPath); err == nil {
			rerr := os.Remove(logSigPath)
			if rerr != nil {
				log.Log.Reason(err).Errorf("could not delete serial console term file: %s", logSigPath)
				return
			}
			log.Log.V(3).Infof("serial console term file deleted: %s", logSigPath)
		}
	}
	// Create a second termination file for the unlikely case where virt-launcher-monitor
	// has enough time to create and remove the termination file before virt-tail (asynchronously started)
	// notices it.
	createSerialConsoleTermFile(uid, "-done")
}

func main() {

	containerDiskDir := pflag.String("container-disk-dir", "/var/run/kubevirt/container-disks", "Base directory for container disk data")
	keepAfterFailure := pflag.Bool("keep-after-failure", false, "virt-launcher will be kept alive after failure for debugging if set to true")
	uid := pflag.String("uid", "", "UID of the VirtualMachineInstance")

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
			log.Log.V(2).Infof("set log verbosity to %d", verbosity)
		} else {
			log.Log.Warningf("failed to set log verbosity. The value of logVerbosity label should be an integer, got %s instead.", verbosityStr)
		}
	}

	exitCode, err := RunAndMonitor(*containerDiskDir, *uid)
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
func RunAndMonitor(containerDiskDir, uid string) (int, error) {
	defer removeSerialConsoleTermFile(uid)
	defer cleanupContainerDiskDirectory(containerDiskDir)
	defer terminateIstioProxy()
	args := removeArg(os.Args[1:], "--keep-after-failure")

	go func() {
		created := false
		i := 0
		for i < 100 && !created {
			i = i + 1
			created, err := createSerialConsoleTermFile(uid, "")
			if err != nil || !created {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	cmd := exec.Command("/usr/bin/virt-launcher", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE},
	}
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

	dumpLogFile(passtLogFile)

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
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		period := make(chan struct{}, 1)

		go func() {
			period <- struct{}{}
			for range ticker.C {
				period <- struct{}{}
			}
		}()

		for {
			select {
			case <-timeout:
				return 1, err
			case <-period:
				pid, _ := findPid(qemuProcessCommandPrefix)
				if pid == 0 {
					return exitCode, nil
				}
			}
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

type isRetriable func(error) bool
type function func() error

func retryOnError(shouldRetry isRetriable, f function) error {
	var lastErr error
	retries := 4
	sleep := 10 * time.Millisecond

	backOff := func() time.Duration {
		const factor = 5
		sleep *= time.Duration(factor)
		return sleep
	}

	for retries > 0 {
		err := f()
		if err != nil {
			if !shouldRetry(err) {
				return err
			}
			lastErr = err
		} else {
			return nil
		}
		time.Sleep(backOff())
		retries--
	}

	return lastErr
}

func terminateIstioProxy() {
	httpClient := &http.Client{Timeout: httpRequestTimeout}
	if istioProxyPresent(httpClient) {
		serviceUnavailable := fmt.Errorf("service unavailable")
		isRetriable := func(err error) bool {
			var errno syscall.Errno
			if errors.As(err, &errno) {
				return errno == syscall.ECONNRESET || errno == syscall.ECONNREFUSED
			}
			return serviceUnavailable == err

		}
		err := retryOnError(isRetriable, func() error {
			resp, err := httpClient.Post(fmt.Sprintf("http://localhost:%d/quitquitquit", envoyMergedPrometheusTelemetryPort), "", nil)
			if err != nil {
				log.Log.Reason(err).Error("failed to request istio-proxy termination, retrying...")
				return err
			}

			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Log.Errorf("status code received: %d", resp.StatusCode)
				if resp.StatusCode == http.StatusServiceUnavailable {
					return serviceUnavailable
				}
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
		var errno syscall.Errno
		if errors.As(err, &errno) {
			return errno == syscall.ECONNRESET || errno == syscall.ECONNREFUSED
		}

		return false
	}
	err := retryOnError(isRetriable, func() error {
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
	return err == nil
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

func dumpLogFile(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Log.Reason(err).Errorf("failed to open file %s", filePath)
			return
		}
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Log.Reason(err).Errorf("failed to close file: %s", filePath)
		}
	}()

	log.Log.Infof("dump log file: %s", filePath)
	const bufferSize = 1024
	const maxBufferSize = 512 * bufferSize
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, bufferSize), maxBufferSize)
	for scanner.Scan() {
		log.Log.Info(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Log.Reason(err).Errorf("failed to read file %s", filePath)
	}
}
