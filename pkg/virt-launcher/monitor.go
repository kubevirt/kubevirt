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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

type OnShutdownCallback func(pid int)
type OnGracefulShutdownCallback func()

type monitor struct {
	timeout                  time.Duration
	pid                      int
	pidDir                   string
	domainName               string
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
	if err := diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(baseDir); err != nil {
		return err
	}
	return nil
}

func InitializeConsoleLogFile(baseDir string) error {
	logPath := filepath.Join(baseDir, "virt-serial0-log")

	_, err := os.Stat(logPath)
	if errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(logPath)
		if err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err = diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(logPath); err != nil {
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
	err = diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(baseDir)
	if err != nil {
		return err
	}
	return nil
}

func NewProcessMonitor(domainName string,
	pidDir string,
	gracePeriod int,
	finalShutdownCallback OnShutdownCallback,
	gracefulShutdownCallback OnGracefulShutdownCallback) ProcessMonitor {
	return &monitor{
		domainName:               domainName,
		pidDir:                   pidDir,
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
	} else if cmdserver.ReceivedEarlyExitSignal() {
		log.Log.Infof("received early exit signal - stop waiting for %s", mon.domainName)
		mon.isDone = true
		return
	}

	log.Log.V(4).Infof("Refreshing. domainName %s pid %d", mon.domainName, mon.pid)

	expired := mon.isGracePeriodExpired()

	// is the process there?
	if mon.pid == 0 {
		var err error

		mon.pid, err = FindPid(mon.domainName, mon.pidDir)
		if err != nil {

			log.Log.Infof("Still missing PID for %s, %v", mon.domainName, err)
			// check to see if we've timed out looking for the process
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				log.Log.Infof("%s not found after timeout", mon.domainName)
				mon.isDone = true
			} else if expired {
				log.Log.Infof("%s not found after grace period expired", mon.domainName)
				mon.isDone = true
			} else if mon.gracePeriodStartTime != 0 {
				log.Log.Infof("%s not found after shutdown initiated", mon.domainName)
				mon.isDone = true
			}
			return
		}

		log.Log.Infof("Found PID for %s: %d", mon.domainName, mon.pid)
	}

	exists, isZombie, err := pidExists(mon.pid)
	if err != nil {
		log.Log.Reason(err).Errorf("Error detecting pid (%d) status.", mon.pid)
		return
	}
	if exists == false {
		log.Log.Infof("Process %s and pid %d is gone!", mon.domainName, mon.pid)
		mon.pid = 0
		mon.isDone = true
		return
	}

	if isZombie {
		log.Log.Infof("Process %s and pid %d is a zombie, sending SIGCHLD to pid 1 to reap process", mon.domainName, mon.pid)
		syscall.Kill(1, syscall.SIGCHLD)
		mon.pid = 0
		mon.isDone = true
		return
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
	defer ticker.Stop()
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

	dataBytes, err := os.ReadFile(pathStatus)
	if err != nil {
		return false, false, err
	}

	if strings.Contains(string(dataBytes), "Z (zombie)") {
		isZombie = true
	}

	return exists, isZombie, nil
}

func FindPid(domainName string, pidDir string) (int, error) {
	content, err := os.ReadFile(filepath.Join(pidDir, domainName+".pid"))
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(content))
}
