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
	"syscall"
	"time"

	"kubevirt.io/client-go/log"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

type OnShutdownCallback func(pid int)
type OnGracefulShutdownCallback func()

type monitor struct {
	timeout                  time.Duration
	pid                      int
	cmdlineMatchStr          string
	start                    time.Time
	isDone                   bool
	gracePeriod              int
	gracePeriodStartTime     int64
	finalShutdownCallback    OnShutdownCallback
	gracefulShutdownCallback OnGracefulShutdownCallback
}

type ProcessMonitor interface {
	RunForever(startTimeout time.Duration, signalStopChan chan struct{})
}

func InitializePrivateDirectories(baseDir string) error {
	if err := util.MkdirAllWithNosec(baseDir); err != nil {
		return err
	}
	if err := diskutils.DefaultOwnershipManager.SetFileOwnership(baseDir); err != nil {
		return err
	}
	return nil
}

func InitializeDisksDirectories(baseDir string) error {
	err := os.MkdirAll(baseDir, 0750)
	if err != nil {
		return err
	}

	// #nosec G302: Poor file permissions used with chmod. Using the safe permission setting for a directory.
	err = os.Chmod(baseDir, 0750)
	if err != nil {
		return err
	}
	err = diskutils.DefaultOwnershipManager.SetFileOwnership(baseDir)
	if err != nil {
		return err
	}
	return nil
}

func NewProcessMonitor(cmdlineMatchStr string,
	gracePeriod int,
	finalShutdownCallback OnShutdownCallback,
	gracefulShutdownCallback OnGracefulShutdownCallback) ProcessMonitor {
	return &monitor{
		cmdlineMatchStr:          cmdlineMatchStr,
		gracePeriod:              gracePeriod,
		finalShutdownCallback:    finalShutdownCallback,
		gracefulShutdownCallback: gracefulShutdownCallback,
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
			} else if mon.gracePeriodStartTime != 0 {
				log.Log.Infof("%s not found after shutdown initiated", mon.cmdlineMatchStr)
				mon.isDone = true
			}
			return
		}

		log.Log.Infof("Found PID for %s: %d", mon.cmdlineMatchStr, mon.pid)
	}

	exists, isZombie, err := pidExists(mon.pid)
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

	if isZombie {
		log.Log.Infof("Process %s and pid %d is a zombie, sending SIGCHLD to pid 1 to reap process", mon.cmdlineMatchStr, mon.pid)
		syscall.Kill(1, syscall.SIGCHLD)
		mon.pid = 0
		mon.isDone = true
	}

	if expired {
		log.Log.Infof("Grace Period expired, shutting down.")
		mon.finalShutdownCallback(mon.pid)
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

			mon.gracefulShutdownCallback()
			mon.gracePeriodStartTime = time.Now().UTC().Unix()
		}
	}

	ticker.Stop()
}

func (mon *monitor) RunForever(startTimeout time.Duration, signalStopChan chan struct{}) {
	mon.monitorLoop(startTimeout, signalStopChan)
}

func pidExists(pid int) (exists bool, isZombie bool, err error) {

	pathCmdline := fmt.Sprintf("/proc/%d/cmdline", pid)
	pathStatus := fmt.Sprintf("/proc/%d/status", pid)

	exists, err = diskutils.FileExists(pathCmdline)
	if err != nil {
		return false, false, err
	}
	if exists == false {
		return false, false, nil
	}

	dataBytes, err := ioutil.ReadFile(pathStatus)
	if err != nil {
		return false, false, err
	}

	if strings.Contains(string(dataBytes), "Z (zombie)") {
		isZombie = true
	}

	return exists, isZombie, nil
}

func FindPid(commandNamePrefix string) (int, error) {
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
