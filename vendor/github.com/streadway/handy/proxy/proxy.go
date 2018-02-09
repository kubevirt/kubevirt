// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

/*
Package proxy contains a proxying HTTP transport.
*/
package proxy

import (
	"net/http"
	"net/url"
)

// Transport is an implementation of the http.RoundTripper that uses a user
// supplied generator function to proxy requests to specific destinations.
type Transport struct {
	// Proxy takes an http.Request and provides a URL to use for that request.
	// Note that the semantics are different from http.DefaultTransport: this
	// proxy is always invoked. If Proxy is nil, requests to the Transport are
	// unaltered.
	Proxy func(*http.Request) (*url.URL, error)

	// Next is the http.RoundTripper to which requests are forwarded.  If Next
	// is nil, http.DefaultTransport is used.
	Next http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Proxy != nil {
		url, err := t.Proxy(req)
		if err != nil {
			return nil, err
		}
		req.URL = url
	}
	if t.Next != nil {
		return t.Next.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
