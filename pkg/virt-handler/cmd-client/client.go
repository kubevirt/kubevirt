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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	com "kubevirt.io/kubevirt/pkg/handler-launcher-com"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
	// add older version when supported
	// don't use the variable in pkg/handler-launcher-com/cmd/v1/version.go in order to detect version mismatches early
	supportedCmdVersions = []uint32{1}
)

const StandardLauncherSocketFileName = "launcher-sock"

type MigrationOptions struct {
	Bandwidth               resource.Quantity
	ProgressTimeout         int64
	CompletionTimeoutPerGiB int64
	UnsafeMigration         bool
	AllowAutoConverge       bool
}

type LauncherClient interface {
	SyncVirtualMachine(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error
	PauseVirtualMachine(vmi *v1.VirtualMachineInstance) error
	UnpauseVirtualMachine(vmi *v1.VirtualMachineInstance) error
	SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error
	ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error
	KillVirtualMachine(vmi *v1.VirtualMachineInstance) error
	MigrateVirtualMachine(vmi *v1.VirtualMachineInstance, options *MigrationOptions) error
	CancelVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error
	SetVirtualMachineGuestTime(vmi *v1.VirtualMachineInstance) error
	DeleteDomain(vmi *v1.VirtualMachineInstance) error
	GetDomain() (*api.Domain, bool, error)
	GetDomainStats() (*stats.DomainStats, bool, error)
	GetGuestInfo() (*v1.VirtualMachineInstanceGuestAgentInfo, error)
	GetUsers() (v1.VirtualMachineInstanceGuestOSUserList, error)
	GetFilesystems() (v1.VirtualMachineInstanceFileSystemList, error)
	Ping() error
	Close()
}

type VirtLauncherClient struct {
	v1client cmdv1.CmdClient
	conn     *grpc.ClientConn
}

const (
	shortTimeout time.Duration = 5 * time.Second
	longTimeout  time.Duration = 20 * time.Second
)

func ListAllSockets(baseDir string) ([]string, error) {
	var socketFiles []string

	socketsDir := filepath.Join(baseDir, "sockets")
	exists, err := diskutils.FileExists(socketsDir)
	if err != nil {
		return nil, err
	}

	if exists == false {
		return socketFiles, nil
	}

	directories, err := ioutil.ReadDir(socketsDir)
	if err != nil {
		return nil, err
	}
	for _, dir := range directories {
		if !dir.IsDir() {
			// legacy support.
			// The old way of handling launcher sockets was to
			// dump them all in the same directory. So if we encounter
			// a legacy socket, still process it. This is necessary
			// for update support.
			socketFiles = append(socketFiles, filepath.Join(socketsDir, dir.Name()))
			continue
		}
		files, err := ioutil.ReadDir(filepath.Join(socketsDir, dir.Name()))
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			socketFiles = append(socketFiles, filepath.Join(socketsDir, dir.Name(), file.Name()))
		}
	}
	return socketFiles, nil
}

func SocketsDirectory(baseDir string) string {
	return filepath.Join(baseDir, "sockets")
}

func SocketFromUID(baseDir string, uid string, isHost bool) string {
	sockFile := StandardLauncherSocketFileName
	legacySockFile := uid + "_sock"
	if isHost {
		// legacy support.
		// The old way of handling launcher sockets was to
		// dump them all in the same directory. So if we encounter
		// a legacy socket, still process it. This is necessary
		// for update support.
		legacySock := filepath.Join(SocketsDirectory(baseDir), legacySockFile)
		exists, _ := diskutils.FileExists(legacySock)
		if exists {
			return legacySock
		}

		return filepath.Join(SocketsDirectory(baseDir), uid, sockFile)
	} else {
		return filepath.Join(SocketsDirectory(baseDir), sockFile)
	}
}

func NewClient(socketPath string) (LauncherClient, error) {
	// dial socket
	conn, err := grpcutil.DialSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Infof("failed to dial cmd socket: %s", socketPath)
		return nil, err
	}

	// create info client and find cmd version to use
	infoClient := info.NewCmdInfoClient(conn)
	return NewClientWithInfoClient(infoClient, conn)
}

func NewClientWithInfoClient(infoClient info.CmdInfoClient, conn *grpc.ClientConn) (LauncherClient, error) {
	ctx, _ := context.WithTimeout(context.Background(), shortTimeout)
	info, err := infoClient.Info(ctx, &info.CmdInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("could not check cmd server version: %v", err)
	}
	version, err := com.GetHighestCompatibleVersion(info.SupportedCmdVersions, supportedCmdVersions)
	if err != nil {
		return nil, err
	}

	// create cmd client
	switch version {
	case 1:
		client := cmdv1.NewCmdClient(conn)
		return newV1Client(client, conn), nil
	default:
		return nil, fmt.Errorf("cmd client version %v not implemented yet", version)
	}
}

func newV1Client(client cmdv1.CmdClient, conn *grpc.ClientConn) LauncherClient {
	return &VirtLauncherClient{
		v1client: client,
		conn:     conn,
	}
}

func (c *VirtLauncherClient) Close() {
	c.conn.Close()
}

func (c *VirtLauncherClient) genericSendVMICmd(cmdName string,
	cmdFunc func(ctx context.Context, request *cmdv1.VMIRequest, opts ...grpc.CallOption) (*cmdv1.Response, error),
	vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {

	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return err
	}

	request := &cmdv1.VMIRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
		Options: options,
	}

	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
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
	} else if response != nil && response.Success != true {
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
			// e.g. v1client connection closing
			return true
		}

	}

	return false
}

