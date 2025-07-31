package tls

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/util"
)

var _ = Describe("CaManager", func() {

	var configMap *k8sv1.ConfigMap
	var manager ClientCAManager
	var store cache.Store

	BeforeEach(func() {
		ca, err := triple.NewCA("first", time.Hour)
		Expect(err).ToNot(HaveOccurred())
		configMap = &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            util.ExtensionAPIServerAuthenticationConfigMap,
				Namespace:       metav1.NamespaceSystem,
				ResourceVersion: "1",
			},
			Data: map[string]string{
				util.RequestHeaderClientCAFileKey: string(cert.EncodeCertPEM(ca.Cert)),
			},
		}
		store = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		Expect(store.Add(configMap)).To(Succeed())
		manager = NewKubernetesClientCAManager(store)
	})

	It("should load an initial CA", func() {
		cert, err := manager.GetCurrent()
		Expect(err).ToNot(HaveOccurred())
		Expect(cert.Subjects()[0]).To(ContainSubstring("first"))
	})

	It("should detect updates on the informer and update the CA", func() {
		newCA, err := triple.NewCA("new", time.Hour)
		Expect(err).ToNot(HaveOccurred())
		configMap.Data[util.RequestHeaderClientCAFileKey] = string(cert.EncodeCertPEM(newCA.Cert))
		configMap.ObjectMeta.ResourceVersion = "2"
		cert, err := manager.GetCurrent()
		Expect(err).ToNot(HaveOccurred())
		Expect(cert.Subjects()[0]).To(ContainSubstring("new"))
	})

	It("should detect invalid CAs and recover later", func() {
		By("injecting an invalid CA")
		configMap.Data[util.RequestHeaderClientCAFileKey] = string("garbage")
		configMap.ObjectMeta.ResourceVersion = "2"
		_, err := manager.GetCurrent()
		Expect(err).To(HaveOccurred())
		By("repairing the CA")
		configMap.ObjectMeta.ResourceVersion = "3"
		newCA, err := triple.NewCA("new", time.Hour)
		Expect(err).ToNot(HaveOccurred())
		configMap.Data[util.RequestHeaderClientCAFileKey] = string(cert.EncodeCertPEM(newCA.Cert))
		cert, err := manager.GetCurrent()
		Expect(err).ToNot(HaveOccurred())
		Expect(cert.Subjects()[0]).To(ContainSubstring("new"))
	})

	It("should detect if the is no CA provided", func() {
		delete(configMap.Data, util.RequestHeaderClientCAFileKey)
		_, err := manager.GetCurrent()
		Expect(err).To(HaveOccurred())
	})

	It("should detect if the config map is missing", func() {
		Expect(store.Delete(configMap)).To(Succeed())
		_, err := manager.GetCurrent()
		Expect(err).To(HaveOccurred())
	})

	It("should return the last result if the resource version of the map did not change", func() {
		By("first loading the valid CA")
		_, err := manager.GetCurrent()
		Expect(err).ToNot(HaveOccurred())
		By("changing the content but not increasing the resource version")
		configMap.Data[util.RequestHeaderClientCAFileKey] = string("garbage")
		cert, err := manager.GetCurrent()
		Expect(err).ToNot(HaveOccurred())
		Expect(cert.Subjects()[0]).To(ContainSubstring("first"))
	})
})

var _ = Describe("KubernetesCAManager", func() {

	prepareManagerComplex := func(f func(*k8sv1.ConfigMap)) KubernetesCAManager {
		ca, err := triple.NewCA("first", time.Hour)
		Expect(err).ToNot(HaveOccurred())
		configMap := &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            util.ExtensionAPIServerAuthenticationConfigMap,
				Namespace:       metav1.NamespaceSystem,
				ResourceVersion: "1",
			},
			Data: map[string]string{
				util.RequestHeaderClientCAFileKey: string(cert.EncodeCertPEM(ca.Cert)),
			},
		}
		f(configMap)
		store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		Expect(store.Add(configMap)).To(Succeed())
		return NewKubernetesClientCAManager(store)
	}

	prepareManager := func(data string) KubernetesCAManager {
		return prepareManagerComplex(func(cm *k8sv1.ConfigMap) {
			cm.Data["requestheader-allowed-names"] = data
		})
	}

	DescribeTable("should return zero CNs", func(data string) {
		manager := prepareManager(data)
		Expect(manager.GetCNs()).To(BeEmpty())
	},
		Entry("with empty", ""),
		Entry("with empty slice", "[]"),
	)

	DescribeTable("should return", func(data string, expected []string) {
		manager := prepareManager(data)
		Expect(manager.GetCNs()).To(ConsistOf(expected))
	},
		Entry("with one element", `["one"]`, []string{"one"}),
		Entry("with two elements", `["one", "two"]`, []string{"one", "two"}),
	)

	DescribeTable("should return error", func(data string) {
		manager := prepareManager(data)
		_, err := manager.GetCNs()
		Expect(err).To(HaveOccurred())
	},
		Entry("with malformed", `["one",]`),
		Entry("with malformed string", `[one]`),
		Entry("with no array", "one"),
	)

	It(`should error when no "requestheader-allowed-names" is not specified`, func() {
		manager := prepareManagerComplex(func(cm *k8sv1.ConfigMap) {})
		_, err := manager.GetCNs()
		Expect(err).To(MatchError(ContainSubstring("requestheader-allowed-names not found in")))
	})
})
