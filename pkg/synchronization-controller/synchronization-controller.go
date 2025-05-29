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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package synchronization

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

const (
	defaultTimeout = 30

	MyPodIP = "MY_POD_IP"

	noSourceStatusErrorMsg               = "must pass source status"
	noTargetStatusErrorMsg               = "must pass target status"
	unableToLocateVMIMigrationIDErrorMsg = "unable to locate VMI for migrationID %s"

	successMessage = "success"
)

type SynchronizationController struct {
	client   kubecli.KubevirtClient
	stopChan chan struct{}
	errChan  chan error
	connChan chan io.ReadWriteCloser

	vmiInformer       cache.SharedIndexInformer
	migrationInformer cache.SharedIndexInformer

	listener        net.Listener
	bindAddress     string
	bindPort        int
	clientTLSConfig *tls.Config
	serverTLSConfig *tls.Config
	timeout         int

	queue     workqueue.TypedRateLimitingInterface[string]
	hasSynced func() bool

	syncOutboundConnectionMap  *sync.Map
	syncReceivingConnectionMap *sync.Map
	grpcServer                 *grpc.Server
}

func NewSynchronizationController(
	client kubecli.KubevirtClient,
	vmiInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	clientTLSConfig,
	serverTLSConfig *tls.Config,
	bindAddress string,
	bindPort int,
) (*SynchronizationController, error) {
	syncController := &SynchronizationController{
		vmiInformer:       vmiInformer,
		migrationInformer: migrationInformer,
		clientTLSConfig:   clientTLSConfig,
		serverTLSConfig:   serverTLSConfig,
		timeout:           defaultTimeout,
		bindAddress:       bindAddress,
		bindPort:          bindPort,
		client:            client,
	}

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "sync-vmi-status"},
	)
	syncController.queue = queue

	syncController.hasSynced = func() bool {
		return vmiInformer.HasSynced()
	}

	syncController.syncOutboundConnectionMap = &sync.Map{}
	syncController.syncReceivingConnectionMap = &sync.Map{}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    syncController.addVmiFunc,
		DeleteFunc: syncController.deleteVmiFunc,
		UpdateFunc: syncController.updateVmiFunc,
	})
	if err != nil {
		return nil, err
	}

	if err := syncController.migrationInformer.AddIndexers(map[string]cache.IndexFunc{
		"byUID":               indexByMigrationUID,
		"byVMIName":           indexByVmiName,
		"byTargetMigrationID": indexByTargetMigrationID,
		"bySourceMigrationID": indexBySourceMigrationID,
	}); err != nil {
		return nil, err
	}

	if _, err := syncController.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    syncController.addMigrationFunc,
		DeleteFunc: syncController.deleteMigrationFunc,
		UpdateFunc: syncController.updateMigrationFunc,
	}); err != nil {
		return nil, err
	}

	syncController.grpcServer = grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLSConfig)))
	syncv1.RegisterSynchronizeServer(syncController.grpcServer, syncController)

	return syncController, nil
}

func (s *SynchronizationController) addVmiFunc(addObj interface{}) {
	s.enqueueVirtualMachineInstance(addObj)
}

func (s *SynchronizationController) deleteVmiFunc(addObj interface{}) {
	s.enqueueVirtualMachineInstance(addObj)
}

func (s *SynchronizationController) updateVmiFunc(_, curr interface{}) {
	s.enqueueVirtualMachineInstance(curr)
}

func (s *SynchronizationController) enqueueVirtualMachineInstance(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)
	if ok {
		key, err := controller.KeyFunc(vmi)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("failed to extract key from virtualmachine.")
			return
		}
		s.queue.Add(key)
	}
}

func (s *SynchronizationController) addMigrationFunc(addObj interface{}) {
	s.enqueueVirtualMachineInstanceFromMigration(addObj)
}

func (s *SynchronizationController) deleteMigrationFunc(delObj interface{}) {
	// Clean up any synchronization connections in the map.
	s.enqueueVirtualMachineInstanceFromMigration(delObj)
	// Close any connections associated with this migration.
	migration, ok := delObj.(*virtv1.VirtualMachineInstanceMigration)
	if ok {
		if !migration.IsDecentralized() {
			return
		}
		if migration.Spec.Receive != nil {
			s.closeConnectionForMigrationID(s.syncReceivingConnectionMap, migration.Spec.Receive.MigrationID)
		} else if migration.Spec.SendTo != nil {
			s.closeConnectionForMigrationID(s.syncOutboundConnectionMap, migration.Spec.SendTo.MigrationID)
		}
	}
}

