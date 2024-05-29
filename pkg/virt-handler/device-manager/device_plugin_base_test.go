package device_manager

import (
	"errors"
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
				stop: stop,
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
})
