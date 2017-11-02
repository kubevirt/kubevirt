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

package console

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"sync"

	"github.com/gorilla/websocket"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Console struct {
}

func (c *Console) FlagSet() *flag.FlagSet {
	cf := flag.NewFlagSet("console", flag.ExitOnError)
	cf.StringP("device", "d", "", "Console to connect to")

	return cf
}

func (c *Console) Usage() string {
	usage := "Connect to a serial console on a VM:\n\n"
	usage += "Examples:\n"
	usage += "# Connect to the console 'serial0' on the VM 'myvm':\n"
	usage += "virtctl console myvm --device serial0\n\n"
	usage += "Options:\n"
	usage += c.FlagSet().FlagUsages()
	return usage
}

func (c *Console) Run(flags *flag.FlagSet) int {

	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	namespace, _ := flags.GetString("namespace")
	device, _ := flags.GetString("device")
	if namespace == "" {
		namespace = v1.NamespaceDefault
	}
	if len(flags.Args()) != 2 {
		log.Println("VM name is missing")
		return 1
	}
	vm := flags.Arg(1)

	config, err := clientcmd.BuildConfigFromFlags(server, kubeconfig)
	if err != nil {
		log.Println(err)
		return 1
	}

	err = ConnectToConsole(config, namespace, vm, device, TerminalWebsocketCallback)
	if err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

func ConnectToConsole(config *rest.Config, namespace string, name string, console string, callback RoundTripCallback) error {

	// Create a round tripper with all necessary kubernetes security details
	wrappedRoundTripper, err := RoundTripperFromConfig(config, callback)
	if err != nil {
		return err
	}

	// Create the basic console request
	req, err := RequestFromConfig(config, name, namespace, console)
	if err != nil {
		return err
	}

	// Do the call and process the websocket connection with the callback
	_, err = wrappedRoundTripper.RoundTrip(req)

	if err != nil {
		return err
	}
	return nil
}

func NewWebsocketCallback(in io.ReadCloser, out io.WriteCloser, stopChan chan struct{}) RoundTripCallback {

	return func(ws *websocket.Conn, resp *http.Response, err error) error {

		if err != nil {
			if resp != nil && resp.StatusCode != http.StatusOK {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				return fmt.Errorf("Can't connect to console (%d): %s\n", resp.StatusCode, buf.String())
			}
			return fmt.Errorf("Can't connect to console: %s\n", err.Error())
		}

		writeStop := make(chan struct{})
		readStop := make(chan struct{})

		go func() {
			defer close(readStop)
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					out.Write(message)
					return
				}
				_, err = out.Write(message)
				if err == io.EOF {
					return
				}
			}
		}()

		buf := make([]byte, 1024, 1024)

		// Synchronize writes for final close announcements
		var writeMux sync.Mutex
		writeProtected := func(messageType int, data []byte) error {
			writeMux.Lock()
			defer writeMux.Unlock()
			return ws.WriteMessage(websocket.TextMessage, data)
		}

		go func() {
			defer close(writeStop)
			for {
				n, err := in.Read(buf)
				if err != nil && err != io.EOF {
					log.Println(err)
					return
				}

				// TODO move this to the TerminalWebsocketCallback
				if buf[0] == 29 {
					return
				}

				// If there is nothing more to write and we have reached EOF, return
				if n == 0 && err == io.EOF {
					return
				}

				err = writeProtected(websocket.TextMessage, buf[0:n])
				if err != nil && err != io.EOF {
					log.Println(err)
					return
				}
			}
		}()

		select {
		case <-stopChan:
		case <-readStop:
		case <-writeStop:
		}

		err = writeProtected(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			return fmt.Errorf("Error on close announcement: %s", err.Error())
		}
		select {
		case <-readStop:
		case <-time.After(time.Second):
		}
		return nil
	}
}

func TerminalWebsocketCallback(ws *websocket.Conn, resp *http.Response, connErr error) error {

	stopChan := make(chan struct{}, 1)

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(stopChan)
	}()

	// If there is no obvious connection error, set up the terminal
	if connErr == nil {
		state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("Make raw terminal failed: %s", err)
		}
		defer terminal.Restore(int(os.Stdin.Fd()), state)
		fmt.Fprint(os.Stderr, "Escape sequence is ^]")
	}

	return NewWebsocketCallback(os.Stdin, os.Stdout, stopChan)(ws, resp, connErr)
}

func RequestFromConfig(config *rest.Config, vm string, namespace string, device string) (*http.Request, error) {

	u, err := url.Parse(config.Host)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, fmt.Errorf("Unsupported Protocol %s", u.Scheme)
	}

	u.Path = fmt.Sprintf("/apis/kubevirt.io/v1alpha1/namespaces/%s/virtualmachines/%s/console", namespace, vm)
	if device != "" {
		u.RawQuery = "console=" + device
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}

	return req, nil
}

func RoundTripperFromConfig(config *rest.Config, callback RoundTripCallback) (http.RoundTripper, error) {

	// Configure TLS
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}

	// Configure the websocket dialer
	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}

	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	rt := &WebsocketRoundTripper{
		Do:     callback,
		Dialer: dialer,
	}

	// Make sure we inherit all relevant security headers
	return rest.HTTPWrappersForConfig(config, rt)
}

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer conn.Close()
	}
	return resp, d.Do(conn, resp, err)
}
