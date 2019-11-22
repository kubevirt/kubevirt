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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virtlauncher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

type OnShutdownCallback func(pid int)

type monitor struct {
	timeout                     time.Duration
	pid                         int
	cmdlineMatchStr             string
	start                       time.Time
	isDone                      bool
	gracePeriod                 int
	gracePeriodStartTime        int64
	gracefulShutdownTriggerFile string
	shutdownCallback            OnShutdownCallback
}

type ProcessMonitor interface {
	RunForever(startTimeout time.Duration, signalStopChan chan struct{})
}

func GracefulShutdownTriggerDir(baseDir string) string {
	return filepath.Join(baseDir, "graceful-shutdown-trigger")
}

func GracefulShutdownTriggerFromNamespaceName(baseDir string, namespace string, name string) string {
	triggerFile := namespace + "_" + name
	return filepath.Join(baseDir, "graceful-shutdown-trigger", triggerFile)
}

func VmGracefulShutdownTriggerClear(baseDir string, vmi *v1.VirtualMachineInstance) error {
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

	triggerFile := GracefulShutdownTriggerFromNamespaceName(baseDir, namespace, domain)

	return diskutils.RemoveFile(triggerFile)
}

func GracefulShutdownTriggerClear(triggerFile string) error {
	return diskutils.RemoveFile(triggerFile)
}

func VmHasGracefulShutdownTrigger(baseDir string, vmi *v1.VirtualMachineInstance) (bool, error) {
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

	return hasGracefulShutdownTrigger(baseDir, namespace, domain)
}

func hasGracefulShutdownTrigger(baseDir string, namespace string, name string) (bool, error) {
	triggerFile := GracefulShutdownTriggerFromNamespaceName(baseDir, namespace, name)

	return diskutils.FileExists(triggerFile)
}

func GracefulShutdownTriggerInitiate(triggerFile string) error {
	exists, err := diskutils.FileExists(triggerFile)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	f, err := os.Create(triggerFile)
	if err != nil {
		return err
	}
	f.Close()

	return nil
}

func InitializePrivateDirectories(baseDir string) error {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}
	if err := diskutils.DefaultOwnershipManager.SetFileOwnership(baseDir); err != nil {
		return err
	}
	return nil
}

func InitializeDisksDirectories(baseDir string) error {
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return err
	}

	err = os.Chmod(baseDir, 0755)
	if err != nil {
		return err
	}
	err = diskutils.DefaultOwnershipManager.SetFileOwnership(baseDir)
	if err != nil {
		return err
	}
	return nil
}

func InitializeSharedDirectories(baseSharedDir string, baseNetworkDir string) error {
	err := os.MkdirAll(watchdog.WatchdogFileDirectory(baseSharedDir), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(GracefulShutdownTriggerDir(baseSharedDir), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(cmdclient.SocketsDirectory(baseSharedDir), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(baseNetworkDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func NewProcessMonitor(cmdlineMatchStr string,
	gracefulShutdownTriggerFile string,
	gracePeriod int,
	shutdownCallback OnShutdownCallback) ProcessMonitor {
	return &monitor{
		cmdlineMatchStr:             cmdlineMatchStr,
		gracePeriod:                 gracePeriod,
		gracefulShutdownTriggerFile: gracefulShutdownTriggerFile,
		shutdownCallback:            shutdownCallback,
	}
}

func (mon *monitor) isGracePeriodExpired() bool {
	if mon.gracePeriodStartTime != 0 {
		now := time.Now().UTC().Unix()
		if (now - mon.gracePeriodStartTime) > int64(mon.gracePeriod) {
			return true
		}
	}
	return false
}

func (mon *monitor) refresh() {
	if mon.isDone {
		log.Log.Error("Called refresh after done!")
		return
	}

	log.Log.V(4).Infof("Refreshing. CommandPrefix %s pid %d", mon.cmdlineMatchStr, mon.pid)

	expired := mon.isGracePeriodExpired()

	// is the process there?
	if mon.pid == 0 {
		var err error

		mon.pid, err = FindPid(mon.cmdlineMatchStr)
		if err != nil {

			log.Log.Infof("Still missing PID for %s, %v", mon.cmdlineMatchStr, err)
			// check to see if we've timed out looking for the process
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				log.Log.Infof("%s not found after timeout", mon.cmdlineMatchStr)
				mon.isDone = true
			} else if expired {
				log.Log.Infof("%s not found after grace period expired", mon.cmdlineMatchStr)
				mon.isDone = true
			}
			return
		}

		log.Log.Infof("Found PID for %s: %d", mon.cmdlineMatchStr, mon.pid)
	}

	exists, err := pidExists(mon.pid)
	if err != nil {
		log.Log.Reason(err).Errorf("Error detecting pid (%d) status.", mon.pid)
		return
	}
	if exists == false {
		log.Log.Infof("Process %s and pid %d is gone!", mon.cmdlineMatchStr, mon.pid)
		mon.pid = 0
		mon.isDone = true
		return
	}

	if expired {
		log.Log.Infof("Grace Period expired, shutting down.")
		mon.shutdownCallback(mon.pid)
	}

	return
}

func (mon *monitor) monitorLoop(startTimeout time.Duration, signalStopChan chan struct{}) {
	// random value, no real rationale
	rate := 1 * time.Second

	timeoutRepr := fmt.Sprintf("%v", startTimeout)
	if startTimeout == 0 {
		timeoutRepr = "disabled"
	}
	log.Log.Infof("Monitoring loop: rate %v start timeout %s", rate, timeoutRepr)

	ticker := time.NewTicker(rate)

	mon.isDone = false
	mon.timeout = startTimeout
	mon.start = time.Now()

	for !mon.isDone {
		select {
		case <-ticker.C:
			mon.refresh()
		case <-signalStopChan:
			if mon.gracePeriodStartTime != 0 {
				continue
			}

			err := GracefulShutdownTriggerInitiate(mon.gracefulShutdownTriggerFile)
			if err != nil {
				log.Log.Reason(err).Errorf("Error detected attempting to initialize graceful shutdown using trigger file %s.", mon.gracefulShutdownTriggerFile)
			}
			mon.gracePeriodStartTime = time.Now().UTC().Unix()
		}
	}

	ticker.Stop()
}

func (mon *monitor) RunForever(startTimeout time.Duration, signalStopChan chan struct{}) {

	mon.monitorLoop(startTimeout, signalStopChan)
}

func readProcCmdline(pathname string) ([]string, error) {
	content, err := ioutil.ReadFile(pathname)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(content), "\x00"), nil
}

func pidExists(pid int) (bool, error) {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)

	exists, err := diskutils.FileExists(path)
	if err != nil {
		return false, err
	}
	if exists == false {
		return false, nil
	}

	return true, nil
}

func FindPid(commandNamePrefix string) (int, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
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
