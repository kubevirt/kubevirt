"""
KubeVirt wrapper for bazeldnf tar2files with /lib64 usrmerge path normalization.

CentOS Stream 9 packages like libgcc and libtirpc install shared libraries to
./lib64/ inside their RPM tar, because /lib64 is a usrmerge symlink to /usr/lib64.
bazeldnf v0.99.2-rc1 has a bug in PrefixFilter (pkg/rpm/tar.go): it detects
./lib64 entries when searching for /usr/lib64 prefix, but does not normalize the
path before the fileMap lookup, so the files are reported as missing.

This wrapper adds a Python preprocessing step that renames ./lib64/* -> ./usr/lib64/*
in the combined rpmtree tar before tar2files extraction runs.

"""

load("@bazeldnf//bazeldnf:defs.bzl", _tar2files_orig = "tar2files")

_NORMALIZE_TAR_PY = """\
import tarfile
import sys


def normalize_path(p):
    if p.startswith("./lib64/") or p == "./lib64":
        return "./usr" + p[1:]
    if p.startswith("lib64/") or p == "lib64":
        return "usr/" + p
    return p


src_path, dst_path = sys.argv[1], sys.argv[2]
seen = set()
with tarfile.open(src_path, "r") as src, tarfile.open(dst_path, "w") as dst:
    for m in src.getmembers():
        m.name = normalize_path(m.name)
        # Skip duplicates: the usrmerge ./lib64 symlink renames to ./usr/lib64,
        # which already exists as a real directory entry.
        if m.name in seen:
            continue
        seen.add(m.name)
        # Only pass a file object for regular files; for symlinks/dirs/hardlinks
        # extractfile() would try to follow the link and fail with renamed paths.
        if m.isreg():
            dst.addfile(m, src.extractfile(m))
        else:
            dst.addfile(m)
"""

def _normalize_tar_impl(ctx):
    """Normalize ./lib64/* -> ./usr/lib64/* paths in an rpmtree tar."""
    src = ctx.files.src[0]
    out = ctx.outputs.out

    script = ctx.actions.declare_file(ctx.label.name + "_normalize_tar.py")
    ctx.actions.write(output = script, content = _NORMALIZE_TAR_PY)

    ctx.actions.run_shell(
        inputs = [src, script],
        outputs = [out],
        command = "python3 {script} {src} {dst}".format(
            script = script.path,
            src = src.path,
            dst = out.path,
        ),
        mnemonic = "NormalizeTar",
        progress_message = "Normalizing lib64 paths in %s" % src.short_path,
    )
    return [DefaultInfo(files = depset([out]))]

_normalize_tar = rule(
    implementation = _normalize_tar_impl,
    attrs = {
        "src": attr.label(allow_single_file = True),
        "out": attr.output(mandatory = True),
    },
)

def tar2files(name, files = None, tar = None, **kwargs):
    """tar2files wrapper that normalizes ./lib64 -> ./usr/lib64 for usrmerge compat.

    Drop-in replacement for @bazeldnf//bazeldnf:defs.bzl tar2files. Inserts a
    Python-based path-normalization step before the underlying _tar2files rule so
    that RPMs shipping files under ./lib64/ (usrmerge packages) are correctly
    extracted to ./usr/lib64/.

    Args:
        name: Target name.
        files: Dict of prefix -> list of file paths to extract (required).
        tar:   Label of the rpmtree tar to extract from (required).
        **kwargs: Forwarded to the underlying tar2files rule (e.g. visibility).
    """
    if not files:
        fail("files is a required attribute")
    if not tar:
        fail("tar is a required attribute")

    normalized_name = name + "__lib64_normalized"
    _normalize_tar(
        name = normalized_name,
        src = tar,
        out = normalized_name + ".tar",
    )

    _tar2files_orig(
        name = name,
        files = files,
        tar = ":" + normalized_name,
        **kwargs
    )