func (s *SynchronizationController) closeConnectionForMigrationID(syncMap *sync.Map, migrationID string) {
	obj, loaded := syncMap.LoadAndDelete(migrationID)
	if loaded {
		log.Log.V(4).Infof("closing connection associated with migrationID %s", migrationID)
		outboundConnection, ok := obj.(*SynchronizationConnection)
		if ok {
			if err := outboundConnection.Close(); err != nil {
				log.Log.Warningf("unable to close connection for migrationID %s, %v", migrationID, err)
			}
		} else {
			log.Log.Warningf("unable to close connection for migrationID %s, type is %v", migrationID, obj)
		}
	}
}

func (s *SynchronizationController) updateMigrationFunc(_, curr interface{}) {
	s.enqueueVirtualMachineInstanceFromMigration(curr)
}

func (s *SynchronizationController) enqueueVirtualMachineInstanceFromMigration(obj interface{}) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
	if ok {
		key := controller.NamespacedKey(migration.Namespace, migration.Spec.VMIName)
		s.queue.Add(key)
	}
}

func (s *SynchronizationController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer s.queue.ShutDown()
	defer s.grpcServer.Stop()
	defer s.closeConnections()

	log.Log.Info("Starting vmi controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, s.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(s.runWorker, time.Second, stopCh)
	}

	conn, err := s.createTcpListener()
	if err != nil {
		s.errChan <- err
	} else {
		go func() {
			s.grpcServer.Serve(conn)
		}()
	}

	<-stopCh
	log.Log.Info("stopping vmi status synchronization controller.")
}

func (s *SynchronizationController) closeConnections() {
	if s.listener != nil {
		s.listener.Close()
	}
	s.syncOutboundConnectionMap.Range(closeMapConnections)
	s.syncReceivingConnectionMap.Range(closeMapConnections)
}

func closeMapConnections(k, obj interface{}) bool {
	outboundConnection, ok := obj.(*SynchronizationConnection)
	if ok {
		if err := outboundConnection.Close(); err != nil {
			log.Log.Warningf("unable to close connection for VMI %s during shutdown, %v", k, err)
		}
	} else {
		log.Log.Warningf("unable to close connection for VMI %s during shutdown", k)
	}
	return true
}

func (s *SynchronizationController) runWorker() {
	for s.Execute() {
	}
}

func (s *SynchronizationController) Execute() bool {
	key, quit := s.queue.Get()
	if quit {
		return false
	}

	defer s.queue.Done(key)
	err := s.execute(key)

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		s.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		s.queue.Forget(key)
	}
	return true
}

