// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

/*
Package rewrite contains filters to handle HTTP rewrites
*/
package rewrite

import (
	"net/http"
)

// Method modifies the http.Request.Method for POST requests to the form value
// "_method" only if that value is one of PUT, PATCH or DELETE.
func Method(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			switch _method := r.FormValue("_method"); _method {
			case "PUT", "PATCH", "DELETE":
				r.Method = _method
			}
		}
		next.ServeHTTP(w, r)
	})
}
