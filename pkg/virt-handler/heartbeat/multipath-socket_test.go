package heartbeat_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/heartbeat"
	"kubevirt.io/kubevirt/pkg/virt-handler/heartbeat/filewatcher"
)

var _ = Describe("MonitorMultipathSocket", func() {
	var (
		monitorMultipath *heartbeat.MonitorMultipathSocket

		isMounted           bool
		runDirPath          string
		multipathSocketPath string
	)

	createFakeSocket := func() {
		_, err := os.Create(multipathSocketPath)
		Expect(err).ToNot(HaveOccurred())
	}

	removeFakeSocket := func() {
		Expect(os.Remove(multipathSocketPath)).To(Succeed())
	}

	BeforeEach(func() {
		isMounted = false
		runDirPath = GinkgoT().TempDir()
		multipathSocketPath = filepath.Join(runDirPath, "multipathd.socket")

		mounter := heartbeat.NewMockmounter(gomock.NewController(GinkgoT()))
		mounter.EXPECT().IsMounted(gomock.Any()).
			DoAndReturn(func(_ *safepath.Path) (bool, error) {
				return isMounted, nil
			}).AnyTimes()
		mounter.EXPECT().Mount(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_, _ *safepath.Path, _ bool) {
				isMounted = true
			}).Return(exec.Command("/bin/true")).AnyTimes()
		mounter.EXPECT().Umount(gomock.Any()).
			Do(func(_ *safepath.Path) {
				isMounted = false
			}).Return(exec.Command("/bin/true")).AnyTimes()

		monitorMultipath = &heartbeat.MonitorMultipathSocket{
			Watcher:             filewatcher.New(multipathSocketPath, 1*time.Second),
			MultipathSocketPath: multipathSocketPath,
			HostDir:             filepath.Join(GinkgoT().TempDir(), "pr"),
			Mounter:             mounter,
		}
	})

	AfterEach(func() {
		Expect(monitorMultipath.Watcher.Errors).To(BeEmpty())
		monitorMultipath.Stop()
	})

	It("It should create the host dir for the persistent reservation", func() {
		go monitorMultipath.Run()
		Eventually(monitorMultipath.HostDir, 5*time.Second, 1*time.Second).Should(BeADirectory())
	})

	It("It should create the mount when the socket appears", func() {
		go monitorMultipath.Run()
		createFakeSocket()
		Eventually(func(g Gomega) {
			g.Expect(isMounted).To(BeTrue())
			g.Expect(multipathSocketPath).To(BeAnExistingFile())
		}, 10*time.Second, time.Second).Should(Succeed())
	})

	It("It should create mount when the socket already exists", func() {
		createFakeSocket()
		go monitorMultipath.Run()
		Eventually(func(g Gomega) {
			g.Expect(isMounted).To(BeTrue())
			g.Expect(multipathSocketPath).To(BeAnExistingFile())
		}, 10*time.Second, time.Second).Should(Succeed())
	})

	It("It should remove the mount when the socket is removed and recreate it", func() {
		createFakeSocket()
		go monitorMultipath.Run()
		Eventually(func(g Gomega) {
			g.Expect(isMounted).To(BeTrue())
			g.Expect(multipathSocketPath).To(BeAnExistingFile())
		}, 10*time.Second, time.Second).Should(Succeed())
		removeFakeSocket()
		Eventually(func(g Gomega) {
			g.Expect(isMounted).To(BeFalse())
			g.Expect(multipathSocketPath).ToNot(BeAnExistingFile())
		}, 10*time.Second, time.Second).Should(Succeed())
		createFakeSocket()
		Eventually(func(g Gomega) {
			g.Expect(isMounted).To(BeTrue())
			g.Expect(multipathSocketPath).To(BeAnExistingFile())
		}, 10*time.Second, time.Second).Should(Succeed())
	})

	It("It should keep the mount when another file in run is removed", func() {
		createFakeSocket()
		go monitorMultipath.Run()
		Eventually(func(g Gomega) {
			g.Expect(isMounted).To(BeTrue())
			g.Expect(multipathSocketPath).To(BeAnExistingFile())
		}, 10*time.Second, time.Second).Should(Succeed())
		file := filepath.Join(runDirPath, "test")
		_, err := os.Create(file)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.Remove(file)).To(Succeed())
		Expect(isMounted).To(BeTrue())
		Expect(multipathSocketPath).To(BeAnExistingFile())
	})
})
