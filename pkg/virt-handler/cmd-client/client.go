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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/api/core/v1"
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
	legacyBaseDir        = "/var/run/kubevirt"
	podsBaseDir          = "/pods"
)

const StandardLauncherSocketFileName = "launcher-sock"
const StandardInitLauncherSocketFileName = "launcher-init-sock"
const StandardLauncherUnresponsiveFileName = "launcher-unresponsive"

const MultiThreadedQemuMigrationAnnotation = "kubevirt.io/multiThreadedQemuMigration"

type MigrationOptions struct {
	Bandwidth                resource.Quantity
	ProgressTimeout          int64
	CompletionTimeoutPerGiB  int64
	UnsafeMigration          bool
	AllowAutoConverge        bool
	AllowPostCopy            bool
	ParallelMigrationThreads *uint
}

type LauncherClient interface {
	SyncVirtualMachine(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error
	PauseVirtualMachine(vmi *v1.VirtualMachineInstance) error
	UnpauseVirtualMachine(vmi *v1.VirtualMachineInstance) error
	FreezeVirtualMachine(vmi *v1.VirtualMachineInstance, unfreezeTimeoutSeconds int32) error
	UnfreezeVirtualMachine(vmi *v1.VirtualMachineInstance) error
	SyncMigrationTarget(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error
	SoftRebootVirtualMachine(vmi *v1.VirtualMachineInstance) error
	SignalTargetPodCleanup(vmi *v1.VirtualMachineInstance) error
	ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error
	KillVirtualMachine(vmi *v1.VirtualMachineInstance) error
	MigrateVirtualMachine(vmi *v1.VirtualMachineInstance, options *MigrationOptions) error
	CancelVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error
	FinalizeVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error
	HotplugHostDevices(vmi *v1.VirtualMachineInstance) error
	DeleteDomain(vmi *v1.VirtualMachineInstance) error
	GetDomain() (*api.Domain, bool, error)
	GetDomainStats() (*stats.DomainStats, bool, error)
	GetGuestInfo() (*v1.VirtualMachineInstanceGuestAgentInfo, error)
	GetUsers() (v1.VirtualMachineInstanceGuestOSUserList, error)
	GetFilesystems() (v1.VirtualMachineInstanceFileSystemList, error)
	Exec(string, string, []string, int32) (int, string, error)
	Ping() error
	GuestPing(string, int32) error
	Close()
	VirtualMachineMemoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error
	GetQemuVersion() (string, error)
	SyncVirtualMachineCPUs(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error
	GetSEVInfo() (*v1.SEVPlatformInfo, error)
	GetLaunchMeasurement(*v1.VirtualMachineInstance) (*v1.SEVMeasurementInfo, error)
	InjectLaunchSecret(*v1.VirtualMachineInstance, *v1.SEVSecretOptions) error
	SyncVirtualMachineMemory(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error
}

type VirtLauncherClient struct {
	v1client cmdv1.CmdClient
	conn     *grpc.ClientConn
}

const (
	shortTimeout time.Duration = 5 * time.Second
	longTimeout  time.Duration = 20 * time.Second
)

func SetLegacyBaseDir(baseDir string) {
	legacyBaseDir = baseDir
}

func SetPodsBaseDir(baseDir string) {
	podsBaseDir = baseDir
}

func ListAllSockets() ([]string, error) {
	var socketFiles []string

	socketsDir := filepath.Join(legacyBaseDir, "sockets")
	exists, err := diskutils.FileExists(socketsDir)
	if err != nil {
		return nil, err
	}

	if exists == false {
		return socketFiles, nil
	}

	files, err := os.ReadDir(socketsDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || !strings.Contains(file.Name(), "_sock") {
			continue
		}
		// legacy support.
		// The old way of handling launcher sockets was to
		// dump them all in the same directory. So if we encounter
		// a legacy socket, still process it. This is necessary
		// for update support.
		socketFiles = append(socketFiles, filepath.Join(socketsDir, file.Name()))
	}

	podsDir := podsBaseDir
	dirs, err := os.ReadDir(podsDir)
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		socketPath := SocketFilePathOnHost(dir.Name())
		exists, err = diskutils.FileExists(socketPath)
		if err != nil {
			return socketFiles, err
		}

		if exists {
			socketFiles = append(socketFiles, socketPath)
		}
	}

	return socketFiles, nil
}

func LegacySocketsDirectory() string {
	return filepath.Join(legacyBaseDir, "sockets")
}

func IsLegacySocket(socket string) bool {
	if filepath.Base(socket) == StandardLauncherSocketFileName {
		return false
	}

	return true
}

func SocketMonitoringEnabled(socket string) bool {
	if filepath.Base(socket) == StandardLauncherSocketFileName {
		return true
	}
	return false
}

func IsSocketUnresponsive(socket string) bool {
	file := filepath.Join(filepath.Dir(socket), StandardLauncherUnresponsiveFileName)
	exists, _ := diskutils.FileExists(file)
	// if the unresponsive socket monitor marked this socket
	// as being unresponsive, return true
	if exists {
		return true
	}

	exists, _ = diskutils.FileExists(socket)
	// if the socket file doesn't exist, it's definitely unresponsive as well
	if !exists {
		return true
	}

	return false
}

func MarkSocketUnresponsive(socket string) error {
	file := filepath.Join(filepath.Dir(socket), StandardLauncherUnresponsiveFileName)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func SocketDirectoryOnHost(podUID string) string {
	return fmt.Sprintf("/%s/%s/volumes/kubernetes.io~empty-dir/sockets", podsBaseDir, string(podUID))
}

func SocketFilePathOnHost(podUID string) string {
	return fmt.Sprintf("%s/%s", SocketDirectoryOnHost(podUID), StandardLauncherSocketFileName)
}

// gets the cmd socket for a VMI
func FindPodDirOnHost(vmi *v1.VirtualMachineInstance) (string, error) {

	// It is possible for multiple pods to be active on a single VMI
	// during migrations. This loop will discover the active pod on
	// this particular local node if it exists. A active pod not
	// running on this node will not have a kubelet pods directory,
	// so it will not be found.
	for podUID := range vmi.Status.ActivePods {
		socketPodDir := SocketDirectoryOnHost(string(podUID))
		exists, _ := diskutils.FileExists(socketPodDir)
		if exists {
			return socketPodDir, nil
		}
	}

	return "", fmt.Errorf("No command socketdir for vmi %s", vmi.UID)
}

// gets the cmd socket for a VMI
func FindSocketOnHost(vmi *v1.VirtualMachineInstance) (string, error) {
	if string(vmi.UID) != "" {
		legacySockFile := string(vmi.UID) + "_sock"
		legacySock := filepath.Join(LegacySocketsDirectory(), legacySockFile)
		exists, _ := diskutils.FileExists(legacySock)
		if exists {
			return legacySock, nil
		}
	}

	socketsFound := 0
	foundSocket := ""
	// It is possible for multiple pods to be active on a single VMI
	// during migrations. This loop will discover the active pod on
	// this particular local node if it exists. A active pod not
	// running on this node will not have a kubelet pods directory,
	// so it will not be found.
	for podUID := range vmi.Status.ActivePods {
		socket := SocketFilePathOnHost(string(podUID))
		exists, _ := diskutils.FileExists(socket)
		if exists {
			foundSocket = socket
			socketsFound++
		}
	}

	if socketsFound == 1 {
		return foundSocket, nil
	} else if socketsFound > 1 {
		return "", fmt.Errorf("Found multiple sockets for vmi %s/%s. waiting for only one to exist", vmi.Namespace, vmi.Name)
	}

	return "", fmt.Errorf("No command socket found for vmi %s", vmi.UID)
}

func SocketOnGuest() string {
	sockFile := StandardLauncherSocketFileName
	return filepath.Join(LegacySocketsDirectory(), sockFile)
}

func UninitializedSocketOnGuest() string {
	sockFile := StandardInitLauncherSocketFileName
	return filepath.Join(LegacySocketsDirectory(), sockFile)
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
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()
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
func IsUnimplemented(err error) bool {
	if grpcStatus, ok := status.FromError(err); ok {
		if grpcStatus.Code() == codes.Unimplemented {
			return true
		}
	}
	return false
}
func handleError(err error, cmdName string, response *cmdv1.Response) error {
	if IsDisconnected(err) {
		return err
	} else if IsUnimplemented(err) {
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

func (c *VirtLauncherClient) FreezeVirtualMachine(vmi *v1.VirtualMachineInstance, unfreezeTimeoutSeconds int32) error {
	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return err
	}

	request := &cmdv1.FreezeRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
		UnfreezeTimeoutSeconds: unfreezeTimeoutSeconds,
	}

	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()
	response, err := c.v1client.FreezeVirtualMachine(ctx, request)

	err = handleError(err, "Freeze", response)
	return err
}

func (c *VirtLauncherClient) UnfreezeVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("Unfreeze", c.v1client.UnfreezeVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) VirtualMachineMemoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error {
	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return err
	}

	request := &cmdv1.MemoryDumpRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
		DumpPath: dumpPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()
	response, err := c.v1client.VirtualMachineMemoryDump(ctx, request)
	err = handleError(err, "Memorydump", response)
	return err
}

func (c *VirtLauncherClient) SoftRebootVirtualMachine(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("SoftReboot", c.v1client.SoftRebootVirtualMachine, vmi, &cmdv1.VirtualMachineOptions{})
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

func (c *VirtLauncherClient) SyncMigrationTarget(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {
	return c.genericSendVMICmd("SyncMigrationTarget", c.v1client.SyncMigrationTarget, vmi, options)
}

func (c *VirtLauncherClient) SyncVirtualMachineCPUs(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {
	return c.genericSendVMICmd("SyncVirtualMachineCPUs", c.v1client.SyncVirtualMachineCPUs, vmi, options)
}

func (c *VirtLauncherClient) SignalTargetPodCleanup(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("SignalTargetPodCleanup", c.v1client.SignalTargetPodCleanup, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) FinalizeVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("FinalizeVirtualMachineMigration", c.v1client.FinalizeVirtualMachineMigration, vmi, &cmdv1.VirtualMachineOptions{})
}

func (c *VirtLauncherClient) HotplugHostDevices(vmi *v1.VirtualMachineInstance) error {
	return c.genericSendVMICmd("HotplugHostDevices", c.v1client.HotplugHostDevices, vmi, &cmdv1.VirtualMachineOptions{})
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

func (c *VirtLauncherClient) GetQemuVersion() (string, error) {
	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	versionResponse, err := c.v1client.GetQemuVersion(ctx, request)
	var response *cmdv1.Response
	if versionResponse != nil {
		response = versionResponse.Response
	}
	if err = handleError(err, "GetQemuVersion", response); err != nil {
		return "", err
	}

	if versionResponse != nil && versionResponse.Version != "" {
		return versionResponse.Version, nil
	}

	log.Log.Reason(err).Error("error getting the qemu version")
	return "", errors.New("error getting the qemu version")
}

func (c *VirtLauncherClient) GetDomainStats() (*stats.DomainStats, bool, error) {
	stats := &stats.DomainStats{}
	exists := false

	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	domainStatsResponse, err := c.v1client.GetDomainStats(ctx, request)
	var response *cmdv1.Response
	if domainStatsResponse != nil {
		response = domainStatsResponse.Response
	}

	if err = handleError(err, "GetDomainStats", response); err != nil {
		return stats, exists, err
	}

	if domainStatsResponse.DomainStats != "" {
		if err := json.Unmarshal([]byte(domainStatsResponse.DomainStats), stats); err != nil {
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

// Exec the command with args on the guest and return the resulting status code, stdOut and error
func (c *VirtLauncherClient) Exec(domainName, command string, args []string, timeoutSeconds int32) (int, string, error) {
	request := &cmdv1.ExecRequest{
		DomainName:     domainName,
		Command:        command,
		Args:           args,
		TimeoutSeconds: int32(timeoutSeconds),
	}
	exitCode := -1
	stdOut := ""

	ctx, cancel := context.WithTimeout(
		context.Background(),
		// we give the context a bit more time as the timeout should kick
		// on the actual execution
		time.Duration(timeoutSeconds)*time.Second+shortTimeout,
	)
	defer cancel()

	resp, err := c.v1client.Exec(ctx, request)
	if resp == nil {
		return exitCode, stdOut, err
	}

	exitCode = int(resp.ExitCode)
	stdOut = resp.StdOut

	return exitCode, stdOut, err
}

func (c *VirtLauncherClient) GuestPing(domainName string, timeoutSeconds int32) error {
	request := &cmdv1.GuestPingRequest{
		DomainName:     domainName,
		TimeoutSeconds: timeoutSeconds,
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		// we give the context a bit more time as the timeout should kick
		// on the actual execution
		time.Duration(timeoutSeconds)*time.Second+shortTimeout,
	)
	defer cancel()

	_, err := c.v1client.GuestPing(ctx, request)
	return err
}

func (c *VirtLauncherClient) GetSEVInfo() (*v1.SEVPlatformInfo, error) {
	request := &cmdv1.EmptyRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	sevInfoResponse, err := c.v1client.GetSEVInfo(ctx, request)
	if err = handleError(err, "GetSEVInfo", sevInfoResponse.GetResponse()); err != nil {
		return nil, err
	}

	sevPlatformInfo := &v1.SEVPlatformInfo{}
	if err := json.Unmarshal(sevInfoResponse.GetSevInfo(), sevPlatformInfo); err != nil {
		log.Log.Reason(err).Error("error unmarshalling SEV info response")
		return nil, err
	}

	return sevPlatformInfo, nil
}

func (c *VirtLauncherClient) GetLaunchMeasurement(vmi *v1.VirtualMachineInstance) (*v1.SEVMeasurementInfo, error) {
	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return nil, err
	}

	request := &cmdv1.VMIRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	launchMeasurementRespose, err := c.v1client.GetLaunchMeasurement(ctx, request)
	if err = handleError(err, "GetLaunchMeasurement", launchMeasurementRespose.GetResponse()); err != nil {
		return nil, err
	}

	sevMeasurementInfo := &v1.SEVMeasurementInfo{}
	if err := json.Unmarshal(launchMeasurementRespose.GetLaunchMeasurement(), sevMeasurementInfo); err != nil {
		log.Log.Reason(err).Error("error unmarshalling launch measurement response")
		return nil, err
	}

	return sevMeasurementInfo, nil
}

func (c *VirtLauncherClient) InjectLaunchSecret(vmi *v1.VirtualMachineInstance, sevSecretOptions *v1.SEVSecretOptions) error {
	vmiJson, err := json.Marshal(vmi)
	if err != nil {
		return err
	}

	optionsJson, err := json.Marshal(sevSecretOptions)
	if err != nil {
		return err
	}

	request := &cmdv1.InjectLaunchSecretRequest{
		Vmi: &cmdv1.VMI{
			VmiJson: vmiJson,
		},
		Options: optionsJson,
	}

	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	response, err := c.v1client.InjectLaunchSecret(ctx, request)

	return handleError(err, "InjectLaunchSecret", response)
}

func (c *VirtLauncherClient) SyncVirtualMachineMemory(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {
	return c.genericSendVMICmd("SyncVirtualMachineMemory", c.v1client.SyncVirtualMachineMemory, vmi, options)
}