func (c *VirtLauncherClient) SyncVirtualMachine(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {
	return c.genericSendVMICmd("SyncVMI", c.v1client.SyncVirtualMachine, vmi, options)
}

func (c *VirtLauncherClient) PauseVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Pause", c.v1client.PauseVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) UnpauseVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Unpause", c.v1client.UnpauseVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Shutdown", c.v1client.ShutdownVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) KillVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Kill", c.v1client.KillVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) DeleteDomain(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Delete", c.v1client.DeleteVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
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

	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()
	response, err := c.v1client.MigrateVirtualMachine(ctx, request)

	err = handleError(err, "Migrate", response)
	return err

}

func (c *VirtLauncherClient) CancelVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("CancelMigration", c.v1client.CancelVirtualMachineMigration, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("SyncMigrationTarget", c.v1client.SyncMigrationTarget, vmi, &cmdv1.VirtualMachineOptions{})

}

func (c *VirtLauncherClient) SetVirtualMachineGuestTime(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("SetVirtualMachineGuestTime", c.v1client.SetVirtualMachineGuestTime, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) GetDomain() (*api.Domain, bool, error) {

	domain := &api.Domain{}
	exists := false

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	domainResponse, err := c.v1client.GetDomain(ctx, request)
	var response *cmdv1.Response
	if domainResponse != nil {
		response = domainResponse.Response
	}

	if err = handleError(err, "GetDomain", response); err != nil {
		return domain, exists, err
	}

	if domainResponse.Domain != "" {
		if err := json.Unmarshal([]byte(domainResponse.Domain), domain); err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	domainStatsRespose, err := c.v1client.GetDomainStats(ctx, request)
	var response *cmdv1.Response
	if domainStatsRespose != nil {
		response = domainStatsRespose.Response
	}

	if err = handleError(err, "GetDomainStats", response); err != nil {
		return stats, exists, err
	}

	if domainStatsRespose.DomainStats != "" {
		if err := json.Unmarshal([]byte(domainStatsRespose.DomainStats), stats); err != nil {
			log.Log.Reason(err).Error("error unmarshalling domain")
			return stats, exists, err
		}
		exists = true
	}
	return stats, exists, nil
}

func (c *VirtLauncherClient) Ping() error {
	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()
	response, err := c.v1client.Ping(ctx, request)

	err = handleError(err, "Ping", response)
	return err
}

// GetGuestInfo is a counterpart for virt-launcher call to gather guest agent data
func (c *VirtLauncherClient) GetGuestInfo() (*v1.VirtualMachineInstanceGuestAgentInfo, error) {
	guestInfo := &v1.VirtualMachineInstanceGuestAgentInfo{}

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	gaRespose, err := c.v1client.GetGuestInfo(ctx, request)
	var response *cmdv1.Response
	if gaRespose != nil {
		response = gaRespose.Response
	}

	if err = handleError(err, "GetGuestInfo", response); err != nil {
		return guestInfo, err
	}

	if gaRespose.GuestInfoResponse != "" {
		if err := json.Unmarshal([]byte(gaRespose.GetGuestInfoResponse()), guestInfo); err != nil {
			log.Log.Reason(err).Error("error unmarshalling guest agent response")
			return guestInfo, err
		}
	}
	return guestInfo, nil
}

// GetUsers returns the list of the active users on the guest machine
func (c *VirtLauncherClient) GetUsers() (v1.VirtualMachineInstanceGuestOSUserList, error) {
	userList := []v1.VirtualMachineInstanceGuestOSUser{}

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	uResponse, err := c.v1client.GetUsers(ctx, request)
	var response *cmdv1.Response
	if uResponse != nil {
		response = uResponse.Response
	}

	if err = handleError(err, "GetUsers", response); err != nil {
		return v1.VirtualMachineInstanceGuestOSUserList{}, err
	}

	if uResponse.GetGuestUserListResponse() != "" {
		if err := json.Unmarshal([]byte(uResponse.GetGuestUserListResponse()), &userList); err != nil {
			log.Log.Reason(err).Error("error unmarshalling guest user list response")
			return v1.VirtualMachineInstanceGuestOSUserList{}, err
		}
	}

	guestUserList := v1.VirtualMachineInstanceGuestOSUserList{
		Items: userList,
	}

	return guestUserList, nil
}

// GetFilesystems returns the list of active filesystems on the guest machine
func (c *VirtLauncherClient) GetFilesystems() (v1.VirtualMachineInstanceFileSystemList, error) {
	fsList := []v1.VirtualMachineInstanceFileSystem{}

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	fsResponse, err := c.v1client.GetFilesystems(ctx, request)
	var response *cmdv1.Response
	if fsResponse != nil {
		response = fsResponse.Response
	}

	if err = handleError(err, "GetFilesystems", response); err != nil {
		return v1.VirtualMachineInstanceFileSystemList{}, err
	}

	if fsResponse.GetGuestFilesystemsResponse() != "" {
		if err := json.Unmarshal([]byte(fsResponse.GetGuestFilesystemsResponse()), &fsList); err != nil {
			log.Log.Reason(err).Error("error unmarshalling guest filesystem list response")
			return v1.VirtualMachineInstanceFileSystemList{}, err
		}
	}

	filesystemList := v1.VirtualMachineInstanceFileSystemList{
		Items: fsList,
	}

	return filesystemList, nil
}
