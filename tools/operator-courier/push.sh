#!/usr/bin/env bash
set -e

if [[ -z "$QUAY_USERNAME" ]] || [[ -z "$QUAY_PASSWORD" ]] || [[ -z "$QUAY_REPOSITORY" ]]; then
	echo "please set QUAY_USERNAME, QUAY_PASSWORD and QUAY_REPOSITORY"
	exit 1
fi

if [[ -z "$package_name" ]] || [[ -z "$csv_version" ]]; then
	echo "please set package_name and csv_version"
	exit 1
fi

echo "getting auth token from Quay"
AUTH_TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
	{
		"user": {
			"username": "'"${QUAY_USERNAME}"'",
			"password": "'"${QUAY_PASSWORD}"'"
		}
	}' | jq -r '.token')

echo "pushing bundle"
docker run \
	-e QUAY_USERNAME="${QUAY_USERNAME}" \
	-e QUAY_PASSWORD="${QUAY_PASSWORD}" \
	-e QUAY_REPOSITORY="${QUAY_REPOSITORY}" \
	operator-courier push "/manifests" "$QUAY_REPOSITORY" "$package_name" "$csv_version" "$AUTH_TOKEN"
echo "bundle pushed"
