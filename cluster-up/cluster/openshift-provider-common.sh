ln_kubeconfig () {
    kube_dir=/root/.kube
    kubeconfig=/root/install/auth/kubeconfig
    ${KUBEVIRTCI_PATH}/container.sh mkdir -p $kube_dir
    ${KUBEVIRTCI_PATH}/container.sh ln -s $kubeconfig $kube_dir
}
