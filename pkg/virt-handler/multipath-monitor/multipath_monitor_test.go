package multipath_monitor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/filewatcher"
	multipath_monitor "kubevirt.io/kubevirt/pkg/virt-handler/multipath-monitor"
)

var _ = Describe("MultipathSocketMonitor", func() {
	var (
		mounted                atomic.Bool
		runDirPath             string
		multipathSocketPath    string
		multipathSocketMonitor *multipath_monitor.MultipathSocketMonitor
	)

	isMounted := func() bool {
		return mounted.Load()
	}

	createSocket := func() {
		_, err := os.Create(multipathSocketPath)
		Expect(err).ToNot(HaveOccurred())
	}

	removeSocket := func() {
		Expect(os.Remove(multipathSocketPath)).To(Succeed())
	}

	BeforeEach(func() {
		mounted.Store(false)
		runDirPath = GinkgoT().TempDir()
		multipathSocketPath = filepath.Join(runDirPath, "multipathd.socket")

		mounter := multipath_monitor.NewMockmounter(gomock.NewController(GinkgoT()))
		mounter.EXPECT().IsMounted(gomock.Any()).
			DoAndReturn(func(_ *safepath.Path) (bool, error) {
				return mounted.Load(), nil
			}).AnyTimes()
		mounter.EXPECT().Mount(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_, _ *safepath.Path, _ bool) *exec.Cmd {
				mounted.Store(true)
				return exec.Command("/bin/true")
			}).AnyTimes()
		mounter.EXPECT().Umount(gomock.Any()).
			DoAndReturn(func(_ *safepath.Path) *exec.Cmd {
				mounted.Store(false)
				return exec.Command("/bin/true")
			}).AnyTimes()

		multipathSocketMonitor = &multipath_monitor.MultipathSocketMonitor{
			Watcher:             filewatcher.New(multipathSocketPath, 10*time.Millisecond),
			MultipathSocketPath: multipathSocketPath,
			HostDir:             filepath.Join(GinkgoT().TempDir(), "pr"),
			Mounter:             mounter,
		}
	})

	AfterEach(func() {
		multipathSocketMonitor.Close()
		Eventually(multipathSocketMonitor.Watcher.Events).Should(BeClosed())
		Eventually(multipathSocketMonitor.Watcher.Errors).Should(BeClosed())
	})

	It("It should create the host dir for the persistent reservation", func() {
		multipathSocketMonitor.Run()
		Eventually(multipathSocketMonitor.HostDir).Should(BeADirectory())
	})

	It("It should create the mount when the socket appears", func() {
		multipathSocketMonitor.Run()
		createSocket()
		Eventually(isMounted).Should(BeTrue())
	})

	It("It should create mount when the socket already exists", func() {
		createSocket()
		multipathSocketMonitor.Run()
		Eventually(isMounted).Should(BeTrue())
	})

	It("It should remove the mount when the socket is removed and recreate it", func() {
		multipathSocketMonitor.Run()
		createSocket()
		Eventually(isMounted).Should(BeTrue())
		removeSocket()
		Eventually(isMounted).Should(BeFalse())
		createSocket()
		Eventually(isMounted).Should(BeTrue())
	})

	It("It should keep the mount when another file in run is removed", func() {
		multipathSocketMonitor.Run()
		createSocket()
		Eventually(isMounted).Should(BeTrue())
		file := filepath.Join(runDirPath, "test")
		_, err := os.Create(file)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.Remove(file)).To(Succeed())
		Expect(isMounted()).To(BeTrue())
	})

	It("It should unmount when it is closed", func() {
		multipathSocketMonitor.Run()
		createSocket()
		Eventually(isMounted).Should(BeTrue())
		multipathSocketMonitor.Close()
		Expect(isMounted()).To(BeFalse())
	})
})
