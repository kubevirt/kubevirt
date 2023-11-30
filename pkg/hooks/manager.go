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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	hooksV1alpha3 "kubevirt.io/kubevirt/pkg/hooks/v1alpha3"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

const dialSockErr = "Failed to Dial hook socket: %s"

type callBackClient struct {
	SocketPath           string
	Version              string
	subscribedHookPoints []*hooksInfo.HookPoint
}

var manager Manager
var once sync.Once

type (
	Manager interface {
		Collect(uint, time.Duration) error
		OnDefineDomain(*virtwrapApi.DomainSpec, *v1.VirtualMachineInstance) (string, error)
		PreCloudInitIso(*v1.VirtualMachineInstance, *cloudinit.CloudInitData) (*cloudinit.CloudInitData, error)
		Shutdown() error
	}
	hookManager struct {
		CallbacksPerHookPoint     map[string][]*callBackClient
		hookSocketSharedDirectory string
	}
)

func GetManager() Manager {
	once.Do(func() {
		manager = newManager(HookSocketsSharedDirectory)
	})
	return manager
}

func newManager(baseDir string) *hookManager {
	return &hookManager{CallbacksPerHookPoint: make(map[string][]*callBackClient), hookSocketSharedDirectory: baseDir}
}

func (m *hookManager) Collect(numberOfRequestedHookSidecars uint, timeout time.Duration) error {
	callbacksPerHookPoint, err := m.collectSideCarSockets(numberOfRequestedHookSidecars, timeout)
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
func (m *hookManager) collectSideCarSockets(numberOfRequestedHookSidecars uint, timeout time.Duration) (map[string][]*callBackClient, error) {
	callbacksPerHookPoint := make(map[string][]*callBackClient)
	processedSockets := make(map[string]bool)

	timeoutCh := time.After(timeout)

	for uint(len(processedSockets)) < numberOfRequestedHookSidecars {
		sockets, err := os.ReadDir(m.hookSocketSharedDirectory)
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

				callBackClient, notReady, err := processSideCarSocket(filepath.Join(m.hookSocketSharedDirectory, socket.Name()))
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
		log.Log.Reason(err).Infof(dialSockErr, socketPath)
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

	// The order matters. We should match newer versions first.
	supportedVersions := []string{
		hooksV1alpha3.Version,
		hooksV1alpha2.Version,
		hooksV1alpha1.Version,
	}

	for _, version := range supportedVersions {
		if _, found := versionsSet[version]; found {
			return &callBackClient{
				SocketPath:           socketPath,
				Version:              version,
				subscribedHookPoints: info.GetHookPoints(),
			}, false, nil
		}
	}

	return nil, false,
		fmt.Errorf("Hook sidecar does not expose a supported version. Exposed versions: %v, supported versions: %v",
			info.GetVersions(), supportedVersions)
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

func (m *hookManager) OnDefineDomain(domainSpec *virtwrapApi.DomainSpec, vmi *v1.VirtualMachineInstance) (string, error) {
	domainSpecXML, err := xml.MarshalIndent(domainSpec, "", "\t")
	if err != nil {
		return "", fmt.Errorf("Failed to marshal domain spec: %v", domainSpec)
	}

	callbacks, found := m.CallbacksPerHookPoint[hooksInfo.OnDefineDomainHookPointName]
	if !found {
		return string(domainSpecXML), nil
	}

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VMI spec: %v, err: %v", vmi, err)
	}

	for _, callback := range callbacks {
		domainSpecXML, err = m.onDefineDomainCallback(callback, domainSpecXML, vmiJSON)
		if err != nil {
			return "", err
		}
	}

	return string(domainSpecXML), nil
}

func (m *hookManager) onDefineDomainCallback(callback *callBackClient, domainSpecXML, vmiJSON []byte) ([]byte, error) {
	conn, err := grpcutil.DialSocketWithTimeout(callback.SocketPath, 1)
	if err != nil {
		log.Log.Reason(err).Errorf(dialSockErr, callback.SocketPath)
		return nil, err
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
			log.Log.Reason(err).Error("Failed to call OnDefineDomain")
			return nil, err
		}
		domainSpecXML = result.GetDomainXML()
	case hooksV1alpha2.Version:
		client := hooksV1alpha2.NewCallbacksClient(conn)
		result, err := client.OnDefineDomain(ctx, &hooksV1alpha2.OnDefineDomainParams{
			DomainXML: domainSpecXML,
			Vmi:       vmiJSON,
		})
		if err != nil {
			log.Log.Reason(err).Error("Failed to call OnDefineDomain")
			return nil, err
		}
		domainSpecXML = result.GetDomainXML()
	case hooksV1alpha3.Version:
		client := hooksV1alpha3.NewCallbacksClient(conn)
		result, err := client.OnDefineDomain(ctx, &hooksV1alpha3.OnDefineDomainParams{
			DomainXML: domainSpecXML,
			Vmi:       vmiJSON,
		})
		if err != nil {
			log.Log.Reason(err).Error("Failed to call OnDefineDomain")
			return nil, err
		}
		domainSpecXML = result.GetDomainXML()
	default:
		log.Log.Errorf("Unsupported callback version: %s", callback.Version)
	}

	return domainSpecXML, nil
}

func preCloudInitIsoDataToJSON(vmi *v1.VirtualMachineInstance, cloudInitData *cloudinit.CloudInitData) ([]byte, []byte, []byte, error) {
	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal VMI spec, err: %v", err)
	}

	// To be backward compatible to sidecar hooks still expecting to receive the cloudinit data as a
	// CloudInitNoCloudSource object,
	// we need to construct a CloudInitNoCloudSource object with the user- and networkdata from the
	// cloudInitData object.
	cloudInitNoCloudSource := v1.CloudInitNoCloudSource{
		UserData:    cloudInitData.UserData,
		NetworkData: cloudInitData.NetworkData,
	}
	cloudInitNoCloudSourceJSON, err := json.Marshal(cloudInitNoCloudSource)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal CloudInitNoCloudSource: %v, err: %v", cloudInitNoCloudSource, err)
	}

	cloudInitDataJSON, err := json.Marshal(cloudInitData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal CloudInitData, err: %v", err)
	}

	return cloudInitDataJSON, cloudInitNoCloudSourceJSON, vmiJSON, nil
}

