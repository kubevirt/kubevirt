/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package virtexportserver

import (
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func successHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}

func newTestServer(token string) *exportServer {
	config := ExportServerConfig{
		ArchiveHandler: func(string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		DirHandler: func(string, string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		FileHandler: func(string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		GzipHandler: func(string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		TokenGetter: func() (string, error) {
			return token, nil
		},
	}
	s := NewExportServer(config)
	return s.(*exportServer)
}

var _ = Describe("exportserver", func() {
	DescribeTable("should handle", func(vi VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Volumes = []VolumeInfo{vi}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri, nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("x-kubevirt-export-token", token)
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(Equal("OK"))
	},
		Entry("archive URI",
			VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
	)

	DescribeTable("should handle (query param version)", func(vi VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Volumes = []VolumeInfo{vi}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri+"?x-kubevirt-export-token="+token, nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(Equal("OK"))
	},
		Entry("archive URI",
			VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
	)

	DescribeTable("should fail bad token", func(vi VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Volumes = []VolumeInfo{vi}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri, nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("x-kubevirt-export-token", "bar")
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
	},
		Entry("archive URI",
			VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
	)

	DescribeTable("should fail bad token (query param version)", func(vi VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Volumes = []VolumeInfo{vi}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri+"?x-kubevirt-export-token=bar", nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
	},
		Entry("archive URI",
			VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
	)

})
