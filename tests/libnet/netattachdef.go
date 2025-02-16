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

package libnet

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

const (
	// ResourceNameAnnotation represents a resource name that is associated with the network.
	// It could be found on NetworkAttachmentDefinition objects.
	ResourceNameAnnotation = "k8s.v1.cni.cncf.io/resourceName"
)

type pluginConfOption func(map[string]interface{})

func NewBridgeNetAttachDef(name, bridgeName string, opts ...pluginConfOption) *nadv1.NetworkAttachmentDefinition {
	const cniPluginType = "bridge"

	pluginConfig := NewNetPluginConfig(
		cniPluginType,
		map[string]interface{}{"bridge": bridgeName},
	)
	for _, f := range opts {
		f(pluginConfig)
	}
	return NewNetAttachDef(name, NewNetConfig(name, pluginConfig))
}

func NewSriovNetAttachDef(name string, vlanID int, opts ...pluginConfOption) *nadv1.NetworkAttachmentDefinition {
	const pluginType = "sriov"
	pluginConf := map[string]interface{}{"vlan": vlanID}
	for _, f := range opts {
		f(pluginConf)
	}
	return NewNetAttachDef(name, NewNetConfig("sriov", NewNetPluginConfig(pluginType, pluginConf)))
}

func NewPasstNetAttachDef(name string) *nadv1.NetworkAttachmentDefinition {
	const pluginType = "kubevirt-passt-binding"
	return NewNetAttachDef(name, NewNetConfig(name, NewNetPluginConfig(pluginType, nil)))
}

func NewMacvtapNetAttachDef(name, lowerDevice string) *nadv1.NetworkAttachmentDefinition {
	const pluginType = "macvtap"
	netAttachDef := NewNetAttachDef(name, NewNetConfig(name, NewNetPluginConfig(pluginType, nil)))
	netAttachDef.Annotations = map[string]string{ResourceNameAnnotation: "macvtap.network.kubevirt.io/" + lowerDevice}
	return netAttachDef
}

func NewNetAttachDef(name, config string) *nadv1.NetworkAttachmentDefinition {
	return &nadv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       nadv1.NetworkAttachmentDefinitionSpec{Config: config},
	}
}

func NewNetConfig(name string, pluginsConfigs ...map[string]interface{}) string {
	const cniVersion = "0.3.1"
	netConfig := map[string]interface{}{
		"cniVersion": cniVersion,
		"name":       name,
	}
	switch len(pluginsConfigs) {
	case 0:
		panic("network configuration requires at least one plugin")
	case 1:
		// The SR-IOV CNI used at the moment is (for unknown reason) no supporting the new specification
		// with `plugins`. Therefore, the older format is kept until this is resolved.
		maps.Copy(netConfig, pluginsConfigs[0])
	default:
		netConfig["plugins"] = pluginsConfigs
	}

	rawNetConfig, err := json.Marshal(netConfig)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal:\n%s\nerror: %v", netConfig, err))
	}
	return string(rawNetConfig)
}

func NewNetPluginConfig(cniType string, conf map[string]interface{}) map[string]interface{} {
	if conf == nil {
		conf = map[string]interface{}{}
	}
	conf["type"] = cniType
	return conf
}

func WithMTU(mtu int) pluginConfOption {
	return func(conf map[string]interface{}) {
		conf["mtu"] = mtu
	}
}

func WithVLAN(vlanID int) pluginConfOption {
	return func(conf map[string]interface{}) {
		conf["vlan"] = vlanID
	}
}

func WithMacSpoofChk(enabled bool) pluginConfOption {
	return func(conf map[string]interface{}) {
		conf["macspoofchk"] = enabled
	}
}

func WithIPAM(ipam map[string]string) pluginConfOption {
	return func(conf map[string]interface{}) {
		conf["ipam"] = ipam
	}
}

func WithLinkState() pluginConfOption {
	return func(conf map[string]interface{}) {
		conf["link_state"] = "enable"
	}
}

func CreateNetAttachDef(
	ctx context.Context, namespace string, netAttachDef *nadv1.NetworkAttachmentDefinition,
) (*nadv1.NetworkAttachmentDefinition, error) {
	kvclient := kubevirt.Client()
	return kvclient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Create(
		ctx, netAttachDef, metav1.CreateOptions{},
	)
}
