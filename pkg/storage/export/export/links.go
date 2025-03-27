/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package export

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	caBundle             = "ca-bundle"
	routeCAConfigMapName = "kube-root-ca.crt"
	routeCaKey           = "ca.crt"
	subjectAltNameId     = "2.5.29.17"

	apiGroup              = "export.kubevirt.io"
	apiVersion            = "v1beta1"
	exportResourceName    = "virtualmachineexports"
	gv                    = apiGroup + "/" + apiVersion
	externalUrlLinkFormat = "/api/" + gv + "/namespaces/%s/" + exportResourceName + "/%s"
	internal              = "internal"
	external              = "external"
)

func (ctrl *VMExportController) getInteralLinks(pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, service *corev1.Service, getVolumeName getExportVolumeName, export *exportv1.VirtualMachineExport) (*exportv1.VirtualMachineExportLink, error) {
	internalCert, err := ctrl.internalExportCa()
	if err != nil {
		return nil, err
	}
	host := fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
	return ctrl.getLinks(pvcs, exporterPod, export, host, internal, internalCert, getVolumeName)
}

func (ctrl *VMExportController) getExternalLinks(pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, getVolumeName getExportVolumeName, export *exportv1.VirtualMachineExport) (*exportv1.VirtualMachineExportLink, error) {
	urlPath := fmt.Sprintf(externalUrlLinkFormat, export.Namespace, export.Name)
	externalLinkHost, cert := ctrl.getExternalLinkHostAndCert()
	if externalLinkHost != "" {
		hostAndBase := path.Join(externalLinkHost, urlPath)
		return ctrl.getLinks(pvcs, exporterPod, export, hostAndBase, external, cert, getVolumeName)
	}
	return nil, nil
}

func (ctrl *VMExportController) getLinks(pvcs []*corev1.PersistentVolumeClaim, exporterPod *corev1.Pod, export *exportv1.VirtualMachineExport, hostAndBase, linkType, cert string, getVolumeName getExportVolumeName) (*exportv1.VirtualMachineExportLink, error) {
	const scheme = "https://"
	if exporterPod == nil {
		return nil, nil
	}

	paths := CreateServerPaths(ContainerEnvToMap(exporterPod.Spec.Containers[0].Env))
	exportLink := &exportv1.VirtualMachineExportLink{
		Cert: cert,
	}

	if paths.VMURI != "" {
		exportLink.Manifests = append(exportLink.Manifests, exportv1.VirtualMachineExportManifest{
			Type: exportv1.AllManifests,
			Url:  scheme + path.Join(hostAndBase, linkType, paths.VMURI),
		})
	}
	if paths.SecretURI != "" {
		exportLink.Manifests = append(exportLink.Manifests, exportv1.VirtualMachineExportManifest{
			Type: exportv1.AuthHeader,
			Url:  scheme + path.Join(hostAndBase, linkType, paths.SecretURI),
		})
	}

	for _, pvc := range pvcs {
		if pvc == nil || exporterPod.Status.Phase != corev1.PodRunning {
			continue
		}

		volumeInfo := paths.GetVolumeInfo(pvc.Name)
		if volumeInfo == nil {
			log.Log.Warningf("Volume %s not found in paths", pvc.Name)
			continue
		}

		ev := exportv1.VirtualMachineExportVolume{
			Name: getVolumeName(pvc, export),
		}

		if volumeInfo.RawURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.RawURI),
			})
		}
		if volumeInfo.RawGzURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.RawGzURI),
			})
		}
		if volumeInfo.DirURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.Dir,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.DirURI),
			})
		}
		if volumeInfo.ArchiveURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.ArchiveGz,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.ArchiveURI),
			})
		}

		if len(ev.Formats) == 0 {
			log.Log.Warningf("No formats found for volume %s", pvc.Name)
			continue
		}

		exportLink.Volumes = append(exportLink.Volumes, ev)
	}

	return exportLink, nil
}

func (ctrl *VMExportController) internalExportCa() (string, error) {
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, components.KubeVirtExportCASecretName)
	obj, exists, err := ctrl.ConfigMapInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return "", err
	}
	cm := obj.(*corev1.ConfigMap).DeepCopy()
	bundle := cm.Data[caBundle]
	return strings.TrimSpace(bundle), nil
}

func (ctrl *VMExportController) getExternalLinkHostAndCert() (string, string) {
	for _, obj := range ctrl.IngressCache.List() {
		if ingress, ok := obj.(*networkingv1.Ingress); ok {
			if host := getHostFromIngress(ingress); host != "" {
				cert, _ := ctrl.getIngressCert(host, ingress)
				return host, cert
			}
		}
	}
	for _, obj := range ctrl.RouteCache.List() {
		if route, ok := obj.(*routev1.Route); ok {
			if host := getHostFromRoute(route); host != "" {
				cert, _ := ctrl.getRouteCert(host)
				return host, cert
			}
		}
	}
	return "", ""
}

