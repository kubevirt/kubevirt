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
 * Copyright The KubeVirt Authors.
 *
 */

// Package pipe provides utilities for proxying domain notify connections
// between virt-launcher pods and virt-handler's notify server.
//
// The pipe acts as a bridge, creating a unix socket within the VMI's
// filesystem that proxies to virt-handler's domain-notify.sock.
package pipe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"time"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
)

func NewConnectToNotifyFunc(virtShareDir string) connectFunc {
	return func() (net.Conn, error) {
		conn, err := net.Dial("unix", filepath.Join(virtShareDir, "domain-notify.sock"))
		if err != nil {
			return nil, fmt.Errorf("error connecting to domain-notify.sock: %w", err)
		}
		return conn, err
	}
}

// InjectNotify injects the domain-notify.sock into the VMI pod and listens for connections
func InjectNotify(pod isolation.IsolationResult, virtShareDir string,
	nonRoot bool) (net.Listener, error) {
	root, err := pod.MountRoot()
	if err != nil {
		return nil, err
	}
	socketDir, err := root.AppendAndResolveWithRelativeRoot(virtShareDir)
	if err != nil {
		return nil, err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket for proxy service: %w", err)
	}

	if nonRoot {
		socketPath, err := safepath.JoinNoFollow(socketDir, "domain-notify-pipe.sock")
		if err != nil {
			return nil, err
		}

		err = diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			return nil, fmt.Errorf("unable to change ownership for domain notify: %w", err)
		}
	}

	return listener, nil
}

type connectFunc func() (net.Conn, error)
type proxyFunc func(net.Conn)

func Pipe(ctx context.Context, pipeChan chan net.Conn, proxy proxyFunc) {
	metrics.IncActivePipes()
	defer metrics.DecActivePipes()
	for {
		select {
		case <-ctx.Done():
			return
		case fd, open := <-pipeChan:
			if !open {
				return
			}
			go proxy(fd)
		}
	}
}

func ProxyWithMetric(proxy proxyFunc) proxyFunc {
	return func(c net.Conn) {
		metrics.IncPipeActiveProxies()
		defer metrics.DecPipeActiveProxies()
		proxy(c)
	}
}

// Proxy is blocking, it proxies pipe connection [pipeConn] to given connection [connect](ultimately to notify server)
// pipeConn is closed on success or error
func Proxy(logger *log.FilteredLogger, pipeConn net.Conn, connect connectFunc) {
	defer pipeConn.Close()

	// proxy the VMI domain-notify-pipe.sock to the virt-handler domain-notify.sock
	// so virt-handler receives notifications from the VMI
	notifyConn, err := connect()
	if err != nil {
		logger.Reason(err).Error("error connecting for proxy connection")
		return
	}
	defer notifyConn.Close()

	logger.Infof("Accepted new notify pipe connection")
	copyErr := make(chan error, 2)
	go func() {
		_, err := io.Copy(pipeConn, notifyConn)
		copyErr <- err
	}()
	go func() {
		_, err := io.Copy(notifyConn, pipeConn)
		copyErr <- err
	}()

	// wait until one of the copy routines exit then
	// let the connections close
	if err = <-copyErr; err != nil {
		logger.Reason(err).Infof("closing notify pipe connection")
	} else {
		logger.Infof("gracefully closed notify pipe connection")
	}
}

func ChanFromListener(ctx context.Context, logger *log.FilteredLogger, listener net.Listener) chan net.Conn {
	connectionChan := make(chan net.Conn, 100)
	// Close listener and exit when stop encountered
	go func() {
		<-ctx.Done()
		logger.Infof("closing notify pipe listener for vmi")
		if err := listener.Close(); err != nil {
			logger.Infof("failed closing notify pipe listener for vmi: %v", err)
		}
	}()

	// Listen for new connections,
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					// As Accept blocks, closing it is our mechanism to exit this loop
					return
				}
				logger.Reason(err).Error("Domain pipe accept error encountered.")
				// keep listening until stop invoked
				time.Sleep(1 * time.Second)
			} else {
				connectionChan <- conn
			}
		}
	}()
	return connectionChan
}
