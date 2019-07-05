package operations

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"gopkg.in/yaml.v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/names"
)

func GetConfig() *opv1alpha1.NetworkAddonsConfig {
	By("Getting the current config")

	config := &opv1alpha1.NetworkAddonsConfig{}

	err := framework.Global.Client.Get(context.TODO(), types.NamespacedName{Name: names.OPERATOR_CONFIG}, config)
	if apierrors.IsNotFound(err) {
		return nil
	}
	Expect(err).NotTo(HaveOccurred(), "Failed to fetch Config")

	return config
}

func CreateConfig(configSpec opv1alpha1.NetworkAddonsConfigSpec) {
	By(fmt.Sprintf("Applying NetworkAddonsConfig:\n%s", configSpecToYaml(configSpec)))

	config := &opv1alpha1.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: names.OPERATOR_CONFIG,
		},
		Spec: configSpec,
	}

	err := framework.Global.Client.Create(context.TODO(), config, &framework.CleanupOptions{})
	Expect(err).NotTo(HaveOccurred(), "Failed to create the Config")
}

func UpdateConfig(configSpec opv1alpha1.NetworkAddonsConfigSpec) {
	By(fmt.Sprintf("Updating NetworkAddonsConfig:\n%s", configSpecToYaml(configSpec)))

	// Get current Config
	config := GetConfig()

	// Update the Config with the desired Spec
	config.Spec = configSpec
	err := framework.Global.Client.Update(context.TODO(), config)
	Expect(err).NotTo(HaveOccurred(), "Failed to update the Config")
}

func DeleteConfig() {
	By("Removing NetworkAddonsConfig")

	config := &opv1alpha1.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: names.OPERATOR_CONFIG,
		},
	}

	err := framework.Global.Client.Delete(context.TODO(), config)
	Expect(err).NotTo(HaveOccurred(), "Failed to remove the Config")
}

// Convert NetworkAddonsConfig specification to a yaml format we would expect in a manifest
func configSpecToYaml(configSpec opv1alpha1.NetworkAddonsConfigSpec) string {
	manifest, err := yaml.Marshal(configSpec)
	if err != nil {
		panic(err)
	}

	manifestLines := strings.Split(string(manifest), "\n")

	// We don't want to show non-set (default) values, usually null. Try our best to filter those out.
	manifestLinesWithoutEmptyValues := []string{}
	for _, line := range manifestLines {
		// If root attribute (e.g. ImagePullPolicy) is set to default, drop it. If it
		// is a nested attribute (e.g. KubeMacPool's RangeEnd), keep it.
		rootAttributeSetToDefault := !strings.Contains(line, "  ") && (strings.Contains(line, ": \"\"") || strings.Contains(line, ": null"))
		if line != "" && !rootAttributeSetToDefault {
			manifestLinesWithoutEmptyValues = append(manifestLinesWithoutEmptyValues, line)
		}
	}

	// If any values has been set, return Spec in a nice YAML format
	if len(manifestLinesWithoutEmptyValues) > 0 {
		indentedManifest := strings.TrimSpace(strings.Join(manifestLinesWithoutEmptyValues, "\n"))
		return indentedManifest
	}

	// Note that it is empty otherwise
	return "Empty Spec"
}
