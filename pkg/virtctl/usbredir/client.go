//go:build !s390x

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
 * Copyright the KubeVirt Authors.
 *
 */

package usbredir

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"time"

	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"
)

type ClientConnectFn func(ctx context.Context, device, address string) error

type Client struct {
	// To connect local USB device buffer to the remote VM using the websocket.
	inputReader  *io.PipeReader
	inputWriter  *io.PipeWriter
	outputReader *io.PipeReader
	outputWriter *io.PipeWriter

	listener *net.TCPListener

	// channels
	done   chan struct{}
	stream chan error
	local  chan error
	remote chan error

	ctx context.Context

	LaunchClient  bool
	ClientConnect ClientConnectFn
}

func NewUSBRedirClient(ctx context.Context, address string, stream kvcorev1.StreamInterface) (*Client, error) {
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	k := &Client{
		ctx:           ctx,
		inputReader:   inReader,
		inputWriter:   inWriter,
		outputReader:  outReader,
		outputWriter:  outWriter,
		ClientConnect: clientConnect,
		LaunchClient:  true,
	}

	// Create local TCP server for usbredir client to connect
	if err := k.withLocalTCPClient(address); err != nil {
		return nil, err
	}

	// Start stream with remote usbredir endpoint
	k.withRemoteVMIStream(stream)

	// Connects data from local usbredir data to remote usbredir endpoint
	k.proxyUSBRedir()

	return k, nil
}

// The address for usbredir client to connect
func (k *Client) GetProxyAddress() string {
	if k.listener == nil {
		log.Log.Warning("Calling GetProxyAddress without a functioning Listener")
		return ""
	}
	return k.listener.Addr().String()
}

func (k *Client) withRemoteVMIStream(usbredirStream kvcorev1.StreamInterface) {
	k.stream = make(chan error)

	go func() {
		defer k.outputWriter.Close()
		select {
		case k.stream <- usbredirStream.Stream(
			kvcorev1.StreamOptions{
				In:  k.inputReader,
				Out: k.outputWriter,
			},
		):
		case <-k.ctx.Done():
		}
	}()
}

func (k *Client) withLocalTCPClient(address string) error {
	lnAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return fmt.Errorf("Can't resolve the address: %s", err.Error())
	}

	// The local tcp server is used to proxy between remote websocket and local USB
	k.listener, err = net.ListenTCP("tcp", lnAddr)
	if err != nil {
		return fmt.Errorf("Can't listen: %s", err.Error())
	}

	return nil
}

func (k *Client) proxyUSBRedir() {
	// forward data to/from websocket after usbredir client connects.
	k.done = make(chan struct{}, 1)
	k.remote = make(chan error)
	go func() {
		defer k.inputWriter.Close()
		start := time.Now()

		usbredirConn, err := k.listener.Accept()
		if err != nil {
			log.Log.V(2).Infof("Failed to accept connection: %s", err.Error())
			k.remote <- err
			return
		}
		defer usbredirConn.Close()

		log.Log.V(2).Infof("Connected to %s at %v", usbredirClient, time.Now().Sub(start))

		stream := make(chan error)
		// write to local usbredir from pipeOutReader
		go func() {
			_, err := io.Copy(usbredirConn, k.outputReader)
			stream <- err
		}()

		// read from local usbredir towards pipeInWriter
		go func() {
			_, err := io.Copy(k.inputWriter, usbredirConn)
			stream <- err
		}()

		select {
		case <-k.done: // Wait for local usbredir to complete
		case err = <-stream: // Wait for remote connection to close
			if err == nil {
				// Remote connection closed, report this as error
				err = fmt.Errorf("Remote connection has closed.")
			}
			// Wait for local usbredir to complete
			k.remote <- err
		case <-k.ctx.Done():
		}
	}()
}

func clientConnect(ctx context.Context, device, address string) error {
	bin := usbredirClient
	args := []string{}
	args = append(args, "--device", device, "--to", address)

	log.Log.Infof("port_arg: '%s'", address)
	log.Log.Infof("args: '%v'", args)
	log.Log.Infof("Executing commandline: '%s %v'", bin, args)

	command := exec.CommandContext(ctx, bin, args...)
	output, err := command.CombinedOutput()
	log.Log.V(2).Infof("%v output: %v", bin, string(output))

	return err
}

func (k *Client) Redirect(device string) error {
	// execute local usbredir binary
	address := k.GetProxyAddress()
	k.local = make(chan error)
	if k.LaunchClient {
		go func() {
			defer close(k.done)
			k.local <- k.ClientConnect(k.ctx, device, address)
		}()
	}

	var err error
	select {
	case err = <-k.stream:
	case err = <-k.local:
	case err = <-k.remote:
	case <-k.ctx.Done():
		err = k.ctx.Err()

	}
	return err
}
