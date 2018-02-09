// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

package report

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// JSONMiddleware returns a composable handler factory implementing the JSON
// handler.
func JSONMiddleware(writer io.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		var mu sync.Mutex // serializes encodings
		out := json.NewEncoder(writer)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := &eventRecorder{
				ResponseWriter: w,
				event: Event{
					// Size & Status possiblly overwritten by the ResponseWriter interface
					Status:         200,
					Time:           time.Now().UTC(),
					Method:         r.Method,
					Url:            r.RequestURI,
					Path:           r.URL.Path,
					Proto:          r.Proto,
					Host:           r.Host,
					RemoteAddr:     r.RemoteAddr,
					ForwardedFor:   r.Header.Get("X-Forwarded-For"),
					ForwardedProto: r.Header.Get("X-Forwarded-Proto"),
					Authorization:  r.Header.Get("Authorization"),
					Referrer:       r.Header.Get("Referer"),
					UserAgent:      r.Header.Get("User-Agent"),
					Range:          r.Header.Get("Range"),
					RequestId:      r.Header.Get("X-Request-Id"),
					Region:         r.Header.Get("X-Region"),
					Country:        r.Header.Get("X-Country"),
					City:           r.Header.Get("X-City"),
				},
			}

			start := time.Now()

			next.ServeHTTP(writer, r)

			writer.event.Ms = int(time.Since(start) / time.Millisecond)

			mu.Lock()
			out.Encode(writer.event)
			mu.Unlock()
		})
	}
}

// JSON writes a JSON encoded Event to the provided writer at the
// completion of each request
func JSON(writer io.Writer, next http.Handler) http.Handler {
	return JSONMiddleware(writer)(next)
}
