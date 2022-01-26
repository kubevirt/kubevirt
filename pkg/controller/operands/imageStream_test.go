package operands

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	imagev1 "github.com/openshift/api/image/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
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
		It("should not create the ImageStream resource if the FG is not set", func() {
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
			req := commonTestUtils.NewReq(hco)
			res := handlers[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())

			imageStreamObjects := &imagev1.ImageStreamList{}
			err = cli.List(context.TODO(), imageStreamObjects)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageStreamObjects.Items).To(BeEmpty())
		})

		It("should delete the ImageStream resource if the FG is not set", func() {
			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

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
			req := commonTestUtils.NewReq(hco)
			res := handlers[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())

			imageStreamObjects := &imagev1.ImageStreamList{}
			err = cli.List(context.TODO(), imageStreamObjects)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageStreamObjects.Items).To(BeEmpty())
		})

		It("should delete the ImageStream resource if the FG is not set, and emit event", func() {
			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true
			eventEmitter := commonTestUtils.NewEventEmitterMock()
			cli := commonTestUtils.InitClient([]runtime.Object{hco})
			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			ImageStreamObjects := &imagev1.ImageStreamList{}
			err = cli.List(context.TODO(), ImageStreamObjects)
			Expect(err).ToNot(HaveOccurred())
			Expect(ImageStreamObjects.Items).To(HaveLen(1))
			Expect(ImageStreamObjects.Items[0].Name).Should(Equal("test-image-stream"))

			objectRef, err := reference.GetReference(commonTestUtils.GetScheme(), &ImageStreamObjects.Items[0])
			Expect(err).ToNot(HaveOccurred())
			hco.Status.RelatedObjects = append(hco.Status.RelatedObjects, *objectRef)

			By("check related object - the imageStream ref should be there")
			existingRef, err := objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *objectRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(existingRef).ToNot(BeNil())

			By("Run again, this time when the FG is false")
			eventEmitter.Reset()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = false
			req = commonTestUtils.NewReq(hco)
			err = handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			By("check that the image stream was removed")
			ImageStreamObjects = &imagev1.ImageStreamList{}
			err = cli.List(context.TODO(), ImageStreamObjects)
			Expect(err).ToNot(HaveOccurred())
			Expect(ImageStreamObjects.Items).To(HaveLen(0))

			By("check that the delete event was emitted")
			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed ImageStream test-image-stream",
				},
			}
			Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeTrue())

			By("check that the related object was removed")
			existingRef, err = objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *objectRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(existingRef).To(BeNil())
		})

		It("should not emit event if the FG is not set and the image stream is not exist", func() {
			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()
			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed ImageStream test-image-stream",
				},
			}
			Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeFalse())
		})

		It("should create the ImageStream resource if not exists", func() {
			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true
			cli := commonTestUtils.InitClient([]runtime.Object{hco})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			By("apply the ImageStream CRs")
			req := commonTestUtils.NewReq(hco)
			res := handlers[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeTrue())

			ImageStreamObjects := &imagev1.ImageStreamList{}
			err = cli.List(context.TODO(), ImageStreamObjects)
			Expect(err).ToNot(HaveOccurred())
			Expect(ImageStreamObjects.Items).To(HaveLen(1))
			Expect(ImageStreamObjects.Items[0].Name).Should(Equal("test-image-stream"))

			By("check that the reference is in the related object list")
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

			cli := commonTestUtils.InitClient([]runtime.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true
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

			cli := commonTestUtils.InitClient([]runtime.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true

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

			cli := commonTestUtils.InitClient([]runtime.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true

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
				Expect(tag.From.UID).ShouldNot(BeEmpty())

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
							ImportPolicy: imagev1.TagImportPolicy{Scheduled: true},
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
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true

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
				Expect(tag.ImportPolicy).Should(Equal(imagev1.TagImportPolicy{Insecure: false, Scheduled: true}))

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

			cli := commonTestUtils.InitClient([]runtime.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true

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
							ImportPolicy: imagev1.TagImportPolicy{Insecure: false, Scheduled: true},
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
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true

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
				Expect(tag.ImportPolicy).Should(Equal(imagev1.TagImportPolicy{Insecure: false, Scheduled: true}))

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

			cli := commonTestUtils.InitClient([]runtime.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commonTestUtils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = true

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

	Context("test compareAndUpgradeImageStream", func() {
		required := &imagev1.ImageStream{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testStream",
			},
			Spec: imagev1.ImageStreamSpec{
				Tags: []imagev1.TagReference{
					{
						From: &corev1.ObjectReference{
							Name: "my-image-registry:5000/my-image:v1",
							Kind: "DockerImage",
						},
						ImportPolicy: imagev1.TagImportPolicy{
							Scheduled: true,
						},
						Name: "v1",
					},
					{
						From: &corev1.ObjectReference{
							Name: "my-image-registry:5000/my-image:v2",
							Kind: "DockerImage",
						},
						ImportPolicy: imagev1.TagImportPolicy{
							Scheduled: true,
						},
						Name: "v2",
					},
					{
						From: &corev1.ObjectReference{
							Name: "my-image-registry:5000/my-image:v2",
							Kind: "DockerImage",
						},
						ImportPolicy: imagev1.TagImportPolicy{
							Scheduled: true,
						},
						Name: "latest",
					},
				},
			},
		}

		hook := newIsHook(required)

		It("should do nothing if there is no difference", func() {
			found := required.DeepCopy()

			Expect(hook.compareAndUpgradeImageStream(found)).To(BeFalse())

			Expect(found.Spec.Tags).To(HaveLen(3))

			validateImageStream(found, hook)
		})

		It("should add all tag if missing", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
			}

			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)
		})

		It("should add missing tags", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "latest",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v1",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v1",
						},
					},
				},
			}

			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)
		})

		It("should delete unknown tags", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v1",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v1",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v2",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v3",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v3",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v3",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "latest",
						},
					},
				},
			}
			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)
		})

		It("should fix tag from and import policy, but leave the rest", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							Annotations: map[string]string{
								"test-annotation": "should stay here",
							},
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v45",
								Kind: "somethingElse",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v1",
						},
						{
							Annotations: map[string]string{
								"test-annotation": "should stay here",
							},
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: false,
							},
							Name: "v2",
						},
						{
							Annotations: map[string]string{
								"test-annotation": "should stay here",
							},
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "latest",
						},
					},
				},
			}
			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)

			for _, tag := range found.Spec.Tags {
				Expect(tag.Annotations).Should(HaveLen(1))
				Expect(tag.Annotations).Should(HaveKeyWithValue("test-annotation", "should stay here"))
			}

		})
	})
})

func validateImageStream(found *imagev1.ImageStream, hook *isHooks) {
	ExpectWithOffset(1, found.Spec.Tags).To(HaveLen(3))

	validationTagMap := map[string]bool{
		"v1":     false,
		"v2":     false,
		"latest": false,
	}

	for i := 0; i < 3; i++ {
		tagName := found.Spec.Tags[i].Name
		tag := getTagByName(found.Spec.Tags, tagName)
		Expect(tag).ToNot(BeNil())
		validationTagMap[tagName] = true

		ExpectWithOffset(1, tag.From).Should(Equal(hook.tags[tagName].From))
		ExpectWithOffset(1, tag.ImportPolicy).Should(Equal(imagev1.TagImportPolicy{Scheduled: true}))
	}

	ExpectWithOffset(1, validateAllTags(validationTagMap)).To(BeTrue())
}

func getTagByName(tags []imagev1.TagReference, name string) *imagev1.TagReference {
	for _, tag := range tags {
		if tag.Name == name {
			return &tag
		}
	}
	return nil
}

func validateAllTags(m map[string]bool) bool {
	for _, toughed := range m {
		if !toughed {
			return false
		}
	}
	return true
}
