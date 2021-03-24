package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

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
	return &HeartBeat{
		clientset:               clientset,
		deviceManagerController: deviceManager,
		clusterConfig:           clusterConfig,
		host:                    host,
		// This is a temporary workaround until k8s bug #66525 is resolved
		cpuManagerPaths:           []string{virtutil.CPUManagerPath, virtutil.CPUManagerOS3Path},
		devicePluginPollIntervall: 1 * time.Second,
		devicePluginWaitTimeout:   10 * time.Second,
	}
}

func (h *HeartBeat) Run(heartBeatInterval time.Duration, stopCh chan struct{}) (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		h.heartBeat(heartBeatInterval, stopCh)
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
	wait.JitterUntil(h.do, heartBeatInterval, 1.2, true, stopCh)
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
	cpuManagerEnabled := false
	if h.clusterConfig.CPUManagerEnabled() {
		cpuManagerEnabled = h.isCPUManagerEnabled(h.cpuManagerPaths)
	}
	data = []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%s", "%s": "%t"}, "annotations": {"%s": %s}}}`,
		v1.NodeSchedulable, kubevirtSchedulable,
		v1.CPUManager, cpuManagerEnabled,
		v1.VirtHandlerHeartbeat, string(now),
	))
	_, err = h.clientset.Nodes().Patch(context.Background(), h.host, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", h.host)
		return
	}
	log.DefaultLogger().V(4).Infof("Heartbeat sent")
}

func (h *HeartBeat) isCPUManagerEnabled(cpuManagerPaths []string) bool {
	var cpuManagerOptions map[string]interface{}
	cpuManagerPath, err := detectCPUManagerFile(cpuManagerPaths)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", h.host)
		return false
	}
	// #nosec No risk for path injection. cpuManagerPath is composed of static values from pkg/util
	content, err := ioutil.ReadFile(cpuManagerPath)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", h.host)
		return false
	}

	err = json.Unmarshal(content, &cpuManagerOptions)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", h.host)
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
