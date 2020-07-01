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
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/common"

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
	legacyBaseDir        = "/var/run/kubevirt"
	podsBaseDir          = "/pods"
)

const StandardLauncherSocketFileName = "launcher-sock"
const StandardLauncherUnresponsiveFileName = "launcher-unresponsive"

var DefaultBackoff = wait.Backoff{
	Duration: 100 * time.Millisecond,
	Factor:   2,
	Steps:    5,
}

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
	HandleK8sEvents()
	HandleDomainEvents()
}

type VirtLauncherClient struct {
	v1client  cmdv1.CmdClient
	conn      *grpc.ClientConn
	eventChan chan watch.Event
	recorder  common.KubernetesEventRecorderInterface
	stopChan  chan struct{}
	backoff   wait.Backoff
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

	files, err := ioutil.ReadDir(socketsDir)
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
	dirs, err := ioutil.ReadDir(podsDir)
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
func FindSocketOnHost(vmi *v1.VirtualMachineInstance) (string, error) {
	if string(vmi.UID) != "" {
		legacySockFile := string(vmi.UID) + "_sock"
		legacySock := filepath.Join(LegacySocketsDirectory(), legacySockFile)
		exists, _ := diskutils.FileExists(legacySock)
		if exists {
			return legacySock, nil
		}
	}

	// It is possible for multiple pods to be active on a single VMI
	// during migrations. This loop will discover the active pod on
	// this particular local node if it exists. A active pod not
	// running on this node will not have a kubelet pods directory,
	// so it will not be found.
	for podUID, _ := range vmi.Status.ActivePods {
		socket := SocketFilePathOnHost(string(podUID))
		exists, _ := diskutils.FileExists(socket)
		if exists {
			return socket, nil
		}
	}

	return "", fmt.Errorf("No command socket found for vmi %s", vmi.UID)

}

func SocketOnGuest() string {
	sockFile := StandardLauncherSocketFileName
	return filepath.Join(LegacySocketsDirectory(), sockFile)
}

func NewClient(socketPath string, eventChan chan watch.Event, recorder common.KubernetesEventRecorderInterface) (LauncherClient, error) {
	// dial socket
	conn, err := grpcutil.DialSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Infof("failed to dial cmd socket: %s", socketPath)
		return nil, err
	}

	// create info client and find cmd version to use
	infoClient := info.NewCmdInfoClient(conn)
	return NewClientWithInfoClient(infoClient, conn, eventChan, recorder)
}

func NewClientWithInfoClient(infoClient info.CmdInfoClient, conn *grpc.ClientConn, eventChan chan watch.Event, recorder common.KubernetesEventRecorderInterface) (LauncherClient, error) {
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
		return newV1Client(client, conn, eventChan, recorder), nil
	default:
		return nil, fmt.Errorf("cmd client version %v not implemented yet", version)
	}
}

func newV1Client(client cmdv1.CmdClient, conn *grpc.ClientConn, eventChan chan watch.Event, recorder common.KubernetesEventRecorderInterface) LauncherClient {
	return &VirtLauncherClient{
		v1client:  client,
		conn:      conn,
		eventChan: eventChan,
		recorder:  recorder,
		backoff:   DefaultBackoff,
		stopChan:  make(chan struct{}),
	}
}

func (c *VirtLauncherClient) Close() {
	close(c.stopChan)
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

// HandleDomainEvents will automatically reconnect and keep the stream alive until Close() is called.
func (c *VirtLauncherClient) HandleDomainEvents() {
	stream := func() error {
		cmd, err := c.v1client.HandleDomainEvent(context.Background(), &cmdv1.EmptyRequest{})
		if err != nil {
			return fmt.Errorf("domain event stream: %v", err)
		}
		for {
			msg, err := cmd.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("domain event stream: %v", err)
			}
			err = common.EnqueueHandlerDomainEvent(c.eventChan, msg)
			if err != nil {
				log.DefaultLogger().Reason(err).Error("Failed to enqueue new domain event.")
			}
		}
	}
	c.runStream(stream)
}

