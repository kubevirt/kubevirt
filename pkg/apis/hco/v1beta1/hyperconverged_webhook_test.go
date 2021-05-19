package v1beta1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("Hyperconverged API: Webhook", func() {
	Context("Test Defaulter", func() {
		Context("test default PciHostDevices", func() {
			It("Should add the default PCI Host Devices to empty spec", func() {
				hco := HyperConverged{Spec: HyperConvergedSpec{}}
				hco.Default()
				Expect(hco.Spec.PermittedHostDevices.PciHostDevices).To(HaveLen(len(defaultPciHostDevices)))

				for _, phd := range defaultPciHostDevices {
					Expect(findPciHostDevice(hco.Spec.PermittedHostDevices.PciHostDevices, phd)).Should(BeTrue())
				}
			})

			It("Should add the default PCI Host Devices to empty PermittedHostDevices", func() {
				hco := HyperConverged{
					Spec: HyperConvergedSpec{
						PermittedHostDevices: &PermittedHostDevices{},
					},
				}
				hco.Default()
				Expect(hco.Spec.PermittedHostDevices.PciHostDevices).To(HaveLen(len(defaultPciHostDevices)))

				for _, phd := range defaultPciHostDevices {
					Expect(findPciHostDevice(hco.Spec.PermittedHostDevices.PciHostDevices, phd)).Should(BeTrue())
				}
			})

			It("Should add the default PCI Host Devices to nil PciHostDevices list", func() {
				hco := HyperConverged{
					Spec: HyperConvergedSpec{
						PermittedHostDevices: &PermittedHostDevices{
							PciHostDevices: nil,
						},
					},
				}
				hco.Default()

				Expect(hco.Spec.PermittedHostDevices.PciHostDevices).To(HaveLen(len(defaultPciHostDevices)))

				for _, phd := range defaultPciHostDevices {
					Expect(findPciHostDevice(hco.Spec.PermittedHostDevices.PciHostDevices, phd)).Should(BeTrue())
				}
			})

			It("Should add the default PCI Host Devices to empty PciHostDevices list", func() {
				hco := HyperConverged{
					Spec: HyperConvergedSpec{
						PermittedHostDevices: &PermittedHostDevices{
							PciHostDevices: make([]PciHostDevice, 0),
						},
					},
				}
				hco.Default()
				Expect(hco.Spec.PermittedHostDevices.PciHostDevices).To(HaveLen(len(defaultPciHostDevices)))

				for _, phd := range defaultPciHostDevices {
					Expect(findPciHostDevice(hco.Spec.PermittedHostDevices.PciHostDevices, phd)).Should(BeTrue())
				}
			})

			It("Should add a default PCI Host Device if missing from the PciHostDevices list", func() {
				hco := HyperConverged{
					Spec: HyperConvergedSpec{
						PermittedHostDevices: &PermittedHostDevices{
							PciHostDevices: []PciHostDevice{
								defaultPciHostDevices[0],
							},
						},
					},
				}
				hco.Default()
				Expect(hco.Spec.PermittedHostDevices.PciHostDevices).To(HaveLen(len(defaultPciHostDevices)))

				for _, phd := range defaultPciHostDevices {
					Expect(findPciHostDevice(hco.Spec.PermittedHostDevices.PciHostDevices, phd)).Should(BeTrue())
				}
			})

			It("Should not add a default PCI Host Device if it already in the PciHostDevices list", func() {
				hco := HyperConverged{
					Spec: HyperConvergedSpec{
						PermittedHostDevices: &PermittedHostDevices{
							PciHostDevices: make([]PciHostDevice, len(defaultPciHostDevices)),
						},
					},
				}

				copy(hco.Spec.PermittedHostDevices.PciHostDevices, defaultPciHostDevices)
				hco.Spec.PermittedHostDevices.PciHostDevices[0].Disabled = true

				hco.Default()
				Expect(hco.Spec.PermittedHostDevices.PciHostDevices).To(HaveLen(len(defaultPciHostDevices)))

				for _, phd := range defaultPciHostDevices {
					Expect(findPciHostDevice(hco.Spec.PermittedHostDevices.PciHostDevices, phd)).Should(BeTrue())
				}

				By("check that the Default() function didn't change the modification we made", func() {
					Expect(hco.Spec.PermittedHostDevices.PciHostDevices[0].Disabled).Should(BeTrue())
				})
			})
		})
	})

	Context("Test GetWebhookCertDir", func() {
		It("should return default value, if the env var is not set", func() {
			Expect(GetWebhookCertDir()).Should(Equal(DefaultWebhookCertDir))
		})

		It("should return the value of the env var, if set", func() {
			env := os.Getenv(webHookCertDirEnv)
			defer os.Setenv(webHookCertDirEnv, env)

			const somethingElse = "/something/else"
			os.Setenv(webHookCertDirEnv, somethingElse)
			Expect(GetWebhookCertDir()).Should(Equal(somethingElse))
		})
	})
})
