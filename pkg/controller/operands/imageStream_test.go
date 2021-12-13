package operands

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	imagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("imageStream tests", func() {

	schemeForTest := commonTestUtils.GetScheme()

	var (
		logger            = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("imageStream_test")
		testFilesLocation = getTestFilesLocation() + "/imageStreams"
		hco               = commonTestUtils.NewHco()
		storeOrigFunc     = getImageStreamFileLocation
	)

	AfterEach(func() {
		getImageStreamFileLocation = storeOrigFunc
	})

	Context("test imageStreamHandler", func() {

		It("should create the ImageStream resource if not exists", func() {
			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			cli := commonTestUtils.InitClient([]runtime.Object{})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), ImageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).Should(Equal("test-image-stream"))
			})
		})

		It("should update the ImageStream resource if the docker image was changed", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/old-test-image",
							},
							Name: "latest",
						},
					},
				},
			}
			exists.Labels = getLabels(hco, util.AppComponentCompute)

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("latest"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should update the ImageStream resource if the tag name was changed", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
							},
							Name: "old",
						},
					},
				},
			}
			exists.Labels = getLabels(hco, util.AppComponentCompute)

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("latest"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should remove tags if they are not required", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							Name: "latest",
						},
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/old-test-image",
							},
							Name: "old",
						},
					},
				},
			}
			exists.Labels = getLabels(hco, util.AppComponentCompute)

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("latest"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/test-image"))
				// check that this tag was changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).Should(BeEmpty())

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should not update the imageStream if the tag name and the from.name fields are the same", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Insecure: true, Scheduled: false},
							Name:         "latest",
						},
					},
				},
			}
			exists.Labels = getLabels(hco, util.AppComponentCompute)

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("latest"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).Should(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).Should(Equal(imagev1.TagImportPolicy{Insecure: true, Scheduled: false}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should not update the imageStream if the it not controlled by HCO (even if the details are not the same)", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/old-test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Insecure: true, Scheduled: false},
							Name:         "old",
						},
					},
				},
			}

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("old"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/old-test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).Should(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).Should(Equal(imagev1.TagImportPolicy{Insecure: true, Scheduled: false}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should not update the imageStream if nothing has changed", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Insecure: true, Scheduled: false},
							Name:         "latest",
						},
					},
				},
			}
			exists.Labels = getLabels(hco, util.AppComponentCompute)

			cli := commonTestUtils.InitClient([]runtime.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("latest"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).Should(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).Should(Equal(imagev1.TagImportPolicy{Insecure: true, Scheduled: false}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should update the ImageStream labels", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
							},
							Name: "latest",
						},
					},
				},
			}
			exists.Labels = getLabels(hco, util.AppComponentCompute)
			exists.ObjectMeta.Labels["to-be-removed"] = "test"

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			By("apply the ImageStream CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				err := cli.List(context.TODO(), imageStreamObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).Should(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).Should(Equal("latest"))
				Expect(tag.From.Name).Should(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).To(BeNil())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

				Expect(exists.ObjectMeta.Labels).ToNot(ContainElement("to-be-removed"))
			})
		})
	})
})
