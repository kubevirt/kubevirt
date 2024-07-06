package topology

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"
	"time"

	nodeutils "kubevirt.io/kubevirt/pkg/util/nodes"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/client-go/log"
)

type NodeTopologyUpdater interface {
	Run(interval time.Duration, stopChan <-chan struct{})
}

type nodeTopologyUpdater struct {
	nodeInformer cache.SharedIndexInformer
	hinter       Hinter
	client       kubecli.KubevirtClient
}

type updateStats struct {
	updated int
	skipped int
	error   int
}

func (n *nodeTopologyUpdater) Run(interval time.Duration, stopChan <-chan struct{}) {
	cache.WaitForCacheSync(stopChan, n.nodeInformer.HasSynced)
	wait.JitterUntil(func() {
		nodes := FilterNodesFromCache(n.nodeInformer.GetStore().List(),
			HasInvTSCFrequency,
		)
		stats := n.sync(nodes)
		if stats.updated != 0 || stats.error != 0 {
			log.DefaultLogger().Infof("TSC Frequency node update status: %d updated, %d skipped, %d errors", stats.updated, stats.skipped, stats.error)
		}
	}, interval, 1.2, true, stopChan)
}

func (n *nodeTopologyUpdater) sync(nodes []*v1.Node) *updateStats {
	requiredFrequencies, err := n.requiredFrequencies()
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Skipping TSC frequency updates on all nodes")
		return &updateStats{skipped: len(nodes)}
	}
	stats := &updateStats{}
	for _, node := range nodes {
		nodeCopy, err := calculateNodeLabelChanges(node, requiredFrequencies)
		if err != nil {
			stats.error++
			log.DefaultLogger().Object(node).Reason(err).Error("Could not calculate TSC frequencies for node")
			continue
		}
		if !equality.Semantic.DeepEqual(node.Labels, nodeCopy.Labels) {
			if err := nodeutils.PatchNode(n.client, node, nodeCopy); err != nil {
				stats.error++
				log.DefaultLogger().Object(node).Reason(err).Error("Could not patch TSC frequencies for node")
				continue
			}
			stats.updated++
		} else {
			stats.skipped++
		}
	}
	return stats
}

func calculateNodeLabelChanges(original *v1.Node, requiredFrequencies []int64) (modified *v1.Node, err error) {
	nodeFreq, scalable, err := TSCFrequencyFromNode(original)
	if err != nil {
		log.DefaultLogger().Reason(err).Object(original).Errorf("Can't determine original TSC frequency of node %s", original.Name)
		return nil, err
	}
	freqsOnNode := TSCFrequenciesOnNode(original)
	toAdd, toRemove := CalculateTSCLabelDiff(requiredFrequencies, freqsOnNode, nodeFreq, scalable)
	toAddLabels := ToTSCSchedulableLabels(toAdd)
	toRemoveLabels := ToTSCSchedulableLabels(toRemove)

	nodeCopy := original.DeepCopy()
	for _, freq := range toAddLabels {
		nodeCopy.Labels[freq] = "true"
	}
	for _, freq := range toRemoveLabels {
		delete(nodeCopy.Labels, freq)
	}
	return nodeCopy, nil
}

func (n nodeTopologyUpdater) requiredFrequencies() ([]int64, error) {
	lowestFrequency, err := n.hinter.LowestTSCFrequencyOnCluster()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate lowest TSC frequency for nodes: %v", err)
	}
	return append(n.hinter.TSCFrequenciesInUse(), lowestFrequency), nil
}

func NewNodeTopologyUpdater(clientset kubecli.KubevirtClient, hinter Hinter, nodeInformer cache.SharedIndexInformer) NodeTopologyUpdater {
	return &nodeTopologyUpdater{
		client:       clientset,
		hinter:       hinter,
		nodeInformer: nodeInformer,
	}
}
