#!/bin/bash -e

: {$KUBEVIRTNAMESPACE:="openshift-cnv"}
_kubectl=$(which kubectl)
_oc=$(which oc)

[[ -z ${_kubectl} && -z ${_oc} ]] && echo "OC or KUBECTL is not installed" && exit 1

[ -z "${KUBECONFIG}" ] && echo "KUBECONFIG is not set" && exit 1

rm -rf _out
hack/dockerized hack/build-func-tests.sh
make manifests
rm -f _out/manifests/testing/cdi-*
rm -f _out/manifests/testing/kubevirt-config.yaml
sed -i "s/namespace: kubevirt/namespace: ${KUBEVIRTNAMESPACE}/g" _out/manifests/testing/*.yaml
_out/tests/tests.test --cdi-namespace=${KUBEVIRTNAMESPACE} \
	--deploy-testing-infra \
	--ginkgo.seed=0 \
	--installed-namespace=${KUBEVIRTNAMESPACE} \
	--junit-output=${PWD}/xunit_results.xml \
	--kubeconfig=${KUBECONFIG} \
	--kubectl-path=${_kubectl} \
	--oc-path=${_oc} \
	--path-to-testing-infra-manifests=${PWD}/_out/manifests/testing

