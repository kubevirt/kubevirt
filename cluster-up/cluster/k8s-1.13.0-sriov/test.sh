#!/bin/sh

PATH=~/go/bin:$PATH
go get -u github.com/onsi/ginkgo/ginkgo
FUNC_TEST_ARGS='--ginkgo.noColor --junit-output=/go/src/kubevirt.io/kubevirt/tests.junit.xml --ginkgo.focus=Multus.*sriov --kubeconfig /etc/kubernetes/admin.conf' make functest
