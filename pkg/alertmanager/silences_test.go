package alertmanager_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/alertmanager"
)

const listResp = `[{"id":"bb881d7f-3278-46fd-a638-d42c57f235b6","status":{"state":"active"},"updatedAt":"2024-07-16T11:46:30.653Z","comment":"test purposes","createdBy":"test_user","endsAt":"3000-01-01T00:00:00.000Z","matchers":[{"isEqual":true,"isRegex":false,"name":"alertname","value":"TestAlert"}],"startsAt":"2024-07-16T11:46:30.653Z"}]`

var _ = Describe("Silences", func() {
	var (
		ts  *httptest.Server
		api *alertmanager.Api
	)

	BeforeEach(func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/v2/silence") {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			switch r.Method {
			case http.MethodGet:
				fmt.Fprintln(w, listResp)
			case http.MethodPost:
				w.WriteHeader(http.StatusOK)
			case http.MethodDelete:
				w.WriteHeader(http.StatusOK)
			}
		}))

		api = alertmanager.NewAPI(http.Client{}, ts.URL, "token")
	})

	AfterEach(func() {
		ts.Close()
	})

	It("should successfully GET /api/v2/silences", func() {
		silences, err := api.ListSilences()
		Expect(err).ToNot(HaveOccurred())
		Expect(silences).To(HaveLen(1))

		Expect(silences[0].Status.State).To(Equal("active"))
		Expect(silences[0].EndsAt).To(Equal("3000-01-01T00:00:00.000Z"))

		Expect(silences[0].Matchers).To(HaveLen(1))
		Expect(silences[0].Matchers[0].Name).To(Equal("alertname"))
		Expect(silences[0].Matchers[0].Value).To(Equal("TestAlert"))
	})

	It("should successfully POST /api/v2/silences", func() {
		err := api.CreateSilence(alertmanager.Silence{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should successfully DELETE /api/v2/silences/{id}", func() {
		err := api.DeleteSilence("bb881d7f-3278-46fd-a638-d42c57f235b6")
		Expect(err).ToNot(HaveOccurred())
	})
})
