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

/*
Package appstats profiles the RPC performance of Google App Engine applications.

Reference: https://developers.google.com/appengine/docs/python/tools/appstats

To use this package, change your HTTP handler functions to use this signature:

	func(appengine.Context, http.ResponseWriter, *http.Request)

Register them in the usual way, wrapping them with NewHandler.

Classic App Engine packages are available at https://godoc.org/gopkg.in/mjibson/v1/appstats.


Example

This is a small example using this package.

	package main

	import (
		"net/http"

		"github.com/mjibson/appstats"
		"golang.org/x/net/context"
	)

	func init() {
		http.Handle("/", appstats.NewHandler(Main))
	}

	func Main(c context.Context, w http.ResponseWriter, r *http.Request) {
		// do stuff with c: datastore.Get(c, key, entity)
		w.Write([]byte("success"))
	}


Usage

Use your app, and view the appstats interface at http://localhost:8080/_ah/stats/, or your production URL.


Configuration

Refer to the variables section of the documentation: http://godoc.org/github.com/mjibson/appstats#pkg-variables.


Routing

In general, your app.yaml will not need to change. In the case of conflicting
routes, add the following to your app.yaml:

	handlers:
	- url: /_ah/stats/.*
	  script: _go_app


TODO

Cost calculation is experimental. Currently it only includes write ops (read and small ops are TODO).
*/
package appstats
