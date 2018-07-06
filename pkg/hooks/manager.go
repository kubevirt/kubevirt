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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package hooks

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/api/v1"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	"kubevirt.io/kubevirt/pkg/log"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type callackClient struct {
	SocketPath          string
	Version             string
	subsribedHookPoints []*hooksInfo.HookPoint
}

var manager *Manager
var once sync.Once

type Manager struct {
	callbacksPerHookPoint map[string][]*callackClient
}

func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{callbacksPerHookPoint: make(map[string][]*callackClient)}
	})
	return manager
}

func (m *Manager) Collect(numberOfRequestedHookSidecars uint, timeout time.Duration) error {
	callbacksPerHookPoint, err := collectSideCarSockets(numberOfRequestedHookSidecars, timeout)
	if err != nil {
		return err
	}
	log.Log.Info("Collected all requested hook sidecar sockets")

	sortCallbacksPerHookPoint(callbacksPerHookPoint)
	log.Log.Infof("Sorted all collected sidecar sockets per hook point based on their priority and name: %v", callbacksPerHookPoint)

	m.callbacksPerHookPoint = callbacksPerHookPoint

	return nil
}

// TODO: Handle sockets in parallel, when a socket appears, run a goroutine trying to read Info from it
func collectSideCarSockets(numberOfRequestedHookSidecars uint, timeout time.Duration) (map[string][]*callackClient, error) {
	callbacksPerHookPoint := make(map[string][]*callackClient)
	processedSockets := make(map[string]bool)

	timeoutCh := time.After(timeout)

	for uint(len(processedSockets)) < numberOfRequestedHookSidecars {
		sockets, err := ioutil.ReadDir(HookSocketsSharedDirectory)
		if err != nil {
			return nil, err
		}

		for _, socket := range sockets {
			select {
			case <-timeoutCh:
				return nil, fmt.Errorf("Failed to collect all expected sidecar hook sockets within given timeout")
			default:
				if _, processed := processedSockets[socket.Name()]; processed {
					continue
				}

				callackClient, notReady, err := processSideCarSocket(HookSocketsSharedDirectory + "/" + socket.Name())
				if notReady {
					log.Log.Info("Sidecar server might not be ready yet, retrying in the next iteration")
					continue
				} else if err != nil {
					log.Log.Reason(err).Infof("Failed to process sidecar socket: %s", socket.Name())
					return nil, err
				}

				for _, subsribedHookPoint := range callackClient.subsribedHookPoints {
					callbacksPerHookPoint[subsribedHookPoint.GetName()] = append(callbacksPerHookPoint[subsribedHookPoint.GetName()], callackClient)
				}

				processedSockets[socket.Name()] = true
			}
		}

		time.Sleep(time.Second)
	}

	return callbacksPerHookPoint, nil
}

func processSideCarSocket(socketPath string) (*callackClient, bool, error) {
	conn, err := dialSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Infof("Failed to Dial hook socket: %s", socketPath)
		return nil, true, nil
	}
	defer conn.Close()

	infoClient := hooksInfo.NewInfoClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	info, err := infoClient.Info(ctx, &hooksInfo.InfoParams{})
	if err != nil {
		return nil, false, err
	}

	versionsSet := make(map[string]bool)
	for _, version := range info.GetVersions() {
		versionsSet[version] = true
	}

	if _, found := versionsSet[hooksV1alpha1.Version]; found {
		return &callackClient{
			SocketPath:          socketPath,
			Version:             hooksV1alpha1.Version,
			subsribedHookPoints: info.GetHookPoints(),
		}, false, nil
	} else {
		return nil, false, fmt.Errorf("Hook sidecar does not expose a supported version. Exposed versions: %v, supported versions: %s", versionsSet, hooksV1alpha1.Version)
	}
}

func sortCallbacksPerHookPoint(callbacksPerHookPoint map[string][]*callackClient) {
	for _, callbacks := range callbacksPerHookPoint {
		for _, callback := range callbacks {
			sort.Slice(callbacks, func(i, j int) bool {
				if callback.subsribedHookPoints[i].Priority == callback.subsribedHookPoints[j].Priority {
					return strings.Compare(callback.subsribedHookPoints[i].Name, callback.subsribedHookPoints[j].Name) < 0
				} else {
					return callback.subsribedHookPoints[i].Priority > callback.subsribedHookPoints[j].Priority
				}
			})
		}
	}
}

func (m *Manager) OnDefineDomain(domainSpec *virtwrapApi.DomainSpec, vmi *v1.VirtualMachineInstance) (*virtwrapApi.DomainSpec, error) {
	if callbacks, found := m.callbacksPerHookPoint[hooksInfo.OnDefineDomainHookPointName]; found {
		for _, callback := range callbacks {
			if callback.Version == hooksV1alpha1.Version {
				domainSpecXML, err := xml.Marshal(domainSpec)
				if err != nil {
					return nil, fmt.Errorf("Failed to marshal domain spec: %v", domainSpec)
				}
				vmiJSON, err := json.Marshal(vmi)
				if err != nil {
					return nil, fmt.Errorf("Failed to marshal VMI spec: %v", vmi)
				}

				conn, err := dialSocket(callback.SocketPath)
				if err != nil {
					log.Log.Reason(err).Infof("Failed to Dial hook socket: %s", callback.SocketPath)
					return nil, err
				}
				defer conn.Close()

				client := hooksV1alpha1.NewCallbacksClient(conn)

				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				result, err := client.OnDefineDomain(ctx, &hooksV1alpha1.OnDefineDomainParams{
					DomainXML: domainSpecXML,
					Vmi:       vmiJSON,
				})
				if err != nil {
					return nil, err
				}

				newDomainSpecXML := result.GetDomainXML()
				newDomainSpec := domainSchema.DomainSpec{}
				err = xml.Unmarshal(newDomainSpecXML, &newDomainSpec)
				if err != nil {
					return nil, fmt.Errorf("Failed to unmarshal given domain spec: %s", newDomainSpecXML)
				}

				domainSpec = &newDomainSpec
			} else {
				panic("Should never happen, version compatibility check is done during Info call")
			}
		}
	}

	return domainSpec, nil
}

func dialSocket(socketPath string) (*grpc.ClientConn, error) {
	return grpc.Dial(
		socketPath,
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
		grpc.WithTimeout(time.Second),
	)
}
