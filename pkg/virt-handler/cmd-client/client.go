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

package cmdclient

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"syscall"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type Reply struct {
	Success bool
	Message string
	Domain  *api.Domain
}

type Args struct {
	// used for domain management
	VMI *v1.VirtualMachineInstance
}

type LauncherClient interface {
	SyncVirtualMachine(vmi *v1.VirtualMachineInstance) error
	SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error
	ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error
	KillVirtualMachine(vmi *v1.VirtualMachineInstance) error
	MigrateVirtualMachine(vmi *v1.VirtualMachineInstance) error
	DeleteDomain(vmi *v1.VirtualMachineInstance) error
	GetDomain() (*api.Domain, bool, error)
	Ping() error
	Close()
}

type VirtLauncherClient struct {
	client *rpc.Client
}

func ListAllSockets(baseDir string) ([]string, error) {
	var socketFiles []string

	fileDir := filepath.Join(baseDir, "sockets")
	exists, err := diskutils.FileExists(fileDir)
	if err != nil {
		return nil, err
	}

	if exists == false {
		return socketFiles, nil
	}

	files, err := ioutil.ReadDir(fileDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		socketFiles = append(socketFiles, filepath.Join(fileDir, file.Name()))
	}
	return socketFiles, nil
}

func SocketsDirectory(baseDir string) string {
	return filepath.Join(baseDir, "sockets")
}

func SocketFromUID(baseDir string, uid string) string {
	sockFile := uid + "_sock"
	return filepath.Join(SocketsDirectory(baseDir), sockFile)
}

func GetClient(socketPath string) (LauncherClient, error) {
	conn, err := rpc.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	return &VirtLauncherClient{client: conn}, nil
}

func (c *VirtLauncherClient) Close() {
	c.client.Close()
}

func (c *VirtLauncherClient) genericSendCmd(args *Args, cmd string) (*Reply, error) {
	reply := &Reply{}

	err := c.client.Call(cmd, args, reply)
	if IsDisconnected(err) {
		return reply, err
	} else if err != nil {
		msg := fmt.Sprintf("unknown error encountered sending command %s: %s", cmd, err.Error())
		return reply, fmt.Errorf(msg)
	} else if reply.Success != true {
		return reply, fmt.Errorf("server error. command %s failed: %q", cmd, reply.Message)
	}
	return reply, nil
}

func (c *VirtLauncherClient) ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	cmd := "Launcher.Shutdown"

	args := &Args{
		VMI: vmi,
	}
	_, err := c.genericSendCmd(args, cmd)

	return err
}

func (c *VirtLauncherClient) MigrateVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	cmd := "Launcher.Migrate"

	args := &Args{
		VMI: vmi,
	}
	_, err := c.genericSendCmd(args, cmd)

	return err
}

func (c *VirtLauncherClient) KillVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	cmd := "Launcher.Kill"

	args := &Args{
		VMI: vmi,
	}
	_, err := c.genericSendCmd(args, cmd)

	return err
}

func (c *VirtLauncherClient) DeleteDomain(vmi *v1.VirtualMachineInstance) error {
	cmd := "Launcher.Delete"

	args := &Args{
		VMI: vmi,
	}
	_, err := c.genericSendCmd(args, cmd)

	return err
}

func (c *VirtLauncherClient) GetDomain() (*api.Domain, bool, error) {
	domain := &api.Domain{}
	cmd := "Launcher.GetDomain"
	exists := false

	args := &Args{}

	reply, err := c.genericSendCmd(args, cmd)
	if err != nil {
		return nil, exists, err
	}

	if reply.Domain != nil {
		domain = reply.Domain
		exists = true
	}
	return domain, exists, nil

}
func (c *VirtLauncherClient) SyncVirtualMachine(vmi *v1.VirtualMachineInstance) error {

	cmd := "Launcher.Sync"

	args := &Args{
		VMI: vmi,
	}

	_, err := c.genericSendCmd(args, cmd)

	return err
}

func (c *VirtLauncherClient) SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error {

	cmd := "Launcher.SyncMigrationTarget"

	args := &Args{
		VMI: vmi,
	}

	_, err := c.genericSendCmd(args, cmd)

	return err
}

func IsDisconnected(err error) bool {
	if err == nil {
		return false
	}
	if err == rpc.ErrShutdown || err == io.ErrUnexpectedEOF || err == io.EOF {
		return true
	}

	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
			// catches "connection reset by peer"
			if syscallErr.Err == syscall.ECONNRESET {
				return true
			}
		}
	}

	return false
}

func (c *VirtLauncherClient) Ping() error {
	cmd := "Launcher.Ping"
	args := &Args{}
	_, err := c.genericSendCmd(args, cmd)

	return err
}
