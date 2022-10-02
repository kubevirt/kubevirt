package heartbeat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"k8s.io/utils/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

const failedSetCPUManagerLabelFmt = "failed to set a cpu manager label on host %s"

type HeartBeat struct {
	clientset                           k8scli.CoreV1Interface
	deviceManagerController             device_manager.DeviceControllerInterface
	clusterConfig                       *virtconfig.ClusterConfig
	host                                string
	cpuManagerPaths                     []string
	devicePluginPollIntervall           time.Duration
	devicePluginWaitTimeout             time.Duration
	isVirtualizationSupportedOnNodeFunc func() bool
	isVirtualizationSupportedOnNode     *bool
}

func NewHeartBeat(clientset k8scli.CoreV1Interface, deviceManager device_manager.DeviceControllerInterface, clusterConfig *virtconfig.ClusterConfig, host string) *HeartBeat {
	return &HeartBeat{
		clientset:               clientset,
		deviceManagerController: deviceManager,
		clusterConfig:           clusterConfig,
		host:                    host,
		// This is a temporary workaround until k8s bug #66525 is resolved
		cpuManagerPaths:                     []string{virtutil.CPUManagerPath, virtutil.CPUManagerOS3Path},
		devicePluginPollIntervall:           1 * time.Second,
		devicePluginWaitTimeout:             10 * time.Second,
		isVirtualizationSupportedOnNodeFunc: isVirtualizationSupportedOnNodeImpl,
	}
}

func (h *HeartBeat) Run(heartBeatInterval time.Duration, stopCh chan struct{}) (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		h.heartBeat(heartBeatInterval, stopCh)
		//ensure that the node is getting marked as unschedulable when removed
		labelNodeDone := h.labelNodeUnschedulable()
		<-labelNodeDone
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

func (h *HeartBeat) labelNodeUnschedulable() (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		now, err := json.Marshal(metav1.Now())
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Can't determine date")
			return
		}
		var data []byte
		cpuManagerEnabled := false
		if h.clusterConfig.CPUManagerEnabled() {
			cpuManagerEnabled = h.isCPUManagerEnabled(h.cpuManagerPaths)
		}
		data = []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%s", "%s": "%t"}, "annotations": {"%s": %s}}}`,
			v1.NodeSchedulable, "false",
			v1.CPUManager, cpuManagerEnabled,
			v1.VirtHandlerHeartbeat, string(now),
		))
		_, err = h.clientset.Nodes().Patch(context.Background(), h.host, types.StrategicMergePatchType, data, metav1.PatchOptions{})
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", h.host)
			return
		}
		close(done)
	}()
	return done
}

// waitForDevicePlugins gives the device plugins additional time to successfully connect to the kubelet.
// If the connection can not be established it just delays the heartbeat start for devicePluginWaitTimeout.
func (h *HeartBeat) waitForDevicePlugins(stopCh chan struct{}) {
	_ = utilwait.PollImmediate(h.devicePluginPollIntervall, h.devicePluginWaitTimeout, func() (done bool, err error) {
		select {
		case <-stopCh:
			return true, nil
		default:
		}
		return h.deviceManagerController.Initialized(), nil
	})
}

type annotationType map[string]string
type labelType map[string]string

func (h *HeartBeat) do() {
	now, err := json.Marshal(metav1.Now())
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Can't determine date")
		return
	}

	var patchedLabels labelType = map[string]string{}
	var patchedAnnotations annotationType = map[string]string{}

	h.setSchedulable(patchedLabels, patchedAnnotations)
	h.setCpuManager(patchedLabels)
	h.setHeartbeat(patchedAnnotations, string(now))

	marshalledLabels, err := json.Marshal(patchedLabels)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("cannot marshall labels")
		return
	}

	marshalledAnnotations, err := json.Marshal(patchedAnnotations)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("cannot marshall annotations")
		return
	}

	var data []byte
	// Label the node if cpu manager is running on it
	// This is a temporary workaround until k8s bug #66525 is resolved
	data = []byte(fmt.Sprintf(`{"metadata": { "labels": %s, "annotations": %s }}`,
		string(marshalledLabels),
		string(marshalledAnnotations),
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

func isVirtualizationSupportedOnNodeImpl() bool {
	// If an error occurs in this process (can't parse info file / any other unexpected failure) this function will
	// return true just to stay on the safe side. false will be returned only if we are certain that virtualization
	// is not supported.

	const cpuInfoFilePath = "/proc/cpuinfo"

	cpuInfoFile, err := os.Open(cpuInfoFilePath)
	if err != nil {
		log.Log.Reason(err).Errorf("cannot open %s", cpuInfoFilePath)
		return true
	}
	defer cpuInfoFile.Close()

	scanner := bufio.NewScanner(cpuInfoFile)
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, []byte("vmx")) || bytes.Contains(line, []byte("svm")) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Log.Reason(err).Errorf("error while parsing %s", cpuInfoFilePath)
		return true
	}

	return false
}

func (h *HeartBeat) setSchedulable(labels labelType, annotations annotationType) {
	isKubevirtSchedulable := true

	if !h.deviceManagerController.Initialized() {
		isKubevirtSchedulable = false
	}

	if isKubevirtSchedulable && !h.clusterConfig.AllowEmulation() {
		// Host's virtualization capabilities are assumed to never change. Therefore use cached result if exists.
		var isVirtualizationSupported bool
		if h.isVirtualizationSupportedOnNode == nil {
			isVirtualizationSupported = h.isVirtualizationSupportedOnNodeFunc()
			h.isVirtualizationSupportedOnNode = pointer.Bool(isVirtualizationSupported)
		} else {
			isVirtualizationSupported = *h.isVirtualizationSupportedOnNode
		}

		if !isVirtualizationSupported {
			isKubevirtSchedulable = false
			annotations[v1.LabellerSkipNodeAnnotation] = fmt.Sprintf("%t", true)
		}
	}

	labels[v1.NodeSchedulable] = fmt.Sprintf("%t", isKubevirtSchedulable)
}

func (h *HeartBeat) setCpuManager(labels labelType) {
	cpuManagerEnabled := false

	if h.clusterConfig.CPUManagerEnabled() {
		cpuManagerEnabled = h.isCPUManagerEnabled(h.cpuManagerPaths)
	}

	labels[v1.CPUManager] = fmt.Sprintf("%t", cpuManagerEnabled)
}

func (h *HeartBeat) setHeartbeat(annotations annotationType, timestamp string) {
	annotations[v1.VirtHandlerHeartbeat] = timestamp
}
