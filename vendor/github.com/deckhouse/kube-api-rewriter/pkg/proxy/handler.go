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
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/deckhouse/kube-api-rewriter/pkg/labels"
	logutil "github.com/deckhouse/kube-api-rewriter/pkg/log"
	"github.com/deckhouse/kube-api-rewriter/pkg/rewriter"
)

type ProxyMode string

const (
	// ToOriginal mode indicates that resource should be restored when passed to target and renamed when passing back to client.
	ToOriginal ProxyMode = "original"
	// ToRenamed mode indicates that resource should be renamed when passed to target and restored when passing back to client.
	ToRenamed ProxyMode = "renamed"
)

func ToTargetAction(proxyMode ProxyMode) rewriter.Action {
	if proxyMode == ToRenamed {
		return rewriter.Rename
	}
	return rewriter.Restore
}

func FromTargetAction(proxyMode ProxyMode) rewriter.Action {
	if proxyMode == ToRenamed {
		return rewriter.Restore
	}
	return rewriter.Rename
}

type Handler struct {
	Name string
	// ProxyPass is a target http client to send requests to.
	// An allusion to nginx proxy_pass directive.
	TargetClient    *http.Client
	TargetURL       *url.URL
	ProxyMode       ProxyMode
	Rewriter        *rewriter.RuleBasedRewriter
	MetricsProvider MetricsProvider
	streamHandler   *StreamHandler
	m               sync.Mutex
}

