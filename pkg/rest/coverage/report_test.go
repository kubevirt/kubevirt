package coverage

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"net/url"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	AuditV1 "k8s.io/apiserver/pkg/apis/audit/v1"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("REST API coverage report", func() {

	Context("With pets audit log", func() {

		Context("With reading an audit file", func() {

			var tmpFile string

			BeforeEach(func() {
				f, err := ioutil.TempFile("/tmp", "report")
				Expect(err).NotTo(HaveOccurred())
				tmpFile = f.Name()
			})

			AfterEach(func() {
				err := os.Remove(tmpFile)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should generate a coverage report", func() {
				var stats map[string]float64

				By("Reading audit log file")
				auditLogs, err := ioutil.ReadFile(auditLogPath)
				Expect(err).NotTo(HaveOccurred())

				By("Generating a coverage report")
				err = GenerateReport(string(auditLogs), petStoreSwaggerPath, "", tmpFile, true)
				Expect(err).NotTo(HaveOccurred())

				By("Checking generated report")
				data, err := ioutil.ReadFile(tmpFile)
				Expect(err).NotTo(HaveOccurred())

				err = json.Unmarshal(data, &stats)
				Expect(err).NotTo(HaveOccurred())
				Expect(int(stats["total"])).To(Equal(44))
			})
		})

		table.DescribeTable("Should return correct swagger path based on audit URL", func(URI string, objRef *AuditV1.ObjectReference, swaggerPath string) {
			path := getSwaggerPath(URI, objRef)
			Expect(path).To(Equal(swaggerPath))
		},
			table.Entry(
				"With an empty namespace",
				"/pets/bite",
				&AuditV1.ObjectReference{Name: "bite"},
				"/pets/{name}",
			),
			table.Entry(
				"With defined namespace",
				"/pets/namespace/default/bite",
				&AuditV1.ObjectReference{Name: "bite"},
				"/pets/namespace/default/{name}",
			),
			table.Entry(
				"With empty object reference",
				"/pets",
				&AuditV1.ObjectReference{},
				"/pets",
			),
			table.Entry(
				"With VMI list request",
				"/apis/kubevirt.io/v1alpha3/namespaces/kubevirt-test-default/virtualmachineinstances",
				&AuditV1.ObjectReference{
					Resource:  "virtualmachineinstances",
					Namespace: "kubevirt-test-default",
				},
				"/apis/kubevirt.io/v1alpha3/namespaces/{namespace}/virtualmachineinstances",
			),
			table.Entry(
				"With VMI get request",
				"/apis/kubevirt.io/v1alpha3/namespaces/kubevirt/virtualmachineinstances/testvmi22gsnklt2flhqflcnp8jpmq6fkj72szv8h9sn26z2hdhkm6l",
				&AuditV1.ObjectReference{
					Resource:  "virtualmachineinstances",
					Namespace: "kubevirt",
					Name:      "testvmi22gsnklt2flhqflcnp8jpmq6fkj72szv8h9sn26z2hdhkm6l",
				},
				"/apis/kubevirt.io/v1alpha3/namespaces/{namespace}/virtualmachineinstances/{name}",
			),
		)

		table.DescribeTable("Should translate k8s verb to HTTP method", func(verb string, httpMethod string) {
			Expect(getHTTPMethod(verb)).To(Equal(httpMethod))
		},
			table.Entry("With get verb", "get", "GET"),
			table.Entry("With list verb", "list", "GET"),
			table.Entry("With watch verb", "watch", "GET"),
			table.Entry("With watchList verb", "watchList", "GET"),
			table.Entry("With create verb", "create", "POST"),
			table.Entry("With delete verb", "delete", "DELETE"),
			table.Entry("With deletecollection verb", "deletecollection", "DELETE"),
			table.Entry("With update verb", "update", "PUT"),
			table.Entry("With patch verb", "patch", "PATCH"),
			table.Entry("With invalid verb", "invalid", ""),
			table.Entry("With empty verb", "", ""),
		)

		Context("With matching query params", func() {

			It("Should match query params", func() {
				reqStats := RequestStats{
					Query: map[string]int{
						"limit": 0,
						"tags":  0,
					},
					ParamsHit: 0,
				}
				vals := url.Values{
					"limit": []string{"100"},
					"tags":  []string{"color"},
				}
				By("Matching the first time it should increase ParamsHit")
				matchQueryParams(vals, &reqStats)
				Expect(reqStats.ParamsHit).To(Equal(2))
				Expect(reqStats.Query["limit"]).To(Equal(1))
				Expect(reqStats.Query["tags"]).To(Equal(1))

				By("Matching the second time it should not increase ParamsHit")
				matchQueryParams(vals, &reqStats)
				Expect(reqStats.ParamsHit).To(Equal(2))
				Expect(reqStats.Query["limit"]).To(Equal(2))
				Expect(reqStats.Query["tags"]).To(Equal(2))
			})

			It("Should not increase ParamsHit for undefined query params", func() {
				reqStats := RequestStats{
					Query:     map[string]int{"limit": 0},
					ParamsHit: 0,
				}
				vals := url.Values{"unknown": []string{"test"}}
				matchQueryParams(vals, &reqStats)
				Expect(reqStats.ParamsHit).To(Equal(0))
				Expect(reqStats.Query["limit"]).To(Equal(0))
				_, exists := reqStats.Query["unknown"]
				Expect(exists).To(BeFalse())
			})

		})

		Context("With matching and extracting body params", func() {

			It("Should match body params", func() {
				reqObject := runtime.Unknown{
					Raw: []byte(
						`{
							"pet": {
								"name": "bite",
								"kind": {
									"color": "red",
									"origin": {
										"country": "unknown",
										"region": "west"
									},
									"profile": {
										"size": "small"
									}
								}
							}
						}`,
					),
				}
				reqStats := RequestStats{
					Body:      map[string]int{},
					ParamsHit: 0,
				}

				By("Matching the first time it should increase ParamsHit")
				err := matchBodyParams(&reqObject, &reqStats)
				Expect(err).NotTo(HaveOccurred())
				Expect(reqStats.ParamsHit).To(Equal(5))
				Expect(reqStats.Body["pet.name"]).To(Equal(1))
				Expect(reqStats.Body["pet.kind.color"]).To(Equal(1))
				Expect(reqStats.Body["pet.kind.origin.country"]).To(Equal(1))
				Expect(reqStats.Body["pet.kind.origin.region"]).To(Equal(1))
				Expect(reqStats.Body["pet.kind.profile.size"]).To(Equal(1))

				By("Matching the second time it should not increase ParamsHit")
				matchBodyParams(&reqObject, &reqStats)
				Expect(reqStats.ParamsHit).To(Equal(5))
				Expect(reqStats.Body["pet.name"]).To(Equal(2))
				Expect(reqStats.Body["pet.kind.color"]).To(Equal(2))
				Expect(reqStats.Body["pet.kind.origin.country"]).To(Equal(2))
				Expect(reqStats.Body["pet.kind.origin.region"]).To(Equal(2))
				Expect(reqStats.Body["pet.kind.profile.size"]).To(Equal(2))
			})

			It("Should return an error for invalid body params", func() {
				reqObject := runtime.Unknown{
					Raw: []byte(
						`{
							"pet": {
								name: bite,
							}
						}`,
					),
				}
				reqStats := RequestStats{
					Body:      map[string]int{},
					ParamsHit: 0,
				}

				By("Passing invalid json object")
				err := matchBodyParams(&reqObject, &reqStats)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With coverage calculation", func() {

			It("Should calculate REST coverage", func() {
				expectedResult := map[string]float64{
					"/pets:GET":           66.67,
					"/pets:POST":          28.57,
					"/pets/{name}:PATCH":  100,
					"/pets/{name}:DELETE": 100,
					"/pets/{name}:GET":    0,
					"total":               63.16,
				}

				restAPIStats := map[string]map[string]*RequestStats{
					"/pets": map[string]*RequestStats{
						"POST": &RequestStats{
							ParamsHit:    1,
							ParamsNum:    6,
							MethodCalled: true,
						},
						"GET": &RequestStats{
							ParamsHit:    1,
							ParamsNum:    2,
							MethodCalled: true,
						},
					},
					"/pets/{name}": map[string]*RequestStats{
						"PATCH": &RequestStats{
							ParamsHit:    10,
							ParamsNum:    6,
							MethodCalled: true,
						},
						"DELETE": &RequestStats{
							ParamsHit:    0,
							ParamsNum:    0,
							MethodCalled: true,
						},
						"GET": &RequestStats{
							ParamsHit:    0,
							ParamsNum:    1,
							MethodCalled: false,
						},
					},
				}
				result := calculateCoverage(restAPIStats)
				for k, v := range result {
					result[k] = math.Round(v*100) / 100
				}
				Expect(result).To(Equal(expectedResult))
			})

		})
	})

})
