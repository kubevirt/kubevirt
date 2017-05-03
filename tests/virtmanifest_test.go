package tests_test

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/util/json"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-manifest"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Virtmanifest", func() {
	Context("Manifest Service", func() {
		flag.Parse()

		var manifestClient *rest.RESTClient
		var vm *v1.VM

		BeforeEach(func() {
			tests.MustCleanup()

			var err error
			var masterUrl *url.URL
			masterUrl, err = url.Parse(flag.Lookup("master").Value.String())
			Expect(err).ToNot(HaveOccurred())
			hostParts := strings.Split(masterUrl.Host, ":")
			Expect(len(hostParts)).To(Equal(2))

			manifestClient, err = kubecli.GetRESTClientFromFlags(fmt.Sprintf("http://%s:8186", hostParts[0]), "")
			Expect(err).ToNot(HaveOccurred())

			vm = tests.NewRandomVM()
		})

		It("Should report server status", func() {
			ref := map[string]string{"status": "ok"}
			data := map[string]string{}

			res, err := manifestClient.Get().RequestURI("/api/v1/status").DoRaw()
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(res, &data)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal(ref))
		})

		It("Should return YAML if requested", func() {
			ref := "status: ok\n"
			res, err := manifestClient.Get().RequestURI("/api/v1/status").SetHeader("Accept", "application/yaml").DoRaw()
			Expect(err).ToNot(HaveOccurred())

			Expect(string(res)).To(Equal(ref))
		})

		It("Should map a VM manifest", func() {
			vmName := vm.ObjectMeta.Name
			mappedVm := v1.VM{}

			request, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			res, err := manifestClient.Post().SetHeader("Content-type", "application/json").Resource("manifest").Body(request).DoRaw()
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(res, &mappedVm)
			Expect(mappedVm.ObjectMeta.Name).To(Equal(vmName))
			Expect(mappedVm.Spec.Domain.Type).To(Equal("qemu"))
		})

		It("Should map PersistentVolumeClaims", func() {
			mappedVm := v1.VM{}
			vm.Spec.Domain.Devices.Disks = []v1.Disk{v1.Disk{
				Device: "disk",
				Type:   virt_manifest.Type_PersistentVolumeClaim,
				Source: v1.DiskSource{Name: "test"},
				Target: v1.DiskTarget{Bus: "scsi", Device: "vda"},
			}}

			request, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			res, err := manifestClient.Post().SetHeader("Content-type", "application/json").Resource("manifest").Body(request).DoRaw()
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(res, &mappedVm)
			Expect(len(mappedVm.Spec.Domain.Devices.Disks)).To(Equal(1))
			Expect(mappedVm.Spec.Domain.Devices.Disks[0].Type).To(Equal(virt_manifest.Type_PersistentVolumeClaim))
		})
	})
})