func (s *SynchronizationController) execute(key string) error {
	// Fetch the latest VMI state from cache
	obj, exists, _ := s.vmiInformer.GetStore().GetByKey(key)
	if !exists {
		return nil
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	migration, err := s.getMigrationForVMI(vmi)
	if err != nil {
		return err
	}
	if migration != nil && migration.IsDecentralized() {
		if migration.IsSource() {
			if err := s.handleSourceState(vmi); err != nil {
				return err
			}
		}
		if migration.IsTarget() {
			return s.handleTargetState(vmi)
		}
		return nil
	} else {
		// No migration found don't do anything
		log.Log.Object(vmi).V(4).Info("no decentralized migration found for VMI")
		return nil
	}
}

func (s *SynchronizationController) getMigrationIDFromUID(migrationUID types.UID) (string, error) {
	objs, err := s.migrationInformer.GetIndexer().ByIndex("byUID", string(migrationUID))
	if err != nil {
		return "", err
	}
	if len(objs) > 1 {
		return "", fmt.Errorf("found more than one migration with same UID")
	}
	if len(objs) == 0 {
		return "", nil
	}
	migration, ok := objs[0].(*virtv1.VirtualMachineInstanceMigration)
	if !ok {
		return "", fmt.Errorf("found unknown object in migration cache")
	}
	var migrationID string
	if migration.Spec.Receive != nil {
		migrationID = migration.Spec.Receive.MigrationID
	}
	if migration.Spec.SendTo != nil {
		migrationID = migration.Spec.SendTo.MigrationID
	}
	return migrationID, nil
}

func (s *SynchronizationController) getOutboundSourceConnection(vmi *virtv1.VirtualMachineInstance, migrationState *virtv1.VirtualMachineInstanceMigrationState) (*SynchronizationConnection, error) {
	if migrationState.TargetState.SyncAddress == nil || *migrationState.TargetState.SyncAddress == "" {
		return nil, nil
	}
	return s.getOutboundConnection(vmi, migrationState.SourceState.MigrationUID, *migrationState.TargetState.SyncAddress, s.syncOutboundConnectionMap)
}

func (s *SynchronizationController) getOutboundTargetConnection(vmi *virtv1.VirtualMachineInstance, migrationState *virtv1.VirtualMachineInstanceMigrationState) (*SynchronizationConnection, error) {
	if migrationState.SourceState.SyncAddress == nil || *migrationState.SourceState.SyncAddress == "" {
		return nil, nil
	}
	return s.getOutboundConnection(vmi, migrationState.TargetState.MigrationUID, *migrationState.SourceState.SyncAddress, s.syncReceivingConnectionMap)
}

func (s *SynchronizationController) getOutboundConnection(vmi *virtv1.VirtualMachineInstance, migrationUID types.UID, syncAddress string, connectionMap *sync.Map) (*SynchronizationConnection, error) {
	if migrationUID == "" {
		return nil, nil
	}
	migrationID, err := s.getMigrationIDFromUID(migrationUID)
	if err != nil {
		return nil, err
	}
	log.Log.Object(vmi).V(4).Infof("found migration ID %s", migrationID)
	obj, ok := connectionMap.Load(migrationID)
	if !ok {
		grpcClientConnection, err := s.createOutboundConnection(syncAddress)
		if err != nil {
			return nil, err
		}
		conn := &SynchronizationConnection{
			migrationID:          migrationID,
			grpcClientConnection: grpcClientConnection,
		}
		connectionMap.Store(migrationID, conn)
		return conn, nil
	}
	outboundSyncConnection, ok := obj.(*SynchronizationConnection)
	if !ok {
		return nil, fmt.Errorf("found unknown object in outbound connection cache %#v", outboundSyncConnection)
	}
	return outboundSyncConnection, nil
}

func (s *SynchronizationController) handleSourceState(vmi *virtv1.VirtualMachineInstance) error {
	var outboundConnection *SynchronizationConnection
	var err error
	if vmi.Status.MigrationState == nil {
		// No migration state, don't do anything
		return nil
	}
	if vmi.Status.MigrationState.SourceState == nil || vmi.Status.MigrationState.TargetState == nil {
		// No migration state, don't do anything
		return nil
	}
	sourceState := vmi.Status.MigrationState.SourceState
	if sourceState.SyncAddress == nil || *sourceState.SyncAddress == "" {
		syncAddress, err := s.getMyURL()
		if err != nil {
			return err
		}
		sourceState.SyncAddress = &syncAddress
	}
	targetState := vmi.Status.MigrationState.TargetState
	if targetState.SyncAddress != nil && sourceState.MigrationUID != "" {
		if outboundConnection, err = s.getOutboundSourceConnection(vmi, vmi.Status.MigrationState); err != nil {
			return err
		}
	}
	if outboundConnection == nil {
		log.Log.Object(vmi).V(4).Info("no synchronization connection found for source, doing nothing")
		return nil
	}
	vmiStatusJson, err := json.Marshal(vmi.Status)
	if err != nil {
		return err
	}
	client := syncv1.NewSynchronizeClient(outboundConnection.grpcClientConnection)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeout)*time.Second)
	defer cancel()

	if _, err := client.SyncSourceMigrationStatus(ctx, &syncv1.VMIStatusRequest{
		MigrationID: outboundConnection.migrationID,
		VmiStatus: &syncv1.VMIStatus{
			VmiStatusJson: vmiStatusJson,
		},
	}); err != nil {
		return err
	}
	return nil
}

