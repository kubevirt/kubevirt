# Technical Design Input

# Hyperconverged Kubevirt cluster-wide Crypto Policy Setting

## Abstract

Starting from OCP/OKD 4.6, a [cluster-wide API](https://github.com/openshift/enhancements/blob/master/enhancements/kube-apiserver/tls-config.md) is available for cluster administrators to set TLS profiles for OCP/OKD core components. HCO, as an OCP/OKD layered product, will follow along OCP/OKD crypto policy cluster-wide setting, and use the same profile configured for the cluster’s control plane. Configuration of a TLS security profile ensures that OCP/OKD, as well as HCO and its sibling operators and operands, use cryptographic libraries that do not allow known insecure protocols, ciphers, or algorithms.

## Motivation

A really security-conscious cluster-admin, is seeking an option to configure ciphers that would apply to all OpenShift-proper components both for internal and external traffic (on OCP/OKD).

In an OpenShift cluster (OCP/OKD), secure communications is expected between system level pods and application level pods within the cluster (in other words east-west traffic in the control plane, system plane and data plane). In addition, secure communication is expected for traffic that ingress into the cluster from external clients as well as egressing from the cluster to external entities.
This RFE proposes to have all the HyperConverged Kubevirt components comply with Cluster-wide cryptographic policies on OpenShift (OCP/OKD) that provide a convenient way to ensure that OpenShift system pods and application pods can use cryptographic libraries that do not allow known insecure protocols, ciphers, or algorithms.

### Goals

1. Describe how a configuration of a TLS security profile for OCP/OKD control plane will be propagated to all the Hyperconverged Kubevirt control plane components, using existing OCP/OKD API.

### Non-Goals

1. Describe the technical implementation of TLS security policy for each operand of a sibling operator of HCO (just a quick hint).
2. The Mozilla Recommended Configurations mandate also `Certificate type` and `Certificate lifespan` as part of each profile while this is not covered by [OpenShift implementation](https://github.com/openshift/enhancements/blob/master/enhancements/kube-apiserver/tls-config.md) so it's not covered by this API.

## Overview

Currently, TLS security policy can be configured for the following OCP/OKD core components:

* Ingress Controller
* Kubelet
* API Server

While the TLS security policy for the Ingress Controller and the kubelet are independent, configuring TLS security policy for the API Server is automatically propagating it for the following core components:

* Kubernetes API server
* Kubernetes controller manager
* Kubernetes scheduler
* OpenShift API server
* OpenShift OAuth API server
* OpenShift OAuth server

The TLS security profiles are based on [Mozilla Recommended Configurations](https://wiki.mozilla.org/Security/Server_Side_TLS):
* Old - intended for use with legacy clients of libraries; requires a minimum TLS version of 1.0
* Intermediate - the default profile for all components; requires a minimum TLS version of 1.2
* Modern - intended for use with clients that don’t need backward compatibility; requires a minimum TLS version of 1.3. Unsupported in OCP/OKD 4.8 and below.

## API Schema

```bash
$ oc explain apiserver.spec.tlsSecurityProfile
KIND:     APIServer
VERSION:  config.openshift.io/v1

RESOURCE: tlsSecurityProfile <Object>

DESCRIPTION:
     tlsSecurityProfile specifies settings for TLS connections for externally
     exposed servers. If unset, a default (which may change between releases) is
     chosen. Note that only Old, Intermediate and Custom profiles are currently
     supported, and the maximum available MinTLSVersions is VersionTLS12.

FIELDS:
   custom	<>
     custom is a user-defined TLS security profile. Be extremely careful using a
     custom profile as invalid configurations can be catastrophic. An example
     custom profile looks like this:
     ciphers: - ECDHE-ECDSA-CHACHA20-POLY1305 - ECDHE-RSA-CHACHA20-POLY1305 -
     ECDHE-RSA-AES128-GCM-SHA256 - ECDHE-ECDSA-AES128-GCM-SHA256 minTLSVersion:
     TLSv1.1

   intermediate	<>
     intermediate is a TLS security profile based on:
     https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28recommended.29
     and looks like this (yaml):
     ciphers: - TLS_AES_128_GCM_SHA256 - TLS_AES_256_GCM_SHA384 -
     TLS_CHACHA20_POLY1305_SHA256 - ECDHE-ECDSA-AES128-GCM-SHA256 -
     ECDHE-RSA-AES128-GCM-SHA256 - ECDHE-ECDSA-AES256-GCM-SHA384 -
     ECDHE-RSA-AES256-GCM-SHA384 - ECDHE-ECDSA-CHACHA20-POLY1305 -
     ECDHE-RSA-CHACHA20-POLY1305 - DHE-RSA-AES128-GCM-SHA256 -
     DHE-RSA-AES256-GCM-SHA384 minTLSVersion: TLSv1.2

   modern	<>
     modern is a TLS security profile based on:
     https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility and
     looks like this (yaml):
     ciphers: - TLS_AES_128_GCM_SHA256 - TLS_AES_256_GCM_SHA384 -
     TLS_CHACHA20_POLY1305_SHA256 minTLSVersion: TLSv1.3 NOTE: Currently
     unsupported.

   old	<>
     old is a TLS security profile based on:
     https://wiki.mozilla.org/Security/Server_Side_TLS#Old_backward_compatibility
     and looks like this (yaml):
     ciphers: - TLS_AES_128_GCM_SHA256 - TLS_AES_256_GCM_SHA384 -
     TLS_CHACHA20_POLY1305_SHA256 - ECDHE-ECDSA-AES128-GCM-SHA256 -
     ECDHE-RSA-AES128-GCM-SHA256 - ECDHE-ECDSA-AES256-GCM-SHA384 -
     ECDHE-RSA-AES256-GCM-SHA384 - ECDHE-ECDSA-CHACHA20-POLY1305 -
     ECDHE-RSA-CHACHA20-POLY1305 - DHE-RSA-AES128-GCM-SHA256 -
     DHE-RSA-AES256-GCM-SHA384 - DHE-RSA-CHACHA20-POLY1305 -
     ECDHE-ECDSA-AES128-SHA256 - ECDHE-RSA-AES128-SHA256 -
     ECDHE-ECDSA-AES128-SHA - ECDHE-RSA-AES128-SHA - ECDHE-ECDSA-AES256-SHA384 -
     ECDHE-RSA-AES256-SHA384 - ECDHE-ECDSA-AES256-SHA - ECDHE-RSA-AES256-SHA -
     DHE-RSA-AES128-SHA256 - DHE-RSA-AES256-SHA256 - AES128-GCM-SHA256 -
     AES256-GCM-SHA384 - AES128-SHA256 - AES256-SHA256 - AES128-SHA - AES256-SHA
     - DES-CBC3-SHA minTLSVersion: TLSv1.0

   type	<string>
     type is one of Old, Intermediate, Modern or Custom. Custom provides the
     ability to specify individual TLS security profile parameters. Old,
     Intermediate and Modern are TLS security profiles based on:
     https://wiki.mozilla.org/Security/Server_Side_TLS#Recommended_configurations
     The profiles are intent based, so they may change over time as new ciphers
     are developed and existing ciphers are found to be insecure. Depending on
     precisely which ciphers are available to a process, the list may be
     reduced. Note that the Modern profile is currently not supported because it
     is not yet well adopted by common software libraries.

```


Example for using old TLS security profile:
```yaml
apiVersion: config.openshift.io/v1
kind: APIServer
 ...
spec:
  tlsSecurityProfile:
    old: {}
    type: Old
 ...
```

Example for using a custom TLS security profile:
```yaml
apiVersion: config.openshift.io/v1
kind: APIServer
metadata:
  name: cluster
spec:
  tlsSecurityProfile:
    type: Custom
    custom:
      ciphers:
      - ECDHE-ECDSA-CHACHA20-POLY1305
      - ECDHE-RSA-CHACHA20-POLY1305
      - ECDHE-RSA-AES128-GCM-SHA256
      - ECDHE-ECDSA-AES128-GCM-SHA256
      minTLSVersion: VersionTLS11
```

*When specifying a _Custom_ profile, a ciphers list must be provided, as well as a minimum TLS version.*


## Proposal

Hyperconverged-cluster-operator (HCO) will read the global configuration for TLS security profile of the APIServer, without storing it in HCO CR, and will propagate the .spec.tlsSecurityProfile stanza to all underlying HCO managed custom resources, and will reconcile them, to achieve alignment and consistency of security profiles between Hyperconverged Kubevirt and OCP/OKD control plane components.
The idea is that the code to read the OpenShift API can be coded once in HCO without the need to replicate it for each single component. Moreover, a few components like kubevirt/kubevirt are pretty independent of OpenShift-specific APIs so HCO looks like a good candidate for this.

The underlying HCO managed custom resources to which HCO will propagate the security profile stanza to, are:
* `kubevirt.kubevirt.io/kubevirt-kubevirt-hyperconverged`
* `cdis.cdi.kubevirt.io/cdi-kubevirt-hyperconverged`
* `networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io/cluster`
* `ssps.ssp.kubevirt.io/ssp-kubevirt-hyperconverged`

HCO will use an informer to watch for changes on APIServer cluster custom resources for changes in `spec.tlsSecurityProfile` API and eventually propagate it to other operators at any time.

Example for KV CR with the new stanza
```
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  annotations:
    kubevirt.io/latest-observed-api-version: v1
    kubevirt.io/storage-observed-api-version: v1alpha3
  labels:
    app: kubevirt-hyperconverged
    app.kubernetes.io/component: compute
    app.kubernetes.io/managed-by: hco-operator
    app.kubernetes.io/part-of: hyperconverged-cluster
    app.kubernetes.io/version: v4.10.0
  name: kubevirt-kubevirt-hyperconverged
  namespace: kubevirt-hyperconverged
  ownerReferences:
  - apiVersion: hco.kubevirt.io/v1beta1
    blockOwnerDeletion: true
    controller: true
    kind: HyperConverged
    name: kubevirt-hyperconverged
spec:
  certificateRotateStrategy:
    <redacted>
  configuration:
    <redacted>
    machineType: pc-q35-rhel8.4.0
    migrations:
      <redacted>
    network:
    defaultNetworkInterface: masquerade
    obsoleteCPUModels:
      <redacted>
    selinuxLauncherType: virt_launcher.process
    smbios:
      <redacted>
  customizeComponents: {}
  tlsSecurityProfile:
    type: Intermediate
    intermediate: {}
  uninstallStrategy: BlockUninstallIfWorkloadsExist
  workloadUpdateStrategy:
    batchEvictionInterval: 1m0s
    batchEvictionSize: 10
    workloadUpdateMethods:
    - LiveMigrate
```


HCO will then be reconciling this value for all underlying HCO managed CRs, in their respective stanza.

The custom resources that the TLS Security Profile stanza will be propagated to, are:
* `KubeVirt kubevirt.io/v1 - kubevirt.kubevirt.io/kubevirt-kubevirt-hyperconverged`
* `CDI cdi.kubevirt.io/v1beta1 - cdis.cdi.kubevirt.io/cdi-kubevirt-hyperconverged`
* `NetworkAddonsConfig networkaddonsoperator.network.kubevirt.io/v1 - networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io/cluster`
* `SSP ssp.kubevirt.io/v1beta1 - ssps.ssp.kubevirt.io/ssp-kubevirt-hyperconverged`


## Notes

1. Any modification for the TLS Security Profile for HCO managed components must be made through APIServer CR and they can happen at any time (day-2 configurations).
2. Modification of TLS Security Profile for a specific component will be possible by using a [jsonpatch annotation](https://github.com/kubevirt/hyperconverged-cluster-operator/blob/main/docs/cluster-configuration.md#jsonpatch-annotations) on HCO CR. However, this configuration is unsupported and should block upgrades.
3. This feature/configuration is only applicable to OCP/OKD platform, not plain kubernetes, since APIServer (apiversion: config.openshift.io/v1) is an openshift resource only.
4. [https://wiki.mozilla.org/Security/Server_Side_TLS#Recommended_configurations](https://wiki.mozilla.org/Security/Server_Side_TLS#Recommended_configurations) also mentions Certificate type and Certificate lifespan but [https://github.com/openshift/enhancements/blob/master/enhancements/kube-apiserver/tls-config.md](https://github.com/openshift/enhancements/blob/master/enhancements/kube-apiserver/tls-config.md) doesn't, so we can safely ignore them


## Open Questions
1. In plain kubernetes HCO deployment, will we use the default intermediate profile for all HCO managed components, or give the option for the administrator to set it on the HCO CR, which will propagate to all components?
   1. The TLS Security Profile configuration feature will be available only on OCP/OKD platforms.
   2. We can decide to expose the API to let the user optionally fine-tune/override the configuration for all the Hyperconverged Kubevirt having HCO reading the OpenShift API server one as its default only if the user didn't express custom values. This will work also on plain k8s.

2. [OpenShift API design](https://github.com/openshift/enhancements/blob/master/enhancements/kube-apiserver/tls-config.md) explicitly mentions API servers, but it never mentions admission webhooks that are also TLS protected.
Is there any technical reason for us to ignore the admission webhooks? 

## Implementation hints

If the TLS implementation is based on the standard golang crypto library (as expected), the TLS configuration is performed via tls.Config struct: [https://pkg.go.dev/crypto/tls#Config](https://pkg.go.dev/crypto/tls#Config)
which contains `MinVersion` and `CipherSuites` which should be tuned according to the chosen tlsSecurityProfile.

For instance intermediate will set: [https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28recommended.29](https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28recommended.29)
If you can/want to import `github.com/openshift/api/config/v1`, the three standard configurations lists are already coded [here](https://github.com/openshift/api/blob/master/config/v1/types_tlssecurityprofile.go#L203)



