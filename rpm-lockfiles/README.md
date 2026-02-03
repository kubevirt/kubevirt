# RPM Lock Files

This directory contains JSON lock files that specify exact RPM package versions
for reproducible container image builds. These files replace the bazeldnf-generated
rpmtree definitions.

## File Format

Each lock file is named `<package_set>-<architecture>.lock.json` and contains:

```json
{
  "schema_version": "1.0",
  "architecture": "x86_64",
  "package_set": "launcherbase",
  "generated": "2026-02-03T12:00:00Z",
  "generator": "rpm-freeze-native.sh",
  "packages": [
    {
      "name": "libvirt-client",
      "epoch": "0",
      "version": "11.9.0",
      "release": "1.el9",
      "arch": "x86_64",
      "sha256": "abc123...",
      "filename": "libvirt-client-11.9.0-1.el9.x86_64.rpm"
    }
  ]
}
```

## Package Sets

| Package Set | Description | Architectures |
|-------------|-------------|---------------|
| testimage | Test tooling image | x86_64, aarch64, s390x |
| libvirt-devel | Build-time libvirt headers | x86_64, aarch64, s390x |
| sandboxroot | Build sandbox environment | x86_64, aarch64, s390x |
| launcherbase | virt-launcher base | x86_64, aarch64, s390x |
| handlerbase | virt-handler base | x86_64, aarch64, s390x |
| passt_tree | Passt networking | x86_64, aarch64, s390x |
| libguestfs-tools | Disk management tools | x86_64, s390x |
| exportserverbase | Export server base | x86_64, aarch64, s390x |
| pr-helper | PR helper | x86_64, aarch64 |
| sidecar-shim | Sidecar base | x86_64, aarch64, s390x |

## Generating Lock Files

To regenerate all lock files:

```bash
hack/dockerized "./hack/rpm-freeze-all.sh"
```

To regenerate a specific package set:

```bash
hack/dockerized "./hack/rpm-freeze-native.sh x86_64 launcherbase"
```

## Verification

To verify lock files match cached RPMs:

```bash
hack/dockerized "./hack/rpm-verify.sh"
```

## Updating Package Versions

1. Edit version variables in `hack/rpm-packages.sh`
2. Run `./hack/rpm-freeze-all.sh` to regenerate lock files
3. Review and commit the updated lock files
