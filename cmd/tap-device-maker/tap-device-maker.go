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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/golang/glog"
	"github.com/songgao/water"
)

func createTapDevice(name string, isMultiqueue bool) error {
	var err error = nil
	config := water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:    name,
			Persist: true,
			Permissions: &water.DevicePermissions{
				Owner: 107,
				Group: 107,
			},
			MultiQueue: isMultiqueue,
		},
	}

	_, err = water.New(config)
	return err
}

func configLogger() error {
	if err := flag.Set("component", "tap-maker"); err != nil {
		return fmt.Errorf("failed to config the 'component' flag on the logger: %v", err)
	}
	if err := flag.Set("logtostderr", "true"); err != nil {
		return fmt.Errorf("failed to config the 'logtostderr' flag on the logger: %v", err)
	}
	return nil
}

func main() {
	tapName := flag.String("tap-name", "tap0", "override the name of the tap device.")
	isMultiqueue := flag.Bool("multiqueue", false, "override the multi-queue flag of the tap device. Defaults to 'false'")

	if err := configLogger(); err != nil {
		os.Exit(1)
	}

	for fd := 3; fd < 256; fd++ { _ = syscall.Close(fd) }

	flag.Parse()
	glog.V(4).Info("Started app")
	if err := createTapDevice(*tapName, *isMultiqueue); err != nil {
		glog.Fatalf("error creating tap device: %v", err)
	}
	glog.V(2).Infof("Successfully created tap device: %s; isMultiqueue: %t", *tapName, *isMultiqueue)
	glog.Flush()
}
