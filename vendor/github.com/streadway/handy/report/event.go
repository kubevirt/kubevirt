// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

/*
Package report organizes textual reporting from the HTTP context.
*/
package report

import (
	"net/http"
	"time"
)

// Event contains significant fields from the request or response to report
type Event struct {
	Time           time.Time `json:"time,omitempty"`
	Method         string    `json:"method,omitempty"`
	Url            string    `json:"url,omitempty"`
	Path           string    `json:"path,omitempty"`
	Proto          string    `json:"proto,omitempty"`
	Status         int       `json:"status,omitempty"`
	Ms             int       `json:"ms"`
	Size           int64     `json:"size"`
	RemoteAddr     string    `json:"remote_addr,omitempty"`
	ForwardedFor   string    `json:"forwarded_for,omitempty"`
	ForwardedProto string    `json:"forwarded_proto,omitempty"`
	Range          string    `json:"range,omitempty"`
	Host           string    `json:"host,omitempty"`
	Referrer       string    `json:"referrer,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	Authorization  string    `json:"authorization,omitempty"`
	Region         string    `json:"region,omitempty"`
	Country        string    `json:"country,omitempty"`
	City           string    `json:"city,omitempty"`
	RequestId      string    `json:"request_id,omitempty"`
}

type eventRecorder struct {
	http.ResponseWriter
	event Event
}

// Write sums the writes to produce the actual number of bytes written
func (e *eventRecorder) Write(b []byte) (int, error) {
	n, err := e.ResponseWriter.Write(b)
	e.event.Size += int64(n)
	return n, err
}

// WriteHeader captures the status code.  On success, this method may not be
// called so initialize your event struct with the status value you wish to
// report on success,like 200.
func (e *eventRecorder) WriteHeader(code int) {
	e.event.Status = code
	e.ResponseWriter.WriteHeader(code)
}
