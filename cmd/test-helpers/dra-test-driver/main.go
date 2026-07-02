/*
 * Copyright 2023 The Kubernetes Authors.
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
 */

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"

	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/profiles"
	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/profiles/network"
)

const (
	DriverPluginCheckpointFile = "checkpoint.json"
	driverName                 = "test.kubevirt.io"
)

type Config struct {
	nodeName   string
	driverName string

	coreclient    coreclientset.Interface
	cancelMainCtx func(error)
	profile       profiles.Profile
}

const (
	cdiRoot                     = "/etc/cdi"
	numDevices                  = 10
	kubeletPluginsDirectoryPath = kubeletplugin.KubeletPluginsDir
)

func (c Config) DriverPluginPath() string {
	return filepath.Join(kubeletPluginsDirectoryPath, c.driverName)
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		fmt.Fprintf(os.Stderr, "Error: NODE_NAME environment variable is required\n")
		os.Exit(1)
	}

	config := &Config{
		nodeName:   nodeName,
		driverName: driverName,
	}

	ctx := context.Background()

	var err error
	config.coreclient, err = newKubeClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	config.profile = network.NewProfile(config.nodeName, numDevices)

	if err = RunPlugin(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newKubeClient() (coreclientset.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("create in-cluster config: %w", err)
	}

	client, err := coreclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	return client, nil
}

func RunPlugin(ctx context.Context, config *Config) error {
	logger := klog.FromContext(ctx)

	err := os.MkdirAll(config.DriverPluginPath(), 0750)
	if err != nil {
		return err
	}

	info, err := os.Stat(cdiRoot)
	switch {
	case err != nil && os.IsNotExist(err):
		err := os.MkdirAll(cdiRoot, 0750)
		if err != nil {
			return err
		}
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("path for cdi file generation is not a directory: '%v'", err)
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	ctx, cancel := context.WithCancelCause(ctx)
	config.cancelMainCtx = cancel

	driver, err := NewDriver(ctx, config)
	if err != nil {
		return err
	}

	<-ctx.Done()
	stop()
	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error(err, "error from context")
	}

	err = driver.Shutdown(logger)
	if err != nil {
		logger.Error(err, "Unable to cleanly shutdown driver")
	}

	return nil
}
