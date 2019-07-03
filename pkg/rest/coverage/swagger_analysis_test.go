package coverage

import (
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Swagger analysis", func() {

	Context("With swagger petstore", func() {

		Context("Without URI filter", func() {

			It("Should build full REST API structure", func() {
				expectedRestAPI := map[string]map[string]*RequestStats{
					"/pets": map[string]*RequestStats{
						"POST": &RequestStats{
							Body:         map[string]int{},
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    6,
							MethodCalled: false,
							Path:         "/pets",
							Method:       "POST",
						},
						"GET": &RequestStats{
							Body: nil,
							Query: map[string]int{
								"tags":  0,
								"limit": 0,
							},
							ParamsHit:    0,
							ParamsNum:    2,
							MethodCalled: false,
							Path:         "/pets",
							Method:       "GET",
						},
					},
					"/pets/{name}": map[string]*RequestStats{
						"GET": &RequestStats{
							Body:         nil,
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    0,
							MethodCalled: false,
							Path:         "/pets/{name}",
							Method:       "GET",
						},
						"DELETE": &RequestStats{
							Body:         nil,
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    0,
							MethodCalled: false,
							Path:         "/pets/{name}",
							Method:       "DELETE",
						},
						"PATCH": &RequestStats{
							Body:         map[string]int{},
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    6,
							MethodCalled: false,
							Path:         "/pets/{name}",
							Method:       "PATCH",
						},
					},
				}

				restAPIStats, err := getRESTApiStats(petStoreSwaggerPath, "")
				Expect(err).NotTo(HaveOccurred(), "request stats structure should be initialized")
				Expect(restAPIStats).To(Equal(expectedRestAPI), "request stats values should be equal to expected values")
			})

		})

		Context("With URI filter", func() {

			It("Should build filtered REST API structure", func() {
				expectedRestAPI := map[string]map[string]*RequestStats{
					"/pets/{name}": map[string]*RequestStats{
						"GET": &RequestStats{
							Body:         nil,
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    0,
							MethodCalled: false,
							Path:         "/pets/{name}",
							Method:       "GET",
						},
						"DELETE": &RequestStats{
							Body:         nil,
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    0,
							MethodCalled: false,
							Path:         "/pets/{name}",
							Method:       "DELETE",
						},
						"PATCH": &RequestStats{
							Body:         map[string]int{},
							Query:        map[string]int{},
							ParamsHit:    0,
							ParamsNum:    6,
							MethodCalled: false,
							Path:         "/pets/{name}",
							Method:       "PATCH",
						},
					},
				}

				restAPIStats, err := getRESTApiStats(petStoreSwaggerPath, "/pets/{name}")
				Expect(err).NotTo(HaveOccurred(), "request stats structure should be initialized")
				Expect(restAPIStats).To(Equal(expectedRestAPI), "request stats values should be equal to expected values")

			})
		})

		It("Should add swagger params", func() {
			document, err := loads.JSONSpec(petStoreSwaggerPath)
			Expect(err).NotTo(HaveOccurred())

			By("Resolving referenced definitions")
			reqStats := RequestStats{
				Body:         nil,
				Query:        map[string]int{},
				ParamsHit:    0,
				ParamsNum:    0,
				MethodCalled: false,
				Path:         "/pets",
				Method:       "POST",
			}
			expectedReqStats := reqStats
			expectedReqStats.Body = map[string]int{}
			expectedReqStats.ParamsNum = 6

			addSwaggerParams(&reqStats, document.Analyzer.ParamsFor("POST", "/pets"), document.Spec().Definitions)
			Expect(reqStats).To(Equal(expectedReqStats), "request stats values should be equal to expected values")

			By("Not resolving referenced definitions")
			reqStats = RequestStats{
				Body:         nil,
				Query:        map[string]int{},
				ParamsHit:    0,
				ParamsNum:    0,
				MethodCalled: false,
				Path:         "/pets",
				Method:       "GET",
			}
			expectedReqStats = reqStats
			expectedReqStats.Query = map[string]int{
				"tags":  0,
				"limit": 0,
			}
			expectedReqStats.ParamsNum = 2

			addSwaggerParams(&reqStats, document.Analyzer.ParamsFor("GET", "/pets"), document.Spec().Definitions)
			Expect(reqStats).To(Equal(expectedReqStats), "request stats values should be equal to expected values")
		})

		It("Should count params from referenced models", func() {
			document, err := loads.JSONSpec(petStoreSwaggerPath)
			Expect(err).NotTo(HaveOccurred(), "swagger json file should be open")

			params := document.Analyzer.ParamsFor("POST", "/pets")
			pCnt := countRefParams(params["body#Pet"].Schema, document.Spec().Definitions)
			Expect(pCnt).To(Equal(6), "reference params number should be equal to expected value")

			pCnt = countRefParams(params["body#Pet"].Schema, spec.Definitions{})
			Expect(pCnt).To(Equal(0), "reference params number for non-existence definition should not be increased")
		})
	})
})
