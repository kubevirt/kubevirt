// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

/*
Package cors contains filters to handle CORS related requests defined from
http://www.w3.org/TR/cors/
*/
package cors

import (
	"net/http"
	"strconv"
	"time"
)

// DefaultAllowOrigin sets Access-Control-Allow-Origin when not
// configured.
const DefaultAllowOrigin = "*"

// Config parameterizes CORS behavior.
type Config struct {
	// AllowOrigin transforms a request into the Access-Control-Allow-Origin
	// header, default is full access "*".
	AllowOrigin func(*http.Request) string
}

// Middleware returns a middleware that applies Config to the request.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	const maxAge = 10 * time.Minute
	age := strconv.Itoa(int(maxAge / time.Second))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := DefaultAllowOrigin
			if cfg.AllowOrigin != nil {
				origin = cfg.AllowOrigin(r)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Encoding, Authorization, Content-Type, Origin")
			w.Header().Set("Access-Control-Allow-Origin", origin)

			switch r.Method {
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			case "OPTIONS":
				if r.Header.Get("Access-Control-Request-Method") == "GET" {
					w.Header().Set("Access-Control-Max-Age", age)
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
			case "HEAD", "GET":
				next.ServeHTTP(w, r)
			}
		})
	}
}

// Get implements a simple read-only access control policy handling preflight
// and normal requests with a cache age of 10 minutes for preflight requests.
// Methods other than HEAD, OPTIONS, GET will return 405.
//
// The origin parameter should be the case-insentive fully qualified origin
// domain to match or '*' to match any domain.
func Get(origin string, next http.Handler) http.Handler {
	return Middleware(Config{
		AllowOrigin: func(*http.Request) string { return origin },
	})(next)
}
