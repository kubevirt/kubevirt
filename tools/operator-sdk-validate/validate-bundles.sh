#!/usr/bin/env bash
set -e

OPERATOR_SDK_VERSION=1.5.0
LOWEST_VERSION_TO_VALIDATE=1.3.0
ALL_VERSIONS=( $(ls -d /manifests/*/ | sort -V | cut -d '/' -f 3) )

curl -L -o /usr/local/bin/operator-sdk \
  "https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_linux_amd64"
sudo chmod +x /usr/local/bin/operator-sdk

function ver
{
  printf "%03d%03d%03d%03d" $(echo "$1" | tr '.' ' ')
}

for version in "${ALL_VERSIONS[@]}"
do
   if [ "$(ver ${version})" -ge "$(ver ${LOWEST_VERSION_TO_VALIDATE})" ]
   then
     set -x
     operator-sdk bundle validate /manifests/${version}
     set +x
   fi
done
