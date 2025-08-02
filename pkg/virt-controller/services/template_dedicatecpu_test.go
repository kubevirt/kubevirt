package services

import (
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/testutils"

	"go.uber.org/mock/gomock"
)

var _ = Describe("Dedicated CPU", func() {

	var (
		svc        TemplateService
		virtClient *kubecli.MockKubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))

		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
		}

		config, _, _ := testutils.NewFakeClusterConfigUsingKVWithCPUArch(kv, runtime.GOARCH)
		svc = NewTemplateService("kubevirt/virt-launcher",
			240,
			"/var/run/kubevirt",
			"/var/run/kubevirt-ephemeral-disks",
			"/var/run/kubevirt/container-disks",
			v1.HotplugDiskDir,
			"pull-secret-1",
			cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
			virtClient,
			config,
			107,
			"kubevirt/vmexport",
			cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
			cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
			WithSidecarCreator(
				func(vmi *v1.VirtualMachineInstance, _ *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
					return hooks.UnmarshalHookSidecarList(vmi)
				}),
			WithNetBindingPluginMemoryCalculator(&stubNetBindingPluginMemoryCalculator{}),
		)
	})

	DescribeTable("should multiple all node affinity rules and append one of cpumanager label", func(affinity k8sv1.Affinity,
		expectedNodeSelectorTerms ...k8sv1.NodeSelectorTerm) {
		vmi := v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testvmi",
				Namespace: "default",
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores:                 2,
						DedicatedCPUPlacement: true,
					},
				},
				Affinity: &affinity,
			},
		}

		pod, err := svc.RenderLaunchManifest(&vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(ContainElements(
			expectedNodeSelectorTerms,
		))
	},
		Entry("without user provided affinity", k8sv1.Affinity{},
			k8sv1.NodeSelectorTerm{
				MatchExpressions: []k8sv1.NodeSelectorRequirement{
					{
						Key:      "node-labeller.kubevirt.io/obsolete-host-model",
						Operator: "DoesNotExist",
						Values:   nil,
					},
					{
						Key:      v1.DeprecatedCPUManager,
						Operator: k8sv1.NodeSelectorOpIn,
						Values:   []string{"true"},
					},
				},
			},
			k8sv1.NodeSelectorTerm{
				MatchExpressions: []k8sv1.NodeSelectorRequirement{
					{
						Key:      "node-labeller.kubevirt.io/obsolete-host-model",
						Operator: "DoesNotExist",
						Values:   nil,
					},
					{
						Key:      v1.CPUManager,
						Operator: k8sv1.NodeSelectorOpIn,
						Values:   []string{"true"},
					},
				},
			}),
		Entry("with user provided affinity", k8sv1.Affinity{
			NodeAffinity: &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "test",
									Operator: k8sv1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
		},
			k8sv1.NodeSelectorTerm{
				MatchExpressions: []k8sv1.NodeSelectorRequirement{
					{
						Key:      "test",
						Operator: k8sv1.NodeSelectorOpExists,
					},
					{
						Key:      "node-labeller.kubevirt.io/obsolete-host-model",
						Operator: "DoesNotExist",
						Values:   nil,
					},
					{
						Key:      v1.DeprecatedCPUManager,
						Operator: k8sv1.NodeSelectorOpIn,
						Values:   []string{"true"},
					},
				},
			},
			k8sv1.NodeSelectorTerm{
				MatchExpressions: []k8sv1.NodeSelectorRequirement{
					{
						Key:      "test",
						Operator: k8sv1.NodeSelectorOpExists,
					},
					{
						Key:      "node-labeller.kubevirt.io/obsolete-host-model",
						Operator: "DoesNotExist",
						Values:   nil,
					},
					{
						Key:      v1.CPUManager,
						Operator: k8sv1.NodeSelectorOpIn,
						Values:   []string{"true"},
					},
				},
			},
		),
	)
})