func (s *SynchronizationController) handleTargetState(vmi *virtv1.VirtualMachineInstance) error {
	if vmi.Status.MigrationState == nil {
		// No migration state, don't do anything
		return nil
	}
	if vmi.Status.MigrationState.TargetState == nil || vmi.Status.MigrationState.SourceState == nil {
		// No migration state, don't do anything
		return nil
	}

	var outboundConnection *SynchronizationConnection
	var err error
	sourceState := vmi.Status.MigrationState.SourceState
	targetState := vmi.Status.MigrationState.TargetState
	if targetState.SyncAddress == nil || *targetState.SyncAddress == "" {
		syncAddress, err := s.getMyURL()
		if err != nil {
			return err
		}
		targetState.SyncAddress = &syncAddress
	}

	if sourceState.SyncAddress != nil && targetState.MigrationUID != "" {
		if outboundConnection, err = s.getOutboundTargetConnection(vmi, vmi.Status.MigrationState); err != nil {
			return err
		}
	}
	if outboundConnection == nil {
		log.Log.Object(vmi).V(4).Info("no synchronization connection found for target, doing nothing")
		return nil
	}
	vmiStatusJson, err := json.Marshal(vmi.Status)
	if err != nil {
		return err
	}
	client := syncv1.NewSynchronizeClient(outboundConnection.grpcClientConnection)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeout)*time.Second)
	defer cancel()

	_, err = client.SyncTargetMigrationStatus(ctx, &syncv1.VMIStatusRequest{
		MigrationID: outboundConnection.migrationID,
		VmiStatus: &syncv1.VMIStatus{
			VmiStatusJson: vmiStatusJson,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizationController) getMigrationForVMI(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstanceMigration, error) {
	objects, err := s.migrationInformer.GetIndexer().ByIndex("byVMIName", vmi.Name)
	if err != nil {
		return nil, err
	}
	if len(objects) > 0 {
		count := 0
		var res *virtv1.VirtualMachineInstanceMigration
		for _, migrationObj := range objects {
			migration, ok := migrationObj.(*virtv1.VirtualMachineInstanceMigration)
			if !ok {
				return nil, fmt.Errorf("not a virtual machine instance migration")
			}
			if migration.Namespace == vmi.Namespace {
				count++
				res = migration
			}
		}
		if count > 1 {
			return nil, fmt.Errorf("found more than one migration pointing to same VMI")
		} else if count == 0 {
			return nil, nil
		}
		return res, nil
	}
	return nil, nil
}

func (s *SynchronizationController) getMyURL() (string, error) {
	myIp := os.Getenv(MyPodIP)
	if myIp != "" {
		names, err := net.LookupAddr(myIp)
		if err != nil {
			return "", err
		}
		for _, name := range names {
			log.Log.V(4).Infof("found DNS name for my IP address: %s", name)
			return fmt.Sprintf("%s:%d", name, s.bindPort), nil
		}
		return fmt.Sprintf("%s:%d", myIp, s.bindPort), nil
	}
	if s.listener == nil {
		return fmt.Sprintf("%s:%d", s.bindAddress, s.bindPort), nil
	}
	// TODO figure out how to get my URL with or without submariner (url changes based on export)
	return s.listener.Addr().String(), nil
}

func (s *SynchronizationController) createOutboundConnection(connectionURL string) (*grpc.ClientConn, error) {
	logger := log.Log.With("outbound", connectionURL)
	logger.Info("creating new synchronization grpc connection")

	client, err := grpc.NewClient(connectionURL, grpc.WithTransportCredentials(credentials.NewTLS(s.clientTLSConfig)))
	return client, err
}

func (s *SynchronizationController) createTcpListener() (net.Listener, error) {
	if s.listener != nil {
		return s.listener, nil
	}
	var ln net.Listener
	var err error
	addr := net.JoinHostPort(s.bindAddress, strconv.Itoa(s.bindPort))
	ln, err = net.Listen("tcp", addr)
	if err != nil {
		log.Log.Reason(err).Error("failed to create tcp listener")
		return nil, err
	}
	s.listener = ln
	return ln, nil
}

func (s *SynchronizationController) findTargetMigrationFromMigrationID(migrationID string) (*virtv1.VirtualMachineInstanceMigration, error) {
	return s.findMigrationFromMigrationIDByIndex("byTargetMigrationID", migrationID)
}

func (s *SynchronizationController) findSourceMigrationFromMigrationID(migrationID string) (*virtv1.VirtualMachineInstanceMigration, error) {
	return s.findMigrationFromMigrationIDByIndex("bySourceMigrationID", migrationID)
}

func (s *SynchronizationController) findMigrationFromMigrationIDByIndex(indexName, migrationID string) (*virtv1.VirtualMachineInstanceMigration, error) {
	objs, err := s.migrationInformer.GetIndexer().ByIndex(indexName, migrationID)
	if err != nil {
		return nil, err
	}
	if len(objs) > 1 {
		log.Log.Warningf("found multiple migrations for migrationID %s, picking first one", migrationID)
	}
	for _, obj := range objs {
		migration, _ := obj.(*virtv1.VirtualMachineInstanceMigration)
		return migration, nil
	}
	return nil, nil
}

func (s *SynchronizationController) SyncSourceMigrationStatus(ctx context.Context, request *syncv1.VMIStatusRequest) (*syncv1.VMIStatusResponse, error) {
	if request.VmiStatus == nil || len(request.VmiStatus.VmiStatusJson) == 0 {
		return &syncv1.VMIStatusResponse{
			Message: noSourceStatusErrorMsg,
		}, fmt.Errorf(noSourceStatusErrorMsg)
	}
	migration, err := s.findTargetMigrationFromMigrationID(request.MigrationID)
	if migration == nil {
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID),
		}, fmt.Errorf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID)
	}
	key := controller.NamespacedKey(migration.Namespace, migration.Spec.VMIName)
	log.Log.Object(migration).V(5).Infof("looking up VMI %s", key)
	obj, exists, err := s.vmiInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		if err == nil {
			err = fmt.Errorf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID)
		}
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID),
		}, err
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	remoteStatus := &virtv1.VirtualMachineInstanceStatus{}
	if err := json.Unmarshal(request.VmiStatus.VmiStatusJson, remoteStatus); err != nil {
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf("unable to unmarshal vmistatus for migrationID %s", request.MigrationID),
		}, err
	}
	if remoteStatus.MigrationState == nil {
		return &syncv1.VMIStatusResponse{
			Message: noSourceStatusErrorMsg,
		}, fmt.Errorf(noSourceStatusErrorMsg)
	}
	origVMI := vmi.DeepCopy()
	if vmi.Status.MigrationState == nil {
		vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{}
	}
	log.Log.Object(vmi).V(5).Infof("vmi migration source state: %#v", vmi.Status.MigrationState.SourceState)
	log.Log.Object(vmi).V(5).Infof("remote migration source state: %#v", remoteStatus.MigrationState.SourceState)
	vmi.Status.MigrationState.SourceState = remoteStatus.MigrationState.SourceState.DeepCopy()
	if !apiequality.Semantic.DeepEqual(origVMI.Status, vmi.Status) {
		if err := s.patchVMI(origVMI, vmi); err != nil {
			return &syncv1.VMIStatusResponse{
				Message: fmt.Sprintf("unable to synchronize VMI for migrationID %s", request.MigrationID),
			}, err
		}
		log.Log.Object(vmi).With("MigrationID", request.MigrationID).V(5).Info("successfully patched VMI with source state")
	}
	log.Log.Object(vmi).V(5).Info("returning success to grpc caller, source")
	return &syncv1.VMIStatusResponse{
		Message: successMessage,
	}, nil
}

