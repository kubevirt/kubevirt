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
 * Copyright The KubeVirt Authors.
 *
 */

package synchronization

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/kubevirt/pkg/controller"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

const (
	defaultTimeout = 30

	MyPodIP = "MY_POD_IP"

	noSourceStatusErrorMsg                        = "must pass source status"
	noTargetStatusErrorMsg                        = "must pass target status"
	sourceUnableToLocateVMIMigrationIDErrorMsg    = "source: unable to locate VMI for migrationID %s"
	targetUnableToLocateVMIMigrationIDErrorMsg    = "target: unable to locate VMI for migrationID %s"
	sourceUnableToLocateVMIMigrationIDErrorMsgVMI = "source: unable to locate VMI for migrationID %s, vmi: %s"
	targetUnableToLocateVMIMigrationIDErrorMsgVMI = "target: unable to locate VMI for migrationID %s, vmi: %s"

	waitingForSyncErrorMessage = "waiting for incoming synchronization, unable to proceed"

	successMessage = "success"

	maxCloseRetries = 10

	SynchronizationFinalizer = "synchronization.kubevirt.io/migrationFinalizer"

	// Migration informer index names. Use these constants for AddIndexers and ByIndex
	// so a typo cannot silently return empty results.
	migrationIndexByUID               = "byUID"
	migrationIndexByActiveVMIName     = "byActiveVMIName"
	migrationIndexByTargetMigrationID = "byTargetMigrationID"
	migrationIndexBySourceMigrationID = "bySourceMigrationID"
)

type SynchronizationController struct {
	client kubecli.KubevirtClient

	vmiInformer       cache.SharedIndexInformer
	migrationInformer cache.SharedIndexInformer

	listener                 net.Listener
	bindAddress              string
	bindPort                 int
	ip                       string
	clientTLSConfig          *tls.Config
	serverTLSConfig          *tls.Config
	migrationClientTLSConfig *tls.Config
	migrationServerTLSConfig *tls.Config
	timeout                  int

	queue     workqueue.TypedRateLimitingInterface[string]
	hasSynced func() bool

	syncOutboundConnectionMap  *sync.Map
	syncReceivingConnectionMap *sync.Map
	failedCloseConnections     *sync.Map
	grpcServer                 *grpc.Server

	tunnelManager *MigrationTunnelManager
}

// ProxyInitConfig selects whether the migration-data proxy is enabled and which
// secondary interfaces are required (vs falling back to the pod IP).
type ProxyInitConfig struct {
	Enabled                      bool
	RequireMigrationInterface    bool
	RequireCrossClusterInterface bool
}

