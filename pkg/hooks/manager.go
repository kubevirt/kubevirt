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
	"sort"
	"strings"
	"sync"
	"time"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type callBackClient struct {
	SocketPath           string
	Version              string
	subscribedHookPoints []*hooksInfo.HookPoint
}

var manager *Manager
var once sync.Once

type Manager struct {
	CallbacksPerHookPoint map[string][]*callBackClient
}

func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{CallbacksPerHookPoint: make(map[string][]*callBackClient)}
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

	m.CallbacksPerHookPoint = callbacksPerHookPoint

	return nil
}

// TODO: Handle sockets in parallel, when a socket appears, run a goroutine trying to read Info from it
func collectSideCarSockets(numberOfRequestedHookSidecars uint, timeout time.Duration) (map[string][]*callBackClient, error) {
	callbacksPerHookPoint := make(map[string][]*callBackClient)
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

				callBackClient, notReady, err := processSideCarSocket(HookSocketsSharedDirectory + "/" + socket.Name())
				if notReady {
					log.Log.Info("Sidecar server might not be ready yet, retrying in the next iteration")
					continue
				} else if err != nil {
					log.Log.Reason(err).Infof("Failed to process sidecar socket: %s", socket.Name())
					return nil, err
				}

				for _, subscribedHookPoint := range callBackClient.subscribedHookPoints {
					callbacksPerHookPoint[subscribedHookPoint.GetName()] = append(callbacksPerHookPoint[subscribedHookPoint.GetName()], callBackClient)
				}

				processedSockets[socket.Name()] = true
			}
		}

		time.Sleep(time.Second)
	}

	return callbacksPerHookPoint, nil
}

func processSideCarSocket(socketPath string) (*callBackClient, bool, error) {
	conn, err := grpcutil.DialSocketWithTimeout(socketPath, 1)
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

	if _, found := versionsSet[hooksV1alpha2.Version]; found {
		return &callBackClient{
			SocketPath:           socketPath,
			Version:              hooksV1alpha2.Version,
			subscribedHookPoints: info.GetHookPoints(),
		}, false, nil
	} else if _, found := versionsSet[hooksV1alpha1.Version]; found {
		return &callBackClient{
			SocketPath:           socketPath,
			Version:              hooksV1alpha1.Version,
			subscribedHookPoints: info.GetHookPoints(),
		}, false, nil
	} else {
		return nil, false,
			fmt.Errorf("Hook sidecar does not expose a supported version. Exposed versions: %v, supported versions: %v",
				info.GetVersions(), []string{hooksV1alpha1.Version, hooksV1alpha2.Version})
	}
}

func sortCallbacksPerHookPoint(callbacksPerHookPoint map[string][]*callBackClient) {
	for _, callbacks := range callbacksPerHookPoint {
		for _, callback := range callbacks {
			sort.Slice(callback.subscribedHookPoints, func(i, j int) bool {
				if callback.subscribedHookPoints[i].Priority == callback.subscribedHookPoints[j].Priority {
					return strings.Compare(callback.subscribedHookPoints[i].Name, callback.subscribedHookPoints[j].Name) < 0
				} else {
					return callback.subscribedHookPoints[i].Priority > callback.subscribedHookPoints[j].Priority
				}
			})
		}
	}
}

func (m *Manager) OnDefineDomain(domainSpec *virtwrapApi.DomainSpec, vmi *v1.VirtualMachineInstance) (string, error) {
	domainSpecXML, err := xml.Marshal(domainSpec)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal domain spec: %v", domainSpec)
	}
	if callbacks, found := m.CallbacksPerHookPoint[hooksInfo.OnDefineDomainHookPointName]; found {
		for _, callback := range callbacks {
			if callback.Version == hooksV1alpha1.Version || callback.Version == hooksV1alpha2.Version {
				vmiJSON, err := json.Marshal(vmi)
				if err != nil {
					return "", fmt.Errorf("Failed to marshal VMI spec: %v", vmi)
				}

				conn, err := grpcutil.DialSocketWithTimeout(callback.SocketPath, 1)
				if err != nil {
					log.Log.Reason(err).Infof("Failed to Dial hook socket: %s", callback.SocketPath)
					return "", err
				}
				defer conn.Close()

				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				switch callback.Version {
				case hooksV1alpha1.Version:
					client := hooksV1alpha1.NewCallbacksClient(conn)
					result, err := client.OnDefineDomain(ctx, &hooksV1alpha1.OnDefineDomainParams{
						DomainXML: domainSpecXML,
						Vmi:       vmiJSON,
					})
					if err != nil {
						return "", err
					}
					domainSpecXML = result.GetDomainXML()
				case hooksV1alpha2.Version:
					client := hooksV1alpha2.NewCallbacksClient(conn)
					result, err := client.OnDefineDomain(ctx, &hooksV1alpha2.OnDefineDomainParams{
						DomainXML: domainSpecXML,
						Vmi:       vmiJSON,
					})
					if err != nil {
						return "", err
					}
					domainSpecXML = result.GetDomainXML()
				default:
					panic("Should never happen, version compatibility check is done during Info call")
				}
			}
		}
	}
	return string(domainSpecXML), nil
}

