# mirror-csv-release
A helper script to mirror an operator-based application to a different repository.

Operator-based applications come with a metadata image shipping its [CSV](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#what-is-a-cluster-service-version-csv) file for [OLM](https://docs.openshift.com/container-platform/4.2/operators/understanding-olm/olm-understanding-olm.html#olm-overview_olm-understanding-olm).

This script assumes that the metadata image is based on [the operator registry image](https://github.com/operator-framework/operator-registry).
Giving the URL of an existing metadata image, this tool extracts the list of images listed in the CSV (the CSV should provide the list of [all the needed images](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/contributors/design-proposals/related-images.md)) and mirrors all of them to a different repository.
The metadata image is patched to point to the new registry, rebuilt and published there as well.

## Usage
```
./mirror_csv_release.sh --help

Usage:

mirror_csv_release.sh [options] SOURCE_BUNDLE_REGISTRY DEST_PREFIX
mirror_csv_release.sh [options] --appregistry DEST_PREFIX

Mirror container images listed in an operator bundle.

Positional arguments:
    [SOURCE_BUNDLE_REGISTRY]
        Will be used for:
          - Extract bundle files
          - Get the list of images listed in the bundle
          - Replacement string when replace the source registry and namespace
            with the destination registry and namespace.
        SOURCE_BUNDLE_REGISTRY should be omitted when fetching the content directly from an appregistry (--appregistry)

        [e.g quay.io/openshift-cnv/container-native-virtualization-hco-bundle-registry:v2.2.0-181]

    DEST_PREFIX
       Will replace the PREFIX in the pull URL of the images that were found
       in the csv files.

       [e.g quay.io/tiraboschi/]

Optional arguments:
    --dest-secret USERNAME[:PASSWORD]
        for accessing the destination registry

    --version-filter
        to mirror just a specific version

    --appregistry
        to fetch the source CSVs from the specified appregistry instead of a bundle image

    -d,--debug
        run in debug mode

    --dry-run
        dry-run mode

    --baseurl
        appregistry API baseurl, used only in appregistry mode
        default: https://quay.io/cnr

    --appregistry-name
        appregistry name, used only in appregistry mode
        default: redhat-operators

    --package-name)
        package name, used only in appregistry mode
        default: kubevirt-hyperconverged

    --packageversion
        package version, used only in appregistry mode
        default: 1.0.0

    --bundle-registry-name
        name of the destination bundle registry image, used only in appregistry mode
        default: bundle-registry

    --bundle-registry-tag
        tag of the destination bundle registry image, used only in appregistry mode
        default: 1.0.0

Example:
    mirror_csv_release.sh  --version-filter 2.2.0 quay.io/openshift-cnv/container-native-virtualization-hco-bundle-registry:v2.2.0-181 quay.io/tiraboschi/
    mirror_csv_release.sh  --appregistry --version-filter 2.2.0 --packageversion 4.0.0 --bundle-registry-name my-bundle-registry --bundle-registry-tag 1.0.0 quay.io/tiraboschi/
```

## Requirements
To be installed on the bastion host used for the mirroring process:
+ [Podman](https://github.com/containers/libpod)
+ [Skopeo >= 0.1.40](https://github.com/containers/skopeo)
+ Python3

On RHEL8/Centos8/Fedora systems you can install them with:
`sudo dnf install podman skopeo python3`
