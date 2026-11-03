/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package eventsclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api2 "kubevirt.io/client-go/api"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Notify", func() {

	Describe("K8s Events", func() {
		var client *Notifier
		var recorder *record.FakeRecorder
		var vmiStore cache.Store

		BeforeEach(func() {
			stop := make(chan struct{})
			eventChan := make(chan watch.Event, 100)
			shareDir, err := os.MkdirTemp("", "kubevirt-share")
			Expect(err).ToNot(HaveOccurred())

			recorder = record.NewFakeRecorder(10)
			recorder.IncludeObject = true
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()

			go func(rec record.EventRecorder, store cache.Store) {
				notifyserver.RunServer(shareDir, stop, eventChan, rec, store)
			}(recorder, vmiStore)
			// mimic pipe
			notifyServer := filepath.Join(shareDir, "domain-notify.sock")
			pipePath := filepath.Join(shareDir, "domain-notify-pipe.sock")
			Expect(os.Symlink(notifyServer, pipePath)).To(Succeed())

			client = NewNotifier(shareDir)

			DeferCleanup(func() {
				close(stop)
				client.Close()
				os.RemoveAll(shareDir)
			})
		})

		It("Should send a k8s event", func() {

			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			err := client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

			Expect(recorder.Events).To(Receive(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage))))
		})

	})

	Describe("Version mismatch", func() {

		var err error
		var ctrl *gomock.Controller
		var infoClient *info.MockNotifyInfoClient

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			infoClient = info.NewMockNotifyInfoClient(ctrl)
		})

		It("Should report error when server version mismatches", func() {

			fakeResponse := info.NotifyInfoResponse{
				SupportedNotifyVersions: []uint32{42},
			}
			infoClient.EXPECT().Info(gomock.Any(), gomock.Any()).Return(&fakeResponse, nil)

			By("Initializing the notifier")
			_, err = negotiateVersion(infoClient)

			Expect(err).To(HaveOccurred(), "Should have returned error about incompatible versions")
			Expect(err.Error()).To(ContainSubstring("no compatible version found"), "Expected error message to contain 'no compatible version found'")

		})
	})

	Describe("isTransientError", func() {
		DescribeTable("should classify errors correctly",
			func(err error, expected bool) {
				Expect(isTransientError(err)).To(Equal(expected))
			},
			Entry("nil error", nil, false),
			Entry("Unavailable is transient", status.Errorf(codes.Unavailable, "unavailable"), true),
			Entry("DeadlineExceeded is transient", status.Errorf(codes.DeadlineExceeded, "timeout"), true),
			Entry("Canceled is not transient", status.Errorf(codes.Canceled, "canceled"), false),
			Entry("Aborted is transient", status.Errorf(codes.Aborted, "aborted"), true),
			Entry("InvalidArgument is not transient", status.Errorf(codes.InvalidArgument, "bad request"), false),
			Entry("Internal is not transient", status.Errorf(codes.Internal, "internal error"), false),
			Entry("NotFound is not transient", status.Errorf(codes.NotFound, "not found"), false),
			Entry("PermissionDenied is not transient", status.Errorf(codes.PermissionDenied, "denied"), false),
			Entry("non-gRPC error is transient", errors.New("connection refused"), true),
			Entry("context.DeadlineExceeded is transient", context.DeadlineExceeded, true),
		)
	})

	Describe("gRPC status error handling", func() {
		newTestNotifier := func(fake *fakeNotifyClient) *Notifier {
			return &Notifier{
				client: &notifyClient{
					v1client:        fake,
					conn:            &grpc.ClientConn{},
					intervalTimeout: 100 * time.Millisecond,
					sendTimeout:     1 * time.Second,
					totalTimeout:    3 * time.Second,
				},
			}
		}

		Context("SendDomainEvent", func() {
			It("should propagate business errors immediately", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventErr: status.Errorf(codes.InvalidArgument, "invalid domain event"),
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).To(MatchError(ContainSubstring("InvalidArgument")))
				Expect(err).To(MatchError(ContainSubstring("invalid domain event")))
			})

			It("should propagate Internal errors from server", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventErr: status.Errorf(codes.Internal, "server crashed"),
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).To(MatchError(ContainSubstring("Internal")))
				Expect(err).To(MatchError(ContainSubstring("server crashed")))
			})

			It("should succeed when server returns no error", func() {
				client := newTestNotifier(&fakeNotifyClient{})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).ToNot(HaveOccurred())
			})

			It("should fall back to Response.Success for old servers", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventResponse: &notifyv1.Response{Success: false, Message: "legacy error"},
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).To(MatchError(ContainSubstring("legacy error")))
			})

			It("should succeed when a migrated server returns an empty response without Success field", func() {
				client := newTestNotifier(&fakeNotifyClient{
					domainEventResponse: &notifyv1.Response{},
				})

				domain := api.NewMinimalDomain("test")
				err := client.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("SendK8sEvent", func() {
			It("should propagate business errors immediately", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventErr: status.Errorf(codes.InvalidArgument, "invalid k8s event"),
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).To(MatchError(ContainSubstring("InvalidArgument")))
				Expect(err).To(MatchError(ContainSubstring("invalid k8s event")))
			})

			It("should propagate Internal errors from server", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventErr: status.Errorf(codes.Internal, "handler error"),
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).To(MatchError(ContainSubstring("Internal")))
				Expect(err).To(MatchError(ContainSubstring("handler error")))
			})

			It("should succeed when server returns no error", func() {
				client := newTestNotifier(&fakeNotifyClient{})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).ToNot(HaveOccurred())
			})

			It("should fall back to Response.Success for old servers", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventResponse: &notifyv1.Response{Success: false, Message: "legacy k8s error"},
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).To(MatchError(ContainSubstring("legacy k8s error")))
			})

			It("should succeed when a migrated server returns an empty response without Success field", func() {
				client := newTestNotifier(&fakeNotifyClient{
					k8sEventResponse: &notifyv1.Response{},
				})

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := client.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("transient error retry", func() {
			var shareDir string
			var grpcServer *grpc.Server
			var sock net.Listener
			var retryClient *Notifier

			startRetryServer := func(server notifyv1.NotifyServer) *Notifier {
				shareDir = GinkgoT().TempDir()

				grpcServer = grpc.NewServer()
				info.RegisterNotifyInfoServer(grpcServer, &testInfoServer{})
				notifyv1.RegisterNotifyServer(grpcServer, server)

				var err error
				sock, err = net.Listen("unix", filepath.Join(shareDir, "domain-notify.sock"))
				Expect(err).ToNot(HaveOccurred())
				go func() { _ = grpcServer.Serve(sock) }()

				Expect(os.Symlink(
					filepath.Join(shareDir, "domain-notify.sock"),
					filepath.Join(shareDir, "domain-notify-pipe.sock"),
				)).To(Succeed())

				retryClient = NewNotifier(shareDir)
				retryClient.SetCustomTimeouts(50*time.Millisecond, 1*time.Second, 5*time.Second)
				return retryClient
			}

			AfterEach(func() {
				if retryClient != nil {
					retryClient.Close()
				}
				if grpcServer != nil {
					grpcServer.Stop()
				}
				if sock != nil {
					sock.Close()
				}
			})

			It("SendDomainEvent should fail a transient error once and succeed on the next send", func() {
				var callCount atomic.Int32
				server := &callbackNotifyServer{
					onDomainEvent: func(_ context.Context, _ *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
						if callCount.Add(1) == 1 {
							return nil, status.Errorf(codes.Unavailable, "temporary failure")
						}
						return &notifyv1.Response{Success: true}, nil
					},
				}
				startRetryServer(server)
				domain := api.NewMinimalDomain("test")

				Expect(retryClient.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})).
					To(MatchError(ContainSubstring("Unavailable")))
				Expect(retryClient.SendDomainEvent(context.Background(), watch.Event{Type: watch.Modified, Object: domain})).
					To(Succeed())
				Expect(callCount.Load()).To(Equal(int32(2)))
			})

			It("SendDomainEvent should stop when its context is cancelled", func() {
				server := &callbackNotifyServer{
					onDomainEvent: func(ctx context.Context, _ *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
						<-ctx.Done()
						return nil, ctx.Err()
					},
				}
				startRetryServer(server)
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				err := retryClient.SendDomainEvent(ctx, watch.Event{Type: watch.Modified, Object: api.NewMinimalDomain("test")})

				Expect(err).To(MatchError(ContainSubstring("Canceled")))
			})

			It("SendK8sEvent should succeed after a transient error on retry", func() {
				var callCount atomic.Int32
				server := &callbackNotifyServer{
					onK8SEvent: func(_ context.Context, _ *notifyv1.K8SEventRequest) (*notifyv1.Response, error) {
						if callCount.Add(1) == 1 {
							return nil, status.Errorf(codes.Unavailable, "temporary failure")
						}
						return &notifyv1.Response{Success: true}, nil
					},
				}
				startRetryServer(server)

				vmi := libvmi.New(libvmi.WithName("test-vmi"))
				vmi.UID = "1234"
				err := retryClient.SendK8sEvent(vmi, "Normal", "TestReason", "test message")

				Expect(err).ToNot(HaveOccurred())
				Expect(callCount.Load()).To(BeNumerically(">=", int32(2)))
			})
		})
	})
})

