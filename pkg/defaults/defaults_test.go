package defaults_test

import (
	"context"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/libds"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Defaults", func() {
	Context("Architecture", func() {
		Context("VirtualMachine", func() {
			var (
				clusterConfig *virtconfig.ClusterConfig
				virtClient    *kubecli.MockKubevirtClient
			)

			const (
				userProvidedArch     = "userArch"
				templateProvidedArch = "arm64"
				configProvidedArch   = "configArch"
			)

			BeforeEach(func() {
				ctrl := gomock.NewController(GinkgoT())
				virtClient = kubecli.NewMockKubevirtClient(ctrl)
				virtClient.EXPECT().CdiClient().Return(cdifake.NewSimpleClientset()).AnyTimes()

				var kvStore cache.Store
				clusterConfig, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{},
					},
					Status: v1.KubeVirtStatus{
						DefaultArchitecture: configProvidedArch,
					},
				})
			})

			createDataSource := func(options ...libds.Option) *cdiv1beta1.DataSource {
				GinkgoHelper()
				ds := libds.New(options...)
				ds, err := virtClient.CdiClient().CdiV1beta1().DataSources(ds.Namespace).Create(context.Background(), ds, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return ds
			}

			It("should ignore unknown arch provided by DataSource", func() {
				ds := createDataSource(libds.WithTemplateArchLabel("foobar"))
				vm := libvmi.NewVirtualMachine(
					libvmi.New(),
					libvmi.WithDataVolumeTemplate(
						libdv.NewDataVolume(
							libdv.WithDataVolumeSourceRef("DataSource", ds.Namespace, ds.Name),
						),
					),
				)
				defaults.SetVirtualMachineDefaults(vm, clusterConfig, virtClient)
				Expect(vm.Spec.Template.Spec.Architecture).To(Equal(configProvidedArch))
			})

			DescribeTable("should default to", func(createVM func() *v1.VirtualMachine, expectedArch string) {
				vm := createVM()
				defaults.SetVirtualMachineDefaults(vm, clusterConfig, virtClient)
				Expect(vm.Spec.Template.Spec.Architecture).To(Equal(expectedArch))
			},
				Entry("user provided value when provided", func() *v1.VirtualMachine {
					ds := createDataSource(
						libds.WithNamespace("ds-namespace"),
						libds.WithTemplateArchLabel(templateProvidedArch),
					)
					return libvmi.NewVirtualMachine(
						libvmi.New(
							libvmi.WithArchitecture(userProvidedArch),
						),
						libvmi.WithDataVolumeTemplate(
							libdv.NewDataVolume(
								libdv.WithDataVolumeSourceRef("DataSource", ds.Namespace, ds.Name),
							),
						),
					)
				}, userProvidedArch),
				Entry("referenced DataSource provided architecture label when not provided by user", func() *v1.VirtualMachine {
					ds := createDataSource(
						libds.WithNamespace("ds-namespace"),
						libds.WithTemplateArchLabel(templateProvidedArch),
					)
					return libvmi.NewVirtualMachine(
						libvmi.New(),
						libvmi.WithDataVolumeTemplate(
							libdv.NewDataVolume(
								libdv.WithDataVolumeSourceRef("DataSource", ds.Namespace, ds.Name),
							),
						),
					)
				}, templateProvidedArch),
				Entry("referenced DataSource (without namespace) provided architecture label when not provided by user", func() *v1.VirtualMachine {
					const vmNamespace = "vm-namespace"
					ds := createDataSource(
						libds.WithNamespace(vmNamespace),
						libds.WithTemplateArchLabel(templateProvidedArch),
					)
					return libvmi.NewVirtualMachine(
						libvmi.New(
							libvmi.WithNamespace(vmNamespace),
						),
						libvmi.WithDataVolumeTemplate(
							libdv.NewDataVolume(
								libdv.WithDataVolumeSourceRef("DataSource", "", ds.Name),
							),
						),
					)
				}, templateProvidedArch),
				Entry("referenced nested DataSource provided architecture label when not provided by user", func() *v1.VirtualMachine {
					nestedDS := createDataSource(
						libds.WithNamespace("ds-namespace"),
						libds.WithTemplateArchLabel(templateProvidedArch),
					)
					ds := createDataSource(
						libds.WithNamespace("ds-namespace"),
						libds.WithDataSourceSource(
							libds.WithDataSource(
								nestedDS.Name,
								nestedDS.Namespace,
							),
						),
					)
					return libvmi.NewVirtualMachine(
						libvmi.New(),
						libvmi.WithDataVolumeTemplate(
							libdv.NewDataVolume(
								libdv.WithDataVolumeSourceRef("DataSource", ds.Namespace, ds.Name),
							),
						),
					)
				}, templateProvidedArch),
				Entry("config arch when not provided by user", func() *v1.VirtualMachine {
					return libvmi.NewVirtualMachine(libvmi.New())
				}, configProvidedArch),
				Entry("runtime arch when not provided by user or config", func() *v1.VirtualMachine {
					clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
					return libvmi.NewVirtualMachine(libvmi.New())
				}, runtime.GOARCH),
			)
		})
	})
})
