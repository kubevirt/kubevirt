#!/bin/bash

while getopts r:t:i:n: flag; do
	case "${flag}" in
	r) LIBVIRT_REPO=${OPTARG} ;;
	t) LIBVIRT_TAG=${OPTARG} ;;
	i) GITHUB_RUN_ID=${OPTARG} ;;
	n) GITHUB_RUN_NUMBER=${OPTARG} ;;
	*)
		echo "Invalid option"
		exit 1
		;;
	esac
done

if [ -z "$LIBVIRT_REPO" ] || [ -z "$LIBVIRT_TAG" ] || [ -z "$GITHUB_RUN_ID" ] || [ -z "$GITHUB_RUN_NUMBER" ]; then
	echo "Usage: $0 -r <LIBVIRT_REPO> -t <LIBVIRT_TAG> -i <GITHUB_RUN_ID> -n <GITHUB_RUN_NUMBER>"
	exit 1
fi

echo "Building custom libvirt RPMs..."

# Clone libvirt repository (needed for submodules)
TAG_REF=${LIBVIRT_TAG}
echo "Cloning libvirt repository for tag: $TAG_REF"

# Clone with minimal depth but fetch the specific tag
git clone --no-single-branch $LIBVIRT_REPO libvirt-src
cd libvirt-src

# Fetch and checkout the specific tag
git fetch origin refs/tags/$TAG_REF:refs/tags/$TAG_REF
git checkout "$TAG_REF"
echo "Checked out libvirt tag: $TAG_REF"
# Get commit info for metadata
COMMIT_SHA=$(git rev-parse HEAD)
COMMIT_SHA_SHORT=$(git rev-parse --short HEAD)
COMMIT_DATE=$(git show -s --format=%ci HEAD)
echo "Tag: $TAG_REF"
echo "Commit: $COMMIT_SHA ($COMMIT_DATE)"

# Updating the release field in libvirt.spec.in to include the short commit hash
echo "Local commits to track release"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
sed -i "s/Release: 1%{?dist}/Release: 1.$COMMIT_SHA_SHORT%{?dist}/" libvirt.spec.in
git commit -am "local change: add last commit hash to dist release"

# Initialize and update submodules
echo "Initializing git submodules..."
git submodule init
git submodule update
echo "Submodules updated successfully"

cd ..

# Create directory for RPMs
mkdir -p rpms-libvirt

# Start build environment
docker run -td \
	--name libvirt-build \
	-v $(pwd)/libvirt-src:/libvirt-src \
	registry.gitlab.com/libvirt/libvirt/ci-centos-stream-9

# Build libvirt RPMs
docker exec -w /libvirt-src libvirt-build bash -c "
  set -e
  echo 'Adding /libvirt-src dirs to git safe directories...'
  git config --global --add safe.directory /libvirt-src
  git config --global --add safe.directory /libvirt-src/subprojects/keycodemapdb

  echo 'Building libvirt from source...'
  meson build
  ninja -C build dist
  
  echo 'Installing build dependencies...'
  dnf update -y
  dnf install -y createrepo hostname rpmdevtools
  dnf builddep -y /libvirt-src/build/libvirt.spec
  
  echo 'Creating RPMs...'
  rpmbuild -ta /libvirt-src/build/meson-dist/libvirt-*.tar.xz
  
  echo 'Creating repository metadata with gzip compression...'
  createrepo --general-compress-type=gz --checksum=sha256 /root/rpmbuild/RPMS/x86_64
  
  echo 'Build completed successfully'
"

# Copy RPMs to local directory
docker cp libvirt-build:/root/rpmbuild/RPMS/. rpms-libvirt/

# Create metadata file
cat >rpms-libvirt/build-info.json <<EOF
{
  "libvirt_version": "0:11.7.0-1.$COMMIT_SHA_SHORT.el9",
  "commit_sha": "$COMMIT_SHA",
  "commit_date": "$COMMIT_DATE",
  "build_date": "$(date -Iseconds)",
  "github_run_id": "${GITHUB_RUN_ID}",
  "github_run_number": "${GITHUB_RUN_NUMBER}",
  "has_submodules": true
}
EOF

# Cleanup build container
docker rm -f libvirt-build

echo "libvirt RPMs built successfully:"
ls -la rpms-libvirt/
cat rpms-libvirt/build-info.json