func (ctrl *VMExportController) getIngressCert(hostName string, ing *networkingv1.Ingress) (string, error) {
	secretName := ""
	for _, tls := range ing.Spec.TLS {
		if tls.SecretName != "" {
			secretName = tls.SecretName
			break
		}
	}
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, secretName)
	obj, exists, err := ctrl.SecretInformer.GetStore().GetByKey(key)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	if secret, ok := obj.(*corev1.Secret); ok {
		return ctrl.getIngressCertFromSecret(secret, hostName)
	}
	return "", nil
}

func (ctrl *VMExportController) getIngressCertFromSecret(secret *corev1.Secret, hostName string) (string, error) {
	certBytes := secret.Data["tls.crt"]
	certs, err := cert.ParseCertsPEM(certBytes)
	if err != nil {
		return "", err
	}
	return ctrl.findCertByHostName(hostName, certs)
}

func (ctrl *VMExportController) getRouteCert(hostName string) (string, error) {
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, routeCAConfigMapName)
	obj, exists, err := ctrl.RouteConfigMapInformer.GetStore().GetByKey(key)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	if cm, ok := obj.(*corev1.ConfigMap); ok {
		cmString := cm.Data[routeCaKey]
		certs, err := cert.ParseCertsPEM([]byte(cmString))
		if err != nil {
			return "", err
		}
		return ctrl.findCertByHostName(hostName, certs)
	}
	return "", fmt.Errorf("not a config map")
}

func (ctrl *VMExportController) findCertByHostName(hostName string, certs []*x509.Certificate) (string, error) {
	now := time.Now()
	var latestCert *x509.Certificate
	for _, cert := range certs {
		if ctrl.matchesOrWildCard(hostName, cert.Subject.CommonName) {
			if latestCert == nil || (cert.NotAfter.After(latestCert.NotAfter) && cert.NotBefore.Before(time.Now())) {
				latestCert = cert
			}
		}
		for _, extension := range cert.Extensions {
			if extension.Id.String() == subjectAltNameId {
				value := strings.Map(func(r rune) rune {
					if unicode.IsPrint(r) && r <= unicode.MaxASCII {
						return r
					}
					return ' '
				}, string(extension.Value))
				names := strings.Split(value, " ")
				for _, name := range names {
					if ctrl.matchesOrWildCard(hostName, name) {
						if latestCert == nil || (cert.NotAfter.After(latestCert.NotAfter) && cert.NotBefore.Before(time.Now())) {
							latestCert = cert
						}
					}
				}
			}
		}
	}
	if latestCert != nil && latestCert.NotAfter.After(now) && latestCert.NotBefore.Before(now) {
		return ctrl.buildPemFromCert(latestCert, certs)
	}
	if len(certs) > 0 {
		return ctrl.buildPemFromAllCerts(certs, now)
	}
	return "", nil
}

func (ctrl *VMExportController) buildPemFromAllCerts(allCerts []*x509.Certificate, now time.Time) (string, error) {
	pemOut := strings.Builder{}
	for _, cert := range allCerts {
		if cert.NotAfter.After(now) && cert.NotBefore.Before(now) {
			pem.Encode(&pemOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		}
	}
	return strings.TrimSpace(pemOut.String()), nil
}

func (ctrl *VMExportController) buildPemFromCert(matchingCert *x509.Certificate, allCerts []*x509.Certificate) (string, error) {
	pemOut := strings.Builder{}
	pem.Encode(&pemOut, &pem.Block{Type: "CERTIFICATE", Bytes: matchingCert.Raw})
	if matchingCert.Issuer.CommonName != matchingCert.Subject.CommonName && !matchingCert.IsCA {
		// lookup issuer recursively, if not found a blank is returned.
		chain, err := ctrl.findCertByHostName(matchingCert.Issuer.CommonName, allCerts)
		if err != nil {
			return "", err
		}
		if _, err := pemOut.WriteString(chain); err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(pemOut.String()), nil
}

func (ctrl *VMExportController) matchesOrWildCard(hostName, compare string) bool {
	wildCard := fmt.Sprintf("*.%s", getDomainFromHost(hostName))
	return hostName == compare || wildCard == compare
}

func getDomainFromHost(host string) string {
	if index := strings.Index(host, "."); index != -1 {
		return host[index+1:]
	}
	return host
}

func getHostFromRoute(route *routev1.Route) string {
	if route.Spec.To.Name == components.VirtExportProxyServiceName {
		if len(route.Status.Ingress) > 0 {
			return route.Status.Ingress[0].Host
		}
	}
	return ""
}

func getHostFromIngress(ing *networkingv1.Ingress) string {
	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		if ing.Spec.DefaultBackend.Service.Name != components.VirtExportProxyServiceName {
			return ""
		}
		return ing.Spec.Rules[0].Host
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil && path.Backend.Service.Name == components.VirtExportProxyServiceName {
				if rule.Host != "" {
					return rule.Host
				}
			}
		}
	}
	return ""
}
