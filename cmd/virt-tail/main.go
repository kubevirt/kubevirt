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

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"
)

var errFileGone = errors.New("file removed or renamed")

// directoryPollInterval is how often to look for the per-VMI log directory.
// guest-console-log is a native sidecar and starts before compute creates it.
const directoryPollInterval = 100 * time.Millisecond

var printLine = func(s string) { fmt.Println(s) }

type VirtTail struct {
	ctx     context.Context
	logFile string
}

func (v *VirtTail) run() error {
	for {
		err := v.tailFile()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			// A missing path is normal at startup and whenever the file is
			// recreated. Exiting makes kubelet back this sidecar off, which
			// blocks every later container in the launcher pod.
			if errors.Is(err, errFileGone) || errors.Is(err, os.ErrNotExist) {
				if errors.Is(err, errFileGone) {
					log.Log.V(4).Infof("file gone, waiting for re-creation")
				} else {
					log.Log.V(4).Infof("log path %s does not exist yet, waiting", v.logFile)
				}
				select {
				case <-v.ctx.Done():
					return nil
				case <-time.After(directoryPollInterval):
				}
				continue
			}
			return err
		}
	}
}

func (v *VirtTail) waitForFile() (*os.File, error) {
	f, err := os.Open(v.logFile)
	if err == nil {
		return f, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	dir := filepath.Dir(v.logFile)
	// compute creates this directory after the sidecar has started. Wait for
	// it instead of failing watcher.Add with ENOENT.
	if err := v.waitForDirectory(dir); err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

	if err := watchDirectory(watcher, dir); err != nil {
		return nil, err
	}

	// Re-check after watch is established to avoid race
	f, err = os.Open(v.logFile)
	if err == nil {
		return f, nil
	}

	for {
		select {
		case <-v.ctx.Done():
			return nil, v.ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return nil, fmt.Errorf("watcher closed")
			}
			if event.Name == v.logFile && event.Has(fsnotify.Create) {
				f, err := os.Open(v.logFile)
				if err == nil {
					return f, nil
				}
				log.Log.V(3).Infof("file appeared but open failed: %v", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil, fmt.Errorf("watcher error channel closed")
			}
			log.Log.V(3).Infof("watcher error: %v", err)
		case <-time.After(5 * time.Second):
			// Polling fallback
			f, err = os.Open(v.logFile)
			if err == nil {
				return f, nil
			}
		}
	}
}

func (v *VirtTail) tailFile() error {
	f, err := v.waitForFile()
	if err != nil {
		return err
	}
	defer f.Close()

	dir := filepath.Dir(v.logFile)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

	if err := watchDirectory(watcher, dir); err != nil {
		return err
	}

	var lineBuf strings.Builder
	defer func() {
		if lineBuf.Len() > 0 {
			printLine(lineBuf.String())
		}
	}()
	const readBufSize = 32 * 1024 // 32 KiB read buffer
	buf := make([]byte, readBufSize)

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			processData(&lineBuf, buf[:n])
		}

		if readErr != nil && readErr != io.EOF {
			return fmt.Errorf("read error: %w", readErr)
		}

		if readErr == io.EOF {
			if err := v.waitForChange(watcher); err != nil {
				return err
			}
		}
	}
}

// waitForDirectory blocks until dir exists. The launcher pod runs this
// sidecar before compute creates /var/run/kubevirt-private/<vmi-uid>.
func (v *VirtTail) waitForDirectory(dir string) error {
	logged := false
	ticker := time.NewTicker(directoryPollInterval)
	defer ticker.Stop()

	for {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("log directory %s is not a directory", dir)
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if !logged {
			log.Log.V(3).Infof("waiting for directory %s to be created", dir)
			logged = true
		}

		select {
		case <-v.ctx.Done():
			return v.ctx.Err()
		case <-ticker.C:
		}
	}
}

// watchDirectory adds dir to watcher. A missing directory is reported as
// errFileGone so run retries instead of exiting.
func watchDirectory(watcher *fsnotify.Watcher, dir string) error {
	if err := watcher.Add(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errFileGone
		}
		return fmt.Errorf("watching directory %s: %w", dir, err)
	}
	return nil
}

func (v *VirtTail) waitForChange(watcher *fsnotify.Watcher) error {
	for {
		select {
		case <-v.ctx.Done():
			return v.ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed")
			}
			if event.Name != v.logFile {
				continue
			}
			if event.Has(fsnotify.Write) {
				return nil
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				return errFileGone
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			log.Log.V(3).Infof("watcher error: %v", err)
		case <-time.After(2 * time.Second):
			return nil
		}
	}
}

func processData(lineBuf *strings.Builder, data []byte) {
	for len(data) > 0 {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			lineBuf.Write(data)
			return
		}
		lineBuf.Write(data[:idx])
		printLine(lineBuf.String())
		lineBuf.Reset()
		data = data[idx+1:]
	}
}

func main() {
	pflag.CommandLine.AddGoFlag(log.VerbosityFlag())
	pflag.CommandLine.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	logFile := pflag.String("logfile", "", "path of the logfile to be streamed")
	pflag.Parse()

	log.InitializeLogging("virt-tail")
	setTailLogverbosity()

	if logFile == nil || *logFile == "" {
		log.Log.V(3).Infof("logfile flags must be provided")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	v := &VirtTail{
		ctx:     ctx,
		logFile: *logFile,
	}

	if err := v.run(); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Log.V(3).Infof("received error: %v", err)
			os.Exit(1)
		}
	}
}

func setTailLogverbosity() {
	if verbosityStr, ok := os.LookupEnv("VIRT_LAUNCHER_LOG_VERBOSITY"); ok {
		if verbosity, err := strconv.Atoi(verbosityStr); err == nil {
			log.Log.SetVerbosityLevel(verbosity)
			log.Log.V(3).Infof("set log verbosity to %d", verbosity)
		} else {
			log.Log.Warningf("failed to set log verbosity. The value of logVerbosity label should be an integer, got %s instead.", verbosityStr)
		}
	}
}
