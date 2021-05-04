set -ex

function main() {
  TARGET_REPO=${TARGET_REPO:-"operator-framework/community-operators"}
  PACKAGE_DIR=${PACKAGE_DIR:-"deploy/olm-catalog/community-kubevirt-hyperconverged"}
  PR_TEMPLATE="https://raw.githubusercontent.com/operator-framework/community-operators/master/docs/pull_request_template.md"
  BOT_USERNAME="hco-bot"

  if [ -z "${TAGGED_VERSION}" ]
  then
    echo "ERROR: Tagged version was not provided."
    exit 1
  fi

  echo "The tagged HCO version is: ${TAGGED_VERSION}"
  BASE_TAGGED_VERSION=${TAGGED_VERSION%.*}.0

  echo "Copy tagged HCO version ${BASE_TAGGED_VERSION} bundle folder"
  if [ ! -d ${PACKAGE_DIR}/${BASE_TAGGED_VERSION} ]
  then
    echo "ERROR: Tagged version not found in ${TARGET_BRANCH}."
    exit 1
  fi

  cp -r ${PACKAGE_DIR}/${BASE_TAGGED_VERSION} ../hco_bundle

  cd ..
  tree hco_bundle

  PR_BODY=$(get_pr_body ${TAGGED_VERSION})

  echo "Switch to stable channel"
  sed -r -i "s/(.*channel.*: ).+/\1\stable/g" hco_bundle/metadata/annotations.yaml

  echo "Add annotation for community-operators index image"
  INDEX_IMAGE_VERSION=$(echo "${BASE_TAGGED_VERSION%.*}+3.4" | bc)
  echo "  com.redhat.openshift.versions: \"v${INDEX_IMAGE_VERSION}\"" >> hco_bundle/metadata/annotations.yaml

  echo "Bump version to ${TAGGED_VERSION}"
  CSV_FILE="hco_bundle/manifests/kubevirt-hyperconverged-operator.v${TAGGED_VERSION}.clusterserviceversion.yaml"
  mv hco_bundle/manifests/kubevirt-hyperconverged-operator.v${BASE_TAGGED_VERSION}.clusterserviceversion.yaml ${CSV_FILE} || true

  sed -i "s/^  name: kubevirt-hyperconverged-operator.*$/  name: kubevirt-hyperconverged-operator.v${TAGGED_VERSION}/g" ${CSV_FILE}
  sed -i "s/value: ${BASE_TAGGED_VERSION}/value: ${TAGGED_VERSION}/g" ${CSV_FILE}
  sed -i "s/version: ${BASE_TAGGED_VERSION}/version: ${TAGGED_VERSION}/g" ${CSV_FILE}

  PREVIOUS_BASE_VERSION=$(echo "${BASE_TAGGED_VERSION%.*}-0.1" | bc).0
  if [ ${TAGGED_VERSION##*.} != "0" ]
  # Add olm.skipRange annotation only if the release is a z-stream.
  then
    echo "Add olm.skipRange annotation to CSV"
    OLM_SKIP_RANGE="'>=${PREVIOUS_BASE_VERSION} <${TAGGED_VERSION}'"
    sed -r -i "s/^  annotations:.*$/  annotations:\n    olm.skipRange: $OLM_SKIP_RANGE/g" ${CSV_FILE}
  fi
  echo "New CSV to publish to community-operators:"
  cat ${CSV_FILE}


  echo "Login to GH account for GH CLI"
  echo ${HCO_BOT_TOKEN} > token.txt
  gh auth login --with-token < token.txt
  rm -f token.txt

  create_pr community-operators
  create_pr upstream-community-operators
}

function create_pr() {
  TARGET_FOLDER=$1

  echo "Clone the community operators repo"
  gh repo clone ${TARGET_REPO}

  echo "Update HCO manifests in ${TARGET_FOLDER}"
  BUNDLE_DIR=${TARGET_REPO##*/}/${TARGET_FOLDER}/community-kubevirt-hyperconverged/${TAGGED_VERSION}
  mkdir -p ${BUNDLE_DIR}
  cp -r hco_bundle/* ${BUNDLE_DIR}

  echo "Open a pull request to community operators"
  cd ${TARGET_REPO##*/}
  git config user.name "hco-bot"
  git config user.email "hco-bot@redhat.com"
  git add .
  BRANCH_NAME=${TARGET_FOLDER%%-*}-release_hco_v${TAGGED_VERSION}
  git checkout -b ${BRANCH_NAME}
  git status
  git commit -asm "Release Kubevirt HCO v${TAGGED_VERSION}"
  git push https://${HCO_BOT_TOKEN}@github.com/${BOT_USERNAME}/${TARGET_REPO##*/}.git
  echo "Create a pull request to community operator with tagged HCO version"
  gh pr create --title "[${TARGET_FOLDER%%-*}]: Release Kubevirt HCO v${TAGGED_VERSION}" --body "${PR_BODY}" \
    --repo ${TARGET_REPO} --head ${BOT_USERNAME}:${BRANCH_NAME}
  cd ..
  rm -rf ${TARGET_REPO##*/}
}

function get_pr_body() {
   wget ${PR_TEMPLATE}
   sed -ir "s/\[ \]/\[x\]/g; 0,/Is operator/d" pull_request_template.md
   sed -r "1s/^/Release Kubevirt HCO v$1\n/" pull_request_template.md
   rm -f pull_request_template.md
}

main