func NewSynchronizationController(
	client kubecli.KubevirtClient,
	vmiInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	clientTLSConfig,
	serverTLSConfig,
	migrationClientTLSConfig,
	migrationServerTLSConfig *tls.Config,
	bindAddress string,
	bindPort int,
	ip string,
	proxyConfig *ProxyInitConfig,
) (*SynchronizationController, error) {
	syncController := &SynchronizationController{
		vmiInformer:              vmiInformer,
		migrationInformer:        migrationInformer,
		clientTLSConfig:          clientTLSConfig,
		serverTLSConfig:          serverTLSConfig,
		migrationClientTLSConfig: migrationClientTLSConfig,
		migrationServerTLSConfig: migrationServerTLSConfig,
		timeout:                  defaultTimeout,
		bindAddress:              bindAddress,
		bindPort:                 bindPort,
		client:                   client,
		ip:                       ip,
	}

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "sync-vmi-status"},
	)
	syncController.queue = queue

	syncController.hasSynced = func() bool {
		return vmiInformer.HasSynced() && migrationInformer.HasSynced()
	}

	syncController.syncOutboundConnectionMap = &sync.Map{}
	syncController.syncReceivingConnectionMap = &sync.Map{}
	syncController.failedCloseConnections = &sync.Map{}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    syncController.addVmiFunc,
		DeleteFunc: syncController.deleteVmiFunc,
		UpdateFunc: syncController.updateVmiFunc,
	})
	if err != nil {
		return nil, err
	}

	if err := syncController.migrationInformer.AddIndexers(map[string]cache.IndexFunc{
		"byActiveVMIName":     indexByActiveVmiName,
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

	// Initialize migration tunnel manager for terminating TLS with virt-handlers
	syncController.tunnelManager = NewMigrationTunnelManager(migrationClientTLSConfig, migrationServerTLSConfig)

	if ip == "" && bindAddress == "" {
		return nil, fmt.Errorf("synchronization controller requires a pod IP or bind address")
	}
	podIP := ip
	if podIP == "" {
		podIP = bindAddress
	}

	if proxyConfig != nil && proxyConfig.Enabled {
		// Proxy was explicitly selected: fail fast on misconfiguration (e.g. a
		// required Multus interface is missing) rather than starting without the
		// tunnel and causing opaque migration failures later.
		if err := syncController.initMigrationProxy(podIP, bindAddress, bindPort, proxyConfig); err != nil {
			return nil, fmt.Errorf("migration proxy required but failed to initialize: %w", err)
		}
	} else {
		if migrationIP, err := interfaceIP(virtv1.MigrationInterfaceName); err == nil {
			syncController.bindAddress = migrationIP
			log.Log.Infof("gRPC server will bind to migration network: %s:%d", migrationIP, bindPort)
		} else if ip != "" {
			// Match historic Direct behavior: prefer pod IP over 0.0.0.0 when no
			// dedicated migration interface is present.
			syncController.bindAddress = ip
			log.Log.Infof("gRPC server will bind to pod IP: %s:%d", ip, bindPort)
		} else {
			log.Log.Infof("gRPC server will bind to: %s:%d", bindAddress, bindPort)
		}
		log.Log.V(2).Info("Decentralized live migration datapath is Direct; migration proxy disabled")
	}

	return syncController, nil
}

func (s *SynchronizationController) initMigrationProxy(podIP, fallbackBind string, bindPort int, cfg *ProxyInitConfig) error {
	migrationIP, err := resolveProxyAddress(virtv1.MigrationInterfaceName, podIP, cfg.RequireMigrationInterface)
	if err != nil {
		return fmt.Errorf("local migration address: %w", err)
	}
	peerIP, err := resolveProxyAddress(virtv1.CrossClusterMigrationInterfaceName, podIP, cfg.RequireCrossClusterInterface)
	if err != nil {
		return fmt.Errorf("peer sync address: %w", err)
	}

	s.tunnelManager.Initialize(migrationIP, peerIP)
	log.Log.Infof("Migration tunnel manager initialized with local=%s peer=%s", migrationIP, peerIP)

	// When a dedicated cross-cluster network is configured, bind only there.
	// Otherwise bind on the pod IP (Service/Ingress reachable).
	if cfg.RequireCrossClusterInterface {
		s.bindAddress = peerIP
		log.Log.Infof("gRPC server will bind to crosscluster network: %s:%d", peerIP, bindPort)
	} else {
		s.bindAddress = podIP
		if s.bindAddress == "" {
			s.bindAddress = fallbackBind
		}
		log.Log.Infof("gRPC server will bind to pod network: %s:%d", s.bindAddress, bindPort)
	}
	return nil
}

// resolveProxyAddress returns the IP on ifaceName when present. If the interface
// is missing and required is false, podIP is used. If required is true, a missing
// or unaddressed interface is an error.
func resolveProxyAddress(ifaceName, podIP string, required bool) (string, error) {
	ip, err := interfaceIP(ifaceName)
	if err == nil {
		return ip, nil
	}
	if required {
		return "", err
	}
	if podIP == "" {
		return "", fmt.Errorf("%s not available and pod IP is empty: %w", ifaceName, err)
	}
	return podIP, nil
}

// IsTunnelInitialized checks if the migration proxy has been initialized with network IPs
func (s *SynchronizationController) IsTunnelInitialized() bool {
	return s.tunnelManager != nil && s.tunnelManager.IsInitialized()
}

// setupTargetProxiesForOutbound starts the target tunnel so inbound per-channel streams
// can be forwarded to virt-handler. The status sent to the source keeps the real
// virt-handler ports (protocol mapping); the source sync controller rewrites its local
// VMI to point virt-handler at source-side listeners.
func (s *SynchronizationController) setupTargetProxiesForOutbound(
	migration *virtv1.VirtualMachineInstanceMigration,
	vmi *virtv1.VirtualMachineInstance,
) error {
	if !s.IsTunnelInitialized() || vmi.Status.MigrationState == nil || vmi.Status.MigrationState.TargetState == nil {
		return nil
	}
	if vmi.Status.MigrationState.TargetState.NodeAddress == nil ||
		vmi.Status.MigrationState.TargetState.DirectMigrationNodePorts == nil {
		return nil
	}
	if migration.Spec.Receive == nil {
		return fmt.Errorf("did not find receiving migration when setting up target proxy")
	}
	return s.startTargetTunnel(migration, vmi)
}

// setupTargetProxiesFromSource starts target tunnel based on received source sync address.
// This is called on the target side when receiving source migration status.
func (s *SynchronizationController) setupTargetProxiesFromSource(
	migration *virtv1.VirtualMachineInstanceMigration,
	vmi *virtv1.VirtualMachineInstance,
	remoteStatus *virtv1.VirtualMachineInstanceStatus,
) error {
	if s.tunnelManager == nil || remoteStatus.MigrationState == nil || remoteStatus.MigrationState.SourceState == nil {
		return nil
	}
	if !s.tunnelManager.IsInitialized() {
		return nil
	}
	if remoteStatus.MigrationState.SourceState.SyncAddress == nil {
		return nil
	}
	log.Log.Object(migration).V(3).Infof("Received source sync address: %s",
		*remoteStatus.MigrationState.SourceState.SyncAddress)

	if vmi.Status.MigrationState == nil ||
		vmi.Status.MigrationState.TargetState == nil ||
		vmi.Status.MigrationState.TargetState.NodeAddress == nil ||
		vmi.Status.MigrationState.TargetState.DirectMigrationNodePorts == nil {
		return nil
	}
	if migration.Spec.Receive == nil {
		return nil
	}
	return s.startTargetTunnel(migration, vmi)
}

// startTargetTunnel starts (or refreshes) the target-side tunnel that dials local
// virt-handler. Callers must ensure Spec.Receive and TargetState dial coordinates
// are present.
func (s *SynchronizationController) startTargetTunnel(
	migration *virtv1.VirtualMachineInstanceMigration,
	vmi *virtv1.VirtualMachineInstance,
) error {
	migrationID := migration.Spec.Receive.MigrationID
	targetIP := *vmi.Status.MigrationState.TargetState.NodeAddress
	targetPorts := vmi.Status.MigrationState.TargetState.DirectMigrationNodePorts

	ports, err := portMapToInt(targetPorts)
	if err != nil {
		log.Log.Object(migration).Reason(err).Error("Failed to convert target virt-handler ports")
		return err
	}

	if _, err := s.tunnelManager.StartTargetTunnel(migrationID, targetIP, ports); err != nil {
		log.Log.Object(migration).Reason(err).Error("Failed to start target tunnel")
		return err
	}

	log.Log.Object(migration).V(3).Infof("Started target tunnel forwarding to virt-handler %s ports %v",
		targetIP, targetPorts)
	return nil
}

// setupSourceProxiesFromTarget starts source tunnel based on received target state
// This is called on the source side when receiving target migration status
func (s *SynchronizationController) setupSourceProxiesFromTarget(
	migration *virtv1.VirtualMachineInstanceMigration,
	vmi *virtv1.VirtualMachineInstance,
	remoteStatus *virtv1.VirtualMachineInstanceStatus,
) error {
	if s.tunnelManager == nil || remoteStatus.MigrationState == nil || remoteStatus.MigrationState.TargetState == nil {
		return nil
	}

	// Check if tunnel manager is initialized
	if !s.tunnelManager.IsInitialized() {
		return nil
	}

	// Extract target virt-handler port map (protocol channels) from received TargetState
	if remoteStatus.MigrationState.TargetState.NodeAddress == nil ||
		remoteStatus.MigrationState.TargetState.DirectMigrationNodePorts == nil {
		return nil
	}

	// Extract migrationID from spec.sendTo.migrationID (source side)
	if migration.Spec.SendTo == nil {
		return nil
	}
	migrationID := migration.Spec.SendTo.MigrationID

	targetVirtHandlerIP := *remoteStatus.MigrationState.TargetState.NodeAddress
	targetVirtHandlerPorts := remoteStatus.MigrationState.TargetState.DirectMigrationNodePorts

	log.Log.Object(migration).V(3).Infof("Received target virt-handler address: %s, ports: %v",
		targetVirtHandlerIP, targetVirtHandlerPorts)

	// Convert from API format (map[string]int) to internal format (map[int]int)
	targetVirtHandlerPortsInt, err := portMapToInt(targetVirtHandlerPorts)
	if err != nil {
		log.Log.Object(migration).Reason(err).Error("Failed to convert target virt-handler ports")
		return err
	}

	// Prefer the remote SyncAddress when dialing — local status may still be stale
	// because TargetState is copied onto the VMI only after this setup completes.
	migrationState := vmi.Status.MigrationState.DeepCopy()
	if remoteStatus.MigrationState.TargetState.SyncAddress != nil &&
		*remoteStatus.MigrationState.TargetState.SyncAddress != "" {
		if migrationState.TargetState == nil {
			migrationState.TargetState = &virtv1.VirtualMachineInstanceMigrationTargetState{}
		}
		migrationState.TargetState.SyncAddress = remoteStatus.MigrationState.TargetState.SyncAddress
	}

	conn, err := s.getOutboundSourceConnection(vmi, migrationState)
	if err != nil {
		log.Log.Object(migration).Reason(err).Error("Failed to get gRPC connection to target sync controller")
		return err
	}
	if conn == nil {
		return fmt.Errorf("no outbound gRPC connection to target sync controller yet")
	}

	// Start source tunnel: listeners on the internal migration network (Multus or
	// pod IP); each accepted connection opens its own MigrationTunnel stream on
	// the shared control-plane gRPC connection.
	tunnel, err := s.tunnelManager.StartSourceTunnel(migrationID, conn.grpcClientConnection, targetVirtHandlerPortsInt)
	if err != nil {
		log.Log.Object(migration).Reason(err).Error("Failed to start source tunnel")
		return err
	}

	sourceTunnelPorts := tunnel.GetListenerPorts()
	migrationIP := s.tunnelManager.MigrationIP()

	log.Log.Object(migration).V(3).Infof("Started source tunnel on internal migration network (%s) ports: %v",
		migrationIP, sourceTunnelPorts)

	// Rewrite local VMI so source virt-handler dials the source sync controller listeners
	vmi.Status.MigrationState.TargetNodeAddress = migrationIP
	vmi.Status.MigrationState.TargetDirectMigrationNodePorts = portMapToString(sourceTunnelPorts)

	log.Log.Object(migration).V(3).Infof("Writing source sync internal migration network address to local VMI: %s, ports: %v",
		migrationIP, sourceTunnelPorts)

	return nil
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
			migrationID := migration.Spec.Receive.MigrationID
			log.Log.V(4).Object(migration).Infof("closing receiving connection for migrationID %s", migrationID)
			if err := s.closeConnectionForMigrationID(s.syncReceivingConnectionMap, migrationID); err != nil {
				log.Log.Reason(err).Infof("unable to close connection for migrationID %s, possibly leaked connection", migrationID)
			}

			// Clean up target tunnel only if migration is actually being deleted (not just temporarily gone from cache)
			// DeletionTimestamp is set when the object is truly being deleted
			if s.tunnelManager != nil && migration.DeletionTimestamp != nil {
				log.Log.V(4).Object(migration).Infof("stopping target tunnel for migration %s", migrationID)
				s.tunnelManager.StopTunnel(migrationID)
			}
		} else if migration.Spec.SendTo != nil {
			migrationID := migration.Spec.SendTo.MigrationID
			log.Log.V(4).Object(migration).Infof("closing outbound connection for migrationID %s", migrationID)
			if err := s.closeConnectionForMigrationID(s.syncOutboundConnectionMap, migrationID); err != nil {
				log.Log.Reason(err).Infof("unable to close connection for migrationID %s, possibly leaked connection", migrationID)
			}

			// Clean up source tunnel only if migration is actually being deleted (not just temporarily gone from cache)
			// DeletionTimestamp is set when the object is truly being deleted
			if s.tunnelManager != nil && migration.DeletionTimestamp != nil {
				log.Log.V(4).Object(migration).Infof("stopping source tunnel for migration %s", migrationID)
				s.tunnelManager.StopTunnel(migrationID)
			}
		}
	}
}

func (s *SynchronizationController) closeConnectionForMigrationID(syncMap *sync.Map, migrationID string) error {
	obj, loaded := syncMap.LoadAndDelete(migrationID)
	if loaded {
		log.Log.V(4).Infof("closing connection associated with migrationID %s", migrationID)
		outboundConnection, ok := obj.(*SynchronizationConnection)
		if ok {
			if err := outboundConnection.Close(); err != nil {
				log.Log.Warningf("unable to close connection for migrationID %s, %v", migrationID, err)
				s.failedCloseConnections.Store(outboundConnection, 0)
				return err
			}
		} else {
			log.Log.Warningf("unable to close connection for migrationID %s, type is %v", migrationID, obj)
			return fmt.Errorf("unknown type %v", obj)
		}
	}
	return nil
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

func (s *SynchronizationController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer controller.HandlePanic()
	defer s.queue.ShutDown()
	defer s.closeConnections()

	log.Log.Info("starting vmi status synchronization controller.")

	// Derive a root context once from stopCh and pass it into workers so cancel
	// is synchronized without a shared runCtx field.
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	go func() {
		select {
		case <-stopCh:
			runCancel()
		case <-runCtx.Done():
		}
	}()

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, s.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(func() { s.runWorker(runCtx) }, time.Second, stopCh)
	}
	go wait.Until(s.runConnectionCleanup, 5*time.Second, stopCh)

	conn, err := s.createTcpListener()
	if err != nil {
		log.Log.Criticalf("received error %v, exiting", err)
		return err
	} else {
		go func() {
			s.grpcServer.Serve(conn)
		}()
	}
	if err := s.rebuildConnectionsAndUpdateSyncAddress(); err != nil {
		return err
	}

	log.Log.Info("waiting on stop signal")
	<-stopCh
	log.Log.Info("normally stopping vmi status synchronization controller.")
	return nil
}

func (s *SynchronizationController) closeConnections() {
	log.Log.V(1).Info("closing listener and grpcserver")
	if s.listener != nil {
		s.listener.Close()
	}
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	log.Log.V(1).Infof("closing outbound connections")
	s.syncOutboundConnectionMap.Range(closeMapConnections)
	log.Log.V(1).Infof("closing inbound connections")
	s.syncReceivingConnectionMap.Range(closeMapConnections)
	log.Log.V(1).Infof("shutting down tunnel manager")
	if s.tunnelManager != nil {
		s.tunnelManager.Shutdown()
	}
}

