// Copyright (c) 2015, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

/*
Package accept contains filters to reject requests without a specified Accept
header with "406 Not Acceptable".
*/
package accept

import (
	"mime"
	"net/http"
	"strings"
)

const (
	// ALL matches all media types.
	ALL = "*/*"
)

var (
	// EventStream matches media typs used for SSE events.
	EventStream = Middleware("text/event-stream", "text/*")

	// HTML matches media typs used for HTML encoded resources.
	HTML = Middleware("text/html")

	// JSON matches media typs used for JSON encoded resources.
	JSON = Middleware("application/json", "application/javascript")

	// Plain matches media typs used for plaintext resources.
	Plain = Middleware("text/plain")

	// XML matches media typs used for XML encoded resources.
	XML = Middleware("application/xhtml+xml", "application/xml")
)

// Middleware returns a composable handler factory to restrict accepted
// media types and respond with "406 Not Acceptable" otherwise.
func Middleware(mediaTypes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !acceptable(r.Header.Get("Accept"), mediaTypes) {
				w.WriteHeader(http.StatusNotAcceptable)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func acceptable(accept string, mediaTypes []string) bool {
	if accept == "" || accept == ALL {
		// The absense of an Accept header is equivalent to "*/*".
		// https://tools.ietf.org/html/rfc2296#section-4.2.2
		return true
	}

	for _, a := range strings.Split(accept, ",") {
		mediaType, _, err := mime.ParseMediaType(a)
		if err != nil {
			continue
		}

		if mediaType == ALL {
			return true
		}

		for _, t := range mediaTypes {
			if mediaType == t {
				return true
			}
		}
	}

	return false
}
