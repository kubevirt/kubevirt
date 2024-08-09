package sev_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/sev"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("SEV Attestation", func() {
	const vmiName = "testvm"
	var vmi *v1.VirtualMachineInstance
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller
	var workDir string

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmi = api.NewMinimalVMI(vmiName)
		workDir = GinkgoT().TempDir()
	})

	Describe("The base 'sev' command", func() {
		Context("Creation", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV)
				Expect(cmd).ToNot(BeNil())
			})
		})
		Context("Running with no subcommand", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
	})

	Describe("The 'fetch-cert-chain' command", func() {
		Context("Creation", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_FETCH_CERT_CHAIN)
				Expect(cmd).ToNot(BeNil())
			})
		})
		Context("With no argument", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_FETCH_CERT_CHAIN)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With VMI name", func() {
			It("should succeed", func() {
				sevPlatformInfo := v1.SEVPlatformInfo{
					PDH:       "somelongpdh",
					CertChain: "somelongcertchain",
				}
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVFetchCertChain(vmi.Name).Return(sevPlatformInfo, nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_FETCH_CERT_CHAIN, vmiName)
				Expect(cmd.Execute()).To(BeNil())
			})
			It("should print cert chain to file", func() {
				sevPlatformInfo := v1.SEVPlatformInfo{
					PDH:       "somelongpdh",
					CertChain: "somelongcertchain",
				}
				var resp v1.SEVPlatformInfo
				fpath := filepath.Join(workDir, "cert_chain.json")
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVFetchCertChain(vmi.Name).Return(sevPlatformInfo, nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_FETCH_CERT_CHAIN, vmiName, "--output", fpath)
				Expect(cmd.Execute()).To(BeNil())

				_, err := os.Stat(fpath)
				Expect(err).ToNot(HaveOccurred(), "File is not created")

				data, err := os.ReadFile(fpath)
				Expect(err).ToNot(HaveOccurred())
				err = json.Unmarshal(data, &resp)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(sevPlatformInfo), "Expected JSON is returned")
			})
			It("should fail if output filepath already exists", func() {
				sevPlatformInfo := v1.SEVPlatformInfo{
					PDH:       "somelongpdh",
					CertChain: "somelongcertchain",
				}
				existingFile := filepath.Join(workDir, "exisiting-file")
				Expect(ioutil.WriteFile(existingFile, []byte("Exists"), 0750)).To(Succeed())

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVFetchCertChain(vmi.Name).Return(sevPlatformInfo, nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_FETCH_CERT_CHAIN, vmiName,
					"--output", existingFile)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
			It("should not create output file on API failure", func() {
				fpath := filepath.Join(workDir, "cert_chain.json")
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVFetchCertChain(vmi.Name).Return(v1.SEVPlatformInfo{}, errors.New("API Error")).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_FETCH_CERT_CHAIN, vmiName, "--output", fpath)
				Expect(cmd.Execute()).ToNot(BeNil())

				_, err := os.Stat(fpath)
				Expect(err).To(HaveOccurred(), "File is created")
			})
		})
	})

	Describe("The 'setup-session' command", func() {
		Context("Creation", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_SETUP_SESSION)
				Expect(cmd).ToNot(BeNil())
			})
		})
		Context("With no argument", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_SETUP_SESSION)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With VMI name and no flags", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_SETUP_SESSION, vmiName)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With only session flag", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_SETUP_SESSION, vmiName,
					"--session", "somelongsessionblob")
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With only guest owner certificate flag", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_SETUP_SESSION, vmiName,
					"--dhcert", "somelongcertificate")
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With session and godh flags", func() {
			It("should succeed", func() {
				sevSessionOptions := &v1.SEVSessionOptions{
					Session: "somelongsessionblob",
					DHCert:  "somelongcertificate",
				}
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVSetupSession(vmi.Name, sevSessionOptions).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_SETUP_SESSION, vmiName,
					"--session", "somelongsessionblob",
					"--dhcert", "somelongcertificate")
				Expect(cmd.Execute()).To(BeNil())
			})
		})
	})

	Describe("The 'query-measurement' command", func() {
		Context("Creation", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_QUERY_MEASUREMENT)
				Expect(cmd).ToNot(BeNil())
			})
		})
		Context("With no argument", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_QUERY_MEASUREMENT)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With VMI name", func() {
			It("should succeed", func() {
				sevMeasurementInfo := v1.SEVMeasurementInfo{
					Measurement: "Somebase64LaunchMeasurement",
					APIMajor:    1,
					APIMinor:    1,
					BuildID:     12345,
					Policy:      12345,
					LoaderSHA:   "loaderbinhash",
				}
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVQueryLaunchMeasurement(vmi.Name).Return(sevMeasurementInfo, nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_QUERY_MEASUREMENT, vmiName)
				Expect(cmd.Execute()).To(BeNil())
			})
			It("should print measurement to file", func() {
				sevMeasurementInfo := v1.SEVMeasurementInfo{
					Measurement: "Somebase64LaunchMeasurement",
					APIMajor:    1,
					APIMinor:    1,
					BuildID:     12345,
					Policy:      12345,
					LoaderSHA:   "loaderbinhash",
				}
				var resp v1.SEVMeasurementInfo
				fpath := filepath.Join(workDir, "measurement.json")
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVQueryLaunchMeasurement(vmi.Name).Return(sevMeasurementInfo, nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_QUERY_MEASUREMENT, vmiName,
					"--output", fpath)
				Expect(cmd.Execute()).To(BeNil())

				_, err := os.Stat(fpath)
				Expect(err).ToNot(HaveOccurred(), "File is not created")

				data, err := os.ReadFile(fpath)
				Expect(err).ToNot(HaveOccurred())
				err = json.Unmarshal(data, &resp)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(sevMeasurementInfo), "Expected JSON is returned")
			})
			It("should fail if output filepath already exists", func() {
				sevMeasurementInfo := v1.SEVMeasurementInfo{
					Measurement: "Somebase64LaunchMeasurement",
					APIMajor:    1,
					APIMinor:    1,
					BuildID:     12345,
					Policy:      12345,
					LoaderSHA:   "loaderbinhash",
				}
				existingFile := filepath.Join(workDir, "exisiting-file")
				Expect(ioutil.WriteFile(existingFile, []byte("Exists"), 0750)).To(Succeed())

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVQueryLaunchMeasurement(vmi.Name).Return(sevMeasurementInfo, nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_QUERY_MEASUREMENT, vmiName,
					"--output", existingFile)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
			It("should not create output file on API failure", func() {
				fpath := filepath.Join(workDir, "measurement.json")
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVQueryLaunchMeasurement(vmi.Name).Return(v1.SEVMeasurementInfo{}, errors.New("API Error")).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_QUERY_MEASUREMENT, vmiName, "--output", fpath)
				Expect(cmd.Execute()).ToNot(BeNil())

				_, err := os.Stat(fpath)
				Expect(err).To(HaveOccurred(), "File is created")
			})
		})
	})

	Describe("The 'inject-secret' command", func() {
		Context("Creation", func() {
			It("should succeed", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_INJECT_SECRET)
				Expect(cmd).ToNot(BeNil())
			})
		})
		Context("With no argument", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_INJECT_SECRET)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With VMI name and no flags", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_INJECT_SECRET, vmiName)
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With only header flag", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_INJECT_SECRET, vmiName,
					"--header", "somelongheaderblob")
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With only secret flag", func() {
			It("should fail", func() {
				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_INJECT_SECRET, vmiName,
					"--secret", "somelongsecretblob")
				Expect(cmd.Execute()).ToNot(BeNil())
			})
		})
		Context("With header and secret flags", func() {
			It("should succeed", func() {
				sevSecretOptions := &v1.SEVSecretOptions{
					Header: "somelongheaderblob",
					Secret: "somelongsecretblob",
				}
				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
				vmiInterface.EXPECT().SEVInjectLaunchSecret(vmi.Name, sevSecretOptions).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand(sev.COMMAND_SEV, sev.COMMAND_INJECT_SECRET, vmiName,
					"--header", "somelongheaderblob",
					"--secret", "somelongsecretblob")
				Expect(cmd.Execute()).To(BeNil())
			})
		})
	})
})