func (h *Handler) Init() {
	if h.MetricsProvider == nil {
		h.MetricsProvider = NewMetricsProvider()
	}
	h.streamHandler = &StreamHandler{
		Rewriter:        h.Rewriter,
		MetricsProvider: h.MetricsProvider,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req == nil {
		slog.Error("req is nil. something wrong")
		return
	}
	if req.URL == nil {
		slog.Error(fmt.Sprintf("req.URL is nil. something wrong. method %s RequestURI '%s' Headers %+v", req.Method, req.RequestURI, req.Header))
		return
	}

	requestHandleStart := time.Now()

	// Step 1. Parse request url, prepare path rewrite.
	targetReq := rewriter.NewTargetRequest(h.Rewriter, req)

	resource := targetReq.ResourceForLog()
	toTargetAction := string(ToTargetAction(h.ProxyMode))
	fromTargetAction := string(FromTargetAction(h.ProxyMode))
	ctx := labels.ContextWithCommon(req.Context(), h.Name, resource, req.Method, WatchLabel(targetReq.IsWatch()), toTargetAction, fromTargetAction)

	logger := LoggerWithCommonAttrs(ctx,
		slog.String("url.path", req.URL.Path),
	)

	metrics := NewProxyMetrics(ctx, h.MetricsProvider)
	metrics.GotClientRequest()

	// Set target address, cleanup RequestURI.
	req.RequestURI = ""
	req.URL.Scheme = h.TargetURL.Scheme
	req.URL.Host = h.TargetURL.Host

	// Log request path.
	rwrReq := " NO"
	if targetReq.ShouldRewriteRequest() {
		rwrReq = "REQ"
	}
	rwrResp := "  NO"
	if targetReq.ShouldRewriteResponse() {
		rwrResp = "RESP"
	}
	if targetReq.Path() != req.URL.Path || targetReq.RawQuery() != req.URL.RawQuery {
		logger.Info(fmt.Sprintf("%s [%s,%s] %s -> %s", req.Method, rwrReq, rwrResp, req.URL.RequestURI(), targetReq.RequestURI()))
	} else {
		logger.Info(fmt.Sprintf("%s [%s,%s] %s", req.Method, rwrReq, rwrResp, req.URL.String()))
	}

	// TODO(development): Mute some logging for development: election, non-rewritable resources.
	isMute := false
	if !targetReq.ShouldRewriteRequest() && !targetReq.ShouldRewriteResponse() {
		isMute = true
	}
	switch resource {
	case "leases":
		isMute = true
	case "endpoints":
		isMute = true
	case "clusterrolebindings":
		isMute = false
	case "clustervirtualmachineimages":
		isMute = false
	case "virtualmachines":
		isMute = false
	case "virtualmachines/status":
		isMute = false
	}
	if isMute {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	logger.Debug(fmt.Sprintf("Request: orig headers: %+v", req.Header))

	// Step 2. Modify request endpoint, headers and body bytes before send it to the target.
	origRequestBytes, rwrRequestBytes, err := h.transformRequest(targetReq, req)
	if err != nil {
		logger.Error(fmt.Sprintf("Error transforming request: %s", req.URL.String()), logutil.SlogErr(err))
		http.Error(w, "can't rewrite request", http.StatusBadRequest)
		metrics.ClientRequestRewriteError()
		return
	}

	logger.Debug(fmt.Sprintf("Request: target headers: %+v", req.Header))

	// Restore req.Body as this reader was read earlier by the transformRequest.
	clientBodyDecision := decisionPass
	if rwrRequestBytes != nil {
		req.Body = BytesCounterReaderWrap(bytes.NewBuffer(rwrRequestBytes))
		metrics.ClientRequestRewriteSuccess()
		clientBodyDecision = decisionRewrite
		// metrics.ClientRequestRewriteDuration()
	} else if origRequestBytes != nil {
		// Fallback to origRequestBytes if body was not rewritten.
		req.Body = BytesCounterReaderWrap(bytes.NewBuffer(origRequestBytes))
	}

	metrics.FromClientBytesAdd(clientBodyDecision, len(origRequestBytes))

	// Step 3. Send request to the target.
	resp, err := h.TargetClient.Do(req)
	if err != nil {
		logger.Error("Error passing request to the target", logutil.SlogErr(err))
		http.Error(w, k8serrors.NewInternalError(err).Error(), http.StatusInternalServerError)
		metrics.TargetResponseError()
		return
	}

	ctx = labels.ContextWithStatus(ctx, resp.StatusCode)
	metrics = NewProxyMetrics(ctx, h.MetricsProvider)
	metrics.ToTargetBytesAdd(clientBodyDecision, CounterValue(req.Body))
	metrics.TargetResponseSuccess(clientBodyDecision)

	// Save original Body to close when handler finishes.
	origRespBody := resp.Body
	defer func() {
		origRespBody.Close()
	}()

	// TODO handle resp.Status 3xx, 4xx, 5xx, etc.

	if req.Method == http.MethodPatch {
		logutil.DebugBodyHead(logger, "Request PATCH", "patch", origRequestBytes)
		logutil.DebugBodyChanges(logger, "Request PATCH", "patch", origRequestBytes, rwrRequestBytes)
	} else {
		logutil.DebugBodyChanges(logger, "Request", resource, origRequestBytes, rwrRequestBytes)
	}

	// Step 5. Handle response: pass through, transform resp.Body, or run stream transformer.

	if !targetReq.ShouldRewriteResponse() {
		ctx = labels.ContextWithDecision(ctx, decisionPass)
		metrics = NewProxyMetrics(ctx, h.MetricsProvider)
		// Pass response as-is without rewriting.
		if targetReq.IsWatch() {
			logger.Debug(fmt.Sprintf("Response decision: PASS STREAM, Status %s, Headers %+v", resp.Status, resp.Header))
		} else {
			logger.Debug(fmt.Sprintf("Response decision: PASS, Status %s, Headers %+v", resp.Status, resp.Header))
		}
		h.passResponse(ctx, targetReq, w, resp, logger)
		metrics.RequestDuration(time.Since(requestHandleStart))
		return
	}

	ctx = labels.ContextWithDecision(ctx, decisionRewrite)
	metrics = NewProxyMetrics(ctx, h.MetricsProvider)

	if targetReq.IsWatch() {
		logger.Debug(fmt.Sprintf("Response decision: REWRITE STREAM, Status %s, Headers %+v", resp.Status, resp.Header))

		h.transformStream(ctx, targetReq, w, resp, logger)
		metrics.RequestDuration(time.Since(requestHandleStart))
		return
	}

	// One-time rewrite is required for client or webhook requests.
	logger.Debug(fmt.Sprintf("Response decision: REWRITE, Status %s, Headers %+v", resp.Status, resp.Header))

	h.transformResponse(ctx, targetReq, w, resp, logger)
	metrics.RequestDuration(time.Since(requestHandleStart))
	return
}

func copyHeader(dst, src http.Header) {
	for header, values := range src {
		// Do not override dst header with the header from the src.
		if len(dst.Values(header)) > 0 {
			continue
		}
		for _, value := range values {
			dst.Add(header, value)
		}
	}
}

// resp.Header.Get("Content-Encoding")
func encodingAwareReaderWrap(bodyReader io.ReadCloser, encoding string) (io.ReadCloser, error) {
	var reader io.ReadCloser
	var err error

	switch encoding {
	case "gzip":
		reader, err = gzip.NewReader(bodyReader)
		if err != nil {
			return nil, fmt.Errorf("errorf making gzip reader: %v", err)
		}
		return io.NopCloser(reader), nil
	case "deflate":
		return flate.NewReader(bodyReader), nil
	}

	return bodyReader, nil
}

// transformRequest transforms request headers and rewrites request payload to use
// request as client to the target.
// TargetMode field defines either transformer should rename resources
// if request is from the client, or restore resources if it is a call
// from the API Server to the webhook.
func (h *Handler) transformRequest(targetReq *rewriter.TargetRequest, req *http.Request) ([]byte, []byte, error) {
	if req == nil || req.URL == nil {
		return nil, nil, fmt.Errorf("http request and URL should not be nil")
	}

	var origBodyBytes []byte
	var rwrBodyBytes []byte
	var err error

	hasPayload := req.Body != nil

	if hasPayload {
		origBodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("read request body: %w", err)
		}
	}

	// Rewrite incoming payload, e.g. create, put, etc.
	if targetReq.ShouldRewriteRequest() && hasPayload {
		switch req.Method {
		case http.MethodPatch:
			rwrBodyBytes, err = h.Rewriter.RewritePatch(targetReq, origBodyBytes)
		default:
			rwrBodyBytes, err = h.Rewriter.RewriteJSONPayload(targetReq, origBodyBytes, ToTargetAction(h.ProxyMode))
		}
		if err != nil {
			return nil, nil, err
		}

		// Put new Body reader to req and fix Content-Length header.
		rwrBodyLen := len(rwrBodyBytes)
		if rwrBodyLen > 0 {
			// Fix content-length if needed.
			req.ContentLength = int64(rwrBodyLen)
			if req.Header.Get("Content-Length") != "" {
				req.Header.Set("Content-Length", strconv.Itoa(rwrBodyLen))
			}
		}
	}

	// TODO Implement protobuf and table rewriting to remove these manipulations with Accept header.
	// TODO Move out to a separate method forceApplicationJSONContent.
	if targetReq.ShouldRewriteResponse() {
		newAccept := make([]string, 0)
		for _, hdr := range req.Header.Values("Accept") {
			// Rewriter doesn't work with protobuf, force JSON in Accept header.
			// This workaround is suitable only for empty body requests: Get, List, etc.
			// A client should be patched to send JSON requests.
			if strings.Contains(hdr, "application/vnd.kubernetes.protobuf") {
				newAccept = append(newAccept, "application/json")
				continue
			}

			// TODO Add rewriting support for Table format.
			// Quickly support kubectl with simple hack
			if strings.Contains(hdr, "application/json") && strings.Contains(hdr, "as=Table") {
				newAccept = append(newAccept, "application/json")
				continue
			}

			newAccept = append(newAccept, hdr)
		}

		req.Header["Accept"] = newAccept

		// Force JSON for watches of core resources and CRDs.
		if targetReq.IsWatch() && (targetReq.IsCRD() || targetReq.IsCore()) {
			if len(req.Header.Values("Accept")) == 0 {
				req.Header["Accept"] = []string{"application/json"}
			}
		}
	}

	// Set new endpoint path and query.
	req.URL.Path = targetReq.Path()
	req.URL.RawQuery = targetReq.RawQuery()

	return origBodyBytes, rwrBodyBytes, nil
}

func (h *Handler) passResponse(ctx context.Context, targetReq *rewriter.TargetRequest, w http.ResponseWriter, resp *http.Response, logger *slog.Logger) {
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	bodyReader := resp.Body

	dst := &immediateWriter{dst: w}

	if logger.Enabled(nil, slog.LevelDebug) {
		if targetReq.IsWatch() {
			dst.chunkFn = func(chunk []byte) {
				logger.Debug(fmt.Sprintf("Pass through response chunk: %s", string(chunk)))
			}
		} else {
			bodyReader = logutil.NewReaderLogger(bodyReader)
		}
	}

	metrics := NewProxyMetrics(ctx, h.MetricsProvider)

	// Wrap body reader with bytes counter to set to_client_bytes metric.
	bytesCounterBody := BytesCounterReaderWrap(bodyReader)

	_, err := io.Copy(dst, bytesCounterBody)
	if err != nil {
		logger.Error(fmt.Sprintf("copy target response back to client: %v", err))
		metrics.RequestHandleError()
	} else {
		metrics.ToClientBytesAdd(CounterValue(bytesCounterBody))
		metrics.RequestHandleSuccess()
	}

	if logger.Enabled(nil, slog.LevelDebug) && !targetReq.IsWatch() {
		logutil.DebugBodyHead(logger,
			fmt.Sprintf("Pass through response: status %d, content-length: '%s'", resp.StatusCode, resp.Header.Get("Content-Length")),
			targetReq.ResourceForLog(),
			logutil.Bytes(bodyReader),
		)
	}

	return
}

// transformResponse rewrites payloads in responses from the target.
//
// ProxyMode field defines either rewriter should restore, or rename resources.
func (h *Handler) transformResponse(ctx context.Context, targetReq *rewriter.TargetRequest, w http.ResponseWriter, resp *http.Response, logger *slog.Logger) {
	metrics := NewProxyMetrics(ctx, h.MetricsProvider)

	var err error
	bytesCounter := BytesCounterReaderWrap(resp.Body)
	// Add gzip decoder if needed.
	bodyReader, err := encodingAwareReaderWrap(bytesCounter, resp.Header.Get("Content-Encoding"))
	if err != nil {
		logger.Error("Error decoding response body", logutil.SlogErr(err))
		http.Error(w, "can't decode response body", http.StatusInternalServerError)
		metrics.RequestHandleError()
		return
	}
	// Close needed for gzip and flate readers.
	defer bodyReader.Close()

	// Step 1. Read response body to buffer.
	origBodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		logger.Error("Error reading response payload", logutil.SlogErr(err))
		http.Error(w, "Error reading response payload", http.StatusBadGateway)
		metrics.RequestHandleError()
		return
	}

	metrics.FromTargetBytesAdd(CounterValue(bytesCounter))

	// Rewrite supports only json responses for now. Pass invalid JSON and non-JSON responses as-is.
	if !gjson.ValidBytes(origBodyBytes) {
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "application/json") {
			logger.Warn(fmt.Sprintf("Will not transform invalid JSON response from target: Content-type=%s", contentType))
		} else {
			logger.Warn(fmt.Sprintf("Will not transform non JSON response from target: Content-type=%s", contentType))
		}

		metrics.TargetResponseInvalidJSON(resp.StatusCode)

		h.passResponse(ctx, targetReq, w, resp, logger)
		return
	}

	// Step 2. Rewrite response JSON.
	rewriteStart := time.Now()
	statusCode := resp.StatusCode
	rwrBodyBytes, err := h.Rewriter.RewriteJSONPayload(targetReq, origBodyBytes, FromTargetAction(h.ProxyMode))
	if err != nil {
		if !errors.Is(err, rewriter.SkipItem) {
			logger.Error("Error rewriting response", logutil.SlogErr(err))
			http.Error(w, "can't rewrite response", http.StatusInternalServerError)
			metrics.RequestHandleError()
			metrics.TargetResponseRewriteError()
			return
		}
		// Return NotFound Status object if rewriter decides to skip resource.
		rwrBodyBytes = notFoundJSON(targetReq.OrigResourceType(), origBodyBytes)
		statusCode = http.StatusNotFound
	}
	metrics.TargetResponseRewriteSuccess()
	metrics.TargetResponseRewriteDuration(time.Since(rewriteStart))

	if targetReq.IsWebhook() {
		logutil.DebugBodyHead(logger, "Response from webhook", targetReq.ResourceForLog(), origBodyBytes)
	}
	logutil.DebugBodyChanges(logger, "Response", targetReq.ResourceForLog(), origBodyBytes, rwrBodyBytes)

	// Step 3. Fix headers before sending response back to the client.
	copyHeader(w.Header(), resp.Header)
	// Fix Content headers.
	// rwrBodyBytes are always decoded from gzip. Delete header to not break our client.
	w.Header().Del("Content-Encoding")
	if rwrBodyBytes != nil {
		w.Header().Set("Content-Length", strconv.Itoa(len(rwrBodyBytes)))
	}
	w.WriteHeader(statusCode)

	// Step 4. Write non-empty rewritten body to the client.
	if rwrBodyBytes != nil {
		copied, err := w.Write(rwrBodyBytes)
		if err != nil {
			logger.Error(fmt.Sprintf("error writing response from target to the client: %v", err))
			metrics.RequestHandleError()
		} else {
			metrics.RequestHandleSuccess()
			metrics.ToClientBytesAdd(copied)
		}
	}

	return
}