func closeMapConnections(k, obj interface{}) bool {
	outboundConnection, ok := obj.(*SynchronizationConnection)
	if ok && outboundConnection != nil {
		log.Log.V(1).Infof("closing connection for migration ID: %s", outboundConnection.migrationID)
		if err := outboundConnection.Close(); err != nil {
			log.Log.Warningf("unable to close connection for VMI %s during shutdown, %v", k, err)
		}
	} else {
		log.Log.Warningf("unable to close connection for VMI %s during shutdown", k)
	}
	return true
}

func (s *SynchronizationController) runWorker(ctx context.Context) {
	for s.processNextWorkItem(ctx) {
	}
}

func (s *SynchronizationController) Execute() bool {
	return s.processNextWorkItem(context.Background())
}

func (s *SynchronizationController) processNextWorkItem(parent context.Context) bool {
	key, quit := s.queue.Get()
	if quit {
		return false
	}

	defer s.queue.Done(key)
	err := s.execute(parent, key)

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		s.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		s.queue.Forget(key)
	}
	return true
}

func (s *SynchronizationController) execute(parent context.Context, key string) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Fetch the latest VMI state from cache
	obj, exists, _ := s.vmiInformer.GetStore().GetByKey(key)
	if !exists {
		// VMI doesn't exist, but we still need to handle migration finalizers
		// for migrations that are in a final state
		return s.handleMigrationFinalizersWithoutVMI(key)
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	// First, handle finalizers for any completed or deleted migrations for this VMI,
	// even if they're no longer the active migration (e.g., virt-controller cleared migration state)
	if err := s.handleFinalizersForCompletedMigrations(vmi); err != nil {
		return err
	}

	migration, err := s.getMigrationForVMI(vmi)
	if err != nil {
		return err
	}
	if migration != nil && migration.IsDecentralized() {
		if err := s.handleMigrationFinalizer(migration); err != nil {
			return err
		}
		if migration.IsDecentralizedSource() {
			if migration.DeletionTimestamp != nil {
				log.Log.V(2).Object(migration).Infof("migration is being deleted, syncing source state before canceling target migration")
				if err := s.handleSourceState(ctx, vmi.DeepCopy(), migration); err != nil {
					return s.updateDecentralizedFailureOnSource(vmi, migration, err)
				}
				if err := s.cancelTargetRemoteMigration(vmi, migration); err != nil {
					return err
				}
				return nil
			}
			err := s.handleSourceState(ctx, vmi, migration)
			return s.updateDecentralizedFailureOnSource(vmi, migration, err)
		}
		if migration.IsDecentralizedTarget() {
			if migration.DeletionTimestamp != nil {
				log.Log.V(2).Object(migration).Infof("migration is being deleted, informing the source that the migration is canceled")
				// migration is being deleted, inform the target that the migration is canceled
				if err := s.cancelSourceRemoteMigration(vmi, migration); err != nil {
					return err
				}
				return nil
			}
			err := s.handleTargetState(ctx, vmi, migration)
			return s.updateDecentralizedFailureOnTarget(vmi, migration, err)
		}
		return nil
	} else {
		// After a target-initiated cancel the source migration object may be removed
		// before the source VMI records abort completion. Push the final source state
		// once more so the target VMI receives EndTimestamp and can finish abort cleanup.
		if needsOrphanSourceAbortSync(vmi) {
			log.Log.Object(vmi).Info("migration object gone after source abort, pushing final source state to target")
			if err := s.handleSourceState(ctx, vmi.DeepCopy(), nil); err != nil {
				return err
			}
		}
		// No migration found don't do anything
		// We should only clear the condition if we are not waiting for synchronization.
		if err := s.clearDecentralizedLiveMigrationFailure(vmi); err != nil {
			return err
		}
		log.Log.Object(vmi).V(4).Info("no active decentralized migration found for VMI")
		return nil
	}
}

func needsOrphanSourceAbortSync(vmi *virtv1.VirtualMachineInstance) bool {
	ms := vmi.Status.MigrationState
	if ms == nil || ms.SourceState == nil || ms.TargetState == nil {
		return false
	}
	if ms.MigrationUID != ms.SourceState.MigrationUID {
		return false
	}
	return ms.AbortRequested && ms.EndTimestamp != nil
}

func (s *SynchronizationController) updateDecentralizedFailureOnSource(vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration, opErr error) error {
	if opErr != nil {
		if err := s.setDecentralizedLiveMigrationFailure(vmi, getErrorMessageForDecentralizedLiveMigrationFailure(opErr)); err != nil {
			return err
		}
		return opErr
	}
	return s.clearDecentralizedLiveMigrationFailure(vmi)
}

func (s *SynchronizationController) updateDecentralizedFailureOnTarget(vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration, opErr error) error {
	// failure from handleTargetState
	if opErr != nil {
		return s.setDecentralizedLiveMigrationFailure(vmi, getErrorMessageForDecentralizedLiveMigrationFailure(opErr))
	}

	// success: special case waiting for sync
	if migration.Status.Phase == virtv1.MigrationWaitingForSync {
		return s.setDecentralizedLiveMigrationFailure(vmi, waitingForSyncErrorMessage)
	}

	// success and not waiting for sync: clear condition
	return s.clearDecentralizedLiveMigrationFailure(vmi)
}

func getErrorMessageForDecentralizedLiveMigrationFailure(err error) string {
	log.Log.V(1).Infof("error message: %v", err)
	// Once we upgrade to golang 1.26, we no longer need to check for x509.HostnameError, since https://github.com/golang/go/issues/76445 will be fixed.
	if errors.As(err, &x509.HostnameError{}) {
		return "x509 hostname error"
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	return message
}

func (s *SynchronizationController) setDecentralizedLiveMigrationFailure(vmi *virtv1.VirtualMachineInstance, errorMessage string) error {
	orgVmi := vmi.DeepCopy()
	condition := &virtv1.VirtualMachineInstanceCondition{
		Type:               virtv1.VirtualMachineInstanceDecentralizedLiveMigrationFailure,
		Status:             k8sv1.ConditionTrue,
		Reason:             virtv1.VirtualMachineInstanceReasonDecentralizedNotMigratable,
		Message:            errorMessage,
		LastTransitionTime: metav1.Now(),
	}
	controller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, condition)
	if err2 := s.patchVMIConditions(context.Background(), orgVmi, vmi); err2 != nil {
		log.Log.Reason(err2).Infof("unable to patch VMI conditions after decentralized live migration failure")
		return err2
	}
	return nil
}

func (s *SynchronizationController) clearDecentralizedLiveMigrationFailure(vmi *virtv1.VirtualMachineInstance) error {
	orgVmi := vmi.DeepCopy()
	controller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstanceDecentralizedLiveMigrationFailure)
	if err := s.patchVMIConditions(context.Background(), orgVmi, vmi); err != nil {
		log.Log.Reason(err).Infof("unable to patch VMI conditions after clearing decentralized live migration failure")
		return err
	}
	return nil
}

