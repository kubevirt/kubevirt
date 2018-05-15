package expose_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	k8sapiv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
)

var _ = Describe("Expose", func() {
	BeforeEach(func() {
		// initialize state before test
		kubecli.CurrentFakeService = &k8sapiv1.Service{}
		kubecli.InvalidFakeClient = false
		kubecli.InvalidFakeResource = false
		kubecli.InvalidFakeService = false
		kubecli.CurrentFakeLabel = map[string]string{"vmname": "testvm"}
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetFakeKubevirtClientFromClientConfig
	})

	Describe("Create an 'expose' command", func() {
		Context("With empty set of flags", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				Expect(cmd).NotTo(BeNil())
			})
		})
	})

	Describe("Run an 'expose' command", func() {
		Context("With a wrong verb", func() {
			It("should fail", func() {
				var flags pflag.FlagSet
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"kaboom", "testvm"})
				Expect(err).NotTo(BeNil())
			})
		})
		Context("With cluster-ip on a vm", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"vm", "testvm"})
				Expect(err).To(BeNil())
				Expect(kubecli.CurrentFakeService.Spec.Type).To(Equal(k8sapiv1.ServiceTypeClusterIP))
				Expect(kubecli.CurrentFakeService.Spec.Selector).To(Equal(kubecli.CurrentFakeLabel))
			})
		})
		Context("With unknown resource", func() {
			It("should fail", func() {
				var flags pflag.FlagSet
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				kubecli.InvalidFakeResource = true
				err := cmd.RunE(cmd, []string{"vm", "unknown"})
				Expect(err).NotTo(BeNil())
			})
		})
		Context("With cluster-ip on an ovm", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"ovm", "testvm"})
				Expect(err).To(BeNil())
				Expect(kubecli.CurrentFakeService.Spec.Type).To(Equal(k8sapiv1.ServiceTypeClusterIP))
				Expect(kubecli.CurrentFakeService.Spec.Selector).To(Equal(kubecli.CurrentFakeLabel))
			})
		})
		Context("With cluster-ip on an vm replica set", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"vmrs", "testvm"})
				Expect(err).To(BeNil())
				Expect(kubecli.CurrentFakeService.Spec.Type).To(Equal(k8sapiv1.ServiceTypeClusterIP))
				Expect(kubecli.CurrentFakeService.Spec.Selector).To(Equal(kubecli.CurrentFakeLabel))
			})
		})
		Context("With invalid type on a vm", func() {
			It("should fail", func() {
				var flags pflag.FlagSet
				if flags.Set("type", "kaboom") != nil {
					Skip("Didn't manage to set flag")
				}
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"vm", "testvm"})
				Expect(err).NotTo(BeNil())
			})
		})
		Context("With node-port on a vm", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				if flags.Set("type", "NodePort") != nil {
					Skip("Didn't manage to set flag")
				}
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"vm", "testvm"})
				Expect(err).To(BeNil())
				Expect(kubecli.CurrentFakeService.Spec.Type).To(Equal(k8sapiv1.ServiceTypeNodePort))
				Expect(kubecli.CurrentFakeService.Spec.Selector).To(Equal(kubecli.CurrentFakeLabel))
			})
		})
		Context("With node-port on an ovm", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				if flags.Set("type", "NodePort") != nil {
					Skip("Didn't manage to set flag")
				}
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"ovm", "testvm"})
				Expect(err).To(BeNil())
				Expect(kubecli.CurrentFakeService.Spec.Type).To(Equal(k8sapiv1.ServiceTypeNodePort))
				Expect(kubecli.CurrentFakeService.Spec.Selector).To(Equal(kubecli.CurrentFakeLabel))
			})
		})
		Context("With node-port on an vm replica set", func() {
			It("should succeed", func() {
				var flags pflag.FlagSet
				if flags.Set("type", "NodePort") != nil {
					Skip("Didn't manage to set flag")
				}
				clientConfig := kubecli.DefaultClientConfig(&flags)
				cmd := expose.NewExposeCommand(clientConfig)
				err := cmd.RunE(cmd, []string{"vmrs", "testvm"})
				Expect(err).To(BeNil())
				Expect(kubecli.CurrentFakeService.Spec.Type).To(Equal(k8sapiv1.ServiceTypeNodePort))
				Expect(kubecli.CurrentFakeService.Spec.Selector).To(Equal(kubecli.CurrentFakeLabel))
			})
		})
	})
})
