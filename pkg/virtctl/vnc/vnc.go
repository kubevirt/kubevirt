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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package vnc

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	LISTEN_TIMEOUT = 60 * time.Second
	FLAG           = "vnc"

	//#### Tiger VNC ####
	//# https://github.com/TigerVNC/tigervnc/releases
	// Compatible with multiple Tiger VNC versions
	TIGER_VNC_PATTERN = `/Applications/TigerVNC Viewer*.app/Contents/MacOS/TigerVNC Viewer`

	//#### Chicken VNC ####
	//# https://sourceforge.net/projects/chicken/
	CHICKEN_VNC = "/Applications/Chicken.app/Contents/MacOS/Chicken"

	//####  Real VNC ####
	//# https://www.realvnc.com/en/connect/download/viewer/macos/
	REAL_VNC = "/Applications/VNC Viewer.app/Contents/MacOS/vncviewer"

	REMOTE_VIEWER = "remote-viewer"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vnc (VMI)",
		Short:   "Open a vnc connection to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := VNC{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type VNC struct {
	clientConfig clientcmd.ClientConfig
}

func (o *VNC) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	vmi := args[0]

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return err
	}

	// setup connection with VM
	vnc, err := virtCli.VirtualMachineInstance(namespace).VNC(vmi)
	if err != nil {
		return fmt.Errorf("Can't access VMI %s: %s", vmi, err.Error())
	}

	lnAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("Can't resolve the address: %s", err.Error())
	}

	// The local tcp server is used to proxy the podExec websock connection to remote-viewer
	ln, err := net.ListenTCP("tcp", lnAddr)
	if err != nil {
		return fmt.Errorf("Can't listen on unix socket: %s", err.Error())
	}
	// End of pre-flight checks. Everything looks good, we can start
	// the goroutines and let the data flow

	//                                       -> pipeInWriter  -> pipeInReader
	// remote-viewer -> unix sock connection
	//                                       <- pipeOutReader <- pipeOutWriter
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()

	k8ResChan := make(chan error)
	listenResChan := make(chan error)
	viewResChan := make(chan error)
	stopChan := make(chan struct{}, 1)
	doneChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)

	go func() {
		// transfer data from/to the VM
		k8ResChan <- vnc.Stream(kubecli.StreamOptions{
			In:  pipeInReader,
			Out: pipeOutWriter,
		})
	}()

	// wait for remote-viewer to connect to our local proxy server
	go func() {
		start := time.Now()
		glog.Infof("connection timeout: %v", LISTEN_TIMEOUT)
		// exit early if spawning remote-viewer fails
		ln.SetDeadline(time.Now().Add(LISTEN_TIMEOUT))

		fd, err := ln.Accept()
		if err != nil {
			glog.V(2).Infof("Failed to accept unix sock connection. %s", err.Error())
			listenResChan <- err
		}
		defer fd.Close()

		glog.V(2).Infof("remote-viewer connected in %v", time.Now().Sub(start))

		// write to FD <- pipeOutReader
		go func() {
			_, err := io.Copy(fd, pipeOutReader)
			readStop <- err
		}()

		// read from FD -> pipeInWriter
		go func() {
			_, err := io.Copy(pipeInWriter, fd)
			writeStop <- err
		}()

		// don't terminate until remote-viewer is done
		<-doneChan
		listenResChan <- err
	}()

	// execute VNC
	go func() {
		defer close(doneChan)
		port := ln.Addr().(*net.TCPAddr).Port
		args := []string{}

		vncBin := ""
		osType := runtime.GOOS
		switch osType {
		case "darwin":
			if matches, err := filepath.Glob(TIGER_VNC_PATTERN); err == nil && len(matches) > 0 {
				// Always use the latest version
				vncBin = matches[len(matches)-1]
				args = tigerVncArgs(port)
			} else if err == filepath.ErrBadPattern {
				viewResChan <- err
				return
			} else if _, err := os.Stat(CHICKEN_VNC); err == nil {
				vncBin = CHICKEN_VNC
				args = chickenVncArgs(port)
			} else if !os.IsNotExist(err) {
				viewResChan <- err
				return
			} else if _, err := os.Stat(REAL_VNC); err == nil {
				vncBin = REAL_VNC
				args = realVncArgs(port)
			} else if !os.IsNotExist(err) {
				viewResChan <- err
				return
			} else if _, err := exec.LookPath(REMOTE_VIEWER); err == nil {
				// fall back to user supplied script/binary in path
				vncBin = REMOTE_VIEWER
				args = remoteViewerArgs(port)
			} else if !os.IsNotExist(err) {
				viewResChan <- err
				return
			}
		case "linux", "windows":
			_, err := exec.LookPath(REMOTE_VIEWER)
			if exec.ErrNotFound == err {
				viewResChan <- fmt.Errorf("could not find the remote-viewer binary in $PATH")
				return
			} else if err != nil {
				viewResChan <- err
				return
			}
			vncBin = REMOTE_VIEWER
			args = remoteViewerArgs(port)
		default:
			viewResChan <- fmt.Errorf("virtctl does not support VNC on %v", osType)
			return
		}

		if vncBin == "" {
			glog.Errorf("No supported VNC app found in %s", osType)
			err = fmt.Errorf("No supported VNC app found in %s", osType)
		} else {
			if glog.V(4) {
				glog.Infof("Executing commandline: '%s %v'", vncBin, args)
			}
			cmnd := exec.Command(vncBin, args...)
			output, err := cmnd.CombinedOutput()
			if err != nil {
				glog.Errorf("%s execution failed: %v, output: %v", vncBin, err, string(output))
			} else {
				glog.V(2).Infof("remote-viewer output: %v", string(output))
			}
		}
		viewResChan <- err
	}()

	go func() {
		defer close(stopChan)
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
	}()

	select {
	case <-stopChan:
	case err = <-readStop:
	case err = <-writeStop:
	case err = <-k8ResChan:
	case err = <-viewResChan:
	case err = <-listenResChan:
	}

	if err != nil {
		return fmt.Errorf("Error encountered: %s", err.Error())
	}
	return nil
}

func tigerVncArgs(port int) (args []string) {
	args = append(args, fmt.Sprintf("127.0.0.1:%d", port))
	if glog.V(4) {
		args = append(args, "Log=*:stderr:100")
	}
	return
}

func chickenVncArgs(port int) (args []string) {
	args = append(args, fmt.Sprintf("127.0.0.1:%d", port))
	return
}

func realVncArgs(port int) (args []string) {
	args = append(args, fmt.Sprintf("127.0.0.1:%d", port))
	args = append(args, "-WarnUnencrypted=0")
	args = append(args, "-Shared=0")
	args = append(args, "-ShareFiles=0")
	if glog.V(4) {
		args = append(args, "-log=*:stderr:100")
	}
	return
}

func remoteViewerArgs(port int) (args []string) {
	args = append(args, fmt.Sprintf("vnc://127.0.0.1:%d", port))
	if glog.V(4) {
		args = append(args, "--debug")
	}
	return
}

func usage() string {
	usage := "  # Connect to 'testvmi' via remote-viewer:\n"
	usage += "  virtctl vnc testvmi"
	return usage
}
