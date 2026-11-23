#!/usr/bin/env python3

import hashlib
import io
import json
import tempfile
import threading
import unittest
from http.server import ThreadingHTTPServer
from pathlib import Path
from tarfile import TarInfo, open as tar_open
from urllib.error import HTTPError
from urllib.request import Request, urlopen

from bazel_image_registry import RegistryHandler, load_layouts


class BazelImageRegistryTest(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp_dir.cleanup)
        self.image_name = "virt-api"
        self.tag = "devel"
        self.config = b'{"architecture":"amd64","os":"linux"}'
        self.layer = b"layer-data"
        self.config_digest = "sha256:" + hashlib.sha256(self.config).hexdigest()
        self.layer_digest = "sha256:" + hashlib.sha256(self.layer).hexdigest()
        self.manifest = json.dumps(
            {
                "schemaVersion": 2,
                "mediaType": "application/vnd.oci.image.manifest.v1+json",
                "config": {
                    "mediaType": "application/vnd.oci.image.config.v1+json",
                    "digest": self.config_digest,
                    "size": len(self.config),
                },
                "layers": [
                    {
                        "mediaType": "application/vnd.oci.image.layer.v1.tar",
                        "digest": self.layer_digest,
                        "size": len(self.layer),
                    }
                ],
            },
            separators=(",", ":"),
        ).encode()
        self.manifest_digest = "sha256:" + hashlib.sha256(self.manifest).hexdigest()
        self.index = json.dumps(
            {
                "schemaVersion": 2,
                "manifests": [
                    {
                        "mediaType": "application/vnd.oci.image.manifest.v1+json",
                        "digest": self.manifest_digest,
                        "size": len(self.manifest),
                    }
                ],
            },
            separators=(",", ":"),
        ).encode()
        self.layout = Path(self.temp_dir.name) / "virt-api.tar"
        self._write_tarball()

        self.images = load_layouts(
            [(self.image_name, self.layout), (self.image_name, "devel_alt", self.layout)],
            "kubevirt",
            self.tag,
        )
        handler = type("TestRegistryHandler", (RegistryHandler,), {"images": self.images})
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.addCleanup(self._stop_server)
        self.base_url = f"http://127.0.0.1:{self.server.server_port}"

    def _write_tarball(self):
        members = {
            "index.json": self.index,
            "blobs/sha256/" + self.manifest_digest.removeprefix("sha256:"): self.manifest,
            "blobs/sha256/" + self.config_digest.removeprefix("sha256:"): self.config,
            "blobs/sha256/" + self.layer_digest.removeprefix("sha256:"): self.layer,
        }
        with tar_open(self.layout, "w") as archive:
            for name, content in members.items():
                info = TarInfo(name)
                info.size = len(content)
                archive.addfile(info, io.BytesIO(content))

    def _stop_server(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)

    def test_serves_v2_ping_and_image_manifest(self):
        with urlopen(self.base_url + "/v2/") as response:
            self.assertEqual(response.status, 200)
            self.assertEqual(response.headers["Docker-Distribution-API-Version"], "registry/2.0")

        request = Request(
            f"{self.base_url}/v2/kubevirt/{self.image_name}/manifests/{self.tag}",
            headers={"Accept": "application/vnd.oci.image.manifest.v1+json"},
        )
        with urlopen(request) as response:
            self.assertEqual(response.read(), self.manifest)
            self.assertEqual(response.headers["Docker-Content-Digest"], self.manifest_digest)

    def test_supports_per_image_tag(self):
        image = load_layouts(
            [(self.image_name, "v0.3.0", self.layout)], "kubevirt", self.tag
        )[f"kubevirt/{self.image_name}"][0]

        self.assertEqual(image.tag, "v0.3.0")

    def test_lists_all_tags_for_an_image(self):
        with urlopen(
            f"{self.base_url}/v2/kubevirt/{self.image_name}/tags/list"
        ) as response:
            self.assertEqual(
                json.loads(response.read()),
                {"name": f"kubevirt/{self.image_name}", "tags": ["devel", "devel_alt"]},
            )

    def test_serves_config_and_layer_blobs(self):
        for digest, expected in (
            (self.config_digest, self.config),
            (self.layer_digest, self.layer),
        ):
            request = f"{self.base_url}/v2/kubevirt/{self.image_name}/blobs/{digest}"
            with urlopen(request) as response:
                self.assertEqual(response.read(), expected)
                self.assertEqual(response.headers["Docker-Content-Digest"], digest)

    def test_reports_unknown_blob_and_rejects_writes(self):
        try:
            urlopen(f"{self.base_url}/v2/kubevirt/{self.image_name}/blobs/sha256:{'f' * 64}")
        except HTTPError as error:
            self.assertEqual(error.code, 404)
            error.close()
        else:
            self.fail("unknown blob request unexpectedly succeeded")

        request = Request(self.base_url + "/v2/kubevirt/virt-api/blobs/sha256:abc", method="PUT")
        try:
            urlopen(request)
        except HTTPError as error:
            self.assertEqual(error.code, 405)
            error.close()
        else:
            self.fail("registry write unexpectedly succeeded")

    def test_supports_head_and_range_requests(self):
        url = f"{self.base_url}/v2/kubevirt/{self.image_name}/blobs/{self.layer_digest}"
        with urlopen(Request(url, method="HEAD")) as response:
            self.assertEqual(response.status, 200)
            self.assertEqual(response.headers["Content-Length"], str(len(self.layer)))

        with urlopen(Request(url, headers={"Range": "bytes=1-4"})) as response:
            self.assertEqual(response.status, 206)
            self.assertEqual(response.read(), self.layer[1:5])


if __name__ == "__main__":
    unittest.main()
