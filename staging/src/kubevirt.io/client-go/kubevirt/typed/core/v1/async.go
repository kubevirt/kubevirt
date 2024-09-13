package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/gorilla/websocket"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rest "k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/subresources"
)

type StreamOptions struct {
	In  io.Reader
	Out io.Writer
}
type StreamInterface interface {
	Stream(options StreamOptions) error
	AsConn() net.Conn
}

type AsyncSubresourceError struct {
	err        string
	StatusCode int
}

func (a *AsyncSubresourceError) Error() string {
	return a.err
}

func (a *AsyncSubresourceError) GetStatusCode() int {
	return a.StatusCode
}

// params are strings with "key=value" format
func AsyncSubresourceHelper(config *rest.Config, resource, namespace, name string, subresource string, queryParams url.Values) (StreamInterface, error) {

	done := make(chan struct{})

	aws := &AsyncWSRoundTripper{
		Connection: make(chan *websocket.Conn),
		Done:       done,
	}
	// Create a round tripper with all necessary kubernetes security details
	wrappedRoundTripper, err := roundTripperFromConfig(config, aws.WebsocketCallback)
	if err != nil {
		return nil, fmt.Errorf("unable to create round tripper for remote execution: %v", err)
	}

	// Create a request out of config and the query parameters
	req, err := RequestFromConfig(config, resource, name, namespace, subresource, queryParams)
	if err != nil {
		return nil, fmt.Errorf("unable to create request for remote execution: %v", err)
	}

	errChan := make(chan error, 1)

	go func() {
		// Send the request and let the callback do its work
		response, err := wrappedRoundTripper.RoundTrip(req)

		if err != nil {
			statusCode := 0
			if response != nil {
				statusCode = response.StatusCode
			}
			errChan <- &AsyncSubresourceError{err: err.Error(), StatusCode: statusCode}
			return
		}

		if response != nil {
			switch response.StatusCode {
			case http.StatusOK:
			case http.StatusNotFound:
				err = &AsyncSubresourceError{err: "Virtual Machine not found.", StatusCode: response.StatusCode}
			case http.StatusInternalServerError:
				err = &AsyncSubresourceError{err: "Websocket failed due to internal server error.", StatusCode: response.StatusCode}
			default:
				err = &AsyncSubresourceError{err: fmt.Sprintf("Websocket failed with http status: %s", response.Status), StatusCode: response.StatusCode}
			}
		} else {
			err = &AsyncSubresourceError{err: "no response received"}
		}
		errChan <- err
	}()

	select {
	case err = <-errChan:
		return nil, err
	case ws := <-aws.Connection:
		return &wsStreamer{
			conn: ws,
			done: done,
		}, nil
	}
}

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer conn.Close()
	}
	return resp, d.Do(conn, resp, err)
}

type AsyncWSRoundTripper struct {
	Done       chan struct{}
	Connection chan *websocket.Conn
}

func (aws *AsyncWSRoundTripper) WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error) error {

	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			return enrichError(err, resp)
		}
		return fmt.Errorf("Can't connect to websocket: %s\n", err.Error())
	}
	aws.Connection <- ws

	// Keep the roundtripper open until we are done with the stream
	<-aws.Done
	return nil
}

func roundTripperFromConfig(config *rest.Config, callback RoundTripCallback) (http.RoundTripper, error) {

	// Configure TLS
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}

	// Configure the websocket dialer
	proxy := http.ProxyFromEnvironment
	if config.Proxy != nil {
		proxy = config.Proxy
	}
	dialer := &websocket.Dialer{
		Proxy:           proxy,
		TLSClientConfig: tlsConfig,
		WriteBufferSize: WebsocketMessageBufferSize,
		ReadBufferSize:  WebsocketMessageBufferSize,
		Subprotocols:    []string{subresources.PlainStreamProtocolName},
	}

	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	rt := &WebsocketRoundTripper{
		Do:     callback,
		Dialer: dialer,
	}

	// Make sure we inherit all relevant security headers
	return rest.HTTPWrappersForConfig(config, rt)
}

func RequestFromConfig(config *rest.Config, resource, name, namespace, subresource string, queryParams url.Values) (*http.Request, error) {

	u, err := url.Parse(config.Host)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, fmt.Errorf("Unsupported Protocol %s", u.Scheme)
	}

	u.Path = path.Join(
		u.Path,
		fmt.Sprintf("/apis/subresources.kubevirt.io/%s/namespaces/%s/%s/%s/%s", v1.ApiStorageVersion, namespace, resource, name, subresource),
	)
	if len(queryParams) > 0 {
		u.RawQuery = queryParams.Encode()
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: map[string][]string{},
	}

	return req, nil
}

// enrichError checks the response body for a k8s Status object and extracts the error from it.
// TODO the k8s http REST client has very sophisticated handling, investigate on how we can reuse it
func enrichError(httpErr error, resp *http.Response) error {
	if resp == nil {
		return httpErr
	}
	httpErr = fmt.Errorf("Can't connect to websocket (%d): %s\n", resp.StatusCode, httpErr)
	status := &metav1.Status{}

	if resp.Header.Get("Content-Type") != "application/json" {
		return httpErr
	}
	// decode, but if the result is Status return that as an error instead.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return httpErr
	}
	err = json.Unmarshal(body, status)
	if err != nil {
		return err
	}
	if status.Kind == "Status" && status.APIVersion == "v1" {
		if status.Status != metav1.StatusSuccess {
			return errors.FromObject(status)
		}
	}
	return httpErr
}
