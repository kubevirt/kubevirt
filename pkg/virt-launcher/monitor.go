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
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	watchdog "kubevirt.io/kubevirt/pkg/watchdog"
)

type monitor struct {
	timeout         time.Duration
	pid             int
	commandPrefix   string
	start           time.Time
	isDone          bool
	forwardedSignal os.Signal
}

type ProcessMonitor interface {
	RunForever(startTimeout time.Duration)
}

func InitializeSharedDirectories(baseDir string) error {
	return os.MkdirAll(watchdog.WatchdogFileDirectory(baseDir), 0755)
}

func NewProcessMonitor(commandPrefix string) ProcessMonitor {
	return &monitor{
		commandPrefix: commandPrefix,
	}
}

func getMyCgroupSlice() (string, error) {
	myPid := os.Getpid()

	_, mySlice, err := isolation.GetDefaultSlice(myPid)
	if err != nil {
		return "", err
	}

	return mySlice, nil
}

func matchPidCgroupSlice(pid int) (bool, error) {

	_, pidSlice, err := isolation.GetDefaultSlice(pid)
	if err != nil {
		return false, err
	}

	mySlice, err := getMyCgroupSlice()
	if err != nil {
		return false, err
	}

	if pidSlice != mySlice {
		return false, nil
	}

	return true, nil
}

func (mon *monitor) refresh() {
	if mon.isDone {
		logging.DefaultLogger().Error().Msg("Called refresh after done!")
		return
	}

	logging.DefaultLogger().Debug().Msgf("Refreshing. CommandPrefix %s pid %d", mon.commandPrefix, mon.pid)

	// is the process there?
	if mon.pid == 0 {
		var err error

		mon.pid, err = findPidInMyCgroup(mon.commandPrefix)
		if err != nil {

			logging.DefaultLogger().Info().Msgf("Still missing PID for %s, %v", mon.commandPrefix, err)
			// check to see if we've timed out looking for the process
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				logging.DefaultLogger().Info().Msgf("%s not found after timeout", mon.commandPrefix)
				mon.isDone = true
			}
			return
		}

		logging.DefaultLogger().Info().Msgf("Found PID for %s: %d", mon.commandPrefix, mon.pid)
	}

	exists, err := pidExistsInMyCgroup(mon.pid)
	if err != nil {
		logging.DefaultLogger().Reason(err).Error().Msgf("Error detecting pid (%d) status.", mon.pid)
		return
	}
	if exists == false {
		logging.DefaultLogger().Info().Msgf("Process %s and pid %d is gone!", mon.commandPrefix, mon.pid)
		mon.pid = 0
		mon.isDone = true
		return
	}

	return
}

func (mon *monitor) monitorLoop(startTimeout time.Duration, signalChan chan os.Signal) {
	// random value, no real rationale
	rate := 1 * time.Second

	timeoutRepr := fmt.Sprintf("%v", startTimeout)
	if startTimeout == 0 {
		timeoutRepr = "disabled"
	}
	logging.DefaultLogger().Info().Msgf("Monitoring loop: rate %v start timeout %s", rate, timeoutRepr)

	ticker := time.NewTicker(rate)

	mon.isDone = false
	mon.timeout = startTimeout
	mon.start = time.Now()

	for !mon.isDone {
		select {
		case <-ticker.C:
			mon.refresh()
		case s := <-signalChan:
			logging.DefaultLogger().Info().Msgf("Received signal %d.", s)
			if mon.pid != 0 {
				mon.forwardedSignal = s.(syscall.Signal)
				syscall.Kill(mon.pid, s.(syscall.Signal))
			}
		}
	}

	ticker.Stop()
	logging.DefaultLogger().Info().Msgf("Exiting...")
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

func pidExistsInMyCgroup(pid int) (bool, error) {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)

	exists, err := diskutils.FileExists(path)
	if err != nil {
		return false, err
	}
	if exists == false {
		return false, nil
	}

	cgroupMatch, err := matchPidCgroupSlice(pid)
	if err != nil {
		return false, err
	}
	if cgroupMatch == false {
		return false, nil
	}
	return true, nil
}

func findPidInMyCgroup(commandNamePrefix string) (int, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		argv, err := readProcCmdline(entry)
		if err != nil {
			return 0, err
		}

		match, _ := filepath.Match(fmt.Sprintf("%s*", commandNamePrefix), filepath.Base(argv[0]))
		// command prefix does not match
		if !match {
			continue
		}

		//   <empty> /    proc     /    $PID   /   cmdline
		// items[0] sep items[1] sep items[2] sep  items[3]
		items := strings.Split(entry, string(os.PathSeparator))
		pid, err := strconv.Atoi(items[2])
		if err != nil {
			return 0, err
		}

		cgroupsMatch, err := matchPidCgroupSlice(pid)
		if err != nil {
			return 0, err
		}

		if cgroupsMatch == false {
			continue
		}

		// everything matched, hooray!
		return pid, nil
	}

	return 0, fmt.Errorf("Process %s not found in /proc", commandNamePrefix)
}
