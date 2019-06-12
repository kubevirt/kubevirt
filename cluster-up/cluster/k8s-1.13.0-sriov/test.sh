#!/bin/sh

FUNC_TEST_ARGS='--ginkgo.noColor --junit-output=/go/src/kubevirt.io/kubevirt/tests.junit.xml --ginkgo.focus=SRIOV --kubeconfig /etc/kubernetes/admin.conf' make functest
