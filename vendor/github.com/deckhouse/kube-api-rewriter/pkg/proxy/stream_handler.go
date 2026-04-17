/*
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	apiutilnet "k8s.io/apimachinery/pkg/util/net"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"

	logutil "github.com/deckhouse/kube-api-rewriter/pkg/log"
	"github.com/deckhouse/kube-api-rewriter/pkg/rewriter"
)

// StreamHandler reads a stream from the target, transforms events
// and sends them to the client.
type StreamHandler struct {
	Rewriter        *rewriter.RuleBasedRewriter
	MetricsProvider MetricsProvider
}

// streamRewriter reads a stream from the src reader, transforms events
// and sends them to the dst writer.
type streamRewriter struct {
	dst          io.Writer
	bytesCounter io.ReadCloser
	src          io.ReadCloser
	rewriter     *rewriter.RuleBasedRewriter
	targetReq    *rewriter.TargetRequest
	decoder      streaming.Decoder
	done         chan struct{}
	log          *slog.Logger
	metrics      *ProxyMetrics
}

// Handle starts a go routine to pass rewritten Watch Events
// from server to client.
// Sources:
// k8s.io/apimachinery@v0.26.1/pkg/watch/streamwatcher.go:100 receive method
// k8s.io/kubernetes@v1.13.0/staging/src/k8s.io/client-go/rest/request.go:537 wrapperFn, create framer.
// k8s.io/kubernetes@v1.13.0/staging/src/k8s.io/client-go/rest/request.go:598 instantiate watch NewDecoder
func (s *StreamHandler) Handle(ctx context.Context, w http.ResponseWriter, resp *http.Response, targetReq *rewriter.TargetRequest) error {
	rewriterInstance := &streamRewriter{
		dst:       w,
		targetReq: targetReq,
		rewriter:  s.Rewriter,
		done:      make(chan struct{}),
		log:       LoggerWithCommonAttrs(ctx),
		metrics:   NewProxyMetrics(ctx, s.MetricsProvider),
	}
	err := rewriterInstance.init(resp)
	if err != nil {
		return err
	}

	rewriterInstance.copyHeaders(w, resp)

	// Start rewriting stream.
	go rewriterInstance.start(ctx)

	<-rewriterInstance.DoneChan()
	return nil
}

// Rewrite reads a watch stream from resp, rewrites its events and writes them to dst.
func (s *StreamHandler) Rewrite(ctx context.Context, dst io.Writer, resp *http.Response, targetReq *rewriter.TargetRequest) error {
	rewriterInstance := &streamRewriter{
		dst:       dst,
		targetReq: targetReq,
		rewriter:  s.Rewriter,
		done:      make(chan struct{}),
		log:       LoggerWithCommonAttrs(ctx),
		metrics:   NewProxyMetrics(ctx, s.MetricsProvider),
	}
	err := rewriterInstance.init(resp)
	if err != nil {
		return err
	}

	rewriterInstance.start(ctx)
	return nil
}
func (s *streamRewriter) init(resp *http.Response) (err error) {
	s.bytesCounter = BytesCounterReaderWrap(resp.Body)
	s.src = s.bytesCounter

	if s.log.Enabled(nil, slog.LevelDebug) {
		s.src = logutil.NewReaderLogger(s.bytesCounter)
	}

	contentType := resp.Header.Get("Content-Type")
	s.decoder, err = createWatchDecoder(s.src, contentType)
	return err
}

func (s *streamRewriter) copyHeaders(w http.ResponseWriter, resp *http.Response) {
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
}

// proxy reads result from the decoder in a loop, rewrites and writes to a client.
// Sources
// k8s.io/apimachinery@v0.26.1/pkg/watch/streamwatcher.go:100 receive method
func (s *streamRewriter) start(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer s.Stop()

	for {
		// Read event from the server.
		var got metav1.WatchEvent
		s.log.Debug("Start decode from stream")
		res, _, err := s.decoder.Decode(nil, &got)
		s.metrics.FromTargetBytesAdd(CounterValue(s.bytesCounter))
		if s.log.Enabled(ctx, slog.LevelDebug) {
			s.log.Debug("Got decoded WatchEvent from stream", slog.Int("bytes_received", CounterValue(s.bytesCounter)))
		}
		CounterReset(s.bytesCounter)

		// Check if context was canceled.
		select {
		case <-ctx.Done():
			s.log.Debug("Context canceled, stop stream rewriter")
			return
		default:
		}

		if err != nil {
			switch err {
			case io.EOF:
				// Watch closed normally.
				s.log.Debug("Catch EOF from target, stop proxying the stream")
			case io.ErrUnexpectedEOF:
				s.log.Error("Unexpected EOF during watch stream event decoding", logutil.SlogErr(err))
			default:
				if apiutilnet.IsProbableEOF(err) || apiutilnet.IsTimeout(err) {
					s.log.Error("Unable to decode an event from the watch stream", logutil.SlogErr(err))
				} else {
					s.log.Error("Unable to decode an event from the watch stream", logutil.SlogErr(err))
				}
			}
			return
		}

		watchEventHandleStart := time.Now()

		var rwrEvent *metav1.WatchEvent
		if res != &got {
			s.log.Warn(fmt.Sprintf("unable to decode to metav1.Event: res=%#v, got=%#v", res, got))
			s.metrics.TargetResponseInvalidJSON(200)
			s.metrics.RequestHandleError()
			// There is nothing to send to the client: no event decoded.
		} else {
			var transformErr error
			rwrEvent, transformErr = s.transformWatchEvent(&got)

			if transformErr != nil && errors.Is(transformErr, rewriter.SkipItem) {
				s.log.Warn(fmt.Sprintf("Watch event '%s': skipped by rewriter", got.Type), logutil.SlogErr(transformErr))
				logutil.DebugBodyHead(s.log, fmt.Sprintf("Watch event '%s' skipped", got.Type), s.targetReq.ResourceForLog(), got.Object.Raw)
				s.metrics.RequestHandleSuccess()
			} else {
				if transformErr != nil {
					s.log.Error(fmt.Sprintf("Watch event '%s': transform error", got.Type), logutil.SlogErr(transformErr))
					logutil.DebugBodyHead(s.log, fmt.Sprintf("Watch event '%s'", got.Type), s.targetReq.ResourceForLog(), got.Object.Raw)
				}
				if rwrEvent == nil {
					// No rewrite, pass original event as-is.
					rwrEvent = &got
				} else {
					// Log changes after rewrite.
					logutil.DebugBodyChanges(s.log, "Watch event", s.targetReq.ResourceForLog(), got.Object.Raw, rwrEvent.Object.Raw)
				}
				// Pass event to the client.
				logutil.DebugBodyHead(s.log, fmt.Sprintf("WatchEvent type '%s' send back to client %d bytes", rwrEvent.Type, len(rwrEvent.Object.Raw)), s.targetReq.ResourceForLog(), rwrEvent.Object.Raw)

				s.writeEvent(rwrEvent)
			}
		}

		s.metrics.RequestDuration(time.Since(watchEventHandleStart))

		// Check if application is stopped before waiting for the next event.
		select {
		case <-s.done:
			return
		default:
		}
	}
}

func (s *streamRewriter) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *streamRewriter) DoneChan() chan struct{} {
	return s.done
}

// createSerializers
// Source
// k8s.io/client-go@v0.26.1/rest/request.go:765 newStreamWatcher
// k8s.io/apimachinery@v0.26.1/pkg/runtime/negotiate.go:70 StreamDecoder
func createWatchDecoder(r io.Reader, contentType string) (streaming.Decoder, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("unexpected media type from the server: %q: %w", contentType, err)
	}

	negotiatedSerializer := scheme.Codecs.WithoutConversion()
	mediaTypes := negotiatedSerializer.SupportedMediaTypes()
	info, ok := runtime.SerializerInfoForMediaType(mediaTypes, mediaType)
	if !ok {
		if len(contentType) != 0 || len(mediaTypes) == 0 {
			return nil, fmt.Errorf("no matching serializer for media type '%s'", contentType)
		}
		info = mediaTypes[0]
	}
	if info.StreamSerializer == nil {
		return nil, fmt.Errorf("no serializer for content type %s", contentType)
	}

	// A chain of the framer and the serializer will split body stream into JSON objects.
	frameReader := info.StreamSerializer.Framer.NewFrameReader(io.NopCloser(r))
	streamingDecoder := streaming.NewDecoder(frameReader, info.StreamSerializer.Serializer)
	return streamingDecoder, nil
}

func (s *streamRewriter) transformWatchEvent(ev *metav1.WatchEvent) (*metav1.WatchEvent, error) {
	switch ev.Type {
	case string(watch.Added), string(watch.Modified), string(watch.Deleted), string(watch.Error), string(watch.Bookmark):
	default:
		return nil, fmt.Errorf("got unknown type in WatchEvent: %v", ev.Type)
	}

	group := gjson.GetBytes(ev.Object.Raw, "apiVersion").String()
	kind := gjson.GetBytes(ev.Object.Raw, "kind").String()
	name := gjson.GetBytes(ev.Object.Raw, "metadata.name").String()
	ns := gjson.GetBytes(ev.Object.Raw, "metadata.namespace").String()

	// TODO add pass-as-is for non rewritable objects.
	if group == "" && kind == "" {
		// Object in event is undetectable, pass this event as-is.
		return nil, fmt.Errorf("object has no apiVersion and kind")
	}
	s.log.Debug(fmt.Sprintf("Receive '%s' watch event with %s/%s %s/%s object", ev.Type, group, kind, ns, name))

	var rwrObjBytes []byte
	var err error
	rewriteStart := time.Now()
	defer func() {
		s.metrics.TargetResponseRewriteDuration(time.Since(rewriteStart))
	}()

	if ev.Type == string(watch.Bookmark) {
		// Temporarily print original BOOKMARK WatchEvent.
		logutil.DebugBodyHead(s.log, fmt.Sprintf("Watch event '%s' from target", ev.Type), s.targetReq.OrigResourceType(), ev.Object.Raw)
		rwrObjBytes, err = s.rewriter.RestoreBookmark(s.targetReq, ev.Object.Raw)
	} else {
		// Restore object in the event. Watch responses are always from the Kubernetes API server, so rename is not needed.
		rwrObjBytes, err = s.rewriter.RewriteJSONPayload(s.targetReq, ev.Object.Raw, rewriter.Restore)
	}
	if err != nil {
		if errors.Is(err, rewriter.SkipItem) {
			s.metrics.TargetResponseRewriteSuccess()
			return nil, err
		}
		s.metrics.TargetResponseRewriteError()
		return nil, fmt.Errorf("rewrite object in WatchEvent '%s': %w", ev.Type, err)
	}

	s.metrics.TargetResponseRewriteSuccess()
	// Prepare rewritten event bytes.
	return &metav1.WatchEvent{
		Type: ev.Type,
		Object: runtime.RawExtension{
			Raw: rwrObjBytes,
		},
	}, nil
}

func (s *streamRewriter) writeEvent(ev *metav1.WatchEvent) {
	rwrEventBytes, err := json.Marshal(ev)
	if err != nil {
		s.log.Error("encode restored event to bytes", logutil.SlogErr(err))
		return
	}

	// Send rewritten event to the client.
	copied, err := s.dst.Write(rwrEventBytes)
	if err != nil {
		s.log.Error("Watch event: error writing event to the client", logutil.SlogErr(err))
		s.metrics.RequestHandleError()
	} else {
		s.metrics.RequestHandleSuccess()
		s.metrics.ToClientBytesAdd(copied)
	}
	// Flush writer to immediately send any buffered content to the client.
	if wr, ok := s.dst.(http.Flusher); ok {
		wr.Flush()
	}
}
