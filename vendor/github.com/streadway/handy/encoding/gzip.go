// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

/*
Package encoding contains Content-Encoding related filters.
*/
package encoding

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

type gzipWriter struct {
	http.ResponseWriter
	sync.Mutex
	io.WriteCloser
	err   error
	level int
	types []string
}

func (w gzipWriter) canGzip() bool {
	if len(w.types) == 0 {
		return true
	}

	contentType := w.Header().Get("Content-Type")
	for _, mediaType := range w.types {
		if strings.Contains(contentType, mediaType) {
			return true
		}
	}

	return false
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	if err := w.init(); err != nil {
		return 0, err
	}
	return w.WriteCloser.Write(b)
}

func (w *gzipWriter) WriteHeader(code int) {
	_ = w.init() // delay error propagation to Write and Close calls
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipWriter) init() error {
	w.Lock()
	defer w.Unlock()
	if w.WriteCloser != nil || w.err != nil { // fast path
		return w.err
	}

	if w.canGzip() {
		w.WriteCloser, w.err = gzip.NewWriterLevel(w.ResponseWriter, w.level)
		if w.err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")
		}
	} else {
		w.WriteCloser = nopCloser{w.ResponseWriter}
	}

	return w.err
}

func (w *gzipWriter) Close() error {
	w.Lock()
	defer w.Unlock()

	switch {
	case w.err != nil:
		return w.err
	case w.WriteCloser != nil:
		return w.WriteCloser.Close()
	}
	return nil
}

// Gzip calls the next handler with a response writer that will compress the
// outbound writes with the default compression level. This filter assumes a
// chunked transfer encoding, so do not add a Content-Length header in the
// terminal handler.
//
// If the request does not accept a gzip encoding, this filter has no effect.
func Gzip(next http.Handler) http.Handler {
	return Gzipper(gzip.DefaultCompression)(next)
}

// GzipTypes sets the gzips the response if the the request Accept-Encoding
// contains 'gzip' and the response 'Content-Type' contains one of the mediaTypes.
// When no or nil mediaTypes are provided, all content types will be gzip encoded.
func GzipTypes(mediaTypes []string, next http.Handler) http.Handler {
	return Gzipper(gzip.DefaultCompression, mediaTypes...)(next)
}

// Gzipper returns a composable middleware function that wraps a given
// http.Handler with outbound Gzip compression using the provided level and
// optional accepted media types.
func Gzipper(level int, mediaTypes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}
			zipper := gzipWriter{level: level, types: mediaTypes, ResponseWriter: w}
			defer zipper.Close()
			next.ServeHTTP(&zipper, r)
		})
	}
}
