package launchsecurity

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/testsuite"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute]AMD Secure Encrypted Virtualization (SEV)", decorators.SEV, decorators.SigCompute, func() {
	const (
		diskSecret = "qwerty123"
	)

	newSEVFedora := func(withES bool, opts ...libvmi.Option) *v1.VirtualMachineInstance {
		const secureBoot = false
		sevOptions := []libvmi.Option{
			libvmi.WithUefi(secureBoot),
			libvmi.WithSEV(withES),
		}
		opts = append(sevOptions, opts...)
		return libvmi.NewFedora(opts...)
	}

	// As per section 6.5 LAUNCH_MEASURE of the AMD SEV specification the launch
	// measurement is calculated as:
	//   HMAC(0x04 || API_MAJOR || API_MINOR || BUILD ||
	//        GCTX.POLICY || GCTX.LD || MNONCE; GCTX.TIK)
	// The implementation is based on
	//   https://blog.hansenpartnership.com/wp-uploads/2020/12/sevsecret.txt
	verifyMeasurement := func(sevMeasurementInfo *v1.SEVMeasurementInfo, tikBase64 string) []byte {
		By("Verifying launch measurement")
		tik, err := base64.StdEncoding.DecodeString(tikBase64)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		h := hmac.New(sha256.New, tik)
		err = binary.Write(h, binary.LittleEndian, uint8(0x04))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		err = binary.Write(h, binary.LittleEndian, uint8(sevMeasurementInfo.APIMajor))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		err = binary.Write(h, binary.LittleEndian, uint8(sevMeasurementInfo.APIMinor))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		err = binary.Write(h, binary.LittleEndian, uint8(sevMeasurementInfo.BuildID))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		err = binary.Write(h, binary.LittleEndian, uint32(sevMeasurementInfo.Policy))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		loaderSHA, err := hex.DecodeString(sevMeasurementInfo.LoaderSHA)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		h.Write(loaderSHA)
		m, err := base64.StdEncoding.DecodeString(sevMeasurementInfo.Measurement)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		nonce := m[32:48]
		h.Write(nonce)
		measure := m[0:32]
		ExpectWithOffset(1, hex.EncodeToString(h.Sum(nil))).To(Equal(hex.EncodeToString(measure)))
		return measure
	}

	// The implementation is based on
	//   https://blog.hansenpartnership.com/wp-uploads/2020/12/sevsecret.txt
	encryptSecret := func(diskSecret string, measure []byte, tikBase64, tekBase64 string) *v1.SEVSecretOptions {
		By("Encrypting launch secret")
		tik, err := base64.StdEncoding.DecodeString(tikBase64)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		tek, err := base64.StdEncoding.DecodeString(tekBase64)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		// AMD SEV specification, section 4.6 Endianness: all integral values
		// passed between the firmware and the CPU driver are little-endian
		// formatted. Applies to UUIDs as well.
		writeUUID := func(w io.Writer, uuid uuid.UUID) {
			var err error
			err = binary.Write(w, binary.LittleEndian, binary.BigEndian.Uint32(uuid[0:4]))
			ExpectWithOffset(2, err).ToNot(HaveOccurred())
			err = binary.Write(w, binary.LittleEndian, binary.BigEndian.Uint16(uuid[4:6]))
			ExpectWithOffset(2, err).ToNot(HaveOccurred())
			err = binary.Write(w, binary.LittleEndian, binary.BigEndian.Uint16(uuid[6:8]))
			ExpectWithOffset(2, err).ToNot(HaveOccurred())
			_, err = w.Write(uuid[8:])
			ExpectWithOffset(2, err).ToNot(HaveOccurred())
		}

		const (
			uuidLen = 16
			sizeLen = 4
		)

		// total length of table: header plus one entry with trailing \0
		l := (uuidLen + sizeLen) + (uuidLen + sizeLen) + len(diskSecret) + 1
		// SEV-ES requires rounding to 16
		l = (l + 15) & ^15

		secret := bytes.NewBuffer(make([]byte, 0, l))
		// 0:16
		writeUUID(secret, uuid.MustParse("{1e74f542-71dd-4d66-963e-ef4287ff173b}"))
		// 16:20
		err = binary.Write(secret, binary.LittleEndian, uint32(l))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		// 20:36
		writeUUID(secret, uuid.MustParse("{736869e5-84f0-4973-92ec-06879ce3da0b}"))
		// 36:40
		err = binary.Write(secret, binary.LittleEndian, uint32(uuidLen+sizeLen+len(diskSecret)+1))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		// 40:40+len(diskSecret)+1
		secret.Write([]byte(diskSecret))
		// write zeroes
		secret.Write(make([]byte, l-secret.Len()))
		ExpectWithOffset(1, secret.Len()).To(Equal(l))

		// The data protection scheme utilizes AES-128 CTR mode:
		//   C = AES-128-CTR(M; K, IV)
		iv := make([]byte, 16)
		_, err = rand.Read(iv)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		aes, err := aes.NewCipher(tek)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ctr := cipher.NewCTR(aes, iv)
		encryptedSecret := make([]byte, secret.Len())
		cipher.Stream.XORKeyStream(ctr, encryptedSecret, secret.Bytes())

		// AMD SEV specification, section 6.6 LAUNCH_SECRET:
		//   Header: FLAGS + IV + HMAC
		header := bytes.NewBuffer(make([]byte, 0, 52))
		// 0:4 FLAGS.COMPRESSED
		err = binary.Write(header, binary.LittleEndian, uint32(0))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		// 4:20 IV
		header.Write(iv)
		// HMAC(0x01 || FLAGS || IV || GUEST_LENGTH ||
		//      TRANS_LENGTH || DATA || MEASURE; GCTX.TIK)
		h := hmac.New(sha256.New, tik)
		err = binary.Write(h, binary.LittleEndian, uint8(0x01))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		h.Write(header.Bytes()[0:20]) // FLAGS || IV
		err = binary.Write(h, binary.LittleEndian, uint32(l))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		err = binary.Write(h, binary.LittleEndian, uint32(l))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		h.Write(encryptedSecret)
		h.Write(measure)
		// 20:52 HMAC
		header.Write(h.Sum(nil))

		return &v1.SEVSecretOptions{
			Secret: base64.StdEncoding.EncodeToString(encryptedSecret),
			Header: base64.StdEncoding.EncodeToString(header.Bytes()),
		}
	}

	parseVirshInfo := func(info string, expectedKeys []string) map[string]string {
		entries := make(map[string]string)
		scanner := bufio.NewScanner(strings.NewReader(info))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			data := strings.Split(line, ":")
			ExpectWithOffset(1, data).To(HaveLen(2))
			entries[strings.TrimSpace(data[0])] = strings.TrimSpace(data[1])
		}
		for _, key := range expectedKeys {
			ExpectWithOffset(1, entries).To(HaveKeyWithValue(key, Not(BeEmpty())))
		}
		return entries
	}

	toUint := func(s string) uint {
		val, err := strconv.Atoi(s)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, val).To(BeNumerically(">=", 0))
		return uint(val)
	}

	prepareSession := func(virtClient kubecli.KubevirtClient, nodeName string, pdh string) (*v1.SEVSessionOptions, string, string) {
		helperPod := tests.RenderPrivilegedPod("sev-helper", []string{"sleep"}, []string{"infinity"})
		helperPod.Spec.NodeName = nodeName

		var err error
		helperPod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(helperPod)).Create(context.Background(), helperPod, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			err := virtClient.CoreV1().Pods(helperPod.Namespace).Delete(context.Background(), helperPod.Name, k8smetav1.DeleteOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}()
		EventuallyWithOffset(1, ThisPod(helperPod), 30).Should(BeInPhase(k8sv1.PodRunning))

		execOnHelperPod := func(command string) (string, error) {
			stdout, err := exec.ExecuteCommandOnPod(
				virtClient,
				helperPod,
				helperPod.Spec.Containers[0].Name,
				[]string{tests.BinBash, "-c", command})
			return strings.TrimSpace(stdout), err
		}

		_, err = execOnHelperPod(fmt.Sprintf("echo %s | base64 --decode > pdh.bin", pdh))
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		_, err = execOnHelperPod("sevctl session pdh.bin 1")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		godh, err := execOnHelperPod("cat vm_godh.b64")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		session, err := execOnHelperPod("cat vm_session.b64")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		tikBase64, err := execOnHelperPod("base64 vm_tik.bin")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		tekBase64, err := execOnHelperPod("base64 vm_tek.bin")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		return &v1.SEVSessionOptions{
			DHCert:  godh,
			Session: session,
		}, tikBase64, tekBase64
	}

	BeforeEach(func() {
		checks.SkipTestIfNoFeatureGate(virtconfig.WorkloadEncryptionSEV)
	})

	Context("[Serial]device management", Serial, func() {
		const (
			sevResourceName = "devices.kubevirt.io/sev"
			sevDevicePath   = "/proc/1/root/dev/sev"
		)

		var (
			virtClient      kubecli.KubevirtClient
			nodeName        string
			isDevicePresent bool
			err             error
		)

		BeforeEach(func() {
			virtClient = kubevirt.Client()

			nodeName = tests.NodeNameWithHandler()
			Expect(nodeName).ToNot(BeEmpty())

			checkCmd := []string{"ls", sevDevicePath}
			_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, checkCmd)
			isDevicePresent = (err == nil)

			if !isDevicePresent {
				By(fmt.Sprintf("Creating a fake SEV device on %s", nodeName))
				mknodCmd := []string{"mknod", sevDevicePath, "c", "10", "124"}
				_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, mknodCmd)
				Expect(err).ToNot(HaveOccurred())
			}

			Eventually(func() bool {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				val, ok := node.Status.Allocatable[sevResourceName]
				return ok && !val.IsZero()
			}, 180*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("Allocatable SEV should not be zero on %s", nodeName))
		})

		AfterEach(func() {
			if !isDevicePresent {
				By(fmt.Sprintf("Removing the fake SEV device from %s", nodeName))
				rmCmd := []string{"rm", "-f", sevDevicePath}
				_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, rmCmd)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should reset SEV allocatable devices when the feature gate is disabled", func() {
			By(fmt.Sprintf("Disabling %s feature gate", virtconfig.WorkloadEncryptionSEV))
			tests.DisableFeatureGate(virtconfig.WorkloadEncryptionSEV)
			Eventually(func() bool {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				val, ok := node.Status.Allocatable[sevResourceName]
				return !ok || val.IsZero()
			}, 180*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("Allocatable SEV should be zero on %s", nodeName))
		})
	})

	Context("lifecycle", func() {
		BeforeEach(func() {
			checks.SkipTestIfNotSEVCapable()
		})

		DescribeTable("should start a SEV or SEV-ES VM",
			func(withES bool, sevstr string) {
				if withES {
					checks.SkipTestIfNotSEVESCapable()
				}
				vmi := newSEVFedora(withES)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Verifying that SEV is enabled in the guest")
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "dmesg | grep --color=never SEV\n"},
					&expect.BExp{R: "AMD Memory Encryption Features active: " + sevstr},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 30)
				Expect(err).ToNot(HaveOccurred())
			},
			// SEV-ES disabled, SEV enabled
			Entry("It should launch with base SEV features enabled", false, "SEV"),
			// SEV-ES enabled
			Entry("It should launch with SEV-ES features enabled", true, "SEV SEV-ES"),
		)

		It("should run guest attestation", func() {
			var (
				expectedSEVPlatformInfo    v1.SEVPlatformInfo
				expectedSEVMeasurementInfo v1.SEVMeasurementInfo
			)

			vmi := newSEVFedora(false, libvmi.WithSEVAttestation())
			vmi = tests.RunVMI(vmi, 30)
			Eventually(ThisVMI(vmi), 60).Should(BeInPhase(v1.Scheduled))

			virtClient, err := kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())

			By("Querying virsh nodesevinfo")
			nodeSevInfo := tests.RunCommandOnVmiPod(vmi, []string{"virsh", "nodesevinfo"})
			Expect(nodeSevInfo).ToNot(BeEmpty())
			entries := parseVirshInfo(nodeSevInfo, []string{"pdh", "cert-chain"})
			expectedSEVPlatformInfo.PDH = entries["pdh"]
			expectedSEVPlatformInfo.CertChain = entries["cert-chain"]

			By("Fetching platform certificates")
			sevPlatformInfo, err := virtClient.VirtualMachineInstance(vmi.Namespace).SEVFetchCertChain(vmi.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(sevPlatformInfo).To(Equal(expectedSEVPlatformInfo))

			By("Setting up session parameters")
			vmi, err = ThisVMI(vmi)()
			Expect(err).ToNot(HaveOccurred())
			sevSessionOptions, tikBase64, tekBase64 := prepareSession(virtClient, vmi.Status.NodeName, sevPlatformInfo.PDH)
			err = virtClient.VirtualMachineInstance(vmi.Namespace).SEVSetupSession(vmi.Name, sevSessionOptions)
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVMI(vmi), 60).Should(And(BeRunning(), HaveConditionTrue(v1.VirtualMachineInstancePaused)))

			By("Querying virsh domlaunchsecinfo 1")
			domLaunchSecInfo := tests.RunCommandOnVmiPod(vmi, []string{"virsh", "domlaunchsecinfo", "1"})
			Expect(domLaunchSecInfo).ToNot(BeEmpty())
			entries = parseVirshInfo(domLaunchSecInfo, []string{
				"sev-measurement", "sev-api-major", "sev-api-minor", "sev-build-id", "sev-policy",
			})
			expectedSEVMeasurementInfo.APIMajor = toUint(entries["sev-api-major"])
			expectedSEVMeasurementInfo.APIMinor = toUint(entries["sev-api-minor"])
			expectedSEVMeasurementInfo.BuildID = toUint(entries["sev-build-id"])
			expectedSEVMeasurementInfo.Policy = toUint(entries["sev-policy"])
			expectedSEVMeasurementInfo.Measurement = entries["sev-measurement"]

			By("Querying the domain loader path")
			domainSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domainSpec.OS.BootLoader).ToNot(BeNil())
			Expect(domainSpec.OS.BootLoader.Path).ToNot(BeEmpty())

			By(fmt.Sprintf("Computing sha256sum %s", domainSpec.OS.BootLoader.Path))
			sha256sum := tests.RunCommandOnVmiPod(vmi, []string{"sha256sum", domainSpec.OS.BootLoader.Path})
			Expect(sha256sum).ToNot(BeEmpty())
			expectedSEVMeasurementInfo.LoaderSHA = strings.TrimSpace(strings.Split(sha256sum, " ")[0])
			Expect(expectedSEVMeasurementInfo.LoaderSHA).To(HaveLen(64))

			By("Querying launch measurement")
			sevMeasurementInfo, err := virtClient.VirtualMachineInstance(vmi.Namespace).SEVQueryLaunchMeasurement(vmi.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(sevMeasurementInfo).To(Equal(expectedSEVMeasurementInfo))
			measure := verifyMeasurement(&sevMeasurementInfo, tikBase64)
			sevSecretOptions := encryptSecret(diskSecret, measure, tikBase64, tekBase64)

			By("Injecting launch secret")
			err = virtClient.VirtualMachineInstance(vmi.Namespace).SEVInjectLaunchSecret(vmi.Name, sevSecretOptions)
			Expect(err).ToNot(HaveOccurred())

			By("Unpausing the VirtualMachineInstance")
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVMI(vmi), 30*time.Second, time.Second).Should(HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

			By("Waiting for the VirtualMachineInstance to become ready")
			libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		})
	})
})