func (s *SynchronizationController) handleFinalizersForCompletedMigrations(vmi *virtv1.VirtualMachineInstance) error {
	// Get all migrations for this VMI name in the same namespace
	objects, err := s.migrationInformer.GetIndexer().ByIndex(migrationIndexByActiveVMIName, vmi.Name)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
		if !ok {
			continue
		}
		// Only handle migrations in the same namespace as the VMI
		if migration.Namespace != vmi.Namespace {
			continue
		}
		// Only handle decentralized migrations
		if !migration.IsDecentralized() {
			continue
		}
		// Only handle migrations that are final or being deleted
		// These need finalizer removal even if they don't match the VMI's current active migration
		if migration.IsFinal() || migration.DeletionTimestamp != nil {
			if err := s.handleMigrationFinalizer(migration); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SynchronizationController) handleMigrationFinalizer(migration *virtv1.VirtualMachineInstanceMigration) error {
	originalMigration := migration.DeepCopy()
	if !migration.IsFinal() && migration.DeletionTimestamp == nil {
		controller.AddFinalizer(migration, SynchronizationFinalizer)
	} else {
		controller.RemoveFinalizer(migration, SynchronizationFinalizer)
	}
	if !apiequality.Semantic.DeepEqual(originalMigration.ObjectMeta, migration.ObjectMeta) {
		log.Log.V(4).Object(migration).Infof("adding or removing finalizer to migration, %v", migration.Finalizers)
		patchSet := patch.New()
		patchSet.AddOption(
			patch.WithReplace("/metadata/finalizers", migration.Finalizers),
		)
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return err
		}
		if _, err := s.client.VirtualMachineInstanceMigration(migration.Namespace).Patch(context.Background(), migration.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizationController) handleMigrationFinalizersWithoutVMI(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// Index keys on Spec.VMIName alone; filter by namespace after lookup.
	objects, err := s.migrationInformer.GetIndexer().ByIndex(migrationIndexByActiveVMIName, name)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
		if !ok {
			continue
		}
		if migration.Namespace != namespace {
			continue
		}
		if !migration.IsDecentralized() {
			continue
		}
		// Handle finalizer for this migration even though VMI is gone
		if err := s.handleMigrationFinalizer(migration); err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizationController) cancelSourceRemoteMigration(vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration) error {
	if vmi == nil || vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceState == nil || vmi.Status.MigrationState.SourceState.MigrationUID == "" {
		return nil
	}
	if migration.UID == vmi.Status.MigrationState.SourceState.MigrationUID {
		return fmt.Errorf("source migration UID %s is the same as the VMI's migration UID %s", migration.UID, vmi.Status.MigrationState.SourceState.MigrationUID)
	}
	log.Log.V(4).Object(migration).Infof("cancelling source remote migration for VMI %s/%s", vmi.Namespace, vmi.Name)
	// Target dials the source and stores that conn in syncReceivingConnectionMap.
	return s.cancelRemoteMigration(vmi.Status.MigrationState.SourceState.MigrationUID, migration.Spec.Receive.MigrationID, s.syncReceivingConnectionMap)
}

func (s *SynchronizationController) cancelTargetRemoteMigration(vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration) error {
	if vmi == nil || vmi.Status.MigrationState == nil || vmi.Status.MigrationState.TargetState == nil || vmi.Status.MigrationState.TargetState.MigrationUID == "" {
		return nil
	}
	if migration.UID == vmi.Status.MigrationState.TargetState.MigrationUID {
		return fmt.Errorf("target migration UID %s is the same as the VMI's migration UID %s", migration.UID, vmi.Status.MigrationState.TargetState.MigrationUID)
	}
	log.Log.V(4).Object(migration).Infof("cancelling target remote migration for VMI %s/%s", vmi.Namespace, vmi.Name)
	// Source dials the target and stores that conn in syncOutboundConnectionMap.
	return s.cancelRemoteMigration(vmi.Status.MigrationState.TargetState.MigrationUID, migration.Spec.SendTo.MigrationID, s.syncOutboundConnectionMap)
}

func (s *SynchronizationController) cancelRemoteMigration(migrationUID types.UID, migrationID string, connectionMap *sync.Map) error {
	obj, ok := connectionMap.Load(migrationID)
	if !ok {
		// No connection found, don't do anything
		return nil
	}
	conn, ok := obj.(*SynchronizationConnection)
	if !ok {
		return fmt.Errorf("found unknown object in outbound connection cache %#v", conn)
	}
	client := syncv1.NewSynchronizeClient(conn.grpcClientConnection)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeout)*time.Second)
	defer cancel()
	_, err := client.CancelMigration(ctx, &syncv1.MigrationCancelRequest{
		MigrationUID: string(migrationUID),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizationController) getMigrationIDFromUID(migrationUID types.UID) (string, error) {
	objs, err := s.migrationInformer.GetIndexer().ByIndex(controller.ByMigrationUIDIndex, string(migrationUID))
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
	if migrationState.TargetState == nil || migrationState.TargetState.SyncAddress == nil || *migrationState.TargetState.SyncAddress == "" {
		return nil, nil
	}
	migrationID, err := s.resolveSourceOutboundMigrationID(migrationState)
	if err != nil {
		return nil, err
	}
	if migrationID == "" {
		return nil, nil
	}
	return s.getOutboundConnectionByMigrationID(vmi, migrationID, *migrationState.TargetState.SyncAddress, s.syncOutboundConnectionMap)
}

func (s *SynchronizationController) resolveSourceOutboundMigrationID(migrationState *virtv1.VirtualMachineInstanceMigrationState) (string, error) {
	if migrationState == nil || migrationState.SourceState == nil {
		return "", nil
	}
	migrationID, err := s.getMigrationIDFromUID(migrationState.SourceState.MigrationUID)
	if err != nil || migrationID != "" {
		return migrationID, err
	}
	// The source migration object may already be deleted after a target-initiated cancel.
	if migrationState.TargetState != nil && migrationState.TargetState.MigrationUID != "" {
		return s.getMigrationIDFromUID(migrationState.TargetState.MigrationUID)
	}
	return "", nil
}

func (s *SynchronizationController) getOutboundTargetConnection(vmi *virtv1.VirtualMachineInstance, migrationState *virtv1.VirtualMachineInstanceMigrationState) (*SynchronizationConnection, error) {
	if migrationState.SourceState == nil || migrationState.SourceState.SyncAddress == nil || *migrationState.SourceState.SyncAddress == "" {
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
	return s.getOutboundConnectionByMigrationID(vmi, migrationID, syncAddress, connectionMap)
}

func (s *SynchronizationController) getOutboundConnectionByMigrationID(vmi *virtv1.VirtualMachineInstance, migrationID string, syncAddress string, connectionMap *sync.Map) (*SynchronizationConnection, error) {
	if migrationID == "" {
		return nil, nil
	}
	log.Log.Object(vmi).V(4).Infof("found migration ID %s", migrationID)
	obj, ok := connectionMap.Load(migrationID)
	if ok {
		outboundSyncConnection, ok := obj.(*SynchronizationConnection)
		if !ok {
			return nil, fmt.Errorf("found unknown object in outbound connection cache %#v", obj)
		}
		if outboundSyncConnection.syncAddress == syncAddress && outboundSyncConnection.grpcClientConnection != nil {
			return outboundSyncConnection, nil
		}
		// Peer address changed or conn is unusable — replace the cached connection.
		log.Log.Object(vmi).V(3).Infof("replacing outbound sync connection for migration ID %s (address %q -> %q)",
			migrationID, outboundSyncConnection.syncAddress, syncAddress)
		connectionMap.Delete(migrationID)
		_ = outboundSyncConnection.Close()
	}

	grpcClientConnection, err := s.createOutboundConnection(syncAddress)
	if err != nil {
		return nil, err
	}
	conn := &SynchronizationConnection{
		migrationID:          migrationID,
		syncAddress:          syncAddress,
		grpcClientConnection: grpcClientConnection,
	}
	if existing, loaded := connectionMap.LoadOrStore(migrationID, conn); loaded {
		// Another goroutine won the race; prefer theirs if it matches the address.
		_ = conn.Close()
		existingConn, ok := existing.(*SynchronizationConnection)
		if !ok {
			return nil, fmt.Errorf("found unknown object in outbound connection cache %#v", existing)
		}
		if existingConn.syncAddress == syncAddress && existingConn.grpcClientConnection != nil {
			return existingConn, nil
		}
		// Winner is stale for this address — replace and return a fresh dial.
		connectionMap.Delete(migrationID)
		_ = existingConn.Close()
		grpcClientConnection, err := s.createOutboundConnection(syncAddress)
		if err != nil {
			return nil, err
		}
		conn = &SynchronizationConnection{
			migrationID:          migrationID,
			syncAddress:          syncAddress,
			grpcClientConnection: grpcClientConnection,
		}
		if existing, loaded := connectionMap.LoadOrStore(migrationID, conn); loaded {
			_ = conn.Close()
			existingConn, ok := existing.(*SynchronizationConnection)
			if !ok {
				return nil, fmt.Errorf("found unknown object in outbound connection cache %#v", existing)
			}
			if existingConn.syncAddress == syncAddress && existingConn.grpcClientConnection != nil {
				return existingConn, nil
			}
			return nil, fmt.Errorf("stale outbound sync connection for migration ID %s after replace race (have %q, want %q)",
				migrationID, existingConn.syncAddress, syncAddress)
		}
		return conn, nil
	}
	return conn, nil
}

func (s *SynchronizationController) handleSourceState(ctx context.Context, vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration) error {
	var outboundConnection *SynchronizationConnection
	var err error
	if vmi.Status.MigrationState == nil {
		// No migration state, don't do anything
		return nil
	}
	if vmi.Status.MigrationState.SourceState == nil {
		// No source state, don't do anything
		return nil
	}

	// Keep original for patching later
	origVMI := vmi
	vmi = vmi.DeepCopy()
	sourceState := vmi.Status.MigrationState.SourceState
	// Always set SyncAddress to our current gRPC synchronization address
	syncAddress, err := s.getLocalSynchronizationAddress()
	if err != nil {
		return err
	}
	syncAddressChanged := sourceState.SyncAddress == nil || *sourceState.SyncAddress != syncAddress
	sourceState.SyncAddress = &syncAddress
	targetState := vmi.Status.MigrationState.TargetState

	if targetState != nil && targetState.SyncAddress != nil && sourceState.MigrationUID != "" {
		if outboundConnection, err = s.getOutboundSourceConnection(vmi, vmi.Status.MigrationState); err != nil {
			return err
		}
	}
	if outboundConnection == nil {
		if !syncAddressChanged {
			log.Log.Object(vmi).V(4).Info("no synchronization connection found for source, doing nothing")
			return nil
		}
		log.Log.Object(vmi).V(4).Infof("updating source SyncAddress to %s (no target connection yet)", *sourceState.SyncAddress)
		if err := s.patchVMI(ctx, origVMI, vmi); err != nil {
			return fmt.Errorf("failed to patch VMI with source sync address: %v", err)
		}
		return nil
	}

	// Create a copy of the status to send via gRPC
	statusToSend := vmi.Status.DeepCopy()

	// If proxy is initialized, replace SourceState.SyncAddress with source sync controller address
	// This tells target sync controller where to connect for synchronization
	if s.IsTunnelInitialized() && statusToSend.MigrationState != nil && statusToSend.MigrationState.SourceState != nil {
		sourceSyncAddress, err := s.getLocalSynchronizationAddress()
		if err != nil {
			return fmt.Errorf("failed to get local synchronization address: %w", err)
		}
		statusToSend.MigrationState.SourceState.SyncAddress = &sourceSyncAddress
		log.Log.Object(migration).Infof("Sending SourceState with source sync address: %s", sourceSyncAddress)
	}

	vmiStatusJson, err := json.Marshal(statusToSend)
	if err != nil {
		return err
	}
	client := syncv1.NewSynchronizeClient(outboundConnection.grpcClientConnection)
	grpcCtx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout)*time.Second)
	defer cancel()

	if _, err := client.SyncSourceMigrationStatus(grpcCtx, &syncv1.VMIStatusRequest{
		MigrationID: outboundConnection.migrationID,
		VmiStatus: &syncv1.VMIStatus{
			VmiStatusJson: vmiStatusJson,
		},
	}); err != nil {
		return err
	}

	if syncAddressChanged {
		if err := s.patchVMI(ctx, origVMI, vmi); err != nil {
			return fmt.Errorf("failed to patch VMI with source sync address: %v", err)
		}
	}

	if migration != nil && migration.IsFinal() {
		if migration.Spec.SendTo != nil {
			migrationID := migration.Spec.SendTo.MigrationID
			log.Log.Object(migration).Infof("completed migration for VMI %s/%s, closing outbound connections", migration.Namespace, migration.Spec.VMIName)
			s.closeConnectionForMigrationID(s.syncOutboundConnectionMap, migrationID)

			// Clean up source tunnel
			if s.tunnelManager != nil {
				s.tunnelManager.StopTunnel(migrationID)
			}
		}
	}

	return nil
}

func (s *SynchronizationController) handleTargetState(ctx context.Context, vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration) error {
	if vmi.Status.MigrationState == nil {
		// No migration state, don't do anything
		return nil
	}
	if vmi.Status.MigrationState.TargetState == nil {
		// No target state, don't do anything
		return nil
	}

	// Keep original for patching later
	origVMI := vmi
	// Work on a copy to avoid modifying the cached object
	vmi = vmi.DeepCopy()

	var outboundConnection *SynchronizationConnection
	var err error
	sourceState := vmi.Status.MigrationState.SourceState
	targetState := vmi.Status.MigrationState.TargetState

	// Always set SyncAddress to our current gRPC synchronization address
	// This handles pod restarts, initial setup, and keeps it current
	syncAddress, err := s.getLocalSynchronizationAddress()
	if err != nil {
		return err
	}
	syncAddressChanged := targetState.SyncAddress == nil || *targetState.SyncAddress != syncAddress
	targetState.SyncAddress = &syncAddress

	// Only attempt outbound connection if SourceState exists
	if sourceState != nil && sourceState.SyncAddress != nil && targetState.MigrationUID != "" {
		if outboundConnection, err = s.getOutboundTargetConnection(vmi, vmi.Status.MigrationState); err != nil {
			return err
		}
	}
	if outboundConnection == nil {
		if !syncAddressChanged {
			return nil
		}
		// No outbound connection yet (SourceState not set), update VMI status locally using patch
		log.Log.Object(vmi).V(4).Infof("updating target SyncAddress to %s (no source connection yet)", *targetState.SyncAddress)
		if err := s.patchVMI(ctx, origVMI, vmi); err != nil {
			return fmt.Errorf("failed to patch VMI with target sync address: %v", err)
		}
		return nil
	}

	// Create a copy of the status to send via gRPC
	statusToSend := vmi.Status.DeepCopy()

	// If proxy is initialized, start target proxies and rewrite addresses
	if err := s.setupTargetProxiesForOutbound(migration, vmi); err != nil {
		return err
	}

	vmiStatusJson, err := json.Marshal(statusToSend)
	if err != nil {
		return err
	}
	client := syncv1.NewSynchronizeClient(outboundConnection.grpcClientConnection)
	grpcCtx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout)*time.Second)
	defer cancel()

	_, err = client.SyncTargetMigrationStatus(grpcCtx, &syncv1.VMIStatusRequest{
		MigrationID: outboundConnection.migrationID,
		VmiStatus: &syncv1.VMIStatus{
			VmiStatusJson: vmiStatusJson,
		},
	})
	if err != nil {
		return err
	}

	// Persist SyncAddress locally even when an outbound connection already exists
	// (pod IP / listen address can change across restarts).
	if syncAddressChanged {
		if err := s.patchVMI(ctx, origVMI, vmi); err != nil {
			return fmt.Errorf("failed to patch VMI with target sync address: %v", err)
		}
	}

	if migration.IsFinal() {
		if migration.Spec.Receive != nil {
			migrationID := migration.Spec.Receive.MigrationID
			log.Log.Object(migration).Infof("completed migration for VMI %s/%s, closing receiving connections", migration.Namespace, migration.Spec.VMIName)
			s.closeConnectionForMigrationID(s.syncReceivingConnectionMap, migrationID)

			// Clean up target tunnel
			if s.tunnelManager != nil {
				s.tunnelManager.StopTunnel(migrationID)
			}
		}
	}

	return nil
}

