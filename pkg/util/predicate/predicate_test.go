package predicate

import (
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("Predicate", func() {

	Describe("When checking a GenerationOrAnnotationChangedPredicate", func() {
		instance := GenerationOrAnnotationChangedPredicate{}

		Context("Where the old object doesn't have metadata", func() {
			It("should return false", func() {
				newObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "foof",
					}}

				updateEvent := event.UpdateEvent{
					ObjectNew: newObj,
					MetaNew:   newObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeFalse())
			})
		})

		Context("Where the new object doesn't have metadata", func() {
			It("should return false", func() {
				oldObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "foof",
					}}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldObj,
					MetaOld:   oldObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeFalse())
			})
		})

		Context("Where both the generation and annotations haven't changed", func() {
			It("should return false", func() {
				oldObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  1,
						Annotations: map[string]string{"key": "value"},
					}}
				newObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  1,
						Annotations: map[string]string{"key": "value"},
					}}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldObj,
					ObjectNew: newObj,
					MetaOld:   oldObj.GetObjectMeta(),
					MetaNew:   newObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeFalse())
			})
		})

		Context("Where the generation hasn't changed and the annotations are empty", func() {
			It("should return false", func() {
				oldObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "foo",
						Namespace:  "foof",
						Generation: 1,
					}}
				newObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "foo",
						Namespace:  "foof",
						Generation: 1,
					}}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldObj,
					ObjectNew: newObj,
					MetaOld:   oldObj.GetObjectMeta(),
					MetaNew:   newObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeFalse())
			})
		})

		Context("Where the generation hasn't changed but an annotation has changed", func() {
			It("should return true", func() {
				oldObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  1,
						Annotations: map[string]string{"key": "old_value"},
					}}
				newObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  1,
						Annotations: map[string]string{"key": "new_value"},
					}}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldObj,
					ObjectNew: newObj,
					MetaOld:   oldObj.GetObjectMeta(),
					MetaNew:   newObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeTrue())
			})
		})

		Context("Where the annotations haven't changed but the generation has changed", func() {
			It("should return true", func() {
				oldObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  1,
						Annotations: map[string]string{"key": "value"},
					}}
				newObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  2,
						Annotations: map[string]string{"key": "value"},
					}}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldObj,
					ObjectNew: newObj,
					MetaOld:   oldObj.GetObjectMeta(),
					MetaNew:   newObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeTrue())
			})
		})

		Context("Where both the generation and annotations have changed", func() {
			It("should return true", func() {
				oldObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  1,
						Annotations: map[string]string{"key": "old_value"},
					}}
				newObj := &v1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "foof",
						Generation:  2,
						Annotations: map[string]string{"key": "new_value"},
					}}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldObj,
					ObjectNew: newObj,
					MetaOld:   oldObj.GetObjectMeta(),
					MetaNew:   newObj.GetObjectMeta(),
				}
				Expect(instance.Create(event.CreateEvent{})).To(BeTrue())
				Expect(instance.Delete(event.DeleteEvent{})).To(BeTrue())
				Expect(instance.Generic(event.GenericEvent{})).To(BeTrue())
				Expect(instance.Update(updateEvent)).To(BeTrue())
			})
		})
	})
})