func (m *Manager) PreCloudInitIso(vmi *v1.VirtualMachineInstance, cloudInitData *cloudinit.CloudInitData) (*cloudinit.CloudInitData, error) {
	if callbacks, found := m.CallbacksPerHookPoint[hooksInfo.PreCloudInitIsoHookPointName]; found {
		for _, callback := range callbacks {
			if callback.Version == hooksV1alpha2.Version {
				var resultData *cloudinit.CloudInitData
				vmiJSON, err := json.Marshal(vmi)
				if err != nil {
					return cloudInitData, fmt.Errorf("Failed to marshal VMI spec: %v", vmi)
				}

				// To be backward compatible to sidecar hooks still expecting to receive the cloudinit data as a CloudInitNoCloudSource object,
				// we need to construct a CloudInitNoCloudSource object with the user- and networkdata from the cloudInitData object.
				cloudInitNoCloudSource := v1.CloudInitNoCloudSource{
					UserData:    cloudInitData.UserData,
					NetworkData: cloudInitData.NetworkData,
				}
				cloudInitNoCloudSourceJSON, err := json.Marshal(cloudInitNoCloudSource)
				if err != nil {
					return cloudInitData, fmt.Errorf("Failed to marshal CloudInitNoCloudSource: %v", cloudInitNoCloudSource)
				}

				cloudInitDataJSON, err := json.Marshal(cloudInitData)
				if err != nil {
					return cloudInitData, fmt.Errorf("Failed to marshal CloudInitData: %v", cloudInitData)
				}

				conn, err := grpcutil.DialSocketWithTimeout(callback.SocketPath, 1)
				if err != nil {
					log.Log.Reason(err).Infof("Failed to Dial hook socket: %s", callback.SocketPath)
					return cloudInitData, err
				}
				defer conn.Close()

				client := hooksV1alpha2.NewCallbacksClient(conn)
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				result, err := client.PreCloudInitIso(ctx, &hooksV1alpha2.PreCloudInitIsoParams{
					CloudInitData:          cloudInitDataJSON,
					CloudInitNoCloudSource: cloudInitNoCloudSourceJSON,
					Vmi:                    vmiJSON,
				})
				if err != nil {
					return cloudInitData, err
				}

				err = json.Unmarshal(result.GetCloudInitData(), &resultData)
				if err != nil {
					log.Log.Reason(err).Infof("Failed to unmarshal CloudInitData result")
					return cloudInitData, err
				}
				if !cloudinit.IsValidCloudInitData(resultData) {
					// Be backwards compatible for hook sidecars still working on CloudInitNoCloudSource objects instead of CloudInitData
					var resultNoCloudSourceData *v1.CloudInitNoCloudSource
					err = json.Unmarshal(result.GetCloudInitNoCloudSource(), &resultNoCloudSourceData)
					if err != nil {
						log.Log.Reason(err).Infof("Failed to unmarshal CloudInitNoCloudSource result")
						return cloudInitData, err
					}
					resultData = &cloudinit.CloudInitData{
						DataSource:  cloudInitData.DataSource,
						UserData:    resultNoCloudSourceData.UserData,
						NetworkData: resultNoCloudSourceData.NetworkData,
					}
				}
				return resultData, nil
			} else {
				panic("Should never happen, version compatibility check is done during Info call")
			}
		}
	}
	return cloudInitData, nil
}
