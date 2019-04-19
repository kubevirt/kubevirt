/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package approver implements an automated approver for kubelet certificates.
package approver

import (
	"crypto/x509"
	"fmt"
	"reflect"
	"strings"

	authorization "k8s.io/api/authorization/v1beta1"
	capi "k8s.io/api/certificates/v1beta1"
	certificatesinformers "k8s.io/client-go/informers/certificates/v1beta1"
	clientset "k8s.io/client-go/kubernetes"
	csr2 "k8s.io/client-go/util/certificate/csr"

	"kubevirt.io/kubevirt/pkg/virt-operator/certificates"
)

type csrRecognizer struct {
	recognize      func(csr *capi.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool
	permission     authorization.ResourceAttributes
	successMessage string
}

type sarApprover struct {
	client      clientset.Interface
	recognizers []csrRecognizer
}

func NewCSRApprovingController(client clientset.Interface, csrInformer certificatesinformers.CertificateSigningRequestInformer) *certificates.CertificateController {
	approver := &sarApprover{
		client:      client,
		recognizers: recognizers(),
	}
	return certificates.NewCertificateController(
		client,
		csrInformer,
		approver.handle,
	)
}

func recognizers() []csrRecognizer {
	recognizers := []csrRecognizer{
		{
			recognize:      isKubeVirtCert,
			permission:     authorization.ResourceAttributes{Group: "certificates.k8s.io", Resource: "certificatesigningrequests", Verb: "create", Subresource: "kubevirt"},
			successMessage: "Auto approving kubevirt.io client certificate after SubjectAccessReview.",
		},
	}
	return recognizers
}

func (a *sarApprover) handle(csr *capi.CertificateSigningRequest) error {
	if len(csr.Status.Certificate) != 0 {
		return nil
	}
	if approved, denied := certificates.GetCertApprovalCondition(&csr.Status); approved || denied {
		return nil
	}
	x509cr, err := csr2.ParseCSR(csr)
	if err != nil {
		return fmt.Errorf("unable to parse csr %q: %v", csr.Name, err)
	}

	tried := []string{}

	for _, r := range a.recognizers {
		if !r.recognize(csr, x509cr) {
			continue
		}

		tried = append(tried, r.permission.Subresource)

		approved, err := a.authorize(csr, r.permission)
		if err != nil {
			return err
		}
		if approved {
			appendApprovalCondition(csr, r.successMessage)
			_, err = a.client.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(csr)
			if err != nil {
				return fmt.Errorf("error updating approval for csr: %v", err)
			}
			return nil
		}
	}

	if len(tried) != 0 {
		return certificates.IgnorableError("recognized csr %q as %v but subject access review was not approved", csr.Name, tried)
	}

	return nil
}

func (a *sarApprover) authorize(csr *capi.CertificateSigningRequest, rattrs authorization.ResourceAttributes) (bool, error) {
	extra := make(map[string]authorization.ExtraValue)
	for k, v := range csr.Spec.Extra {
		extra[k] = authorization.ExtraValue(v)
	}

	sar := &authorization.SubjectAccessReview{
		Spec: authorization.SubjectAccessReviewSpec{
			User:               csr.Spec.Username,
			UID:                csr.Spec.UID,
			Groups:             csr.Spec.Groups,
			Extra:              extra,
			ResourceAttributes: &rattrs,
		},
	}
	sar, err := a.client.AuthorizationV1beta1().SubjectAccessReviews().Create(sar)
	if err != nil {
		return false, err
	}
	return sar.Status.Allowed, nil
}

func appendApprovalCondition(csr *capi.CertificateSigningRequest, message string) {
	csr.Status.Conditions = append(csr.Status.Conditions, capi.CertificateSigningRequestCondition{
		Type:    capi.CertificateApproved,
		Reason:  "AutoApproved",
		Message: message,
	})
}

func hasExactUsages(csr *capi.CertificateSigningRequest, usages []capi.KeyUsage) bool {
	if len(usages) != len(csr.Spec.Usages) {
		return false
	}

	usageMap := map[capi.KeyUsage]struct{}{}
	for _, u := range usages {
		usageMap[u] = struct{}{}
	}

	for _, u := range csr.Spec.Usages {
		if _, ok := usageMap[u]; !ok {
			return false
		}
	}

	return true
}

var virtHandlerUsages = []capi.KeyUsage{
	capi.UsageKeyEncipherment,
	capi.UsageDigitalSignature,
	capi.UsageClientAuth,
	capi.UsageServerAuth,
}

var serverUsages = []capi.KeyUsage{
	capi.UsageKeyEncipherment,
	capi.UsageDigitalSignature,
	capi.UsageServerAuth,
}

func isKubeVirtCert(csr *capi.CertificateSigningRequest, x509cr *x509.CertificateRequest) bool {
	// Only sign for the kubevirt.io:system org
	if !reflect.DeepEqual([]string{"kubevirt.io:system"}, x509cr.Subject.Organization) {
		return false
	}

	// No email needed
	if len(x509cr.EmailAddresses) > 0 {
		return false
	}

	// This is the PodIP, we want to have that on every cert
	if len(x509cr.IPAddresses) != 1 {
		return false
	}

	switch csr.Spec.Username {

	case "system:serviceaccount:kubevirt:kubevirt-handler":
		// virt-handler is server and client
		if !hasExactUsages(csr, virtHandlerUsages) {
			return false
		}
		// virt-handler does not need any dns names, communication only happens via IPs
		if len(x509cr.DNSNames) > 0 {
			return false
		}
		// virt-handlers should have the kubevirt.io:system:nodes prefix in the common name
		if !strings.HasPrefix(x509cr.Subject.CommonName, "kubevirt.io:system:nodes") {
			return false
		}
	case "system:serviceaccount:kubevirt:kubevirt-controller", "system:serviceaccount:kubevirt:kubevirt-operator":
		// virt-controller is only a server
		if !hasExactUsages(csr, serverUsages) {
			return false
		}
		// virt-controller does not need any dns names, communication only happens via IPs
		if len(x509cr.DNSNames) > 0 {
			return false
		}
		// virt-controller acts on the cluster level, not on the node level
		if !strings.HasPrefix(x509cr.Subject.CommonName, "kubevirt.io:system:services") {
			return false
		}
	case "system:serviceaccount:kubevirt:kubevirt-apiserver":
		// virt-api is only a server
		if !hasExactUsages(csr, serverUsages) {
			return false
		}
		// virt-api needs to be reachable via the virt-api service in the install namespace
		if len(x509cr.DNSNames) != 1 {
			return false
		}

		// virt-api acts on the cluster level, not on the node level
		if !strings.HasPrefix(x509cr.Subject.CommonName, "kubevirt.io:system:services") {
			return false
		}
	default:
		return false
	}
	return true
}
