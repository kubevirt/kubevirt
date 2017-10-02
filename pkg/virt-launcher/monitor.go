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
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type monitor struct {
	timeout   time.Duration
	pid       int
	exename   string
	start     time.Time
	isDone    bool
	debugMode bool
}

type ProcessMonitor interface {
	RunForever(startTimeout time.Duration)
}

func NewProcessMonitor(execname string, debugMode bool) ProcessMonitor {
	return &monitor{
		exename:   execname,
		debugMode: debugMode,
	}
}

func (mon *monitor) refresh() {
	if mon.isDone {
		log.Print("Called refresh after done!")
		return
	}

	if mon.debugMode {
		log.Printf("Refreshing executable %s pid %d", mon.exename, mon.pid)
	}

	// is the process there?
	if mon.pid == 0 {
		var err error
		mon.pid, err = pidOf(mon.exename)
		if err == nil {
			log.Printf("Found PID for %s: %d", mon.exename, mon.pid)
		} else {
			if mon.debugMode {
				log.Printf("Missing PID for %s", mon.exename)
			}
			// if the proces is not there yet, is it too late?
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				log.Printf("%s not found after timeout", mon.exename)
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
		log.Printf("Process %s is gone!", mon.exename)
		mon.pid = 0
		mon.isDone = true
		return
	}

	return
}

func (mon *monitor) RunForever(startTimeout time.Duration) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

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

	gotSignal := false
	mon.isDone = false
	mon.timeout = startTimeout
	mon.start = time.Now()

	log.Printf("Waiting forever...")
	for !gotSignal && !mon.isDone {
		select {
		case <-ticker.C:
			mon.refresh()
		case s := <-c:
			log.Print("Got signal: ", s)
			gotSignal = true
			if mon.pid != 0 {
				// forward the signal to the VM process
				// TODO allow a delay here to support graceful shutdown from virt-handler side
				syscall.Kill(mon.pid, s.(syscall.Signal))
			}
		}
	}

	ticker.Stop()
	log.Printf("Exiting...")
}

func readProcCmdline(pathname string) ([]string, error) {
	content, err := ioutil.ReadFile(pathname)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(content), "\x00"), nil
}

func pidOf(exename string) (int, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		argv, err := readProcCmdline(entry)
		if err != nil {
			return 0, err
		}

		// we need to support both
		// - /usr/bin/qemu-system-$ARCH (fedora)
		// - /usr/libexec/qemu-kvm (*EL, CentOS)
		match, _ := filepath.Match(fmt.Sprintf("%s*", exename), filepath.Base(argv[0]))

		if match {
			//   <empty> /    proc     /    $PID   /   cmdline
			// items[0] sep items[1] sep items[2] sep  items[3]
			items := strings.Split(entry, string(os.PathSeparator))
			pid, err := strconv.Atoi(items[2])
			if err != nil {
				return 0, err
			}

			return pid, nil
		}
	}
	return 0, fmt.Errorf("Process %s not found in /proc", exename)
}

func pidExists(pid int) bool {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
