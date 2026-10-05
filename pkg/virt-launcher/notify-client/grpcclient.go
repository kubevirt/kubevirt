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
package eventsclient

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/wait"
	com "kubevirt.io/kubevirt/pkg/handler-launcher-com"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var (
	defaultIntervalTimeout = 1 * time.Second
	defaultSendTimeout     = 5 * time.Second
	defaultTotalTimeout    = 20 * time.Second
)

func newGRPCNotifyClient(virtShareDir string) *notifyClient {
	return &notifyClient{
		pipeSocketPath:  filepath.Join(virtShareDir, "domain-notify-pipe.sock"),
		intervalTimeout: defaultIntervalTimeout,
		sendTimeout:     defaultSendTimeout,
		totalTimeout:    defaultTotalTimeout,
	}
}

var _ notifierClient = &notifyClient{}

type notifyClient struct {
	v1client       notifyv1.NotifyClient
	conn           *grpc.ClientConn
	connLock       sync.Mutex
	pipeSocketPath string

	intervalTimeout time.Duration
	sendTimeout     time.Duration
	totalTimeout    time.Duration
}

func (n *notifyClient) HandleDomainEvent(event *watch.Event) error {
	var domainJSON []byte
	var statusJSON []byte
	var err error

	if event.Type == watch.Error {
		status := event.Object.(*metav1.Status)
		statusJSON, err = json.Marshal(status)
		if err != nil {
			log.Log.Reason(err).Infof("JSON marshal of notify ERROR event failed")
			return err
		}
	} else {
		domain := event.Object.(*api.Domain)
		domainJSON, err = json.Marshal(domain)
		if err != nil {
			log.Log.Reason(err).Infof("JSON marshal of notify event failed")
			return err
		}
	}
	request := notifyv1.DomainEventRequest{
		DomainJSON: domainJSON,
		StatusJSON: statusJSON,
		EventType:  string(event.Type),
	}

	var response *notifyv1.Response
	err = wait.PollImmediately(n.intervalTimeout, n.totalTimeout, func(ctx context.Context) (done bool, err error) {
		n.connLock.Lock()
		defer n.connLock.Unlock()

		err = n.connect()
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to connect to notify server")
			return false, nil
		}

		ctx, cancel := context.WithTimeout(ctx, n.sendTimeout)
		defer cancel()
		response, err = n.v1client.HandleDomainEvent(ctx, &request)
		if err != nil {
			// Retry transient errors (connection issues), propagate other errors immediately
			if isTransientError(err) {
				log.Log.Reason(err).Errorf("failed to notify domain event. closing connection.")
				n.close()
				return false, nil
			}
			log.Log.Reason(err).Errorf("failed to notify domain event")
			return false, err
		}

		return true, nil

	})

	if err != nil {
		if st, ok := grpcstatus.FromError(err); ok {
			return fmt.Errorf("failed to send domain notify event with %s: %w", st.Code(), err)
		}
		return err
	}

	// Fallback for old servers that embed errors in Response instead of using gRPC status codes.
	// The additional response.Message != "" check handles the case where a fully migrated server
	// no longer sets the Success field - in the legacy server, every
	// error response always includes a non-empty Message, so this condition is safe
	if response != nil && !response.Success && response.Message != "" {
		return fmt.Errorf("failed to notify domain event: %s", response.Message)
	}

	return nil
}

func (n *notifyClient) connect() error {
	if n.conn != nil {
		// already connected
		return nil
	}

	socketPath := n.detectSocketPath()

	// dial socket
	conn, err := grpcutil.DialSocketWithTimeout(socketPath, 5)
	if err != nil {
		log.Log.Reason(err).Infof("failed to dial notify socket: %s", socketPath)
		return err
	}

	version, err := negotiateVersion(info.NewNotifyInfoClient(conn))
	if err != nil {
		log.Log.Reason(err).Infof("failed to negotiate version")
		conn.Close()
		return err
	}

	// create cmd v1client
	switch version {
	case 1:
		client := notifyv1.NewNotifyClient(conn)
		n.v1client = client
		n.conn = conn
	default:
		conn.Close()
		return fmt.Errorf("cmd v1client version %v not implemented yet", version)
	}

	log.Log.Infof("Successfully connected to domain notify socket at %s", socketPath)
	return nil
}

func (n *notifyClient) detectSocketPath() string {
	return n.pipeSocketPath
}

func (n *notifyClient) Close() {
	n.connLock.Lock()
	defer n.connLock.Unlock()
	n.close()
}
func (n *notifyClient) close() {
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
}

func (n *notifyClient) HandleK8SEvent(event *k8sv1.Event) error {
	json, err := json.Marshal(event)
	if err != nil {
		return err
	}

	request := notifyv1.K8SEventRequest{
		EventJSON: json,
	}

	var response *notifyv1.Response
	err = wait.PollImmediately(n.intervalTimeout, n.totalTimeout, func(ctx context.Context) (done bool, err error) {
		n.connLock.Lock()
		defer n.connLock.Unlock()

		err = n.connect()
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to connect to notify server")
			return false, nil
		}

		ctx, cancel := context.WithTimeout(ctx, n.sendTimeout)
		defer cancel()
		response, err = n.v1client.HandleK8SEvent(ctx, &request)
		if err != nil {
			// Retry transient errors (connection issues), propagate business errors immediately
			if isTransientError(err) {
				log.Log.Reason(err).Errorf("failed to send k8s notify event. closing connection.")
				n.close()
				return false, nil
			}
			log.Log.Reason(err).Errorf("failed to send k8s notify event")
			return false, err
		}

		return true, nil
	})

	if err != nil {
		if st, ok := grpcstatus.FromError(err); ok {
			return fmt.Errorf("failed to notify k8s event with %s: %w", st.Code(), err)
		}
		return err
	}

	// Fallback for old servers that embed errors in Response instead of using gRPC status codes.
	// The additional response.Message != "" check handles the case where a fully migrated server
	// no longer sets the Success field - in the legacy server, every
	// error response always includes a non-empty Message, so this condition is safe
	if response != nil && !response.Success && response.Message != "" {
		return fmt.Errorf("failed to notify k8s event: %s", response.Message)
	}

	return nil
}

func negotiateVersion(infoClient info.NotifyInfoClient) (uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := infoClient.Info(ctx, &info.NotifyInfoRequest{})
	if err != nil {
		return 0, fmt.Errorf("could not check cmd server version: %v", err)
	}
	version, err := com.GetHighestCompatibleVersion(info.SupportedNotifyVersions, supportedNotifyVersions)
	if err != nil {
		return 0, err
	}

	switch version {
	case 1:
		// fall-through for all supported versions
	default:
		return 0, fmt.Errorf("cmd v1client version %v not implemented yet", version)
	}

	return version, nil
}

func isTransientError(err error) bool {
	st, ok := grpcstatus.FromError(err)
	if !ok {
		// Non-gRPC errors are typically network-level issues which are transient and should be retried
		return true
	}
	//nolint:exhaustive
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted:
		return true
	default:
		return false
	}
}
