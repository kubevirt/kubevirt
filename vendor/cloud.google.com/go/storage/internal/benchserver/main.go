// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main pretends to be the storage backend for the sake of benchmarking.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storage/v1"
)

func main() {
	// Serves the read from an object.
	http.HandleFunc("/some-bucket-name/some-object-name", func(resp http.ResponseWriter, req *http.Request) {
	})
	// Serves the write to an object.
	http.HandleFunc("/b/some-bucket-name/o/", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "application/json")
		ret := &storage.Object{
			ServerResponse: googleapi.ServerResponse{
				Header:         resp.Header(),
				HTTPStatusCode: http.StatusCreated,
			},
		}
		if err := json.NewEncoder(resp).Encode(ret); err != nil {
			log.Fatal(err)
		}
	})
	log.Println("listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
