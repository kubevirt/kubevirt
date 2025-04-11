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
	"strconv"
	"syscall"
	"time"

	"github.com/nxadm/tail"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"kubevirt.io/client-go/log"
)

type VirtTail struct {
	ctx     context.Context
	logFile string
}

func (v *VirtTail) tailLogsWrapper() error {
	location := &tail.SeekInfo{Offset: 0}
	for {
		var err error
		location, err = v.tailLogs(*location)
		if err != nil {
			return err
		}
		if location == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func (v *VirtTail) tailLogs(location tail.SeekInfo) (*tail.SeekInfo, error) {
	t, err := tail.TailFile(v.logFile, tail.Config{
		Location:      &location,
		Follow:        true,
		CompleteLines: true,
		MustExist:     false,
		ReOpen:        true,
		Logger:        tail.DiscardingLogger,
	})
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		serr := t.Stop()
		if serr != nil {
			log.Log.V(3).Infof("tail error: %v", serr)
		}
		if cleanup {
			t.Cleanup()
		}
	}()

	for {
		select {
		case line, ok := <-t.Lines:
			if !ok {
				log.Log.V(4).Info("tail error: chan closed")
				cleanup = false
				return &location, nil
			} else if line != nil {
				location = line.SeekInfo
				if line.Err != nil {
					log.Log.V(3).Infof("tail error: %v", line.Err)
				} else {
					fmt.Println(line.Text)
				}
			}
		case <-v.ctx.Done():
			return nil, v.ctx.Err()
		}
	}
}

func main() {
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)

	v := &VirtTail{
		ctx:     gctx,
		logFile: *logFile,
	}

	g.Go(v.tailLogsWrapper)

	// wait for all errgroup goroutines
	if err := g.Wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Log.V(3).Infof("received error: %v", err)
			os.Exit(1)
		}
		// Exit cleanly on clean termination errors
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
