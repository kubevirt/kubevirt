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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"kubevirt.io/client-go/log"

	kutil "kubevirt.io/kubevirt/pkg/util"
)

const (
	NBDKitExec  = "/usr/sbin/nbdkit"
	NBDFuseExec = "/usr/bin/nbdfuse"
)

type NBDHelper struct {
	guestMemory string
	IsRunning   bool
	stopChan    chan struct{}
}

func NewNBDHelper(guestMemory string) *NBDHelper {
	return &NBDHelper{
		guestMemory: guestMemory,
		stopChan:    make(chan struct{}),
	}
}

func (l *NBDHelper) AllocateMemoryWithNBDFuse() {
	l.startNBDKit()
	l.runNBDFuse()
}

func (l *NBDHelper) startNBDKit() {
	allocator := "allocator=zstd"
	args := []string{"mmeory", l.guestMemory, allocator}
	l.runNBDCommand(NBDKitExec, args)
}

func (l *NBDHelper) runNBDFuse() {
	memFile := fmt.Sprintf("%s/pc.mem", kutil.FileMemoryBackingPath)
	file, err := os.Create(memFile)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create memory file: %s", memFile)
		panic(err)
	}
	defer file.Close()
	args := []string{memFile, "nbd://localhost"}
	l.runNBDCommand(NBDFuseExec, args)
}

func (l *NBDHelper) setNotRunning() {
	l.IsRunning = false
	return
}

func (l *NBDHelper) Close() {
	close(l.stopChan)
}

func (l *NBDHelper) runNBDCommand(execCommand string, args []string) {
	go func() {
		defer l.setNotRunning()
		for {
			exitChan := make(chan struct{})
			cmd := exec.Command(execCommand, args...)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE},
			}

			// connect process's stderr to our own stdout in order to see the logs in the container logs
			reader, err := cmd.StderrPipe()
			if err != nil {
				log.Log.Reason(err).Errorf("failed to start %s", execCommand)
				panic(err)
			}

			baseName := filepath.Base(execCommand)
			go func() {
				scanner := bufio.NewScanner(reader)
				scanner.Buffer(make([]byte, 1024), 512*1024)
				for scanner.Scan() {
					log.Log.Infof("%s::%s", baseName, scanner.Text())
				}

				if err := scanner.Err(); err != nil {
					log.Log.Reason(err).Errorf("failed to read %s logs", baseName)
				}
			}()

			err = cmd.Start()
			if err != nil {
				log.Log.Reason(err).Errorf("failed to start %s", baseName)
				panic(err)
			}

			go func() {
				defer close(exitChan)
				cmd.Wait()
			}()

			select {
			case <-l.stopChan:
				cmd.Process.Kill()
				return
			case <-exitChan:
				log.Log.Errorf("%s exited, restarting", baseName)
			}

			// this sleep is to avoid consuming all resources in the
			// event of a process crash loop.
			time.Sleep(time.Second)
		}
	}()
}
