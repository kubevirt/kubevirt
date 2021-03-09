set -ex

function main() {
  TARGET_REPO=${TARGET_REPO:-"operator-framework/community-operators"}
  PACKAGE_DIR=${PACKAGE_DIR:-"deploy/olm-catalog/community-kubevirt-hyperconverged"}
  PR_TEMPLATE="https://raw.githubusercontent.com/operator-framework/community-operators/master/docs/pull_request_template.md"
  BOT_USERNAME="hco-bot"

  TAGGED_VERSION=${GIT_TAG##*/v}
  echo "The tagged HCO version is: ${TAGGED_VERSION}"
  BASE_TAGGED_VERSION=${TAGGED_VERSION%.*}.0

  echo "Copy tagged HCO version ${BASE_TAGGED_VERSION} bundle folder"
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
  mv hco_bundle/manifests/kubevirt-hyperconverged-operator.v${BASE_TAGGED_VERSION}.clusterserviceversion.yaml ${CSV_FILE}
  sed -r -i "s/${BASE_TAGGED_VERSION}/${TAGGED_VERSION}/g" ${CSV_FILE}

  echo "Add olm.skipRange annotation to CSV"
  PREVIOUS_BASE_VERSION=$(echo "${BASE_TAGGED_VERSION%.*}-0.1" | bc).0
  OLM_SKIP_RANGE="'>=${PREVIOUS_BASE_VERSION} <${TAGGED_VERSION}'"
  sed -r -i "s/^  annotations:.*$/  annotations:\n    olm.skipRange: $OLM_SKIP_RANGE/g" ${CSV_FILE}

  echo "New CSV to publish to community-operators:"
  cat ${CSV_FILE}


  echo "Login to GH account for GH CLI"
  echo ${HCO_TOKEN} > token.txt
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
  cp -r hco_bundle ${TARGET_REPO##*/}/${TARGET_FOLDER}/community-kubevirt-hyperconverged/${TAGGED_VERSION}

  echo "Open a pull request to community operators"
  cd ${TARGET_REPO##*/}
  git config user.name "hco-bot"
  git config user.email "hco-bot@redhat.com"
  git add .
  BRANCH_NAME=${TARGET_FOLDER%%-*}-update_hco_to_${TAGGED_VERSION}
  git checkout -b ${BRANCH_NAME}
  git status
  git commit -asm "Update Kubevirt HCO to ${TAGGED_VERSION}"
  git push https://${HCO_TOKEN}@github.com/${BOT_USERNAME}/${TARGET_REPO##*/}.git
  echo "Create a pull request to community operator with tagged HCO version"
  gh pr create --title "[${TARGET_FOLDER%%-*}]: Update HCO to ${TAGGED_VERSION}" --body "${PR_BODY}" \
    --repo ${TARGET_REPO} --head ${BOT_USERNAME}:${BRANCH_NAME}
  cd ..
  rm -rf ${TARGET_REPO##*/}
}

function get_pr_body() {
   wget ${PR_TEMPLATE}
   sed -ir "s/\[ \]/\[x\]/g; 0,/Is operator/d" pull_request_template.md
   sed -r "1s/^/Update Kubevirt HCO to version $1\n/" pull_request_template.md
   rm -rf pull_request_template.md
}

main
