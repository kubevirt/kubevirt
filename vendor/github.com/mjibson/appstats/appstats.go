/*
 * Copyright (c) 2013 Matt Jibson <matt.jibson@gmail.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package appstats

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/user"
)

var (
	// RecordFraction is the fraction of requests to record.
	// Set to a number between 0.0 (none) and 1.0 (all).
	RecordFraction float64 = 1.0

	// ShouldRecord is the function used to determine if recording will occur
	// for a given request. The default is to use RecordFraction.
	ShouldRecord = DefaultShouldRecord

	// ProtoMaxBytes is the amount of protobuf data to record.
	// Data after this is truncated.
	ProtoMaxBytes = 150

	// Namespace is the memcache namespace under which to store appstats data.
	Namespace = "__appstats__"
)

const (
	serveURL   = "/_ah/stats/"
	detailsURL = serveURL + "details"
	fileURL    = serveURL + "file"
	staticURL  = serveURL + "static/"
)

const (
	statsKey  = "appstats stats"
	headerKey = "appstats header"
)

func init() {
	http.HandleFunc(serveURL, appstatsHandler)
}

// DefaultShouldRecord will record a request based on RecordFraction.
func DefaultShouldRecord(r *http.Request) bool {
	if RecordFraction >= 1.0 {
		return true
	}

	return rand.Float64() < RecordFraction
}

func stats(ctx context.Context) *requestStats {
	return ctx.Value(statsKey).(*requestStats)
}

// header will return the HTTP headers associated with the given
// context. If there are no headers associated, it will return nil.
func header(ctx context.Context) http.Header {
	h, _ := ctx.Value(headerKey).(http.Header)
	return h
}

func override(ctx context.Context, service, method string, in, out proto.Message) error {
	stats := stats(ctx)

	stats.wg.Add(1)
	defer stats.wg.Done()

	if service == "__go__" {
		return appengine.APICall(ctx, service, method, in, out)
	}

	stat := rpcStat{
		Service:   service,
		Method:    method,
		Start:     time.Now(),
		Offset:    time.Since(stats.Start),
		StackData: string(debug.Stack()),
	}
	err := appengine.APICall(ctx, service, method, in, out)
	stat.Duration = time.Since(stat.Start)
	stat.In = in.String()
	stat.Out = out.String()
	stat.Cost = getCost(out)

	if len(stat.In) > ProtoMaxBytes {
		stat.In = stat.In[:ProtoMaxBytes] + "..."
	}
	if len(stat.Out) > ProtoMaxBytes {
		stat.Out = stat.Out[:ProtoMaxBytes] + "..."
	}

	stats.lock.Lock()
	stats.RPCStats = append(stats.RPCStats, stat)
	stats.Cost += stat.Cost
	stats.lock.Unlock()
	return err
}

// newContext creates a new timing-aware context from req.
func newContext(r *http.Request) context.Context {
	ctx := appengine.NewContext(r)

	stats := &requestStats{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
		Start:  time.Now(),
	}

	if u := user.Current(ctx); u != nil {
		stats.User = u.String()
		stats.Admin = u.Admin
	}

	ctx = context.WithValue(ctx, statsKey, stats)
	ctx = context.WithValue(ctx, headerKey, r.Header)
	ctx = appengine.WithAPICallFunc(ctx, override)

	return ctx
}

// WithContext enables profiling of functions without a corresponding request,
// as in the appengine/delay package. method and path may be empty.
func WithContext(ctx context.Context, method, path string, f func(context.Context)) {
	stats := &requestStats{
		Method: method,
		Path:   path,
		Start:  time.Now(),
	}

	if u := user.Current(ctx); u != nil {
		stats.User = u.String()
		stats.Admin = u.Admin
	}

	ctx = context.WithValue(ctx, statsKey, stats)
	ctx = appengine.WithAPICallFunc(ctx, override)

	f(ctx)
	save(ctx)
}

const bufMaxLen = 1000000

func save(ctx context.Context) {
	stats := stats(ctx)
	stats.wg.Wait()
	stats.Duration = time.Since(stats.Start)

	var buf_part, buf_full bytes.Buffer
	full := stats_full{
		Header: header(ctx),
		Stats:  stats,
	}
	if err := gob.NewEncoder(&buf_full).Encode(&full); err != nil {
		log.Errorf(ctx, "appstats Save error: %v", err)
		return
	} else if buf_full.Len() > bufMaxLen {
		// first try clearing stack traces
		for i := range full.Stats.RPCStats {
			full.Stats.RPCStats[i].StackData = ""
		}
		buf_full.Truncate(0)
		gob.NewEncoder(&buf_full).Encode(&full)
	}
	part := stats_part(*stats)
	for i := range part.RPCStats {
		part.RPCStats[i].StackData = ""
		part.RPCStats[i].In = ""
		part.RPCStats[i].Out = ""
	}
	if err := gob.NewEncoder(&buf_part).Encode(&part); err != nil {
		log.Errorf(ctx, "appstats Save error: %v", err)
		return
	}

	item_part := &memcache.Item{
		Key:   stats.PartKey(),
		Value: buf_part.Bytes(),
	}

	item_full := &memcache.Item{
		Key:   stats.FullKey(),
		Value: buf_full.Bytes(),
	}

	log.Infof(ctx, "Saved; %s: %s, %s: %s, link: %v",
		item_part.Key,
		byteSize(len(item_part.Value)),
		item_full.Key,
		byteSize(len(item_full.Value)),
		URL(ctx),
	)

	nc := storeContext(ctx)
	memcache.SetMulti(nc, []*memcache.Item{item_part, item_full})
}

// URL returns the appstats URL for the current request.
func URL(ctx context.Context) string {
	stats := stats(ctx)
	u := url.URL{
		Path:     detailsURL,
		RawQuery: fmt.Sprintf("time=%v", stats.Start.Nanosecond()),
	}
	return u.String()
}

func storeContext(ctx context.Context) context.Context {
	nc, _ := appengine.Namespace(ctx, Namespace)
	return nc
}

// handler is an http.Handler that records RPC statistics.
type handler struct {
	f func(context.Context, http.ResponseWriter, *http.Request)
}

// NewHandler returns a new Handler that will execute f.
func NewHandler(f func(context.Context, http.ResponseWriter, *http.Request)) http.Handler {
	return handler{
		f: f,
	}
}

// NewHandlerFunc returns a new HandlerFunc that will execute f.
func NewHandlerFunc(f func(context.Context, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := handler{
			f: f,
		}
		h.ServeHTTP(w, r)
	}
}

type responseWriter struct {
	http.ResponseWriter

	stats *requestStats
}

func (r responseWriter) Write(b []byte) (int, error) {
	// Emulate the behavior of http.ResponseWriter.Write since it doesn't
	// call our WriteHeader implementation.
	if r.stats.Status == 0 {
		r.WriteHeader(http.StatusOK)
	}

	return r.ResponseWriter.Write(b)
}

func (r responseWriter) WriteHeader(i int) {
	r.stats.Status = i
	r.ResponseWriter.WriteHeader(i)
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ShouldRecord(r) {
		ctx := newContext(r)
		rw := responseWriter{
			ResponseWriter: w,
			stats:          stats(ctx),
		}
		h.f(ctx, rw, r)
		save(ctx)
	} else {
		c := appengine.NewContext(r)
		h.f(c, w, r)
	}
}
