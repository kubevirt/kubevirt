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

package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

const failedSetCPUManagerLabelFmt = "failed to set a cpu manager label on host %s"

type HeartBeat struct {
	clientset                 k8scli.CoreV1Interface
	deviceManagerController   device_manager.DeviceControllerInterface
	clusterConfig             *virtconfig.ClusterConfig
	host                      string
	cpuManagerPaths           []string
	devicePluginPollIntervall time.Duration
	devicePluginWaitTimeout   time.Duration
}

func NewHeartBeat(clientset k8scli.CoreV1Interface, deviceManager device_manager.DeviceControllerInterface, clusterConfig *virtconfig.ClusterConfig, host string) *HeartBeat {
	const cpuManagerOS3Path = virtutil.HostRootMount + "var/lib/origin/openshift.local.volumes/cpu_manager_state"
	const cpuManagerPath = virtutil.KubeletRoot + "/cpu_manager_state"
	return &HeartBeat{
		clientset:               clientset,
		deviceManagerController: deviceManager,
		clusterConfig:           clusterConfig,
		host:                    host,
		// This is a temporary workaround until k8s bug #66525 is resolved
		cpuManagerPaths:           []string{cpuManagerPath, cpuManagerOS3Path},
		devicePluginPollIntervall: 1 * time.Second,
		devicePluginWaitTimeout:   10 * time.Second,
	}
}

func (h *HeartBeat) Run(heartBeatInterval time.Duration, stopCh chan struct{}) (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		h.heartBeat(heartBeatInterval, stopCh)
		//ensure that the node is getting marked as unschedulable when removed
		h.labelNodeUnschedulable()
		close(done)
	}()
	return done
}

func (h *HeartBeat) heartBeat(heartBeatInterval time.Duration, stopCh chan struct{}) {
	// ensure that the node is synchronized with the actual state
	// especially setting the node to unschedulable if device plugins are not yet ready is very important
	// otherwise workloads get scheduled but are immediately terminated by the kubelet
	h.do()
	// Now wait for 10 seconds for the device plugins  to be initialized
	// This is more than fast enough to be not treated as unschedulable by the cluster
	// and ensures that the cluster gets marked as scheduled as soon as the device plugin is ready
	h.waitForDevicePlugins(stopCh)

	// from now on periodically update the node status
	// This sets the heartbeat to:
	// 1 minute with a 1.2 jitter + the time it takes for the heartbeat function to run (sliding == true).
	// So the amount of time between heartbeats randomly varies between 1min and 2min12sec + the heartbeat function execution time.
	wait.JitterUntil(h.do, heartBeatInterval, 1.2, true, stopCh)
}

func (h *HeartBeat) labelNodeUnschedulable() {
	err := retry.OnError(retry.DefaultBackoff, func(err error) bool { return err != nil }, func() error {
		now, err := json.Marshal(metav1.Now())
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Can't determine date")
			return err
		}

		cpuManagerEnabled := h.isCPUManagerEnabled(h.cpuManagerPaths)
		data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%s", "%s": "%t", "%s": "%t"}, "annotations": {"%s": %s}}}`,
			v1.NodeSchedulable, "false",
			v1.DeprecatedCPUManager, cpuManagerEnabled,
			v1.CPUManager, cpuManagerEnabled,
			v1.VirtHandlerHeartbeat, string(now),
		))
		_, err = h.clientset.Nodes().Patch(context.Background(), h.host, types.StrategicMergePatchType, data, metav1.PatchOptions{})
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", h.host)
			return err
		}
		return nil
	})

	if err != nil {
		log.Log.Warningf("Failed to set node %s unschedulable", h.host)
	}
}

// waitForDevicePlugins gives the device plugins additional time to successfully connect to the kubelet.
// If the connection can not be established it just delays the heartbeat start for devicePluginWaitTimeout.
func (h *HeartBeat) waitForDevicePlugins(stopCh chan struct{}) {
	_ = virtwait.PollImmediately(h.devicePluginPollIntervall, h.devicePluginWaitTimeout, func(_ context.Context) (done bool, err error) {
		select {
		case <-stopCh:
			return true, nil
		default:
		}
		return h.deviceManagerController.Initialized(), nil
	})
}

func (h *HeartBeat) do() {
	now, err := json.Marshal(metav1.Now())
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Can't determine date")
		return
	}

	kubevirtSchedulable := "true"
	if !h.deviceManagerController.Initialized() {
		kubevirtSchedulable = "false"
	}

	var data []byte
	// Label the node if cpu manager is running on it
	// This is a temporary workaround until k8s bug #66525 is resolved
	cpuManagerEnabled := h.isCPUManagerEnabled(h.cpuManagerPaths)

	data = []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%s", "%s": "%t", "%s": "%t"}, "annotations": {"%s": %s}}}`,
		v1.NodeSchedulable, kubevirtSchedulable,
		v1.DeprecatedCPUManager, cpuManagerEnabled,
		v1.CPUManager, cpuManagerEnabled,
		v1.VirtHandlerHeartbeat, string(now),
	))
	_, err = h.clientset.Nodes().Patch(context.Background(), h.host, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", h.host)
		return
	}

	// A configuration of mediated devices types on this node depends on the existing node labels
	// and a MediatedDevicesConfiguration in KubeVirt CR.
	// When labels change we should initialize a refresh to create/remove mdev types and start/stop
	// relevant device plugins. This operation should be async.
	if !h.clusterConfig.MediatedDevicesHandlingDisabled() {
		h.deviceManagerController.RefreshMediatedDeviceTypes()
	}

	log.DefaultLogger().V(4).Infof("Heartbeat sent")
}

func (h *HeartBeat) isCPUManagerEnabled(cpuManagerPaths []string) bool {
	var cpuManagerOptions map[string]interface{}
	cpuManagerPath, err := detectCPUManagerFile(cpuManagerPaths)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf(failedSetCPUManagerLabelFmt, h.host)
		return false
	}
	// #nosec No risk for path injection. cpuManagerPath is composed of static values from pkg/util
	content, err := os.ReadFile(cpuManagerPath)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf(failedSetCPUManagerLabelFmt, h.host)
		return false
	}

	err = json.Unmarshal(content, &cpuManagerOptions)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf(failedSetCPUManagerLabelFmt, h.host)
		return false
	}

	if v, ok := cpuManagerOptions["policyName"]; ok && v == "static" {
		log.DefaultLogger().V(4).Infof("Node has CPU Manager running")
		return true
	} else {
		log.DefaultLogger().V(4).Infof("Node has CPU Manager not runnning")
		return false
	}
}

func detectCPUManagerFile(cpuManagerPaths []string) (string, error) {
	for _, path := range cpuManagerPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no cpumanager policy file found")
}
