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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

type monitor struct {
	timeout         time.Duration
	pid             int
	pidFile         string
	start           time.Time
	isDone          bool
	forwardedSignal os.Signal
	debugMode       bool
}

type ProcessMonitor interface {
	RunForever(startTimeout time.Duration)
}

func InitializeSharedDirectories(baseDir string) error {
	err := os.MkdirAll(baseDir+"/qemu-pids", 0755)
	if err != nil {
		return err
	}

	return os.MkdirAll(baseDir+"/watchdog-files", 0755)
}

func WatchdogFileFromNamespaceName(baseDir string, namespace string, name string) string {
	return filepath.Clean(baseDir) + "/watchdog-files/" + namespace + "_" + name
}

func QemuPidfileFromNamespaceName(baseDir string, namespace string, name string) string {
	return filepath.Clean(baseDir) + "/qemu-pids/" + namespace + "_" + name
}

func NewProcessMonitor(pidFile string, debugMode bool) ProcessMonitor {
	return &monitor{
		pidFile:   pidFile,
		debugMode: debugMode,
	}
}

func (mon *monitor) refresh() {
	if mon.isDone {
		log.Print("Called refresh after done!")
		return
	}

	if mon.debugMode {
		log.Printf("Refreshing pidFIle %s pid %d", mon.pidFile, mon.pid)
	}

	// is the process there?
	if mon.pid == 0 {
		var err error
		mon.pid, err = getPidFromFile(mon.pidFile)
		if err == nil {
			log.Printf("Found PID for %s: %d", mon.pidFile, mon.pid)
		} else {
			log.Printf("Still missing PID for %s, %v", mon.pidFile, err)
			// if the proces is not there yet, is it too late?
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				log.Printf("%s not found after timeout", mon.pidFile)
				mon.isDone = true
			}
		}
		return
	}

	// is the process gone? mon.pid != 0 -> mon.pid == 0
	// note libvirt deliver one event for this, but since we need
	// to poll procfs anyway to detect incoming QEMUs after migrations,
	// we choose to not use this. Bonus: we can close the connection
	// and open it only when needed, which is a tiny part of the
	// virt-launcher lifetime.
	if !pidExists(mon.pid) {
		log.Printf("Process with pidfile %s and pid %d is gone!", mon.pidFile, mon.pid)
		mon.pid = 0
		mon.isDone = true
		return
	}

	return
}

func (mon *monitor) monitorLoop(startTimeout time.Duration, signalChan chan os.Signal) {
	// random value, no real rationale
	rate := 500 * time.Millisecond

	if mon.debugMode {
		timeoutRepr := fmt.Sprintf("%v", startTimeout)
		if startTimeout == 0 {
			timeoutRepr = "disabled"
		}
		log.Printf("Monitoring loop: rate %v start timeout %s", rate, timeoutRepr)
	}

	ticker := time.NewTicker(rate)

	mon.isDone = false
	mon.timeout = startTimeout
	mon.start = time.Now()

	log.Printf("Waiting forever...")
	for !mon.isDone {
		select {
		case <-ticker.C:
			mon.refresh()
		case s := <-signalChan:
			log.Print("Got signal: ", s)
			if mon.pid != 0 {
				mon.forwardedSignal = s.(syscall.Signal)
				syscall.Kill(mon.pid, s.(syscall.Signal))
			}
		}
	}

	ticker.Stop()
	log.Printf("Exiting...")
}

func (mon *monitor) RunForever(startTimeout time.Duration) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	mon.monitorLoop(startTimeout, c)
}

func readProcCmdline(pathname string) ([]string, error) {
	content, err := ioutil.ReadFile(pathname)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(content), "\x00"), nil
}

func getPidFromFile(pidFile string) (int, error) {
	exists, err := diskutils.FileExists(pidFile)

	if err != nil {
		return 0, err
	}

	if exists == false {
		return 0, fmt.Errorf("pid file at path %s not found", pidFile)
	}

	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		return 0, err
	}

	if !pidExists(pid) {
		return 0, fmt.Errorf("stale pid detected at pidFile %s", pidFile)
	}

	return pid, nil
}

func pidExists(pid int) bool {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
