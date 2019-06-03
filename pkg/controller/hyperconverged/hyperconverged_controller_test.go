package hyperconverged

import (
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("HyperconvergedController", func() {
	Describe("CR creation functions", func() {
		instance := &hcov1alpha1.HyperConverged{}
		instance.Name = "hyperconverged-cluster"
		appLabel := map[string]string{
			"app": instance.Name,
		}

		Context("KubeVirt Config CR", func() {
			It("should have metadata", func() {
				cr := newKubeVirtConfigForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "kubevirt-config", appLabel)
			})
		})

		Context("KubeVirt CR", func() {
			It("should have metadata", func() {
				cr := newKubeVirtForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "kubevirt-"+instance.Name, appLabel)
			})
		})

		Context("CDI CR", func() {
			It("should have metadata", func() {
				cr := newCDIForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "cdi-"+instance.Name, appLabel)
			})
		})

		Context("Network Addons CR", func() {
			It("should have metadata and spec", func() {
				cr := newNetworkAddonsForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, networkaddonsnames.OPERATOR_CONFIG, appLabel)
				Expect(cr.Spec.Multus).To(Equal(&networkaddons.Multus{}))
				Expect(cr.Spec.LinuxBridge).To(Equal(&networkaddons.LinuxBridge{}))
				Expect(cr.Spec.KubeMacPool).To(Equal(&networkaddons.KubeMacPool{}))
			})
		})

		Context("KubeVirt Common Template Bundle CR", func() {
			It("should have metadata", func() {
				cr := newKubevirtCommonTemplateBundleForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "common-templates-"+instance.Name, appLabel)
				Expect(cr.ObjectMeta.Namespace).To(Equal("openshift"))
			})
		})

		Context("KubeVirt Node Labeller Bundle CR", func() {
			It("should have metadata", func() {
				cr := newKubevirtNodeLabellerBundleForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "node-labeller-"+instance.Name, appLabel)
			})
		})

		Context("KubeVirt Template Validator CR", func() {
			It("should have metadata", func() {
				cr := newKubevirtTemplateValidatorForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "template-validator-"+instance.Name, appLabel)
			})
		})

		Context("KubeVirt Web UI CR", func() {
			It("should have metadata and spec", func() {
				cr := newKWebUIForCR(instance)
				checkMetadataNameAndLabel(cr.ObjectMeta, "kubevirt-web-ui-"+instance.Name, appLabel)
				Expect(cr.ObjectMeta.Name).To(Equal("kubevirt-web-ui-" + instance.Name))
				Expect(cr.ObjectMeta.Labels).To(Equal(appLabel))
				Expect(cr.Spec.OpenshiftMasterDefaultSubdomain).To(Equal(instance.Spec.KWebUIMasterDefaultSubdomain))
				Expect(cr.Spec.PublicMasterHostname).To(Equal(instance.Spec.KWebUIPublicMasterHostname))
				Expect(cr.Spec.Version).To(Equal("automatic"))
			})
		})
	})
})

func checkMetadataNameAndLabel(metadata metav1.ObjectMeta, expectedName string, expectedLabel map[string]string) {
	Expect(metadata.Name).To(Equal(expectedName))
	Expect(metadata.Labels).To(Equal(expectedLabel))
}
