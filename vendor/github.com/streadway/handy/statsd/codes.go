// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

package statsd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// capture records the response code returned through the embedded
// ResponseWriter from the WriteHeader call.
type capture struct {
	http.ResponseWriter
	code int
}

// WriteHeader captures the returned code and delegates
func (w *capture) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func reportCodes(statsd io.Writer, key string, report <-chan time.Time, codes <-chan int) {
	hist := make(map[int]int)
	msg := &bytes.Buffer{}

	for {
		select {
		case sample := <-codes:
			hist[sample] = hist[sample] + 1

		case <-report:
			if len(hist) > 0 {
				for code, count := range hist {
					fmt.Fprintf(msg, "%s.%d:%d|c\n", key, code, count)
				}

				if _, err := statsd.Write(msg.Bytes()); err != nil {
					log.Printf("error: could not write to statsd: %v", err)
				}

				hist = make(map[int]int)
				msg.Reset()
			}
		}
	}
}

// Codes collects and reports the counts of response codes for the handler chain for all requests.
func Codes(statsd io.Writer, key string, interval time.Duration, next http.Handler) http.Handler {
	responses := make(chan int)
	go reportCodes(statsd, key, time.Tick(interval), responses)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// init with 200 so no call to WriteHeader is a success
		collector := &capture{w, 200}
		next.ServeHTTP(collector, r)
		responses <- collector.code
	})
}