func (c *VirtLauncherClient) runStream(stream func() error) {
	runner := func() (bool, error) {
		err := stream()
		if err != nil {
			log.DefaultLogger().Reason(err).Error("Failed to connect to stream.")
		}
		select {
		case <-c.stopChan:
			log.DefaultLogger().Info("Stop for client received")
			return true, nil
		default:
			log.DefaultLogger().Info("No stop for client received, will try to reconnect stream")
			return false, nil
		}
	}

	for {
		err := wait.ExponentialBackoff(c.backoff, runner)
		if err == nil {
			break
		}
		log.DefaultLogger().Reason(err).Error("Failed to connect to stream.")
		time.Sleep(5 * time.Second)
	}
}

func (c *VirtLauncherClient) HandleK8sEvents() {
	stream := func() error {
		cmd, err := c.v1client.HandleK8SEvent(context.Background(), &cmdv1.EmptyRequest{})
		if err != nil {
			return fmt.Errorf("k8s event stream: %v", err)
		}
		for {
			msg, err := cmd.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("k8s event stream: %v", err)
			}
			err = c.recorder.Record(msg)
			if err != nil {
				log.DefaultLogger().Reason(err).Error("Failed to record new k8s domain event.")
				continue
			}
		}
	}
	c.runStream(stream)
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

type ClientFactory struct {
	clients   map[string]LauncherClient
	locks     map[string]*sync.Mutex
	lock      sync.Mutex
	eventChan chan watch.Event
	recorder  common.KubernetesEventRecorderInterface
}

func NewClientFactory(eventChan chan watch.Event, recorderInterface common.KubernetesEventRecorderInterface) *ClientFactory {
	return &ClientFactory{
		clients:   map[string]LauncherClient{},
		locks:     map[string]*sync.Mutex{},
		lock:      sync.Mutex{},
		eventChan: eventChan,
		recorder:  recorderInterface,
	}
}

func (f *ClientFactory) EventChan() chan watch.Event {
	return f.eventChan
}

func (f *ClientFactory) ClientIfExists(socket string) LauncherClient {
	lock := f.getLock(socket)
	lock.Lock()
	defer lock.Unlock()
	if client, exists := f.clients[socket]; exists {
		return client
	}
	f.removeLock(socket)
	return nil
}

func (f *ClientFactory) ClientForSocket(socket string) (LauncherClient, error) {
	lock := f.getLock(socket)
	lock.Lock()
	defer lock.Unlock()
	if client, exists := f.clients[socket]; exists {
		return client, nil
	}
	client, err := NewClient(socket, f.eventChan, f.recorder)
	if err != nil {
		f.removeLock(socket)
		return nil, err
	}
	if !IsLegacySocket(socket) {
		go client.HandleDomainEvents()
		go client.HandleK8sEvents()
	}
	f.clients[socket] = client
	return client, nil
}

func (f *ClientFactory) removeLock(socket string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	delete(f.locks, socket)
}

func (f *ClientFactory) getLock(socket string) *sync.Mutex {
	f.lock.Lock()
	defer f.lock.Unlock()
	if lock, exists := f.locks[socket]; exists {
		return lock
	}
	f.locks[socket] = &sync.Mutex{}
	return f.locks[socket]
}

func (f *ClientFactory) RemoveClient(socket string) {
	// take both locks, to ensure that no one creates new connections on half-delete clients
	f.lock.Lock()
	defer f.lock.Unlock()
	if lock, exists := f.locks[socket]; exists {
		lock.Lock()
		defer lock.Unlock()
	} else {
		return
	}

	if client, exists := f.clients[socket]; exists {
		// async to not block by accident
		go client.Close()
		delete(f.clients, socket)
	}

	delete(f.locks, socket)
}

type VMIClientFactoryImpl struct {
	ClientFactory *ClientFactory
	socketForUID  map[string]string
	lock          sync.Mutex
}

