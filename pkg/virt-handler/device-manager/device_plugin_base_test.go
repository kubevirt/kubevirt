package device_manager

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

type fakeListAndWatchServer struct {
	listAndWatchResponseError error
	count                     int
	grpc.ServerStream
}

func (f *fakeListAndWatchServer) setlistAndWatchResponseError(err error) {
	f.listAndWatchResponseError = err
}

func (f *fakeListAndWatchServer) Send(m *pluginapi.ListAndWatchResponse) error {
	f.count++
	return f.listAndWatchResponseError
}

var _ = Describe("Device plugin base", func() {
	Context("List and Watch", func() {
		var (
			fakeServer   *fakeListAndWatchServer
			devicePlugin *DevicePluginBase
			stop         chan struct{}
		)
		BeforeEach(func() {
			stop = make(chan struct{})
			fakeServer = &fakeListAndWatchServer{
				count: 0,
			}
			devicePlugin = &DevicePluginBase{
				stop:         stop,
				deregistered: make(chan struct{}),
			}
		})
		AfterEach(func() {
			close(stop)
		})

		It("should return an error when the send the request to the server fails", func() {
			errExpected := errors.New("error")
			fakeServer.setlistAndWatchResponseError(errExpected)
			err := devicePlugin.ListAndWatch(nil, fakeServer)
			Expect(err).To(Equal(errExpected))
		})

		It("should gracefully stop the device plugin", func() {
			errChan := make(chan error, 1)
			go func(errChan chan error) {
				errChan <- devicePlugin.ListAndWatch(nil, fakeServer)
			}(errChan)
			stop <- struct{}{}
			Eventually(func() error {
				return (<-errChan)
			}, 5*time.Second).ShouldNot(HaveOccurred())
		})

		It("should send the device infomation to the server", func() {
			errChan := make(chan error, 1)
			devicePlugin.devs = []*pluginapi.Device{
				&pluginapi.Device{}, &pluginapi.Device{}, &pluginapi.Device{},
			}
			go func(errChan chan error) {
				errChan <- devicePlugin.ListAndWatch(nil, fakeServer)
			}(errChan)
			stop <- struct{}{}
			Eventually(func() error {
				return (<-errChan)
			}, 5*time.Second).ShouldNot(HaveOccurred())
			Expect(fakeServer.count).To(Equal(3))
		})
	})

	Context("device plugin lifecycle", func() {
		var dpi *DevicePluginBase
		var devicePath string

		BeforeEach(func() {
			workDir, err := os.MkdirTemp("", "kubevirt-test")
			Expect(err).ToNot(HaveOccurred())

			devicePath = path.Join(workDir, "foo")
			fileObj, err := os.Create(devicePath)
			Expect(err).ToNot(HaveOccurred())
			fileObj.Close()

			dpi = &DevicePluginBase{
				devs:       []*pluginapi.Device{&pluginapi.Device{}},
				socketPath: filepath.Join(workDir, "test.sock"),
				server:     grpc.NewServer([]grpc.ServerOption{}...),
				done:       make(chan struct{}),
				deviceRoot: "/",
				stop:       stop,
			}
			DeferCleanup(func() {
				close(stop)
				os.RemoveAll(workDir)
			})
		})

		It("Should stop if the device plugin socket file is deleted", func() {
			os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)
			var err error
			errChan := make(chan error, 1)
			go func(errChan chan error) {
				errChan <- dpi.healthCheck()
			}(errChan)
			Consistently(func() string {
				return dpi.devs[0].Health
			}, 2*time.Second, 500*time.Millisecond).Should(Equal(pluginapi.Healthy))
			Expect(os.Remove(dpi.socketPath)).To(Succeed())
			Expect(errChan).Should(Receive(&err))
			Expect(err).To(HaveOccurred())
		})

		It("Should monitor health of device node", func() {

			os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

			go dpi.healthCheck()
			Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))

			time.Sleep(1 * time.Second)
			By("Removing a (fake) device node")
			os.Remove(devicePath)

			By("waiting for healthcheck to send Unhealthy message")
			Eventually(func() string {
				return (<-dpi.health).Health
			}, 5*time.Second).Should(Equal(pluginapi.Unhealthy))

			By("Creating a new (fake) device node")
			fileObj, err := os.Create(devicePath)
			Expect(err).ToNot(HaveOccurred())
			fileObj.Close()

			By("waiting for healthcheck to send Healthy message")
			Eventually(func() string {
				return (<-dpi.health).Health
			}, 5*time.Second).Should(Equal(pluginapi.Healthy))
		})
	})

})
