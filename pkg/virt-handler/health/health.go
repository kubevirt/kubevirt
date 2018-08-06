package health

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/devices"
)

func NewReadinessChecker(clientset kubecli.KubevirtClient, host string, config *config.ClusterConfig) *ReadinessChecker {

	return &ReadinessChecker{
		clientset: clientset,
		host:      host,
		plugins: map[string]devices.Device{
			"/dev/kvm": &devices.KVM{
				ClusterConfig: config,
			},
			"/dev/net/tun": &devices.TUN{},
		},
		clock:         clock.RealClock{},
		clusterConfig: config,
	}
}

type ReadinessChecker struct {
	clientset     kubecli.KubevirtClient
	host          string
	plugins       map[string]devices.Device
	clock         clock.Clock
	clusterConfig *config.ClusterConfig
}

// HeartBeat take a heartbeat inverval, a maximum of non-userfacing errors which are
// allowed to happen and a stop channel to stop the heartbeat updates.
// It periodically performs some health checks and updates the kubevirt.io/schedulable according to its checks.
// Further it sets a timestamp on the node so that cluster components can see when it last updated the node.
func (l *ReadinessChecker) HeartBeat(interval time.Duration, maxErrorsPerInterval uint64, stopCh chan struct{}) {
	for {
		wait.JitterUntil(func() {
			schedulable := true

			// Check if the node has all mandatory devices set
			for dev, plugin := range l.plugins {
				if err := plugin.Available(); err != nil {
					log.DefaultLogger().Reason(err).Errorf("Check for mandatory device %s failed", dev)
					schedulable = false
				}
			}

			// Check for networks
			node, err := l.clientset.CoreV1().Nodes().Get(l.host, v12.GetOptions{})
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't fetch node")
				return
			}
			matchingNetworks, err := l.clusterConfig.GetLabelNetworksForNode(node)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine matching networks")
			}

			toPatch := []string{}

			for k, _ := range matchingNetworks {
				fmt.Printf("matching network %s found\n", k)
			}

			networksToDelete, err := l.clusterConfig.GetNotMatchingLabelNetworksOnNode(node)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine networks to delete")
			}

			for _, name := range networksToDelete {
				fmt.Printf("not matching network %s needs to be removed\n", name)
				toPatch = append(toPatch, fmt.Sprintf(`{ "op": "remove", "path": "/metadata/labels/%s" }`, escape(name)))
			}

			networksToAdd, err := l.clusterConfig.GetMissingLabelNetworksOnNode(node)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine networks to add")
			}

			for _, name := range networksToAdd {
				fmt.Printf("network %s needs to be added\n", name)
				toPatch = append(toPatch, fmt.Sprintf(`  { "op": "add", "path": "/metadata/labels/%s", "value": "" }`, escape(name)))
			}

			if len(toPatch) > 0 {
				patch := "[\n" + strings.Join(toPatch, ",\n") + "]"
				fmt.Println(patch)
				_, err = l.clientset.CoreV1().Nodes().Patch(l.host, types.JSONPatchType, []byte(patch))
				if err != nil {
					log.DefaultLogger().Reason(err).Errorf("Failed to patch networks")
				}
			}

			now, err := json.Marshal(v12.Time{Time: l.clock.Now()})
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine date")
				return
			}
			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%t"}, "annotations": {"%s": %s}}}`, v1.NodeSchedulable, schedulable, v1.VirtHandlerHeartbeat, string(now)))
			_, err = l.clientset.CoreV1().Nodes().Patch(l.host, types.StrategicMergePatchType, data)

			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", l.host)
			} else {
				log.DefaultLogger().V(4).Infof("Heartbeat sent")
			}
		}, interval, 1.2, true, stopCh)
	}
}

func escape(name string) string {
	return strings.Replace(name, "/", "~1", -1)
}
