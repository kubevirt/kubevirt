package operands

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	imagev1 "github.com/openshift/api/image/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("imageStream tests", func() {

	schemeForTest := commontestutils.GetScheme()

	var (
		logger            = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("imageStream_test")
		testFilesLocation = getTestFilesLocation() + "/imageStreams"
		storeOrigFunc     = getImageStreamFileLocation
		hco               *hcov1beta1.HyperConverged
	)

	BeforeEach(func() {
		hco = commontestutils.NewHco()
	})

	AfterEach(func() {
		getImageStreamFileLocation = storeOrigFunc
	})

	Context("test imageStreamHandler", func() {
		It("should not create the ImageStream resource if the FG is not set", func() {
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(false)

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			cli := commontestutils.InitClient([]client.Object{})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			req := commontestutils.NewReq(hco)
			res := handlers[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())

			imageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
			Expect(imageStreamObjects.Items).To(BeEmpty())
		})

		It("should delete the ImageStream resource if the FG is not set", func() {
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(false)

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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			req := commontestutils.NewReq(hco)
			res := handlers[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())

			imageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
			Expect(imageStreamObjects.Items).To(BeEmpty())
		})

		It("should delete the ImageStream resource if the FG is not set, and emit event", func() {
			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			hcoNamespace := commontestutils.NewHcoNamespace()
			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
			eventEmitter := commontestutils.NewEventEmitterMock()
			ci := commontestutils.ClusterInfoMock{}
			cli := commontestutils.InitClient([]client.Object{hcoNamespace, hco, ci.GetCSV()})
			handler := NewOperandHandler(cli, commontestutils.GetScheme(), ci, eventEmitter)
			handler.FirstUseInitiation(commontestutils.GetScheme(), ci, hco)

			req := commontestutils.NewReq(hco)
			Expect(handler.Ensure(req)).To(Succeed())

			ImageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
			Expect(ImageStreamObjects.Items).To(HaveLen(1))
			Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))

			objectRef, err := reference.GetReference(commontestutils.GetScheme(), &ImageStreamObjects.Items[0])
			Expect(err).ToNot(HaveOccurred())
			hco.Status.RelatedObjects = append(hco.Status.RelatedObjects, *objectRef)

			By("check related object - the imageStream ref should be there")
			existingRef, err := objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *objectRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(existingRef).ToNot(BeNil())

			By("Run again, this time when the FG is false")
			eventEmitter.Reset()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(false)
			req = commontestutils.NewReq(hco)
			Expect(handler.Ensure(req)).To(Succeed())

			By("check that the image stream was removed")
			ImageStreamObjects = &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
			Expect(ImageStreamObjects.Items).To(BeEmpty())

			By("check that the delete event was emitted")
			expectedEvents := []commontestutils.MockEvent{
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

			hcoNamespace := commontestutils.NewHcoNamespace()
			hco := commontestutils.NewHco()
			ci := commontestutils.ClusterInfoMock{}
			cli := commontestutils.InitClient([]client.Object{hcoNamespace, hco, ci.GetCSV()})

			eventEmitter := commontestutils.NewEventEmitterMock()
			handler := NewOperandHandler(cli, commontestutils.GetScheme(), ci, eventEmitter)
			handler.FirstUseInitiation(commontestutils.GetScheme(), ci, hco)

			req := commontestutils.NewReq(hco)
			Expect(handler.Ensure(req)).To(Succeed())

			expectedEvents := []commontestutils.MockEvent{
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

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
			cli := commontestutils.InitClient([]client.Object{hco})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			req := commontestutils.NewReq(hco)
			res := handlers[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeTrue())

			ImageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
			Expect(ImageStreamObjects.Items).To(HaveLen(1))
			Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))
				// check that this tag was changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).ToNot(BeEmpty())

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).To(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Insecure: false, Scheduled: true}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("old"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/old-test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).To(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Insecure: true, Scheduled: false}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
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

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).To(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Insecure: false, Scheduled: true}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should update the ImageStream labels", func() {

			getImageStreamFileLocation = func() string {
				return testFilesLocation
			}

			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"

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
			expectedLabels := make(map[string]string, len(exists.Labels))
			for k, v := range exists.Labels {
				expectedLabels[k] = v
			}
			exists.Labels[userLabelKey] = userLabelValue
			for k, v := range expectedLabels {
				exists.Labels[k] = "wrong_" + v
			}
			exists.Labels[util.AppLabelManagedBy] = expectedLabels[util.AppLabelManagedBy]

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

				for k, v := range expectedLabels {
					Expect(is.Labels).To(HaveKeyWithValue(k, v))
				}
				Expect(is.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
			})
		})

		Context("imagestream namespace", func() {
			const customNS = "custom-ns"
			It("should create imagestream in a custom namespace", func() {
				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				hco := commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))
			})

			It("should delete an imagestream from one namespace, and create it in another one", func() {
				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				By("create imagestream in the default namespace")
				hco := commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal("test-image-stream-ns"))

				By("replace the image stream with a new one in the custom namespace")
				hco = commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				req = commontestutils.NewReq(hco)
				res = handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects = &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))
			})

			It("should remove an imagestream from a custom namespace, and create it in the default one", func() {
				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				By("create imagestream in a custom namespace")
				hco := commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))

				By("replace the image stream with a new one in the default namespace")
				hco = commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)

				req = commontestutils.NewReq(hco)
				res = handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects = &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal("test-image-stream-ns"))
			})

			It("should remove an imagestream from a custom namespace, and create it in the new custom namespace", func() {
				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				getImageStreamFileLocation = func() string {
					return testFilesLocation
				}

				By("create imagestream in a custom namespace")
				hco := commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := getImageStreamHandlers(logger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))

				By("replace the image stream with a new one in another custom namespace")
				hco = commontestutils.NewHco()
				hco.Spec.FeatureGates.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS + "1")

				req = commontestutils.NewReq(hco)
				res = handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects = &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS + "1"))
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

		hook := newIsHook(required, required.Namespace)

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
				Expect(tag.Annotations).To(HaveLen(1))
				Expect(tag.Annotations).To(HaveKeyWithValue("test-annotation", "should stay here"))
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
