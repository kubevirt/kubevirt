if [[ ${IP_MODE} == "ipv4" ]]; then
    # DIND_SUBNET="10.192.0.0"
    # DIND_SUBNET_SIZE=16
    :
else
    # DinD subnet (expected to be /64)
    DIND_SUBNET="${DIND_SUBNET:-fd00:10::}"
fi

# Apiserver port
# APISERVER_PORT=${APISERVER_PORT:-8080}

# Number of nodes. 0 nodes means just one master node.
# In case of NUM_NODES=0 'node-role.kubernetes.io/master' taint is removed
# from the master node.
NUM_NODES=${NUM_NODES:-0}

# Use non-dockerized build
# KUBEADM_DIND_LOCAL=

# Image name base for k-d-c
DIND_IMAGE_BASE="${DIND_IMAGE_BASE:-sebassch/kubeadm-dind-cluster}"

# Specify DIND image to use. mirantis/kubeadm-dind-cluster:local
# is the one that is built locally using build/build-local.sh
DIND_IMAGE="${DIND_IMAGE:-${DIND_IMAGE_BASE}:latest}"

# Set DOWNLOAD_KUBECTL to non-empty string to download
# kubectl. Should not be used with BUILD_KUBEADM / BUILD_HYPERKUBE
DOWNLOAD_KUBECTL=y

# Set to non-empty string to enable building kubeadm
#BUILD_KUBEADM=n

# Set to non-empty string to enable building hyperkube
#BUILD_HYPERKUBE=n

# Use pre-built Kubernetes binaries (hyperkube and kubeadm) on the
# host, located in the specified directory. When this environment
# variable is set, BUILD_KUBEADM and BUILD_HYPERKUBE will be ignored.
# This will not work with a remote docker engine (e.g. started via
# docker-machine on GCE) unless the file is placed on the target machine.
# DIND_K8S_BIN_DIR="${DIND_K8S_BIN_DIR:-}"

# Set custom URL for Dashboard yaml file
# DASHBOARD_URL="${DASHBOARD_URL:-https://rawgit.com/kubernetes/dashboard/bfab10151f012d1acc5dfb1979f3172e2400aa3c/src/deploy/kubernetes-dashboard.yaml}"

# CNI plugin to use (bridge, flannel, calico, calico-kdd, weave). Defaults to 'bridge'
# In case of 'bridge' plugin, additional hacks are employed to bridge
# DIND containers together.
CNI_PLUGIN="${CNI_PLUGIN:-flannel}"

# When using Calico with Kubernetes as the datastore (calico-kdd) your
# controller manager needs to be started with `--cluster-cidr=192.168.0.0/16`.
# More information here: http://docs.projectcalico.org/v2.3/getting-started/kubernetes/installation/hosted/kubernetes-datastore/
POD_NETWORK_CIDR="10.244.0.0/16"

# Set SKIP_SNAPSHOT to non-empty string to skip making the snapshot.
# This may be useful for CI environment where the cluster is never
# restarted after it's created.
SKIP_SNAPSHOT=y

# Disable parallel running of e2e tests. Use this if you use a resource
# constrained machine for e2e tests and get some flakes.
# DIND_NO_PARALLEL_E2E=y

# Any options to be passed to the docker run both on init and reup.
# By default it's empty
# MASTER_EXTRA_OPTS="  "

# Define which DNS service to run
# possible values are coredns (default) and kube-dns
DNS_SERVICE="${DNS_SERVICE:-coredns}"

# Feature Gates
# This value will be passed to kube-apiserver, kube-controller-manager and kube-scheduler
# you can set special value 'none' not to set any feature gates on them.
# FEATURE_GATES=""

# Kubelet Feature Gates
# you can set special value 'none' not to set any feature gates on kubelet.
# KUBELET_FEATURE_GATES=""

# You can configure extra component args for kube-apiservers
# APISERVER_underscored_option_name will be converted --hyphenated-option-name
# e.g. APISERVER_admission_control=xxx,yyy -> --admission-control=xxx,yyy
# APISERVER_xxx_yyy=zzz

# Extra component args for kube-controller-manager
# CONTROLLER_MANAGER_underscored_option_name will be converted --hyphenated-option-name
# CONTROLLER_MANAGER_xxx=yyy

# Extra component args for kube-scheduler
# SCHEDULER_underscored_option_name will be converted --hyphenated-option-name
# SCHEDULER_xxx=yyy

# Enable Ceph support. DANGER: you must take care of unmapping all
# the RBDs (e.g. by removing all the pods that use RBDs) before
# stopping / restarting the cluster, or they'll get stuck possibly
# blocking even system reboot.
# ENABLE_CEPH=n
