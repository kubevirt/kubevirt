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
	"context"
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nxadm/tail"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"kubevirt.io/client-go/log"
)

type TermFileError struct{}

func (m *TermFileError) Error() string {
	return "termFile got detected"
}

type VirtTail struct {
	ctx     context.Context
	logFile string
	g       *errgroup.Group
}

func (v *VirtTail) checkFile(socketFile string) bool {
	_, err := os.Stat(socketFile)
	return !os.IsNotExist(err)
}

func (v *VirtTail) tailLogs() error {
	t, err := tail.TailFile(v.logFile, tail.Config{
		Follow:        true,
		CompleteLines: true,
		MustExist:     false,
		ReOpen:        true,
		Logger:        tail.DiscardingLogger,
	})
	if err != nil {
		return err
	}
	defer func() {
		serr := t.Stop()
		if serr != nil {
			log.Log.V(3).Infof("tail error: %v", serr)
		}
		t.Cleanup()
	}()

	for {
		select {
		case line, ok := <-t.Lines:
			if !ok {
				log.Log.V(4).Info("tail error: line not ok")
			} else if line != nil {
				if line.Err != nil {
					log.Log.V(3).Infof("tail error: %v", line.Err)
				} else {
					fmt.Println(line.Text)
				}
			}
		case <-v.ctx.Done():
			return v.ctx.Err()
		}
	}
}

func (v *VirtTail) watchFS() error {
	socketFile := strings.TrimSuffix(v.logFile, "-log")
	termFile := v.logFile + "-sigTerm"
	termFileDone := termFile + "-done"
	socketExists := v.checkFile(socketFile)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Log.V(3).Infof("watcher error: %v", err)
		return err
	}
	defer watcher.Close()

	// Add a path.
	dirPath := filepath.Dir(v.logFile)
	found := false
	i := 0
	for i < 30 && !found {
		i = i + 1
		if _, derr := os.Stat(dirPath); derr == nil {
			found = true
			if err = watcher.Add(dirPath); err != nil {
				log.Log.V(3).Infof("watcher error: %v - %s", err, dirPath)
				return err
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !found {
		rerr := errors.New("expected directory is still not ready")
		log.Log.V(3).Infof("watchFS error: %v", rerr)
		return rerr
	}

	// initial timeout for serial console socket creation
	const initialSocketTimeout = time.Second * 20
	socketCheckCh := make(chan int)
	time.AfterFunc(initialSocketTimeout, func() {
		socketCheckCh <- 1
	})

	if v.checkFile(termFileDone) {
		log.Log.V(3).Infof("watchFS error: termFileDone was already there")
		return &TermFileError{}
	}

	// Start listening for events.
	for {
		select {
		case <-socketCheckCh:
			if !socketExists {
				if socketExists = v.checkFile(socketFile); !socketExists {
					rerr := errors.New("socketFile is still not ready")
					log.Log.V(3).Infof("watchFS error: %v", rerr)
					return rerr
				}
			}
			if v.checkFile(termFileDone) {
				log.Log.V(3).Infof("watchFS error: termFileDone was already there")
				return &TermFileError{}
			}
		case event := <-watcher.Events:
			if event.Has(fsnotify.Create) {
				if event.Name == socketFile {
					// socket file got created
					socketExists = true
				}
			} else if event.Has(fsnotify.Remove) {
				if event.Name == socketFile {
					// socket file got deleted, we should quickly terminate
					rerr := errors.New("socketFile got removed")
					log.Log.V(3).Infof("watchFS error: %v", rerr)
					return rerr
				} else if event.Name == termFile {
					// termination file got deleted, we should quickly terminate
					terr := &TermFileError{}
					log.Log.V(3).Infof("watchFS error: %v", terr)
					return terr
				}
			}
		case werr := <-watcher.Errors:
			log.Log.V(3).Infof("watcher error: %v", werr)
			return werr
		case <-v.ctx.Done():
			return v.ctx.Err()
		}
	}
}

func main() {
	// set new default verbosity, was set to 0 by glog
	goflag.Set("v", "2")
	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.CommandLine.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	logFile := pflag.String("logfile", "", "path of the logfile to be streamed")
	pflag.Parse()

	log.InitializeLogging("virt-tail")
	setTailLogverbosity()

	if logFile == nil || *logFile == "" {
		log.Log.V(3).Infof("logfile flags must be provided")
		os.Exit(1)
	}

	// Create context that listens for the interrupt signal from the container runtime.
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	g, gctx := errgroup.WithContext(ctx)

	v := &VirtTail{
		ctx:     gctx,
		logFile: *logFile,
		g:       g,
	}

	g.Go(v.tailLogs)
	g.Go(v.watchFS)

	// wait for all errgroup goroutines
	if err := g.Wait(); err != nil {
		// Exit cleanly on clean termination errors
		if !(errors.Is(err, context.Canceled) || errors.Is(err, &TermFileError{})) {
			log.Log.V(3).Infof("received error: %v", err)
			os.Exit(1)
		}
	}
}

func setTailLogverbosity() {
	// check if virt-launcher verbosity should be changed
	if verbosityStr, ok := os.LookupEnv("VIRT_LAUNCHER_LOG_VERBOSITY"); ok {
		if verbosity, err := strconv.Atoi(verbosityStr); err == nil {
			log.Log.SetVerbosityLevel(verbosity)
			log.Log.V(3).Infof("set log verbosity to %d", verbosity)
		} else {
			log.Log.Warningf("failed to set log verbosity. The value of logVerbosity label should be an integer, got %s instead.", verbosityStr)
		}
	}
}
