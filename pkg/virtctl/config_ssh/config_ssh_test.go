package config_ssh

import (
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("config-ssh", func() {

	It("should not modify content if nothing needs to be done", func() {
		cfg, fileMode, err := loadSSHConfig("testdata/config.1")
		Expect(err).ToNot(HaveOccurred())

		stat, err := os.Stat("testdata/config.1")
		Expect(err).ToNot(HaveOccurred())

		Expect(fileMode).To(BeNumerically("==", stat.Mode()))
		Expect(cfg.Hosts).To(HaveLen(4))
		rawCfg, err := ioutil.ReadFile("testdata/config.1")
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(cfg.String())).To(Equal(strings.TrimSpace(string(rawCfg))))
	})

	It("should return an empty config if the file does not exist", func() {
		cfg, fileMode, err := loadSSHConfig("testdata/nonexistent")
		Expect(err).To(BeNil())
		Expect(fileMode).To(BeNumerically("==", 0600))
		Expect(cfg.Hosts).To(BeEmpty())
	})

	It("should generate a host entry for a VMI", func() {
		host, err := generateHostEntry("virtctl", "mycontext", vmi("testvm", "testnamespace"))
		Expect(err).ToNot(HaveOccurred())
		Expect(host.EOLComment).To(Equal(KubeVirtEOLComment))
		Expect(host.Patterns[0].String()).To(Equal("vmi/testvm.testnamespace.mycontext"))
		Expect(host.Nodes[0].String()).To(Equal("ProxyCommand virtctl port-forward --context mycontext --stdio vmi/testvm.testnamespace %p"))
	})

	It("should generate a host entry for a VM", func() {
		host, err := generateHostEntry("virtctl", "mycontext", vm("testvm", "testnamespace"))
		Expect(err).ToNot(HaveOccurred())
		Expect(host.EOLComment).To(Equal(KubeVirtEOLComment))
		Expect(host.Patterns[0].String()).To(Equal("vm/testvm.testnamespace.mycontext"))
		Expect(host.Nodes[0].String()).To(Equal("ProxyCommand virtctl port-forward --context mycontext --stdio vm/testvm.testnamespace %p"))
	})

	It("should remove host entries from KubeVirt", func() {
		cfg, _, err := loadSSHConfig("testdata/config.1")
		Expect(err).ToNot(HaveOccurred())
		hosts, err := generateHostEntries("virtctl", "mycontext", []unstructured.Unstructured{
			*vmi("myvmi", "mynamespace"),
			*vm("myvm", "mynamespace"),
		})
		Expect(err).ToNot(HaveOccurred())
		cfg.Hosts = append(cfg.Hosts, hosts...)
		Expect(cfg.Hosts).To(HaveLen(6))
		hosts = removeHostEntries(cfg.Hosts)
		Expect(hosts).To(HaveLen(4))
		for _, host := range hosts {
			Expect(host.EOLComment).ToNot(Equal(KubeVirtEOLComment))
		}
	})
})

func vmi(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "VirtualMachineInstance",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}

func vm(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}