func (s *SynchronizationController) SyncTargetMigrationStatus(ctx context.Context, request *syncv1.VMIStatusRequest) (*syncv1.VMIStatusResponse, error) {
	if request.VmiStatus == nil || len(request.VmiStatus.VmiStatusJson) == 0 {
		return &syncv1.VMIStatusResponse{
			Message: noTargetStatusErrorMsg,
		}, fmt.Errorf(noTargetStatusErrorMsg)
	}

	migration, err := s.findSourceMigrationFromMigrationID(request.MigrationID)
	if migration == nil {
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID),
		}, fmt.Errorf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID)
	}

	key := controller.NamespacedKey(migration.Namespace, migration.Spec.VMIName)
	obj, exists, err := s.vmiInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		if err == nil {
			err = fmt.Errorf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID)
		}
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf(unableToLocateVMIMigrationIDErrorMsg, request.MigrationID),
		}, err
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	remoteStatus := &virtv1.VirtualMachineInstanceStatus{}
	if err := json.Unmarshal(request.VmiStatus.VmiStatusJson, remoteStatus); err != nil {
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf("unable to unmarshal vmistatus for migrationID %s", request.MigrationID),
		}, err
	}
	if remoteStatus.MigrationState == nil {
		return &syncv1.VMIStatusResponse{
			Message: noTargetStatusErrorMsg,
		}, fmt.Errorf(noTargetStatusErrorMsg)
	}
	origVMI := vmi.DeepCopy()
	if vmi.Status.MigrationState == nil {
		vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{}
	}

	log.Log.Object(vmi).V(5).Infof("vmi migration target state: %#v", vmi.Status.MigrationState.TargetState)
	log.Log.Object(vmi).V(5).Infof("remote migration target state: %#v", remoteStatus.MigrationState.TargetState)
	vmi.Status.MigrationState.TargetState = remoteStatus.MigrationState.TargetState.DeepCopy()
	if !apiequality.Semantic.DeepEqual(origVMI.Status, vmi.Status) {
		if err := s.patchVMI(origVMI, vmi); err != nil {
			return &syncv1.VMIStatusResponse{
				Message: fmt.Sprintf("unable to synchronize VMI for migrationID %s", request.MigrationID),
			}, err
		}
		log.Log.Object(vmi).With("MigrationID", request.MigrationID).V(5).Info("successfully patched VMI with target state")
	}
	log.Log.Object(vmi).V(5).Info("returning success to grpc caller, target")
	return &syncv1.VMIStatusResponse{
		Message: successMessage,
	}, nil
}