func (f *VMIClientFactoryImpl) RemoveClientForVMI(vmi *v1.VirtualMachineInstance) {
	if vmi.UID == "" {
		return
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	if socket, exists := f.socketForUID[string(vmi.UID)]; exists {
		f.ClientFactory.RemoveClient(socket)
	}
	delete(f.socketForUID, string(vmi.UID))
}

func (f *VMIClientFactoryImpl) ClientForVMIIfExists(vmi *v1.VirtualMachineInstance) (LauncherClient, string, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	socket, exists := f.socketForUID[string(vmi.UID)]
	if !exists {
		return nil, "", false
	}
	client := f.ClientFactory.ClientIfExists(socket)
	if client != nil {
		return client, "", true
	}
	return nil, socket, false
}

func (f *VMIClientFactoryImpl) ClientForVMI(vmi *v1.VirtualMachineInstance) (LauncherClient, string, error) {
	if vmi.UID == "" {
		return nil, "", fmt.Errorf("VMI %s/%s has no UID", vmi.Namespace, vmi.Name)
	}
	socket := func() string {
		f.lock.Lock()
		defer f.lock.Unlock()
		if socket, exists := f.socketForUID[string(vmi.UID)]; exists {
			return socket
		}
		return ""
	}()

	if socket == "" {
		var err error
		socket, err = FindSocketOnHost(vmi)
		if err != nil {
			return nil, "", fmt.Errorf("No client socket for vmi %s/%s found", vmi.Namespace, vmi.Name)
		}
	}

	client, err := f.ClientFactory.ClientForSocket(socket)
	if err != nil {
		return client, "", err
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	f.socketForUID[string(vmi.UID)] = socket
	return client, socket, nil
}

func NewVMIClientFactory(factory *ClientFactory) *VMIClientFactoryImpl {
	return &VMIClientFactoryImpl{
		ClientFactory: factory,
		socketForUID:  map[string]string{},
		lock:          sync.Mutex{},
	}
}

type VMIClientFactory interface {
	ReadOnlyVMIClientFactory
	RemoveClientForVMI(vmi *v1.VirtualMachineInstance)
	// ClientForVMI returns an existing connection or tries to establish a new connection
	ClientForVMI(vmi *v1.VirtualMachineInstance) (LauncherClient, string, error)
}

type ReadOnlyVMIClientFactory interface {
	// ClientForVMIIfExists will return a client connection if one exists
	ClientForVMIIfExists(vmi *v1.VirtualMachineInstance) (LauncherClient, string, bool)
}

type fakeConnection struct {
	Socket string
	client LauncherClient
	err    error
}

type FakeVMIClientFactory struct {
	fakeConnections map[string]fakeConnection
	lock            sync.Mutex
}

func NewFakeVMIClientFactory() *FakeVMIClientFactory {
	return &FakeVMIClientFactory{
		fakeConnections: map[string]fakeConnection{},
	}
}

func (f *FakeVMIClientFactory) RemoveClientForVMI(vmi *v1.VirtualMachineInstance) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if conn, exists := f.fakeConnections[string(vmi.UID)]; exists {
		conn.client.Close()
	}
	delete(f.fakeConnections, string(vmi.UID))
}

func (f *FakeVMIClientFactory) ClientForVMIIfExists(vmi *v1.VirtualMachineInstance) (LauncherClient, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if conn, exists := f.fakeConnections[string(vmi.UID)]; exists {
		return conn.client, true
	}
	return nil, false
}

func (f *FakeVMIClientFactory) ClientForVMI(vmi *v1.VirtualMachineInstance) (LauncherClient, string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if conn, exists := f.fakeConnections[string(vmi.UID)]; exists {
		return conn.client, conn.Socket, conn.err
	}
	return nil, "", fmt.Errorf("no client")
}

func (f *FakeVMIClientFactory) AddFakeClient(uid types.UID, client LauncherClient, socket string, err error) {
	f.fakeConnections[string(uid)] = fakeConnection{
		Socket: socket,
		client: client,
		err:    err,
	}
}
