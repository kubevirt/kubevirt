/*
Copyright 2026 Flant JSC

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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/tidwall/gjson"

	"github.com/deckhouse/kube-api-rewriter/pkg/labels"
	logutil "github.com/deckhouse/kube-api-rewriter/pkg/log"
	"github.com/deckhouse/kube-api-rewriter/pkg/rewriter"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// RoundTripper rewrites Kubernetes API requests and responses without an HTTP sidecar proxy.
// It is intended to wrap kubeclient transport in-process.
type RoundTripper struct {
	Name            string
	Base            http.RoundTripper
	ProxyMode       ProxyMode
	Rewriter        *rewriter.RuleBasedRewriter
	MetricsProvider MetricsProvider

	streamHandler *StreamHandler
}

var _ http.RoundTripper = (*RoundTripper)(nil)

func NewProxyRoundTripper(name string, mode ProxyMode, rules *rewriter.RewriteRules) *RoundTripper {
	rt := &RoundTripper{
		Name:      name,
		ProxyMode: mode,
		Rewriter: &rewriter.RuleBasedRewriter{
			Rules: rules,
		},
	}
	rt.initialize()
	return rt
}

// WrapRESTConfig attaches a transport wrapper to the provided rest.Config.
// For proxy.RoundTripper it also wires the previous transport into Base.
func WrapRESTConfig(cfg *rest.Config, wrapper http.RoundTripper) {
	prev := cfg.WrapTransport
	cfg.ContentType = apiruntime.ContentTypeJSON
	cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if prev != nil {
			rt = prev(rt)
		}

		rwr, ok := wrapper.(*RoundTripper)
		if !ok {
			return wrapper
		}

		copied := *rwr
		copied.initialize()
		copied.Base = rt
		return &copied
	}
}

func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("http request should not be nil")
	}
	if req.URL == nil {
		return nil, fmt.Errorf("http request URL should not be nil")
	}
	if rt.Rewriter == nil {
		return nil, fmt.Errorf("rewriter is required")
	}

	requestHandleStart := time.Now()

	targetReq := rewriter.NewTargetRequest(rt.Rewriter, req)
	if targetReq == nil {
		return rt.base().RoundTrip(req)
	}

	resource := targetReq.ResourceForLog()
	toTargetAction := string(ToTargetAction(rt.ProxyMode))
	fromTargetAction := string(FromTargetAction(rt.ProxyMode))
	ctx := labels.ContextWithCommon(req.Context(), rt.Name, resource, req.Method, WatchLabel(targetReq.IsWatch()), toTargetAction, fromTargetAction)
	logger := LoggerWithCommonAttrs(ctx, slog.String("url.path", req.URL.Path))
	metrics := NewProxyMetrics(ctx, rt.MetricsProvider)
	metrics.GotClientRequest()

	clonedReq := cloneRequest(req)

	helper := &Handler{
		ProxyMode: rt.ProxyMode,
		Rewriter:  rt.Rewriter,
	}
	origRequestBytes, rwrRequestBytes, err := helper.transformRequest(targetReq, clonedReq)
	if err != nil {
		metrics.ClientRequestRewriteError()
		return nil, fmt.Errorf("rewrite request: %w", err)
	}

	clientBodyDecision := decisionPass
	if rwrRequestBytes != nil {
		clonedReq.Body = io.NopCloser(bytes.NewReader(rwrRequestBytes))
		metrics.ClientRequestRewriteSuccess()
		clientBodyDecision = decisionRewrite
	} else if origRequestBytes != nil {
		clonedReq.Body = io.NopCloser(bytes.NewReader(origRequestBytes))
	}

	metrics.FromClientBytesAdd(clientBodyDecision, len(origRequestBytes))

	resp, err := rt.base().RoundTrip(clonedReq)
	if err != nil {
		metrics.TargetResponseError()
		return nil, err
	}

	ctx = labels.ContextWithStatus(ctx, resp.StatusCode)
	metrics = NewProxyMetrics(ctx, rt.MetricsProvider)
	metrics.ToTargetBytesAdd(clientBodyDecision, len(rwrRequestBytesOrOriginal(origRequestBytes, rwrRequestBytes)))
	metrics.TargetResponseSuccess(clientBodyDecision)

	if !targetReq.ShouldRewriteResponse() {
		ctx = labels.ContextWithDecision(ctx, decisionPass)
		NewProxyMetrics(ctx, rt.MetricsProvider).RequestDuration(time.Since(requestHandleStart))
		return resp, nil
	}

	ctx = labels.ContextWithDecision(ctx, decisionRewrite)
	metrics = NewProxyMetrics(ctx, rt.MetricsProvider)

	if targetReq.IsWatch() {
		rwrResp, err := rt.rewriteWatchResponse(ctx, resp, targetReq)
		if err != nil {
			metrics.RequestHandleError()
			return nil, err
		}
		metrics.RequestHandleSuccess()
		metrics.RequestDuration(time.Since(requestHandleStart))
		return rwrResp, nil
	}

	rwrResp, err := rt.rewriteResponse(ctx, resp, targetReq, logger)
	if err != nil {
		metrics.RequestHandleError()
		return nil, err
	}
	metrics.RequestHandleSuccess()
	metrics.RequestDuration(time.Since(requestHandleStart))
	return rwrResp, nil
}

func (rt *RoundTripper) initialize() {
	if rt.Name == "" {
		rt.Name = "kube-api-roundtripper"
	}
	if rt.MetricsProvider == nil {
		rt.MetricsProvider = NewMetricsProvider()
	}
	if rt.streamHandler == nil {
		rt.streamHandler = &StreamHandler{
			Rewriter:        rt.Rewriter,
			MetricsProvider: rt.MetricsProvider,
		}
	}
}

func (rt *RoundTripper) base() http.RoundTripper {
	if rt.Base != nil {
		return rt.Base
	}
	return http.DefaultTransport
}

func cloneRequest(req *http.Request) *http.Request {
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	if req.URL != nil {
		urlCopy := *req.URL
		cloned.URL = &urlCopy
	}
	cloned.RequestURI = ""
	return cloned
}

func rwrRequestBytesOrOriginal(orig, rwr []byte) []byte {
	if rwr != nil {
		return rwr
	}
	return orig
}

func (rt *RoundTripper) rewriteResponse(ctx context.Context, resp *http.Response, targetReq *rewriter.TargetRequest, logger *slog.Logger) (*http.Response, error) {
	metrics := NewProxyMetrics(ctx, rt.MetricsProvider)

	origRespBody := resp.Body
	defer origRespBody.Close()

	bytesCounter := BytesCounterReaderWrap(origRespBody)
	bodyReader, err := encodingAwareReaderWrap(bytesCounter, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	defer bodyReader.Close()

	origBodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("read response payload: %w", err)
	}
	metrics.FromTargetBytesAdd(CounterValue(bytesCounter))

	if !gjson.ValidBytes(origBodyBytes) {
		contentType := resp.Header.Get("Content-Type")
		if contentType != "" {
			logger.Warn(fmt.Sprintf("Will not transform non-JSON or invalid JSON response from target: Content-Type=%s", contentType))
		}
		metrics.TargetResponseInvalidJSON(resp.StatusCode)
		return replaceResponseBody(resp, origBodyBytes, true), nil
	}

	rewriteStart := time.Now()
	statusCode := resp.StatusCode
	rwrBodyBytes, err := rt.Rewriter.RewriteJSONPayload(targetReq, origBodyBytes, FromTargetAction(rt.ProxyMode))
	if err != nil {
		if !errors.Is(err, rewriter.SkipItem) {
			metrics.TargetResponseRewriteError()
			return nil, fmt.Errorf("rewrite response: %w", err)
		}

		rwrBodyBytes = notFoundJSON(targetReq.OrigResourceType(), origBodyBytes)
		statusCode = http.StatusNotFound
	}

	metrics.TargetResponseRewriteSuccess()
	metrics.TargetResponseRewriteDuration(time.Since(rewriteStart))

	logutil.DebugBodyChanges(logger, "Response", targetReq.ResourceForLog(), origBodyBytes, rwrBodyBytes)

	resp = replaceResponseBody(resp, rwrBodyBytes, true)
	resp.StatusCode = statusCode
	resp.Status = fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
	return resp, nil
}

func (rt *RoundTripper) rewriteWatchResponse(ctx context.Context, resp *http.Response, targetReq *rewriter.TargetRequest) (*http.Response, error) {
	origResp := *resp
	pipeReader, pipeWriter := io.Pipe()

	resp.Body = pipeReader
	resp.ContentLength = -1
	resp.Header.Del("Content-Length")
	resp.Header.Del("Content-Encoding")

	go func() {
		defer origResp.Body.Close()
		slog.Debug("watch goroutine: starting stream rewriter")
		if err := rt.streamHandler.Rewrite(ctx, pipeWriter, &origResp, targetReq); err != nil {
			slog.Error("watch goroutine: stream rewriter returned error", "err", err)
			pipeWriter.CloseWithError(err)
			return
		}
		slog.Debug("watch goroutine: stream rewriter finished, closing pipe")
		pipeWriter.Close()
	}()

	return resp, nil
}

func replaceResponseBody(resp *http.Response, body []byte, decoded bool) *http.Response {
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	if decoded {
		resp.Header.Del("Content-Encoding")
	}
	return resp
}
