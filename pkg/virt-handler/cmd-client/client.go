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
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	com "kubevirt.io/kubevirt/pkg/handler-launcher-com"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
	supportedCmdVersions         = []string{"v1"}
	SupportedKubeVirtAPIVersions = []string{"kubevirt.io/v1alpha3"}
	SupportedDomainVersions      = []string{"v1"}
	SupportedDomainStatsVersions = []string{"v1"}
)

type MigrationOptions struct {
	Bandwidth               resource.Quantity
	ProgressTimeout         int64
	CompletionTimeoutPerGiB int64
}

type LauncherClient interface {
	SyncVirtualMachine(vmi *v1.VirtualMachineInstance) error
	SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error
	ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error
	KillVirtualMachine(vmi *v1.VirtualMachineInstance) error
	MigrateVirtualMachine(vmi *v1.VirtualMachineInstance, options *MigrationOptions) error
	CancelVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error
	DeleteDomain(vmi *v1.VirtualMachineInstance) error
	GetDomain() (*api.Domain, bool, error)
	GetDomainStats() (*stats.DomainStats, bool, error)
	Ping() error
	Close()
}

type VirtLauncherClient struct {
	client cmdv1.CmdClient
	conn   *grpc.ClientConn
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

func NewClient(socketPath string) (LauncherClient, error) {
	// prepare cmd clients
	infoClient, client, conn, err := com.NewCmdClients(socketPath)
	if err != nil {
		return nil, err
	}
	return NewClientWithRPCClients(infoClient, client, conn)
}

func NewClientWithRPCClients(infoClient info.CmdInfoClient, client cmdv1.CmdClient, conn *grpc.ClientConn) (LauncherClient, error) {

	// check if the cmd server is compatible
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	info, err := infoClient.Info(ctx, &info.CmdInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("could not check cmd server version: %v", err)
	}
	if !checkVersions(info) {
		return nil, fmt.Errorf("incompatible cmd server: %+v", info)
	}

	return &VirtLauncherClient{
		client: client,
		conn:   conn,
	}, nil
}

func checkVersions(info *info.CmdInfoResponse) bool {
	return com.ContainsVersion(info.SupportedCmdVersions, supportedCmdVersions) &&
		com.ContainsVersion(info.SupportedKubeVirtAPIVersions, SupportedKubeVirtAPIVersions) &&
		com.ContainsVersion(info.SupportedDomainVersions, SupportedDomainVersions) &&
		com.ContainsVersion(info.SupportedDomainStatsVersions, SupportedDomainStatsVersions)
}

func (c *VirtLauncherClient) Close() {
	c.conn.Close()
}

func (c *VirtLauncherClient) genericSendVMICmd(cmdName string,
	cmdFunc func(ctx context.Context, request *cmdv1.VMIRequest, opts ...grpc.CallOption) (*cmdv1.Response, error),
	vmi *v1.VirtualMachineInstance) error {

	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return err
	}

	request := &cmdv1.VMIRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := cmdFunc(ctx, request)

	err = handleError(err, cmdName, response)
	return err
}

func handleError(err error, cmdName string, response *cmdv1.Response) error {
	if IsDisconnected(err) {
		return err
	} else if err != nil {
		msg := fmt.Sprintf("unknown error encountered sending command %s: %s", cmdName, err.Error())
		return fmt.Errorf(msg)
	} else if response.Success != true {
		return fmt.Errorf("server error. command %s failed: %q", cmdName, response.Message)
	}
	return nil
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

	if grpcStatus, ok := status.FromError(err); ok {

		// see https://github.com/grpc/grpc-go/blob/master/codes/codes.go
		// TODO which other codes might be related to disconnection...?
		switch grpcStatus.Code() {
		case codes.Canceled:
			// e.g. client connection closing
			return true
		}

	}

	return false
}

func (c *VirtLauncherClient) SyncVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("SyncVMI", c.client.SyncVirtualMachine, vmi)

}

func (c *VirtLauncherClient) ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Shutdown", c.client.ShutdownVirtualMachine, vmi)
}

func (c *VirtLauncherClient) KillVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Kill", c.client.KillVirtualMachine, vmi)
}

func (c *VirtLauncherClient) DeleteDomain(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Delete", c.client.DeleteVirtualMachine, vmi)
}

func (c *VirtLauncherClient) MigrateVirtualMachine(vmi *v1.VirtualMachineInstance, options *MigrationOptions) error {

	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return err
	}

	optionsJson, err := json.Marshal(options)
	if err != nil {
		return err
	}

	request := &cmdv1.MigrationRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
		Options: optionsJson,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := c.client.MigrateVirtualMachine(ctx, request)

	err = handleError(err, "Migrate", response)
	return err

}

func (c *VirtLauncherClient) CancelVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("CancelMigration", c.client.CancelVirtualMachineMigration, vmi)
}

func (c *VirtLauncherClient) SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("SyncMigrationTarget", c.client.SyncMigrationTarget, vmi)

}

func (c *VirtLauncherClient) GetDomain() (*api.Domain, bool, error) {

	domain := &api.Domain{}
	exists := false

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := c.client.GetDomain(ctx, request)

	if err = handleError(err, "GetDomain", response.Response); err != nil {
		return domain, exists, err
	}

	if response.Domain != "" {
		if err := json.Unmarshal([]byte(response.Domain), domain); err != nil {
			log.Log.Reason(err).Error("error unmarshalling domain")
			return domain, exists, err
		}
		exists = true
	}
	return domain, exists, nil
}

func (c *VirtLauncherClient) GetDomainStats() (*stats.DomainStats, bool, error) {
	stats := &stats.DomainStats{}
	exists := false

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := c.client.GetDomainStats(ctx, request)

	if err = handleError(err, "GetDomainStats", response.Response); err != nil {
		return stats, exists, err
	}

	if response.DomainStats != "" {
		if err := json.Unmarshal([]byte(response.DomainStats), stats); err != nil {
			log.Log.Reason(err).Error("error unmarshalling domain")
			return stats, exists, err
		}
		exists = true
	}
	return stats, exists, nil
}

func (c *VirtLauncherClient) Ping() error {
	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := c.client.Ping(ctx, request)

	err = handleError(err, "Ping", response)
	return err
}