func preCloudInitIsoValidateResult(dataSource cloudinit.DataSourceType, initData, noCloudSource []byte) (*cloudinit.CloudInitData, error) {
	var resultData *cloudinit.CloudInitData
	if err := json.Unmarshal(initData, &resultData); err != nil {
		log.Log.Reason(err).Error("Failed to unmarshal CloudInitData result")
		return nil, err
	}

	if !cloudinit.IsValidCloudInitData(resultData) {
		// Be backwards compatible for hook sidecars still working on CloudInitNoCloudSource objects instead of CloudInitData
		var resultNoCloudSourceData *v1.CloudInitNoCloudSource
		if err := json.Unmarshal(noCloudSource, &resultNoCloudSourceData); err != nil {
			log.Log.Reason(err).Error("Failed to unmarshal CloudInitNoCloudSource result")
			return nil, err
		}
		resultData = &cloudinit.CloudInitData{
			DataSource:  dataSource,
			UserData:    resultNoCloudSourceData.UserData,
			NetworkData: resultNoCloudSourceData.NetworkData,
		}
	}
	return resultData, nil
}

func (m *hookManager) PreCloudInitIso(vmi *v1.VirtualMachineInstance, cloudInitData *cloudinit.CloudInitData) (*cloudinit.CloudInitData, error) {
	callbacks, found := m.CallbacksPerHookPoint[hooksInfo.PreCloudInitIsoHookPointName]
	if !found {
		return cloudInitData, nil
	}

	cloudInitDataJSON, cloudInitNoCloudSourceJSON, vmiJSON, err := preCloudInitIsoDataToJSON(vmi, cloudInitData)
	if err != nil {
		log.Log.Reason(err).Error("Failed to run PreCloudInitIso")
		return cloudInitData, err
	}

	for _, callback := range callbacks {
		switch callback.Version {
		case hooksV1alpha2.Version:
			conn, err := grpcutil.DialSocketWithTimeout(callback.SocketPath, 1)
			if err != nil {
				log.Log.Reason(err).Errorf(dialSockErr, callback.SocketPath)
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
				log.Log.Reason(err).Error("Failed to call PreCloudInitIso")
				return cloudInitData, err
			}
			return preCloudInitIsoValidateResult(cloudInitData.DataSource, result.GetCloudInitData(), result.GetCloudInitNoCloudSource())
		case hooksV1alpha3.Version:
			conn, err := grpcutil.DialSocketWithTimeout(callback.SocketPath, 1)
			if err != nil {
				log.Log.Reason(err).Errorf(dialSockErr, callback.SocketPath)
				return cloudInitData, err
			}
			defer conn.Close()

			client := hooksV1alpha3.NewCallbacksClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			result, err := client.PreCloudInitIso(ctx, &hooksV1alpha3.PreCloudInitIsoParams{
				CloudInitData:          cloudInitDataJSON,
				CloudInitNoCloudSource: cloudInitNoCloudSourceJSON,
				Vmi:                    vmiJSON,
			})
			if err != nil {
				log.Log.Reason(err).Error("Failed to call PreCloudInitIso")
				return cloudInitData, err
			}
			return preCloudInitIsoValidateResult(cloudInitData.DataSource, result.GetCloudInitData(), result.GetCloudInitNoCloudSource())
		default:
			log.Log.Errorf("Unsupported callback version: %s", callback.Version)
		}
	}
	return cloudInitData, nil
}

func (m *hookManager) Shutdown() error {
	callbacks, found := m.CallbacksPerHookPoint[hooksInfo.ShutdownHookPointName]
	if !found {
		return nil
	}
	for _, callback := range callbacks {
		switch callback.Version {
		case hooksV1alpha3.Version:
			conn, err := grpcutil.DialSocketWithTimeout(callback.SocketPath, 1)
			if err != nil {
				log.Log.Reason(err).Error("Failed to run Shutdown")
				return err
			}
			defer conn.Close()

			client := hooksV1alpha3.NewCallbacksClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			if _, err := client.Shutdown(ctx, &hooksV1alpha3.ShutdownParams{}); err != nil {
				log.Log.Reason(err).Error("Failed to run Shutdown")
				return err
			}
		default:
			log.Log.Errorf("Unsupported callback version: %s", callback.Version)
		}
	}
	return nil
}
