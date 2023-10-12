package heartbeat

import (
	"bytes"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// In some environments, sysfs is mounted read-only even for privileged
// containers: https://github.com/containerd/containerd/issues/8445.
// Use the path from the host filesystem.
const ksmPath = "/proc/1/root/sys/kernel/mm/ksm/run"

func loadKSM() (bool, bool) {
	ksmValue, err := os.ReadFile(ksmPath)
	if err != nil {
		log.DefaultLogger().Warningf("An error occurred while reading the ksm module file; Maybe it is not available: %s", err)
		// Only enable for ksm-available nodes
		return false, false
	}

	return true, bytes.Equal(ksmValue, []byte("1\n"))
}

// handleKSM will update the ksm of the node (if available) based on the kv configuration and
// will set the outcome value to the n.KSM struct
// If the node labels match the selector terms, the ksm will be enabled.
// Empty Selector will enable ksm for every node
func handleKSM(node *v1.Node, clusterConfig *virtconfig.ClusterConfig) (bool, bool) {
	available, enabled := loadKSM()
	if !available {
		return enabled, false
	}

	ksmConfig := clusterConfig.GetKSMConfiguration()
	if ksmConfig == nil {
		if disableKSM(node, enabled) {
			return false, false
		} else {
			return enabled, false
		}
	}

	selector, err := metav1.LabelSelectorAsSelector(ksmConfig.NodeLabelSelector)
	if err != nil {
		log.DefaultLogger().Errorf("An error occurred while converting the ksm selector: %s", err)
		return enabled, false
	}

	if !selector.Matches(labels.Set(node.ObjectMeta.Labels)) {
		if disableKSM(node, enabled) {
			return false, false
		} else {
			return enabled, false
		}
	}

	if enableKSM(enabled) {
		return true, true
	} else {
		return enabled, false
	}
}

func enableKSM(enabled bool) bool {
	if !enabled {
		err := os.WriteFile(ksmPath, []byte("1\n"), 0644)
		if err != nil {
			log.DefaultLogger().Errorf("Unable to write ksm: %s", err.Error())
			return false
		}
	}

	log.DefaultLogger().Infof("KSM enabled")
	return true
}

func disableKSM(node *v1.Node, enabled bool) bool {
	if enabled {
		if _, found := node.GetAnnotations()[kubevirtv1.KSMHandlerManagedAnnotation]; found {
			err := os.WriteFile(ksmPath, []byte("0\n"), 0644)
			if err != nil {
				log.DefaultLogger().Errorf("Unable to write ksm: %s", err.Error())
				return false
			}
			return true
		}
	}
	return false
}