func (s *SynchronizationController) patchVMI(origVMI, newVMI *virtv1.VirtualMachineInstance) error {
	patchSet := patch.New()

	if origVMI.Status.MigrationState == nil {
		patchSet.AddOption(
			patch.WithAdd("/status/migrationState", newVMI.Status.MigrationState))
	} else {
		if !apiequality.Semantic.DeepEqual(origVMI.Status.MigrationState.SourceState, newVMI.Status.MigrationState.SourceState) {
			if origVMI.Status.MigrationState.SourceState == nil {
				patchSet.AddOption(
					patch.WithTest("/status/migrationState/sourceState", origVMI.Status.MigrationState.SourceState),
					patch.WithAdd("/status/migrationState/sourceState", newVMI.Status.MigrationState.SourceState))
			} else {
				patchSet.AddOption(
					patch.WithTest("/status/migrationState/sourceState", origVMI.Status.MigrationState.SourceState),
					patch.WithReplace("/status/migrationState/sourceState", newVMI.Status.MigrationState.SourceState),
				)
			}
		}
		if !apiequality.Semantic.DeepEqual(origVMI.Status.MigrationState.TargetState, newVMI.Status.MigrationState.TargetState) {
			if origVMI.Status.MigrationState.TargetState == nil {
				patchSet.AddOption(
					patch.WithTest("/status/migrationState/targetState", origVMI.Status.MigrationState.TargetState),
					patch.WithAdd("/status/migrationState/targetState", newVMI.Status.MigrationState.TargetState))
			} else {
				patchSet.AddOption(
					patch.WithTest("/status/migrationState/targetState", origVMI.Status.MigrationState.TargetState),
					patch.WithReplace("/status/migrationState/targetState", newVMI.Status.MigrationState.TargetState),
				)
			}
		}
	}

	if !patchSet.IsEmpty() {
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return err
		}
		log.Log.Object(origVMI).Infof("patch VMI with %s", string(patchBytes))
		if _, err := s.client.VirtualMachineInstance(origVMI.Namespace).Patch(context.Background(), origVMI.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func indexByMigrationUID(obj interface{}) ([]string, error) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
	if !ok {
		return nil, nil
	}
	return []string{string(migration.UID)}, nil
}

func indexByVmiName(obj interface{}) ([]string, error) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
	if !ok {
		return nil, nil
	}
	return []string{migration.Spec.VMIName}, nil
}

func indexByTargetMigrationID(obj interface{}) ([]string, error) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
	if !ok {
		return nil, nil
	}
	if migration.Spec.Receive != nil {
		return []string{migration.Spec.Receive.MigrationID}, nil
	}
	return []string{}, nil
}

func indexBySourceMigrationID(obj interface{}) ([]string, error) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
	if !ok {
		return nil, nil
	}
	if migration.Spec.SendTo != nil {
		return []string{migration.Spec.SendTo.MigrationID}, nil
	}
	return []string{}, nil
}
