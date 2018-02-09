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

// MaxPacketLen is the number of bytes filled before a packet is flushed before the reporting interval.
const maxPacketLen = 2 ^ 15

func flush(w io.Writer, buf *bytes.Buffer) {
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("error: could not write to statsd: %v", err)
	}
	buf.Reset()
}

func reportDurations(w io.Writer, key string, interval <-chan time.Time, samples <-chan time.Duration) {
	msg := &bytes.Buffer{}

	for {
		select {
		case sample := <-samples:
			if msg.Len() == 0 {
				msg.Write([]byte(key))
			}

			fmt.Fprintf(msg, ":%d|ms", int(sample/time.Millisecond))

			if msg.Len() > maxPacketLen {
				flush(w, msg)
			}

		case <-interval:
			if msg.Len() > 0 {
				flush(w, msg)
			}
		}
	}
}

// Durations writes a statsd formatted packet to the io.Writer with a list of
// durations recorded for each reporting interval or until a packet is filled.
func Durations(statsd io.Writer, key string, interval time.Duration, next http.Handler) http.Handler {
	// buffered - reporting is concurrent with the handler
	durations := make(chan time.Duration, 1)
	go reportDurations(statsd, key, time.Tick(interval), durations)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		durations <- time.Now().Sub(start)
	})
}