type fakeNotifyClient struct {
	domainEventErr      error
	domainEventResponse *notifyv1.Response
	k8sEventErr         error
	k8sEventResponse    *notifyv1.Response
}

func (f *fakeNotifyClient) HandleDomainEvent(
	_ context.Context, _ *notifyv1.DomainEventRequest, _ ...grpc.CallOption,
) (*notifyv1.Response, error) {
	if f.domainEventErr != nil {
		return nil, f.domainEventErr
	}
	if f.domainEventResponse != nil {
		return f.domainEventResponse, nil
	}
	return &notifyv1.Response{Success: true}, nil
}

func (f *fakeNotifyClient) HandleK8SEvent(
	_ context.Context, _ *notifyv1.K8SEventRequest, _ ...grpc.CallOption,
) (*notifyv1.Response, error) {
	if f.k8sEventErr != nil {
		return nil, f.k8sEventErr
	}
	if f.k8sEventResponse != nil {
		return f.k8sEventResponse, nil
	}
	return &notifyv1.Response{Success: true}, nil
}

type testInfoServer struct{}

func (t *testInfoServer) Info(_ context.Context, _ *info.NotifyInfoRequest) (*info.NotifyInfoResponse, error) {
	return &info.NotifyInfoResponse{
		SupportedNotifyVersions: []uint32{1},
	}, nil
}

type callbackNotifyServer struct {
	onDomainEvent func(context.Context, *notifyv1.DomainEventRequest) (*notifyv1.Response, error)
	onK8SEvent    func(context.Context, *notifyv1.K8SEventRequest) (*notifyv1.Response, error)
}

func (s *callbackNotifyServer) HandleDomainEvent(ctx context.Context, req *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
	if s.onDomainEvent != nil {
		return s.onDomainEvent(ctx, req)
	}
	return &notifyv1.Response{Success: true}, nil
}

func (s *callbackNotifyServer) HandleK8SEvent(ctx context.Context, req *notifyv1.K8SEventRequest) (*notifyv1.Response, error) {
	if s.onK8SEvent != nil {
		return s.onK8SEvent(ctx, req)
	}
	return &notifyv1.Response{Success: true}, nil
}