func (s *SynchronizationController) getMigrationForVMI(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstanceMigration, error) {
	objects, err := s.migrationInformer.GetIndexer().ByIndex(migrationIndexByActiveVMIName, vmi.Name)
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
				if migration.IsDecentralizedSource() {
					if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.SourceState != nil && migration.UID == vmi.Status.MigrationState.SourceState.MigrationUID {
						count++
						res = migration
					}
				} else if migration.IsDecentralizedTarget() {
					if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetState != nil && migration.UID == vmi.Status.MigrationState.TargetState.MigrationUID {
						count++
						res = migration
					}
				}
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

func (s *SynchronizationController) rebuildConnectionsAndUpdateSyncAddress() error {
	// Go and find all active migration resources, if they are decentralized rebuild either
	// the incoming or outbound connections, and call sync to update the remote with the new
	// address.
	objs := s.migrationInformer.GetStore().List()
	log.Log.V(4).Infof("rebuilding any connections, and updating remote VMIs, found %d migrations", len(objs))
	for _, obj := range objs {
		migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
		if !ok {
			return fmt.Errorf("unknown object in migration store %v", obj)
		}
		if isOnGoingMigration(migration) {
			vmi, err := s.getVMIFromMigration(migration)
			if err != nil {
				return err
			}
			if vmi == nil {
				// No VMI found, can't update it, so skip it.
				continue
			}
			// ongoing migration.
			if migration.Spec.Receive != nil {
				// We are the target
				log.Log.Object(migration).Object(vmi).Info("found ongoing target migration for vmi, rebuilding connection")
				if err := s.rebuildTargetConnection(migration, vmi); err != nil {
					return err
				}
			} else if migration.Spec.SendTo != nil {
				// We are the source
				log.Log.Object(migration).Object(vmi).Info("found ongoing source migration for vmi, rebuilding connection")
				if err := s.rebuildSourceConnection(migration, vmi); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func isOnGoingMigration(migration *virtv1.VirtualMachineInstanceMigration) bool {
	return migration.IsDecentralized() && migration.Status.Phase != virtv1.MigrationFailed && migration.Status.Phase != virtv1.MigrationSucceeded
}

func (s *SynchronizationController) rebuildTargetConnection(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) error {
	conn, err := s.getOutboundTargetConnection(vmi, vmi.Status.MigrationState)
	if err != nil {
		return err
	}
	if conn == nil {
		return nil
	}
	s.syncReceivingConnectionMap.Store(migration.Spec.Receive.MigrationID, conn)
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetState != nil {
		url, err := s.getLocalSynchronizationAddress()
		if err != nil {
			return err
		}
		origVMI := vmi.DeepCopy()
		vmi.Status.MigrationState.TargetState.SyncAddress = &url
		// patching will cause reconcile loop to connect to remote to update
		if err := s.patchVMI(context.Background(), origVMI, vmi); err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizationController) rebuildSourceConnection(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) error {
	conn, err := s.getOutboundSourceConnection(vmi, vmi.Status.MigrationState)
	if err != nil {
		return err
	}
	if conn == nil {
		return nil
	}
	s.syncOutboundConnectionMap.Store(migration.Spec.SendTo.MigrationID, conn)
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.SourceState != nil {
		url, err := s.getLocalSynchronizationAddress()
		if err != nil {
			return err
		}
		origVMI := vmi.DeepCopy()
		vmi.Status.MigrationState.SourceState.SyncAddress = &url
		// patching will cause reconcile loop to connect to remote to update
		if err := s.patchVMI(context.Background(), origVMI, vmi); err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizationController) getVMIFromMigration(migration *virtv1.VirtualMachineInstanceMigration) (*virtv1.VirtualMachineInstance, error) {
	key := controller.NamespacedKey(migration.Namespace, migration.Spec.VMIName)
	obj, exists, err := s.vmiInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return obj.(*virtv1.VirtualMachineInstance).DeepCopy(), nil
}

func (s *SynchronizationController) getLocalSynchronizationAddress() (string, error) {
	// When tunnel is initialized, use crosscluster IP for gRPC synchronization
	if s.IsTunnelInitialized() {
		return net.JoinHostPort(s.tunnelManager.CrossClusterIP(), strconv.Itoa(s.bindPort)), nil
	}

	if s.ip != "" {
		return net.JoinHostPort(s.ip, strconv.Itoa(s.bindPort)), nil
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
	log.Log.Infof("gRPC server listening on %s", ln.Addr().String())
	return ln, nil
}

func (s *SynchronizationController) findTargetMigrationFromMigrationID(migrationID string) (*virtv1.VirtualMachineInstanceMigration, error) {
	return s.findMigrationFromMigrationIDByIndex(migrationIndexByTargetMigrationID, migrationID)
}

func (s *SynchronizationController) findSourceMigrationFromMigrationID(migrationID string) (*virtv1.VirtualMachineInstanceMigration, error) {
	return s.findMigrationFromMigrationIDByIndex(migrationIndexBySourceMigrationID, migrationID)
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
			Message: fmt.Sprintf(sourceUnableToLocateVMIMigrationIDErrorMsg, request.MigrationID),
		}, fmt.Errorf(sourceUnableToLocateVMIMigrationIDErrorMsg, request.MigrationID)
	}
	key := controller.NamespacedKey(migration.Namespace, migration.Spec.VMIName)
	log.Log.Object(migration).V(5).Infof("looking up VMI %s", key)
	obj, exists, err := s.vmiInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		if err == nil {
			err = fmt.Errorf(sourceUnableToLocateVMIMigrationIDErrorMsgVMI, request.MigrationID, key)
		}
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf(sourceUnableToLocateVMIMigrationIDErrorMsgVMI, request.MigrationID, key),
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
	newVMI := vmi.DeepCopy()
	if newVMI.Status.MigrationState == nil {
		newVMI.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{}
	}

	// Only update SourceState if this migration is still active and matches the VMI's current migration.
	// This prevents stale updates from a completed decentralized migration from interfering with a new compute migration.
	if migration.IsFinal() {
		log.Log.Object(migration).Infof("Migration is final, ignoring source state update for VMI %s/%s", vmi.Namespace, vmi.Name)
		return &syncv1.VMIStatusResponse{
			Message: successMessage,
		}, nil
	}

	// Check if the VMI's current migration matches this migration
	if newVMI.Status.MigrationState.MigrationUID != "" && newVMI.Status.MigrationState.MigrationUID != migration.UID {
		log.Log.Object(migration).Warningf("VMI %s/%s has different migration UID %s, ignoring source state update for migration %s",
			vmi.Namespace, vmi.Name, newVMI.Status.MigrationState.MigrationUID, migration.UID)
		return &syncv1.VMIStatusResponse{
			Message: successMessage,
		}, nil
	}

	// Bind peer only after the update is accepted, and only when the tunnel proxy is active.
	if s.IsTunnelInitialized() {
		p, ok := peer.FromContext(ctx)
		if !ok || p.Addr == nil {
			err := status.Errorf(codes.Unauthenticated,
				"missing peer identity for migration tunnel binding of %s", request.MigrationID)
			return &syncv1.VMIStatusResponse{Message: err.Error()}, err
		}
		if err := s.tunnelManager.BindTunnelPeer(request.MigrationID, p.Addr.String()); err != nil {
			return &syncv1.VMIStatusResponse{Message: err.Error()}, err
		}
	}

	log.Log.Object(newVMI).V(5).Infof("vmi migration source state: %#v", newVMI.Status.MigrationState.SourceState)
	log.Log.Object(newVMI).V(5).Infof("remote migration source state: %#v", remoteStatus.MigrationState.SourceState)

	// If proxy is initialized, handle target-side proxy setup
	if err := s.setupTargetProxiesFromSource(migration, newVMI, remoteStatus); err != nil {
		return nil, err
	}

	newVMI.Status.MigrationState.SourceState = remoteStatus.MigrationState.SourceState.DeepCopy()
	copyLegacySourceFields(newVMI, remoteStatus.MigrationState)
	if len(remoteStatus.MigratedVolumes) > 0 {
		log.Log.Object(newVMI).V(5).Infof("SyncSourceMigrationStatus: Copying migrated volumes to target state, %#v", newVMI.Status.MigratedVolumes)
		newVMI.Status.MigratedVolumes = getMergedSourceMigratedVolumes(newVMI.Status.MigratedVolumes, remoteStatus.MigratedVolumes)
	} else {
		log.Log.Object(newVMI).V(5).Info("SyncSourceMigrationStatus: No source migrated volumes found")
	}
	newVMI.Status.MigrationMethod = remoteStatus.MigrationMethod
	newVMI.Status.MigrationTransport = remoteStatus.MigrationTransport
	if !apiequality.Semantic.DeepEqual(vmi.Status, newVMI.Status) {
		if err := s.patchVMI(ctx, vmi, newVMI); err != nil {
			return &syncv1.VMIStatusResponse{
				Message: fmt.Sprintf("unable to synchronize VMI for migrationID %s", request.MigrationID),
			}, err
		}
		log.Log.Object(newVMI).With("MigrationID", request.MigrationID).V(5).Info("successfully patched VMI with source state")
	}
	log.Log.Object(newVMI).V(5).Info("returning success to grpc caller, source")
	return &syncv1.VMIStatusResponse{
		Message: successMessage,
	}, nil
}

func getMergedTargetMigratedVolumes(vmiMigratedVolumes []virtv1.StorageMigratedVolumeInfo, remoteMigratedVolumes []virtv1.StorageMigratedVolumeInfo) []virtv1.StorageMigratedVolumeInfo {
	remoteVolumeMap := make(map[string]virtv1.StorageMigratedVolumeInfo)
	for _, volume := range remoteMigratedVolumes {
		remoteVolumeMap[volume.VolumeName] = volume
	}
	mergedVolumes := make([]virtv1.StorageMigratedVolumeInfo, 0)
	for _, volume := range vmiMigratedVolumes {
		if remoteVolume, ok := remoteVolumeMap[volume.VolumeName]; ok {
			mergedVolume := virtv1.StorageMigratedVolumeInfo{
				VolumeName: volume.VolumeName,
			}
			if remoteVolume.DestinationPVCInfo != nil {
				mergedVolume.DestinationPVCInfo = remoteVolume.DestinationPVCInfo.DeepCopy()
			}
			if volume.SourcePVCInfo != nil {
				mergedVolume.SourcePVCInfo = volume.SourcePVCInfo.DeepCopy()
			}
			mergedVolumes = append(mergedVolumes, mergedVolume)
		} else {
			mergedVolumes = append(mergedVolumes, volume)
		}
	}
	return mergedVolumes
}

func getMergedSourceMigratedVolumes(vmiMigratedVolumes []virtv1.StorageMigratedVolumeInfo, remoteMigratedVolumes []virtv1.StorageMigratedVolumeInfo) []virtv1.StorageMigratedVolumeInfo {
	remoteVolumeMap := make(map[string]virtv1.StorageMigratedVolumeInfo)
	for _, volume := range remoteMigratedVolumes {
		remoteVolumeMap[volume.VolumeName] = volume
	}
	mergedVolumes := make([]virtv1.StorageMigratedVolumeInfo, 0)
	for _, vmiVolume := range vmiMigratedVolumes {
		if remoteVolume, ok := remoteVolumeMap[vmiVolume.VolumeName]; ok {
			log.Log.V(5).Infof("Merging volume %s", vmiVolume.VolumeName)
			// Found a match merge the current target volume with the incoming source volume
			mergedVolume := virtv1.StorageMigratedVolumeInfo{
				VolumeName: vmiVolume.VolumeName,
			}
			if vmiVolume.SourcePVCInfo != nil {
				mergedVolume.SourcePVCInfo = vmiVolume.SourcePVCInfo.DeepCopy()
			} else {
				mergedVolume.SourcePVCInfo = remoteVolume.SourcePVCInfo.DeepCopy()
			}
			if vmiVolume.DestinationPVCInfo != nil {
				mergedVolume.DestinationPVCInfo = vmiVolume.DestinationPVCInfo.DeepCopy()
			}
			mergedVolumes = append(mergedVolumes, mergedVolume)
			delete(remoteVolumeMap, vmiVolume.VolumeName)
		}
	}
	for _, volume := range remoteVolumeMap {
		mergedVolumes = append(mergedVolumes, volume)
	}
	return mergedVolumes
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
			Message: fmt.Sprintf(targetUnableToLocateVMIMigrationIDErrorMsg, request.MigrationID),
		}, fmt.Errorf(targetUnableToLocateVMIMigrationIDErrorMsg, request.MigrationID)
	}

	key := controller.NamespacedKey(migration.Namespace, migration.Spec.VMIName)
	obj, exists, err := s.vmiInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		if err == nil {
			err = fmt.Errorf(targetUnableToLocateVMIMigrationIDErrorMsgVMI, request.MigrationID, key)
		}
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf(targetUnableToLocateVMIMigrationIDErrorMsgVMI, request.MigrationID, key),
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
	newVMI := vmi.DeepCopy()
	if newVMI.Status.MigrationState == nil {
		newVMI.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{}
	}

	// Only update TargetState if this migration is still active and matches the VMI's current migration.
	// This prevents stale updates from a completed decentralized migration from interfering with a new compute migration.
	if migration.IsFinal() {
		log.Log.Object(migration).Infof("Migration is final, ignoring target state update for VMI %s/%s", vmi.Namespace, vmi.Name)
		return &syncv1.VMIStatusResponse{
			Message: successMessage,
		}, nil
	}

	// Check if the VMI's current migration matches this migration
	if newVMI.Status.MigrationState.MigrationUID != "" && newVMI.Status.MigrationState.MigrationUID != migration.UID {
		log.Log.Object(migration).Warningf("VMI %s/%s has different migration UID %s, ignoring target state update for migration %s",
			vmi.Namespace, vmi.Name, newVMI.Status.MigrationState.MigrationUID, migration.UID)
		return &syncv1.VMIStatusResponse{
			Message: successMessage,
		}, nil
	}

	log.Log.Object(newVMI).V(5).Infof("vmi migration target state: %#v", newVMI.Status.MigrationState.TargetState)
	log.Log.Object(newVMI).V(5).Infof("remote migration target state: %#v", remoteStatus.MigrationState.TargetState)

	// If proxy is initialized, handle source-side proxy setup when receiving TargetState
	if err := s.setupSourceProxiesFromTarget(migration, newVMI, remoteStatus); err != nil {
		return &syncv1.VMIStatusResponse{
			Message: fmt.Sprintf("failed to setup source proxies: %v", err),
		}, err
	}

	newVMI.Status.MigrationState.TargetState = remoteStatus.MigrationState.TargetState.DeepCopy()
	newVMI.Status.MigratedVolumes = getMergedTargetMigratedVolumes(newVMI.Status.MigratedVolumes, remoteStatus.MigratedVolumes)
	// Copy legacy fields
	// When proxy is active, skip proxy-managed fields (TargetNodeAddress, TargetDirectMigrationNodePorts)
	// to avoid overwriting addresses set by setupSourceProxiesFromTarget
	copyLegacyTargetFields(newVMI, remoteStatus.MigrationState, s.IsTunnelInitialized())
	if !apiequality.Semantic.DeepEqual(vmi.Status.MigrationState, newVMI.Status.MigrationState) ||
		!apiequality.Semantic.DeepEqual(vmi.Status.MigratedVolumes, newVMI.Status.MigratedVolumes) {
		if err := s.patchVMI(ctx, vmi, newVMI); err != nil {
			return &syncv1.VMIStatusResponse{
				Message: fmt.Sprintf("unable to synchronize VMI for migrationID %s", request.MigrationID),
			}, err
		}
		log.Log.Object(newVMI).With("MigrationID", request.MigrationID).V(5).Info("successfully patched VMI with target state")
	}
	log.Log.Object(newVMI).V(5).Info("returning success to grpc caller, target")
	return &syncv1.VMIStatusResponse{
		Message: successMessage,
	}, nil
}

func (s *SynchronizationController) patchVMIConditions(ctx context.Context, origVMI, newVMI *virtv1.VirtualMachineInstance) error {
	patchSet := patch.New()
	if !apiequality.Semantic.DeepEqual(origVMI.Status.Conditions, newVMI.Status.Conditions) {
		patchSet.AddOption(
			patch.WithTest("/status/conditions", origVMI.Status.Conditions),
			patch.WithReplace("/status/conditions", newVMI.Status.Conditions),
		)
	}
	if !patchSet.IsEmpty() {
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return err
		}
		log.Log.Object(origVMI).V(5).Infof("patch VMI conditions with %s", string(patchBytes))
		if _, err := s.client.VirtualMachineInstance(origVMI.Namespace).Patch(ctx, origVMI.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizationController) patchVMI(ctx context.Context, origVMI, newVMI *virtv1.VirtualMachineInstance) error {
	if origVMI.Status.MigrationState != nil &&
		origVMI.Status.MigrationState.Completed {
		log.Log.Object(origVMI).V(3).Infof("VMI is completed, skipping patch")
		return nil
	}
	patchSet := patch.New()

	if !apiequality.Semantic.DeepEqual(origVMI.Labels, newVMI.Labels) {
		if len(origVMI.Labels) == 0 {
			patchSet.AddOption(
				patch.WithAdd("/metadata/labels", newVMI.Labels))
		} else {
			patchSet.AddOption(
				patch.WithTest("/metadata/labels", origVMI.Labels),
				patch.WithReplace("/metadata/labels", newVMI.Labels),
			)
		}
	}

	if !apiequality.Semantic.DeepEqual(origVMI.Status.MigrationMethod, newVMI.Status.MigrationMethod) {
		if origVMI.Status.MigrationMethod == "" {
			patchSet.AddOption(
				patch.WithAdd("/status/migrationMethod", newVMI.Status.MigrationMethod))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/migrationMethod", origVMI.Status.MigrationMethod),
				patch.WithReplace("/status/migrationMethod", newVMI.Status.MigrationMethod),
			)
		}
	}

	if !apiequality.Semantic.DeepEqual(origVMI.Status.MigrationTransport, newVMI.Status.MigrationTransport) {
		if origVMI.Status.MigrationTransport == "" {
			patchSet.AddOption(
				patch.WithAdd("/status/migrationTransport", newVMI.Status.MigrationTransport))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/migrationTransport", origVMI.Status.MigrationTransport),
				patch.WithReplace("/status/migrationTransport", newVMI.Status.MigrationTransport),
			)
		}
	}

	if !apiequality.Semantic.DeepEqual(origVMI.Status.MigratedVolumes, newVMI.Status.MigratedVolumes) {
		if origVMI.Status.MigratedVolumes == nil {
			patchSet.AddOption(
				patch.WithAdd("/status/migratedVolumes", newVMI.Status.MigratedVolumes))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/migratedVolumes", origVMI.Status.MigratedVolumes),
				patch.WithReplace("/status/migratedVolumes", newVMI.Status.MigratedVolumes),
			)
		}
	}

	if !apiequality.Semantic.DeepEqual(origVMI.Status.MigrationState, newVMI.Status.MigrationState) {
		if origVMI.Status.MigrationState == nil {
			patchSet.AddOption(
				patch.WithAdd("/status/migrationState", newVMI.Status.MigrationState))
		} else {
			addMigrationStateFieldPatches(patchSet, origVMI.Status.MigrationState, newVMI.Status.MigrationState)
		}
	}

	if !patchSet.IsEmpty() {
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return err
		}
		log.Log.Object(origVMI).V(5).Infof("patch VMI with %s", string(patchBytes))
		if _, err := s.client.VirtualMachineInstance(origVMI.Namespace).Patch(ctx, origVMI.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// addMigrationStateFieldPatches generates individual JSON Patch
// operations for each changed field in MigrationState, rather than a
// single test+replace of the entire object. This prevents spurious
// patch failures when another controller concurrently modifies
// unrelated MigrationState fields (e.g. the virt-handler setting
// ports while the sync controller sets SourceState).
func addMigrationStateFieldPatches(patchSet *patch.PatchSet, origMS, newMS *virtv1.VirtualMachineInstanceMigrationState) {
	if newMS == nil {
		patchSet.AddOption(
			patch.WithTest("/status/migrationState", origMS),
			patch.WithRemove("/status/migrationState"),
		)
		return
	}

	patchMigrationStateField(patchSet, "startTimestamp", origMS.StartTimestamp, newMS.StartTimestamp)
	patchMigrationStateField(patchSet, "endTimestamp", origMS.EndTimestamp, newMS.EndTimestamp)
	patchMigrationStateField(patchSet, "targetNodeDomainReadyTimestamp", origMS.TargetNodeDomainReadyTimestamp, newMS.TargetNodeDomainReadyTimestamp)
	patchMigrationStateField(patchSet, "targetNodeDomainDetected", origMS.TargetNodeDomainDetected, newMS.TargetNodeDomainDetected)
	patchMigrationStateField(patchSet, "targetNodeAddress", origMS.TargetNodeAddress, newMS.TargetNodeAddress)
	patchMigrationStateField(patchSet, "targetDirectMigrationNodePorts", origMS.TargetDirectMigrationNodePorts, newMS.TargetDirectMigrationNodePorts)
	patchMigrationStateField(patchSet, "targetNode", origMS.TargetNode, newMS.TargetNode)
	patchMigrationStateField(patchSet, "targetPod", origMS.TargetPod, newMS.TargetPod)
	patchMigrationStateField(patchSet, "targetAttachmentPodUID", origMS.TargetAttachmentPodUID, newMS.TargetAttachmentPodUID)
	patchMigrationStateField(patchSet, "sourceNode", origMS.SourceNode, newMS.SourceNode)
	patchMigrationStateField(patchSet, "sourcePod", origMS.SourcePod, newMS.SourcePod)
	patchMigrationStateField(patchSet, "completed", origMS.Completed, newMS.Completed)
	patchMigrationStateField(patchSet, "failed", origMS.Failed, newMS.Failed)
	patchMigrationStateField(patchSet, "abortRequested", origMS.AbortRequested, newMS.AbortRequested)
	patchMigrationStateField(patchSet, "abortStatus", origMS.AbortStatus, newMS.AbortStatus)
	patchMigrationStateField(patchSet, "failureReason", origMS.FailureReason, newMS.FailureReason)
	patchMigrationStateField(patchSet, "migrationUid", origMS.MigrationUID, newMS.MigrationUID)
	patchMigrationStateField(patchSet, "mode", origMS.Mode, newMS.Mode)
	patchMigrationStateField(patchSet, "migrationPolicyName", origMS.MigrationPolicyName, newMS.MigrationPolicyName)
	patchMigrationStateField(patchSet, "migrationConfiguration", origMS.VMIMConfigurationOptions, newMS.VMIMConfigurationOptions)
	patchMigrationStateField(patchSet, "targetCPUSet", origMS.TargetCPUSet, newMS.TargetCPUSet)
	patchMigrationStateField(patchSet, "targetNodeTopology", origMS.TargetNodeTopology, newMS.TargetNodeTopology)
	patchMigrationStateField(patchSet, "sourcePersistentStatePVCName", origMS.SourcePersistentStatePVCName, newMS.SourcePersistentStatePVCName)
	patchMigrationStateField(patchSet, "targetPersistentStatePVCName", origMS.TargetPersistentStatePVCName, newMS.TargetPersistentStatePVCName)
	patchMigrationStateField(patchSet, "sourceState", origMS.SourceState, newMS.SourceState)
	patchMigrationStateField(patchSet, "targetState", origMS.TargetState, newMS.TargetState)
	patchMigrationStateField(patchSet, "migrationNetworkType", origMS.MigrationNetworkType, newMS.MigrationNetworkType)
	patchMigrationStateField(patchSet, "targetMemoryOverhead", origMS.TargetMemoryOverhead, newMS.TargetMemoryOverhead)
}

// patchMigrationStateField generates an add or test+replace JSON
// Patch operation for a single MigrationState field (all of which are
// omitempty). Unchanged fields are skipped. Zero originals use add
// (field absent from JSON), non-zero use test+replace for conflict
// detection.
func patchMigrationStateField[T any](patchSet *patch.PatchSet, field string, orig, new T) {
	if apiequality.Semantic.DeepEqual(orig, new) {
		return
	}
	path := "/status/migrationState/" + field
	var zero T
	if apiequality.Semantic.DeepEqual(orig, zero) {
		patchSet.AddOption(patch.WithAdd(path, new))
	} else {
		patchSet.AddOption(
			patch.WithTest(path, orig),
			patch.WithReplace(path, new),
		)
	}
}

func indexByActiveVmiName(obj interface{}) ([]string, error) {
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

// copyLegacyTargetFields copies target migration fields from new API to legacy fields
// If skipProxyFields is true, skips TargetNodeAddress and TargetDirectMigrationNodePorts
// to avoid overwriting proxy-managed addresses
func copyLegacyTargetFields(vmi *virtv1.VirtualMachineInstance, migrationState *virtv1.VirtualMachineInstanceMigrationState, skipProxyFields bool) {
	targetState := migrationState.TargetState
	vmi.Status.MigrationState.TargetNode = targetState.Node
	if targetState.AttachmentPodUID != nil {
		vmi.Status.MigrationState.TargetAttachmentPodUID = *targetState.AttachmentPodUID
	}
	vmi.Status.MigrationState.TargetCPUSet = targetState.CPUSet

	// Skip proxy-managed fields when proxy is active
	if !skipProxyFields {
		vmi.Status.MigrationState.TargetDirectMigrationNodePorts = targetState.DirectMigrationNodePorts
		// Copy TargetState.NodeAddress to TargetNodeAddress ONLY if TargetNodeAddress is currently empty
		// This allows the initial value to be set, but prevents gRPC sync from overwriting
		// the proxy addresses set by sync controllers.
		//
		// Flow:
		// 1. Target virt-handler sets TargetState.NodeAddress (e.g., 10.244.16.96)
		// 2. gRPC sync copies it to TargetNodeAddress (because TargetNodeAddress is empty)
		// 3. Target sync controller overwrites TargetNodeAddress with crosscluster IP (e.g., 172.22.42.1)
		// 4. gRPC sync does NOT overwrite (because TargetNodeAddress is not empty)
		// 5. Source sync controller overwrites TargetNodeAddress with source internal migration network IP (e.g., 10.244.1.237)
		// 6. gRPC sync does NOT overwrite (because TargetNodeAddress is not empty)
		if targetState.NodeAddress != nil && vmi.Status.MigrationState.TargetNodeAddress == "" {
			vmi.Status.MigrationState.TargetNodeAddress = *targetState.NodeAddress
		}
	}

	vmi.Status.MigrationState.TargetNodeDomainDetected = targetState.DomainDetected
	vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp = targetState.DomainReadyTimestamp
	if targetState.NodeTopology != nil {
		vmi.Status.MigrationState.TargetNodeTopology = *targetState.NodeTopology
	}
	if targetState.PersistentStatePVCName != nil {
		vmi.Status.MigrationState.TargetPersistentStatePVCName = *targetState.PersistentStatePVCName
	}
	vmi.Status.MigrationState.TargetPod = targetState.Pod
	copyCommonLegacyFields(vmi.Status.MigrationState, migrationState)
	vmi.Status.MigrationState.Completed = migrationState.Completed
	vmi.Status.MigrationState.Failed = migrationState.Failed
}

func copyLegacySourceFields(vmi *virtv1.VirtualMachineInstance, migrationState *virtv1.VirtualMachineInstanceMigrationState) {
	vmi.Status.MigrationState.SourceNode = migrationState.SourceState.Node
	if migrationState.SourceState.PersistentStatePVCName != nil {
		vmi.Status.MigrationState.SourcePersistentStatePVCName = *migrationState.SourceState.PersistentStatePVCName
	}
	vmi.Status.MigrationState.SourcePod = migrationState.SourceState.Pod
	copyCommonLegacyFields(vmi.Status.MigrationState, migrationState)
	if migrationState.AbortRequested && migrationState.EndTimestamp != nil {
		vmi.Status.MigrationState.Failed = migrationState.Failed
		vmi.Status.MigrationState.Completed = migrationState.Completed
		if migrationState.AbortStatus != "" {
			vmi.Status.MigrationState.AbortStatus = migrationState.AbortStatus
		}
		if migrationState.FailureReason != "" {
			vmi.Status.MigrationState.FailureReason = migrationState.FailureReason
		}
	}
}

func copyCommonLegacyFields(targetMigrationState, sourceMigrationState *virtv1.VirtualMachineInstanceMigrationState) {
	// Copy regular fields.
	if sourceMigrationState.MigrationPolicyName != nil {
		targetMigrationState.MigrationPolicyName = sourceMigrationState.MigrationPolicyName
	}
	if sourceMigrationState.VMIMConfigurationOptions != nil {
		targetMigrationState.VMIMConfigurationOptions = sourceMigrationState.VMIMConfigurationOptions
	}
	if sourceMigrationState.StartTimestamp != nil {
		targetMigrationState.StartTimestamp = sourceMigrationState.StartTimestamp
	}
	if sourceMigrationState.EndTimestamp != nil {
		targetMigrationState.EndTimestamp = sourceMigrationState.EndTimestamp
	}
}

func (s *SynchronizationController) runConnectionCleanup() {
	s.failedCloseConnections.Range(func(k, v interface{}) bool {
		retryCount, ok := v.(int)
		if !ok {
			log.Log.Warningf("invalid retry count type during connection cleanup: %v", v)
			s.failedCloseConnections.Delete(k)
			return true
		}
		if retryCount >= maxCloseRetries {
			log.Log.Warningf("connection for migrationID %s failed to close after %d retries, not attempting to close again", k, retryCount)
			s.failedCloseConnections.Delete(k)
		}
		outboundConnection, ok := k.(*SynchronizationConnection)
		if !ok {
			log.Log.Warningf("invalid outbound connection type during connection cleanup: %v", k)
			s.failedCloseConnections.Delete(k)
			return true
		}
		if err := outboundConnection.Close(); err != nil {
			log.Log.Warningf("unable to close connection for migrationID, trying again: %s, %v", outboundConnection.migrationID, err)
			s.failedCloseConnections.Store(outboundConnection, retryCount+1)
		} else {
			s.failedCloseConnections.Delete(k)
		}
		return true
	})
}

func (s *SynchronizationController) CancelMigration(ctx context.Context, request *syncv1.MigrationCancelRequest) (*syncv1.MigrationCancelResponse, error) {
	migrationUID := request.MigrationUID

	migration, err := s.findMigrationFromMigrationIDByIndex(controller.ByMigrationUIDIndex, migrationUID)
	if err != nil {
		return &syncv1.MigrationCancelResponse{
			Message: fmt.Sprintf("unable to find migration to cancel for migrationUID %s", migrationUID),
		}, err
	}
	if migration != nil {
		log.Log.V(2).Object(migration).Infof("found migration to cancel for migrationUID %s", migrationUID)
		if err := s.client.VirtualMachineInstanceMigration(migration.Namespace).Delete(ctx, migration.Name, metav1.DeleteOptions{}); err != nil {
			return &syncv1.MigrationCancelResponse{
				Message: fmt.Sprintf("unable to cancel migration for migrationUID %s", migrationUID),
			}, err
		}
		log.Log.V(2).Object(migration).Infof("successfully deleted migration %s/%s for migrationUID %s", migration.Namespace, migration.Name, migrationUID)
	}
	return &syncv1.MigrationCancelResponse{
		Message: "migration canceled",
	}, nil
}

func (s *SynchronizationController) MigrationTunnel(stream syncv1.Synchronize_MigrationTunnelServer) error {
	// Source opens one bidi stream per migration channel on the shared control-plane
	// connection. The first frame is OPEN and identifies migration + channel.
	openFrame, err := stream.Recv()
	if err != nil {
		log.Log.Reason(err).Error("MigrationTunnel RPC failed to receive OPEN frame")
		return err
	}

	log.Log.V(4).Infof("Received migration tunnel stream for migration %s channel %d",
		openFrame.MigrationId, openFrame.ChannelId)

	if s.tunnelManager == nil {
		return fmt.Errorf("tunnel manager not initialized")
	}

	peerAddr := ""
	if p, ok := peer.FromContext(stream.Context()); ok && p.Addr != nil {
		peerAddr = p.Addr.String()
	}
	if err := s.tunnelManager.AuthorizeTunnelPeer(openFrame.MigrationId, peerAddr); err != nil {
		log.Log.Reason(err).Errorf("Rejecting MigrationTunnel for migration %s from peer %q",
			openFrame.MigrationId, peerAddr)
		return err
	}

	return s.tunnelManager.HandleInboundChannel(stream, openFrame)
}

// interfaceIP returns a global unicast IP on the named interface.
func interfaceIP(ifaceName string) (string, error) {
	ief, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", fmt.Errorf("%s not found: %w", ifaceName, err)
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return "", fmt.Errorf("%s present but no addresses: %w", ifaceName, err)
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		if !ip.IsGlobalUnicast() {
			continue
		}
		return ip.String(), nil
	}
	return "", fmt.Errorf("no IP found on %s", ifaceName)
}

// portMapToInt converts a port map from API format (map[string]int) to internal format (map[int]int)
func portMapToInt(apiMap map[string]int) (map[int]int, error) {
	if apiMap == nil {
		return nil, nil
	}
	result := make(map[int]int, len(apiMap))
	for portStr, value := range apiMap {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port string %q: %v", portStr, err)
		}
		result[port] = value
	}
	return result, nil
}

// portMapToString converts a port map from internal format (map[int]int) to API format (map[string]int)
func portMapToString(internalMap map[int]int) map[string]int {
	if internalMap == nil {
		return nil
	}
	result := make(map[string]int, len(internalMap))
	for port, value := range internalMap {
		result[strconv.Itoa(port)] = value
	}
	return result
}
