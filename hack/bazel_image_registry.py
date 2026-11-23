#!/usr/bin/env python3
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2026 Red Hat, Inc.
#

"""Serve Bazel OCI layout tarballs through a read-only registry v2 endpoint."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
import tarfile
from dataclasses import dataclass
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote, urlsplit

DIGEST_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
RANGE_RE = re.compile(r"^bytes=(\d*)-(\d*)$")
CHUNK_SIZE = 1024 * 1024


@dataclass(frozen=True)
class ImageLayout:
    """Identify a single image manifest stored in an OCI layout tarball."""

    archive_path: Path
    repository: str
    tag: str
    manifest_digest: str
    manifest: bytes
    manifest_media_type: str
    blob_media_types: dict[str, str]

    def read_blob(
        self, digest: str
    ) -> tuple[tarfile.TarFile, tarfile.ExFileObject, tarfile.TarInfo] | None:
        """Open a blob member in this image archive, if it exists."""
        if not DIGEST_RE.fullmatch(digest):
            return None
        member_name = f"blobs/sha256/{digest.removeprefix('sha256:')}"
        archive = tarfile.open(self.archive_path, "r")
        try:
            member = archive.getmember(member_name)
        except KeyError:
            archive.close()
            return None
        blob = archive.extractfile(member)
        if blob is None:
            archive.close()
            return None
        return archive, blob, member


def _read_tar_member(archive: tarfile.TarFile, name: str) -> bytes:
    try:
        member = archive.getmember(name)
    except KeyError as error:
        raise ValueError(f"OCI layout is missing {name}") from error
    fileobj = archive.extractfile(member)
    if fileobj is None:
        raise ValueError(f"OCI layout member is not a file: {name}")
    with fileobj:
        return fileobj.read()


def load_layouts(
    layouts: list[tuple[str, str, Path] | tuple[str, Path]],
    repository_prefix: str,
    tag: str,
) -> dict[str, list[ImageLayout]]:
    """Load and validate OCI image manifests from Bazel-produced layout tarballs."""
    prefix = repository_prefix.strip("/")
    images: dict[str, list[ImageLayout]] = {}
    for layout in layouts:
        if len(layout) == 2:
            image_name, archive_path = layout
            image_tag = ""
        else:
            image_name, image_tag, archive_path = layout
        if not re.fullmatch(r"[A-Za-z0-9_][A-Za-z0-9_.-]*", image_name):
            raise ValueError(f"Invalid image name: {image_name}")
        archive_path = archive_path.resolve()
        if not archive_path.is_file():
            raise ValueError(f"OCI layout tarball does not exist: {archive_path}")
        repository = "/".join(part for part in (prefix, image_name) if part)
        if any(image.tag == (image_tag or tag) for image in images.get(repository, [])):
            raise ValueError(
                f"Duplicate registry repository and tag: {repository}:{image_tag or tag}"
            )

        with tarfile.open(archive_path, "r") as archive:
            index = json.loads(_read_tar_member(archive, "index.json"))
            descriptors = index.get("manifests", [])
            if len(descriptors) != 1:
                raise ValueError(
                    f"OCI layout {archive_path} must contain exactly one manifest"
                )
            descriptor = descriptors[0]
            manifest_digest = descriptor.get("digest", "")
            if not DIGEST_RE.fullmatch(manifest_digest):
                raise ValueError(f"Invalid manifest digest in {archive_path}")
            manifest_path = (
                "blobs/sha256/" + manifest_digest.removeprefix("sha256:")
            )
            manifest = _read_tar_member(archive, manifest_path)
            calculated_digest = "sha256:" + hashlib.sha256(manifest).hexdigest()
            if calculated_digest != manifest_digest:
                raise ValueError(
                    f"OCI manifest digest mismatch in {archive_path}: "
                    f"{manifest_digest} != {calculated_digest}"
                )
            manifest_data = json.loads(manifest)
            manifest_media_type = manifest_data.get(
                "mediaType", descriptor.get("mediaType", "application/json")
            )
            blob_media_types = {}
            for blob in [manifest_data.get("config", {}), *manifest_data.get("layers", [])]:
                digest = blob.get("digest", "")
                if not DIGEST_RE.fullmatch(digest):
                    raise ValueError(f"Invalid image blob digest in {archive_path}")
                blob_media_types[digest] = blob.get(
                    "mediaType", "application/octet-stream"
                )
                blob_path = "blobs/sha256/" + digest.removeprefix("sha256:")
                try:
                    archive.getmember(blob_path)
                except KeyError as error:
                    raise ValueError(
                        f"OCI layout {archive_path} is missing image blob {digest}"
                    ) from error

        images.setdefault(repository, []).append(
            ImageLayout(
                archive_path=archive_path,
                repository=repository,
                tag=image_tag or tag,
                manifest_digest=manifest_digest,
                manifest=manifest,
                manifest_media_type=manifest_media_type,
                blob_media_types=blob_media_types,
            )
        )
    if not images:
        raise ValueError("At least one image layout is required")
    return images


class RegistryHandler(BaseHTTPRequestHandler):
    """Serve read-only OCI Registry HTTP API requests from Bazel outputs."""

    images: dict[str, list[ImageLayout]] = {}
    server_version = "BazelOCIRegistry/1.0"

    def do_GET(self) -> None:
        self._handle_request(send_body=True)

    def do_HEAD(self) -> None:
        self._handle_request(send_body=False)

    def do_POST(self) -> None:
        self._method_not_allowed()

    do_PUT = do_POST
    do_PATCH = do_POST
    do_DELETE = do_POST

    def _handle_request(self, *, send_body: bool) -> None:
        path = unquote(urlsplit(self.path).path)
        if path in {"/v2", "/v2/"}:
            self._send_bytes(200, b"{}", "application/json", send_body=send_body)
            return

        parts = path.strip("/").split("/")
        if len(parts) < 4 or parts[0] != "v2":
            self._error(404, "NAME_UNKNOWN", "unknown repository", send_body=send_body)
            return
        repository = "/".join(parts[1:-2])
        action, reference = parts[-2:]
        repository_images = self.images.get(repository)
        if repository_images is None:
            self._error(404, "NAME_UNKNOWN", "unknown repository", send_body=send_body)
            return

        if action == "manifests":
            image = next(
                (
                    candidate
                    for candidate in repository_images
                    if reference in {candidate.tag, candidate.manifest_digest}
                ),
                None,
            )
            if image is None:
                self._error(404, "MANIFEST_UNKNOWN", "unknown manifest", send_body=send_body)
                return
            self._send_bytes(
                200,
                image.manifest,
                image.manifest_media_type,
                send_body=send_body,
                digest=image.manifest_digest,
            )
            return
        if action == "blobs":
            self._serve_blob(reference, send_body=send_body)
            return
        if action == "tags" and reference == "list":
            body = json.dumps(
                {"name": repository, "tags": sorted({image.tag for image in repository_images})},
                separators=(",", ":"),
            ).encode()
            self._send_bytes(200, body, "application/json", send_body=send_body)
            return
        self._error(404, "UNSUPPORTED", "unsupported registry endpoint", send_body=send_body)

    def _serve_blob(self, digest: str, *, send_body: bool) -> None:
        for repository_images in self.images.values():
            for image in repository_images:
                blob_entry = image.read_blob(digest)
                if blob_entry is None:
                    continue
                archive, blob, member = blob_entry
                try:
                    self._send_blob(
                        blob,
                        member.size,
                        image.blob_media_types.get(digest, "application/octet-stream"),
                        digest,
                        send_body=send_body,
                    )
                finally:
                    archive.close()
                return
        self._error(404, "BLOB_UNKNOWN", "unknown blob", send_body=send_body)

    def _send_blob(
        self,
        blob: tarfile.ExFileObject,
        size: int,
        media_type: str,
        digest: str,
        *,
        send_body: bool,
    ) -> None:
        start, end = 0, size - 1
        status = 200
        range_header = self.headers.get("Range")
        if range_header:
            match = RANGE_RE.fullmatch(range_header)
            if not match or (not match.group(1) and not match.group(2)):
                blob.close()
                self.send_error(416, "invalid byte range")
                return
            if match.group(1):
                start = int(match.group(1))
                end = int(match.group(2)) if match.group(2) else size - 1
            else:
                suffix_length = int(match.group(2))
                start = max(0, size - suffix_length)
            if start >= size or end < start:
                blob.close()
                self.send_error(416, "byte range is not satisfiable")
                return
            end = min(end, size - 1)
            status = 206

        content_length = end - start + 1
        try:
            self._begin_response(status, media_type, content_length, digest)
            if status == 206:
                self.send_header("Content-Range", f"bytes {start}-{end}/{size}")
            self.end_headers()
            if send_body:
                blob.seek(start)
                remaining = content_length
                while remaining:
                    chunk = blob.read(min(CHUNK_SIZE, remaining))
                    if not chunk:
                        break
                    self.wfile.write(chunk)
                    remaining -= len(chunk)
        finally:
            blob.close()

    def _send_bytes(
        self,
        status: int,
        body: bytes,
        media_type: str,
        *,
        send_body: bool,
        digest: str | None = None,
    ) -> None:
        self._begin_response(status, media_type, len(body), digest)
        self.end_headers()
        if send_body:
            self.wfile.write(body)

    def _begin_response(
        self, status: int, media_type: str, content_length: int, digest: str | None = None
    ) -> None:
        self.send_response(status)
        self.send_header("Content-Type", media_type)
        self.send_header("Content-Length", str(content_length))
        self.send_header("Docker-Distribution-API-Version", "registry/2.0")
        if digest:
            self.send_header("Docker-Content-Digest", digest)

    def _error(self, status: int, code: str, message: str, *, send_body: bool) -> None:
        body = json.dumps(
            {"errors": [{"code": code, "message": message}]}, separators=(",", ":")
        ).encode()
        self._send_bytes(status, body, "application/json", send_body=send_body)

    def _method_not_allowed(self) -> None:
        self.send_response(405)
        self.send_header("Allow", "GET, HEAD")
        self.send_header("Content-Length", "0")
        self.send_header("Docker-Distribution-API-Version", "registry/2.0")
        self.end_headers()

    def log_message(self, format_string: str, *args: object) -> None:
        print(f"registry: {format_string % args}", file=sys.stderr)


def parse_image_layout(value: str) -> tuple[str, str, Path]:
    """Parse an --image NAME[@TAG]=OCI_LAYOUT_TARBALL argument."""
    image_name, separator, path = value.partition("=")
    if not separator or not image_name or not path:
        raise argparse.ArgumentTypeError("expected IMAGE[@TAG]=OCI_LAYOUT_TARBALL")
    image_name, tag_separator, image_tag = image_name.partition("@")
    return image_name, image_tag if tag_separator else "", Path(path)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--listen-address", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=5001)
    parser.add_argument("--tag", default="devel")
    parser.add_argument("--repository-prefix", default="kubevirt")
    parser.add_argument("--image", action="append", type=parse_image_layout, required=True)
    parser.add_argument("--advertise", default="127.0.0.1")
    args = parser.parse_args(argv)

    try:
        images = load_layouts(args.image, args.repository_prefix, args.tag)
        handler = type("BazelRegistryHandler", (RegistryHandler,), {"images": images})
        server = ThreadingHTTPServer((args.listen_address, args.port), handler)
    except (OSError, ValueError, tarfile.TarError, json.JSONDecodeError) as error:
        print(f"Cannot start Bazel image registry: {error}", file=sys.stderr)
        return 1

    scheme = "http"
    print(
        f"Serving {len(images)} read-only Bazel images at {scheme}://{args.advertise}:{args.port}",
        flush=True,
    )
    for repository in sorted(images):
        for image in images[repository]:
            print(
                f"  {scheme}://{args.advertise}:{args.port}/{repository}:{image.tag}",
                flush=True,
            )
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("Stopping Bazel image registry")
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
