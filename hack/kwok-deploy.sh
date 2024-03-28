if [[ ${KWOK_DEPLOY} != "true" ]]; then
  exit
fi

# KWOK repository
KWOK_REPO=kubernetes-sigs/kwok
# Get latest
if [[ -n ${KWOK_LATEST_RELEASE} ]]; then
  KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
fi

_kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok.yaml"

_kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/stage-fast.yaml"

KUBEVIRT_KWOK_DIR=${KUBEVIRT_DIR}/hack/kwok

_kubectl kustomize ${KUBEVIRT_KWOK_DIR}