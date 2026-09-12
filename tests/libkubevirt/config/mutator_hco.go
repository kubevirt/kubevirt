//go:build managed_hco

package config

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

type hcoMutator struct{}

func newConfigMutator() ConfigMutator {
	return &hcoMutator{}
}

func (h *hcoMutator) resolveTarget() (string, string) {
	ns := os.Getenv("HCO_NAMESPACE")
	if ns == "" {
		ns = flags.KubeVirtInstallNamespace
	}
	name := os.Getenv("HCO_NAME")
	if name == "" {
		name = "kubevirt-hyperconverged"
	}
	return ns, name
}

func (h *hcoMutator) Apply(config v1.KubeVirtConfiguration) (*v1.KubeVirt, error) {
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	if equality.Semantic.DeepEqual(kv.Spec.Configuration, config) {
		return kv, nil
	}

	mergePatch := h.buildHCOMergePatch(kv.Spec.Configuration, config)
	if mergePatch == nil {
		return kv, nil
	}

	patchData, err := json.Marshal(mergePatch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal HCO merge patch: %w", err)
	}

	if err := h.patchHCO(patchData); err != nil {
		return nil, err
	}

	kv = h.waitForHCOReconciliation(config)
	return kv, nil
}

func (h *hcoMutator) patchHCO(patchData []byte) error {
	ns, name := h.resolveTarget()

	kubectlPath := ""
	if f := flag.Lookup("kubectl-path"); f != nil {
		kubectlPath = f.Value.String()
	}
	if kubectlPath == "" {
		kubectlPath = "kubectl"
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if f := flag.Lookup("kubeconfig"); f != nil {
			kubeconfig = f.Value.String()
		}
	}

	args := []string{
		"patch", "hyperconverged", name,
		"-n", ns,
		"--type=merge",
		"-p", string(patchData),
	}
	if kubeconfig != "" {
		args = append(args, "--kubeconfig="+kubeconfig)
	}

	log.DefaultLogger().Infof("patching HCO CR %s/%s", ns, name)

	cmd := exec.CommandContext(context.Background(), kubectlPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl patch hyperconverged failed: %w, output: %s",
			err, string(output))
	}
	return nil
}

func (h *hcoMutator) waitForHCOReconciliation(expected v1.KubeVirtConfiguration) *v1.KubeVirt {
	var kv *v1.KubeVirt
	EventuallyWithOffset(2, func() bool {
		kv = libkubevirt.GetCurrentKv(kubevirt.Client())
		return h.configFieldsMatch(kv.Spec.Configuration, expected)
	}, 60*time.Second, 2*time.Second).Should(BeTrue(),
		"HCO did not propagate configuration to KubeVirt CR within timeout")

	return kv
}

func (h *hcoMutator) configFieldsMatch(actual, expected v1.KubeVirtConfiguration) bool {
	if expected.CPUModel != "" && actual.CPUModel != expected.CPUModel {
		return false
	}
	if expected.EvictionStrategy != nil &&
		(actual.EvictionStrategy == nil || *actual.EvictionStrategy != *expected.EvictionStrategy) {
		return false
	}
	if expected.MigrationConfiguration != nil {
		if actual.MigrationConfiguration == nil {
			return false
		}
		mc, ac := expected.MigrationConfiguration, actual.MigrationConfiguration
		if mc.NodeDrainTaintKey != nil &&
			(ac.NodeDrainTaintKey == nil || *mc.NodeDrainTaintKey != *ac.NodeDrainTaintKey) {
			return false
		}
		if mc.AllowPostCopy != nil &&
			(ac.AllowPostCopy == nil || *mc.AllowPostCopy != *ac.AllowPostCopy) {
			return false
		}
		if mc.CompletionTimeoutPerGiB != nil &&
			(ac.CompletionTimeoutPerGiB == nil || *mc.CompletionTimeoutPerGiB != *ac.CompletionTimeoutPerGiB) {
			return false
		}
		if mc.BandwidthPerMigration != nil &&
			(ac.BandwidthPerMigration == nil || !mc.BandwidthPerMigration.Equal(*ac.BandwidthPerMigration)) {
			return false
		}
	}
	if expected.ObsoleteCPUModels != nil {
		if actual.ObsoleteCPUModels == nil {
			return false
		}
		for k, v := range expected.ObsoleteCPUModels {
			if actual.ObsoleteCPUModels[k] != v {
				return false
			}
		}
	}
	return true
}

func (h *hcoMutator) buildHCOMergePatch(oldConfig, newConfig v1.KubeVirtConfiguration) map[string]interface{} {
	virtualization := map[string]interface{}{}

	if newConfig.CPUModel != oldConfig.CPUModel {
		virtualization["virtualMachineOptions"] = map[string]interface{}{
			"defaultCPUModel": newConfig.CPUModel,
		}
	}

	if newConfig.EvictionStrategy != nil &&
		(oldConfig.EvictionStrategy == nil || *oldConfig.EvictionStrategy != *newConfig.EvictionStrategy) {
		virtualization["evictionStrategy"] = string(*newConfig.EvictionStrategy)
	}

	if newConfig.MigrationConfiguration != nil {
		mc := newConfig.MigrationConfiguration
		oldMC := oldConfig.MigrationConfiguration
		liveMigration := map[string]interface{}{}

		if mc.NodeDrainTaintKey != nil &&
			(oldMC == nil || oldMC.NodeDrainTaintKey == nil || *oldMC.NodeDrainTaintKey != *mc.NodeDrainTaintKey) {
			liveMigration["nodeDrainTaintKey"] = *mc.NodeDrainTaintKey
		}
		if mc.AllowPostCopy != nil &&
			(oldMC == nil || oldMC.AllowPostCopy == nil || *oldMC.AllowPostCopy != *mc.AllowPostCopy) {
			liveMigration["allowPostCopy"] = *mc.AllowPostCopy
		}
		if mc.CompletionTimeoutPerGiB != nil &&
			(oldMC == nil || oldMC.CompletionTimeoutPerGiB == nil || *oldMC.CompletionTimeoutPerGiB != *mc.CompletionTimeoutPerGiB) {
			liveMigration["completionTimeoutPerGiB"] = *mc.CompletionTimeoutPerGiB
		}
		if mc.BandwidthPerMigration != nil &&
			(oldMC == nil || oldMC.BandwidthPerMigration == nil || !oldMC.BandwidthPerMigration.Equal(*mc.BandwidthPerMigration)) {
			liveMigration["bandwidthPerMigration"] = mc.BandwidthPerMigration.String()
		}

		if len(liveMigration) > 0 {
			virtualization["liveMigrationConfig"] = liveMigration
		}
	}

	if newConfig.ObsoleteCPUModels != nil &&
		!equality.Semantic.DeepEqual(oldConfig.ObsoleteCPUModels, newConfig.ObsoleteCPUModels) {
		models := make([]string, 0, len(newConfig.ObsoleteCPUModels))
		for model, obsolete := range newConfig.ObsoleteCPUModels {
			if obsolete {
				models = append(models, model)
			}
		}
		virtualization["obsoleteCPUModels"] = models
	}

	if len(virtualization) == 0 {
		return nil
	}

	return map[string]interface{}{
		"spec": map[string]interface{}{
			"virtualization": virtualization,
		},
	}
}
