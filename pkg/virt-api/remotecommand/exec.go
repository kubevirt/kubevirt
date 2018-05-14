/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package remotecommand

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"net"
	"strings"

	"golang.org/x/crypto/ssh"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"

	v12 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

// Executor knows how to execute a command in a container in a pod.
type Executor interface {
	// ExecInContainer executes a command in a container in the pod, copying data
	// between in/out/err and the container's stdin/stdout/stderr.
	ExecInVirtualMachine(vm *v1.VirtualMachine, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize, timeout time.Duration) error
}

// ServeExec handles requests to execute a command in a container. After
// creating/receiving the required streams, it delegates the actual execution
// to the executor.
func ServeExec(w http.ResponseWriter, req *http.Request, executor Executor, vm *v1.VirtualMachine, cmd []string, streamOpts *Options, idleTimeout, streamCreationTimeout time.Duration, supportedProtocols []string) {
	ctx, ok := createStreams(req, w, streamOpts, supportedProtocols, idleTimeout, streamCreationTimeout)
	if !ok {
		// error is handled by createStreams
		return
	}
	defer ctx.conn.Close()

	err := executor.ExecInVirtualMachine(vm, cmd, ctx.stdinStream, ctx.stdoutStream, ctx.stderrStream, ctx.tty, ctx.resizeChan, 0)
	if err != nil {
		if exitErr, ok := err.(exec.ExitError); ok && exitErr.Exited() {
			rc := exitErr.ExitStatus()
			ctx.writeStatus(&apierrors.StatusError{ErrStatus: metav1.Status{
				Status: metav1.StatusFailure,
				Reason: remotecommandconsts.NonZeroExitCodeReason,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    remotecommandconsts.ExitCodeCauseType,
							Message: fmt.Sprintf("%d", rc),
						},
					},
				},
				Message: fmt.Sprintf("command terminated with non-zero exit code: %v", exitErr),
			}})
		} else {
			err = fmt.Errorf("error executing command in container: %v", err)
			runtime.HandleError(err)
			ctx.writeStatus(apierrors.NewInternalError(err))
		}
	} else {
		ctx.writeStatus(&apierrors.StatusError{ErrStatus: metav1.Status{
			Status: metav1.StatusSuccess,
		}})
	}
}

type SSHExecutor struct {
	KubeCli kubecli.KubevirtClient
}

func extractValue(secret *v12.Secret, key string) (string, error) {

	valueBase, ok := secret.Data[key]
	if ok == false {
		return "", fmt.Errorf("key %v does not exist", key)
	}
	return string(valueBase), nil
}

func (e *SSHExecutor) ExecInVirtualMachine(vm *v1.VirtualMachine, cmd []string, in io.Reader, out, errWriteCloser io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize, timeout time.Duration) error {
	var user string
	var password string
	var port string

	for _, volume := range vm.Spec.Volumes {
		if volume.CloudInitNoCloud != nil && volume.CloudInitNoCloud.UserDataSecretRef != nil {
			secret, err := e.KubeCli.CoreV1().Secrets(vm.Namespace).Get(volume.CloudInitNoCloud.UserDataSecretRef.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			user, err = extractValue(secret, "ssh.user")
			if err != nil {
				return err
			}

			password, err = extractValue(secret, "ssh.password")
			if err != nil {
				return err
			}

			port, err = extractValue(secret, "ssh.port")
			if err != nil {
				return err
			}
			break
		}
	}

	ip := vm.Status.Interfaces[0].IP
	tcpConn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		log.Log.Reason(err).Error("Failed to create tcp connection")
		return fmt.Errorf("could not open tcp connection: %v", err)
	}
	defer tcpConn.Close()
	log.Log.V(4).Infof("tcp connection to %v:%v established", ip, "22")

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (string, error) {
				return string(password), nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshCon, chans, reqs, err := ssh.NewClientConn(tcpConn, "whatever", sshConfig)
	if err != nil {
		log.Log.Reason(err).Error("Failed to create ssh connection")
		return fmt.Errorf("failed to create ssh connection: %v", err)
	}
	log.Log.V(4).Info("ssh connection established")
	cli := ssh.NewClient(sshCon, chans, reqs)

	session, err := cli.NewSession()
	if err != nil {
		log.Log.Reason(err).Error("Failed to create session")
		return fmt.Errorf("failed to create session: %v", err)
	}
	log.Log.V(4).Info("ssh session created")
	defer session.Close()

	if tty {
		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
			log.Log.Reason(err).Error("Failed to allocate pseudo terminal")
			return fmt.Errorf("failed to allocate pseudo terminal: %v", err)
		}
		log.Log.V(4).Info("tty created")
	}

	errChan := make(chan error)

	stdin, err := session.StdinPipe()
	if err != nil {
		log.Log.Reason(err).Error("Unable to setup stdin for session")
		return fmt.Errorf("unable to setup stdin for session: %v", err)
	}
	go func() {
		_, err := io.Copy(stdin, in)
		errChan <- err
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Log.Reason(err).Error("Unable to setup stdout for session")
		return fmt.Errorf("unable to setup stdout for session: %v", err)
	}
	go func() {
		_, err := io.Copy(out, stdout)
		errChan <- err
	}()

	stderr, err := session.StderrPipe()
	if err != nil {
		log.Log.Reason(err).Error("Unable to setup stderr for session")
		return fmt.Errorf("unable to setup stderr for session: %v", err)
	}
	go func() {
		_, err := io.Copy(errWriteCloser, stderr)
		errChan <- err
	}()

	go func() {
		errChan <- session.Run(strings.Join(cmd, " "))
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Log.Reason(err).Error("Error in websocket proxy")
			return fmt.Errorf("error in websocket proxy: %v", err)
		}
	}
	return nil
}
