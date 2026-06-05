package virtio_serial

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/textproto"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/time/rate"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
)

const (
	emptyMetricsReply = "<metrics><!-- host metrics not available --><!-- VM metrics not available --></metrics>"
	invalidReqReply   = "INVALID REQUEST"
)

func newFakeMetricsReporter() metricsReporter {
	return func() (*api.Metrics, error) {
		return nil, errors.New("fake empty metrics")
	}
}

func newServer() downwardMetricsServer {
	return downwardMetricsServer{
		rateLimiter:        rate.NewLimiter(rate.Inf, 1),
		maxConnectAttempts: 1,
		virtioSerialSocket: "dumb.sock",
		reportFn:           newFakeMetricsReporter(),
	}
}

var _ = Describe("DownwardMetrics virtio-serial server", func() {
	DescribeTable("Parse request", func(request string, shouldFail bool) {
		err := parseRequest(request)

		if shouldFail {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("Should succeed to parse a valid request", "GET /metrics/XML", false),
		Entry("Should fail to parse an unsupported method", "PUT /metrics/XML", true),
		Entry("Should fail to parse an missing method", "/metrics/XML", true),
		Entry("Should fail to parse an invalid URI with extra space", "GET  /metrics/XML", true),
		Entry("Should fail to parse an invalid URI", "GET \\metrics\\XML", true),
		Entry("Should fail to parse an unsupported path", "GET /foo/bar", true),
	)

	Context("Send a valid request and close the qemu side connection", func() {
		It("Should respond with an empty metrics and two new lines", func() {
			qemu, msrv := net.Pipe()
			reader := textproto.NewReader(bufio.NewReader(qemu))

			done := make(chan struct{})
			ctx, cancelCtx := context.WithCancel(context.Background())
			defer cancelCtx()

			By("Starting the server")
			server := newServer()
			go func() {
				defer close(done)
				server.serve(ctx, msrv)
			}()

			By("Sending the request")
			_, err := qemu.Write([]byte("GET /metrics/XML\n\n"))
			Expect(err).NotTo(HaveOccurred())

			By("Reading the first line response")
			result, err := reader.ReadLine()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(emptyMetricsReply))

			By("Reading the empty line")
			result, err = reader.ReadLine()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(""))

			By("Closing the qemu connection")
			err = qemu.Close()
			Expect(err).NotTo(HaveOccurred())

			By("Should stop gracefully")
			Eventually(done).WithTimeout(5 * time.Second).Should(BeClosed())
		})
	})

	Context("Send an invalid request and close the qemu side connection", func() {
		It("Should respond with an empty metrics and two new lines", func() {
			qemu, msrv := net.Pipe()
			reader := textproto.NewReader(bufio.NewReader(qemu))

			done := make(chan struct{})
			ctx, cancelCtx := context.WithCancel(context.Background())
			defer cancelCtx()

			By("Starting the server")
			server := newServer()
			go func() {
				defer close(done)
				server.serve(ctx, msrv)
			}()

			By("Sending the request")
			_, err := qemu.Write([]byte("GET /foo\n\n"))
			Expect(err).NotTo(HaveOccurred())

			By("Reading the first line response")
			result, err := reader.ReadLine()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(invalidReqReply))

			By("Reading the empty line")
			result, err = reader.ReadLine()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(""))

			By("Closing the qemu connection")
			err = qemu.Close()
			Expect(err).NotTo(HaveOccurred())

			By("Should stop gracefully")
			Eventually(done).WithTimeout(5 * time.Second).Should(BeClosed())
		})
	})

	Context("Send a valid request and signal the server to stop", func() {
		It("Should respond with an empty metrics and two new lines", func() {
			qemu, msrv := net.Pipe()
			reader := textproto.NewReader(bufio.NewReader(qemu))
			defer func() { _ = qemu.Close() }()

			done := make(chan struct{})
			ctx, cancelCtx := context.WithCancel(context.Background())

			By("Starting the server")
			server := newServer()
			go func() {
				defer close(done)
				server.serve(ctx, msrv)
			}()

			By("Sending the request")
			_, err := qemu.Write([]byte("GET /metrics/XML\n\n"))
			Expect(err).NotTo(HaveOccurred())

			By("Reading the first line response")
			result, err := reader.ReadLine()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(emptyMetricsReply))

			By("Reading the empty line")
			result, err = reader.ReadLine()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(""))

			By("Stopping the server")
			cancelCtx()
			Eventually(done).WithTimeout(5 * time.Second).Should(BeClosed())

			By("The connection should be closed")
			_, err = reader.ReadLine()
			Expect(err).To(HaveOccurred())
		})
	})
})