func (h *Handler) transformStream(ctx context.Context, targetReq *rewriter.TargetRequest, w http.ResponseWriter, resp *http.Response, logger *slog.Logger) {
	// Rewrite body as a stream. ServeHTTP will block until context cancel.
	err := h.streamHandler.Handle(ctx, w, resp, targetReq)
	if err != nil {
		logger.Error("Error watching stream", logutil.SlogErr(err))
		http.Error(w, fmt.Sprintf("watch stream: %v", err), http.StatusInternalServerError)
	}
}

type immediateWriter struct {
	dst     io.Writer
	chunkFn func([]byte)
}

func (iw *immediateWriter) Write(p []byte) (n int, err error) {
	n, err = iw.dst.Write(p)

	if iw.chunkFn != nil {
		iw.chunkFn(p)
	}

	if flusher, ok := iw.dst.(http.Flusher); ok {
		flusher.Flush()
	}

	return
}

// notFoundJSON constructs Status response of type NotFound
// for resourceType and object name.
// Example:
//
//	{
//		"kind":"Status",
//		"apiVersion":"v1",
//		"metadata":{},
//		"status":"Failure",
//		"message":"pods \"vmi-router-x9mqwdqwd\" not found",
//		"reason":"NotFound",
//		"details":{"name":"vmi-router-x9mqwdqwd","kind":"pods"},
//		"code":404}
func notFoundJSON(resourceType string, obj []byte) []byte {
	objName := gjson.GetBytes(obj, "metadata.name").String()
	details := fmt.Sprintf(`"details":{"name":"%s","kind":"%s"}`, objName, resourceType)
	message := fmt.Sprintf(`"message":"%s %s not found"`, resourceType, objName)
	notFoundTpl := `{"kind":"Status","apiVersion":"v1",%s,%s,"metadata":{},"status":"Failure","reason":"NotFound","code":404}`
	return []byte(fmt.Sprintf(notFoundTpl, message, details))
}
