#!/usr/bin/env bash

function patch_og() {
  CSV_VERSION=$1

  CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep ${CSV_VERSION})
  ALL_NAMESPACES_INSTALLMODE=$(${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{.spec.installModes[?(@.type=="AllNamespaces")].supported}')
  OG_SPEC=$(${CMD} get og ${HCO_OPERATORGROUP_NAME} -n ${HCO_NAMESPACE} -o jsonpath='{.spec}')
  if [ "${ALL_NAMESPACES_INSTALLMODE}" == "true" ] && [ "${OG_SPEC}" != "{}" ]
  then
    echo "CSV is supporting AllNamespaces InstallMode but OperatorGroup is watching a single namespace. Patching OperatorGroup..."
    ${CMD} patch og "${HCO_OPERATORGROUP_NAME}" -n ${HCO_NAMESPACE} --type json -p '[{"op": "remove", "path": "/spec/targetNamespaces"}]'
    OG_PATCHED=1
  elif [ "${ALL_NAMESPACES_INSTALLMODE}" == "false" ] && [ "${OG_SPEC}" == "{}" ]
  then
    echo "CSV is not supporting AllNamespaces InstallMode, and OperatorGroup is watching all namespaces. Patching OperatorGroup..."
    ${CMD} patch og "${HCO_OPERATORGROUP_NAME}" -n ${HCO_NAMESPACE} --type json -p "[{\"op\": \"replace\", \"path\": \"/spec/targetNamespaces\", \"value\": [${HCO_NAMESPACE}]}]"
    OG_PATCHED=1
  else
    OG_PATCHED=0
  fi
}