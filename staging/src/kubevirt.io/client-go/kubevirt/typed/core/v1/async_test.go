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
 * Copyright The KubeVirt Authors
 *
 */

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
)

const (
	errMsgDefault   = "Can't connect to websocket"
	errMsgValidJson = "You shall not pass"
	errMsgTextPlain = "This works now"

	errJsonUnsupported = `{"custom":"data","doesnt":"work"}`
	errJsonValid       = `{"kind":"Status","apiVersion":"v1","status":"Failure","message":"` +
		errMsgValidJson + `"}`
)

var _ = ginkgo.Describe("async", func() {
	curErr := fmt.Errorf("Just a startup error to compare")
	newResp := func(body []byte) *http.Response {
		if body == nil {
			return nil
		}
		resp := &http.Response{
			Header: http.Header{},
			Body:   ioutil.NopCloser(bytes.NewReader(body)),
		}
		if len(body) < 2 {
			return resp
		}
		// Find content-type on the fly
		switch {
		case json.Unmarshal(body, new(interface{})) == nil:
			resp.Header.Set("Content-Type", "application/json")
		default:
			resp.Header.Set("Content-Type", "text/plain")
		}
		ginkgo.GinkgoWriter.Printf("resp: %+v", resp)
		return resp
	}

	ginkgo.DescribeTable("should enrich error with reponse body when possible",
		func(body []byte, prevErr error, expectErrMsg string) {
			resp := newResp(body)
			err := EnrichError(prevErr, resp)
			gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(expectErrMsg)))
		},
		// Each entry relates to the first argument: Response body
		ginkgo.Entry("body is nil", nil, curErr, curErr.Error()),
		ginkgo.Entry("body is empty", []byte(""), curErr, errMsgDefault),
		ginkgo.Entry("body has valid json Status", []byte(errJsonValid), curErr, errMsgValidJson),
		ginkgo.Entry("body has invalid json Status", []byte(errJsonUnsupported), curErr, errMsgDefault),
		ginkgo.Entry("body has text plain", []byte(errMsgTextPlain), curErr, errMsgTextPlain),
	)
})
