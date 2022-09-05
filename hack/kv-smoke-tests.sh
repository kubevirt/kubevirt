#!/usr/bin/env bash

set -euxo pipefail

INSTALLED_NAMESPACE=${INSTALLED_NAMESPACE:-"kubevirt-hyperconverged"}
OUTPUT_DIR=${ARTIFACT_DIR:-"$(pwd)/_out"}

source hack/common.sh
source cluster/kubevirtci.sh

echo "downloading the test binary"
BIN_DIR="$(pwd)/_out" && mkdir -p "${BIN_DIR}"
export BIN_DIR

TESTS_BINARY="$BIN_DIR/kv_smoke_tests.test"
curl -Lo "$TESTS_BINARY" "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/tests.test"
chmod +x "$TESTS_BINARY"

echo "create testing infrastructure"

cat <<EOF | ${CMD} apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: host-path-disk-alpine
  labels:
    kubevirt.io: ""
    os: "alpine"
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /tmp/hostImages/alpine
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: host-path-disk-custom
  labels:
    kubevirt.io: ""
    os: "custom"
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /tmp/hostImages/custom
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: v1
kind: Service
metadata:
  name: cdi-http-import-server
  namespace: ${INSTALLED_NAMESPACE}
  labels:
    kubevirt.io: "cdi-http-import-server"
spec:
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
  selector:
    kubevirt.io: cdi-http-import-server
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdi-http-import-server
  namespace: ${INSTALLED_NAMESPACE}
  labels:
    kubevirt.io: "cdi-http-import-server"
spec:
  selector:
    matchLabels:
      kubevirt.io: "cdi-http-import-server"
  replicas: 1
  template:
    metadata:
      labels:
        kubevirt.io: cdi-http-import-server
    spec:
      securityContext:
        runAsUser: 0
      serviceAccountName: kubevirt-testing
      containers:
        - name: cdi-http-import-server
          image: quay.io/kubevirt/cdi-http-import-server:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 80
              name: "http"
              protocol: "TCP"
          readinessProbe:
            tcpSocket:
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 10
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: disks-images-provider
  namespace: ${INSTALLED_NAMESPACE}
  labels:
    kubevirt.io: "disks-images-provider"
spec:
  selector:
    matchLabels:
      kubevirt.io: "disks-images-provider"
  template:
    metadata:
      labels:
        name: disks-images-provider
        kubevirt.io: disks-images-provider
      name: disks-images-provider
    spec:
      serviceAccountName: kubevirt-testing
      containers:
        - name: target
          image: quay.io/kubevirt/disks-images-provider:latest
          imagePullPolicy: Always
          volumeMounts:
          - name: images
            mountPath: /hostImages
          - name: local-storage
            mountPath: /local-storage
          securityContext:
            privileged: true
          readinessProbe:
            exec:
              command:
              - cat
              - /ready
            initialDelaySeconds: 10
            periodSeconds: 5
      volumes:
        - name: images
          hostPath:
            path: /tmp/hostImages
            type: DirectoryOrCreate
        - name: local-storage
          hostPath:
            path: /mnt/local-storage
            type: DirectoryOrCreate
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-block-storage-cirros
  labels:
    kubevirt.io: ""
    blockstorage: "cirros"
spec:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 1Gi
  local:
    path: /mnt/local-storage/cirros-block-device
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - node01
  persistentVolumeReclaimPolicy: Retain
  storageClassName: local-block
  volumeMode: Block
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubevirt-testing
  namespace: ${INSTALLED_NAMESPACE}
  labels:
    kubevirt.io: ""
EOF

cat <<EOF | ${CMD} apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubevirt-testing-cluster-admin
  labels:
    kubevirt.io: ""
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: kubevirt-testing
    namespace: ${INSTALLED_NAMESPACE}
EOF


${CMD} create configmap -n ${INSTALLED_NAMESPACE} kubevirt-test-config --from-file=hack/test-config.json --dry-run=client -o yaml | ${CMD} apply -f -

echo "waiting for testing infrastructure to be ready"
${CMD} wait deployment cdi-http-import-server -n "${INSTALLED_NAMESPACE}" --for condition=Available --timeout=10m
${CMD} wait pods -l "kubevirt.io=disks-images-provider" -n "${INSTALLED_NAMESPACE}" --for condition=Ready --timeout=10m

echo "starting tests"
${TESTS_BINARY} \
    -cdi-namespace="$INSTALLED_NAMESPACE" \
    -config=hack/test-config.json \
    -installed-namespace="$INSTALLED_NAMESPACE" \
    -junit-output="${OUTPUT_DIR}/junit_kv_smoke_tests.xml" \
    -kubeconfig="$KUBECONFIG" \
    -ginkgo.focus='(rfe_id:1177)|(rfe_id:273)|(rfe_id:151)' \
    -ginkgo.no-color \
    -ginkgo.seed=0 \
    -ginkgo.skip='(Slirp Networking)|(with CPU spec)|(with TX offload disabled)|(with cni flannel and ptp plugin interface)|(with ovs-cni plugin)|(test_id:1752)|(SRIOV)|(with EFI)|(Operator)|(GPU)|(DataVolume Integration)|(when virt-handler is not responsive)|(with default cpu model)|(should set the default MachineType when created without explicit value)|(should fail to start when a volume is backed by PVC created by DataVolume instead of the DataVolume itself)|(test_id:3468)|(test_id:3466)|(test_id:1015)|(rfe_id:393)|(test_id:4646)|(test_id:4647)|(test_id:4648)|(test_id:4649)|(test_id:4650)|(test_id:4651)|(test_id:4652)|(test_id:4654)|(test_id:4655)|(test_id:4656)|(test_id:4657)|(test_id:4658)|(test_id:4659)|(should obey the disk verification limits in the KubeVirt CR)' \
    -ginkgo.slow-spec-threshold=60s \
    -ginkgo.succinct \
    -ginkgo.flake-attempts=3 \
    -oc-path="$(which oc)" \
    -kubectl-path="$(which oc)" \
    -utility-container-prefix=quay.io/kubevirt \
    -test.timeout=3h \
    -ginkgo.timeout=3h \
    -artifacts=${ARTIFACT_DIR}/kubevirt_dump
