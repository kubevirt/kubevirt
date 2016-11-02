#!/bin/bash
vagrant ssh master -c "sudo cat /etc/kubernetes/kubelet.conf" > cluster/.kubeconfig
kubectl --kubeconfig=cluster/.kubeconfig $@
