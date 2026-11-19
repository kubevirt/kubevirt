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

from __future__ import annotations

import json
import os
import re
import shlex
import subprocess
import sys
import time
from collections.abc import Iterable
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCRIPT_PATH = "./hack/cluster_resync.py"
KUBECTL = "./kubevirtci/cluster-up/kubectl.sh"
DOCKERIZED = "./hack/dockerized"
ROLLOUT_TIMEOUT = "10m"
DIGEST_RE = re.compile(r"^sha256:[0-9a-fA-F]{64}$")
SOURCE_ENV_KEYS = {
    "KUBEVIRT_PROVIDER",
    "docker_prefix",
    "image_prefix",
    "image_prefix_alt",
    "virt_template_version",
    "BAZEL_CS_CONFIG",
    "ARCHITECTURE",
}


@dataclass(frozen=True)
class ImageSpec:
    name: str
    target: str
    workload: str | None = None
    rollout_order: int | None = None
    hints: tuple[str, ...] = ()
    feature_gate: str | None = None
    template: bool = False


IMAGE_SPECS = (
    ImageSpec(
        "virt-operator",
        "//cmd/virt-operator:virt-operator-image",
        workload="deployment/virt-operator",
        rollout_order=0,
    ),
    ImageSpec(
        "virt-api",
        "//cmd/virt-api:virt-api-image",
        workload="deployment/virt-api",
        rollout_order=1,
    ),
    ImageSpec(
        "virt-controller",
        "//cmd/virt-controller:virt-controller-image",
        workload="deployment/virt-controller",
        rollout_order=2,
    ),
    ImageSpec(
        "virt-handler",
        "//cmd/virt-handler:virt-handler-image",
        workload="daemonset/virt-handler",
        rollout_order=5,
        hints=("virt-launcher",),
    ),
    ImageSpec(
        "virt-launcher",
        "//cmd/virt-launcher:virt-launcher-image",
        workload="daemonset/virt-handler",
        rollout_order=5,
    ),
    ImageSpec("virt-exportserver", "//cmd/virt-exportserver:virt-exportserver-image"),
    ImageSpec(
        "virt-exportproxy",
        "//cmd/virt-exportproxy:virt-exportproxy-image",
        workload="deployment/virt-exportproxy",
        rollout_order=3,
    ),
    ImageSpec(
        "virt-synchronization-controller",
        "//cmd/synchronization-controller:virt-synchronization-controller-image",
        workload="deployment/virt-synchronization-controller",
        rollout_order=4,
        feature_gate="DecentralizedLiveMigration",
    ),
    ImageSpec(
        "virt-template-apiserver",
        "//images/virt-template:virt-template-apiserver-image",
        workload="deployment/virt-template-apiserver",
        rollout_order=6,
        template=True,
    ),
    ImageSpec(
        "virt-template-controller",
        "//images/virt-template:virt-template-controller-image",
        workload="deployment/virt-template-controller",
        rollout_order=7,
        template=True,
    ),
)
IMAGE_SPECS_BY_NAME = {spec.name: spec for spec in IMAGE_SPECS}


class ResyncError(RuntimeError):
    def __init__(self, message: str, exit_code: int = 1):
        super().__init__(message)
        self.exit_code = exit_code


@dataclass
class ImagePlan:
    changed: set[str] = field(default_factory=set)
    core_push: list[str] = field(default_factory=list)
    template_push: list[str] = field(default_factory=list)
    registry_digests: dict[str, str] = field(default_factory=dict)


@dataclass(frozen=True)
class BuiltImage:
    digest: str
    architecture: str


def command(
    args: list[str],
    *,
    env: dict[str, str] | None = None,
    check: bool = True,
    capture: bool = True,
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        args,
        cwd=ROOT,
        env=env,
        text=True,
        stdout=subprocess.PIPE if capture else None,
        check=False,
    )
    if check and result.returncode:
        raise ResyncError(f"Command failed ({result.returncode}): {shlex.join(args)}")
    return result


def parse_specs(fast_templates: bool) -> list[tuple[str, str]]:
    return [(spec.name, spec.target) for spec in IMAGE_SPECS if fast_templates or not spec.template]


def select_images(env: dict[str, str]) -> list[tuple[str, str]]:
    use_hints = env.get("USE_HINTS", "true")
    if use_hints not in {"true", "false"}:
        raise ResyncError("USE_HINTS must be true or false", exit_code=2)

    fast_templates = env.get("FAST_VIRT_TEMPLATE_ENABLED") == "true"
    specs = parse_specs(fast_templates)
    raw_requested = env.get("PUSH_TARGETS", "")
    requested = raw_requested.replace(",", " ").split()
    if raw_requested and not requested:
        raise ResyncError("PUSH_TARGETS is empty", exit_code=2)
    selected = set(requested)
    if requested and use_hints == "true":
        for spec in IMAGE_SPECS:
            if spec.name in selected:
                selected.update(spec.hints)

    if not requested:
        return specs

    result = [(image, target) for image, target in specs if image in selected]
    unknown = selected - {image for image, _target in result}
    if unknown:
        raise ResyncError(
            f"Unknown or disabled PUSH_TARGETS: {' '.join(sorted(unknown))}",
            exit_code=2,
        )
    return result


def rollout_targets(changed: set[str]) -> list[str]:
    workloads = {}
    for spec in IMAGE_SPECS:
        if spec.name not in changed or spec.workload is None:
            continue
        if spec.rollout_order is None:
            raise ResyncError(f"No rollout order for {spec.workload}")
        workloads[spec.workload] = spec.rollout_order
    return sorted(workloads, key=workloads.__getitem__)


def bazel_digest_path(target: str) -> Path:
    relative_target = target.removeprefix("//").replace(":", "/")
    return ROOT / "bazel-bin" / f"{relative_target}_digest"


def bazel_image_config_path(target: str) -> Path:
    relative_target = target.removeprefix("//").replace(":", "/")
    return ROOT / "bazel-bin" / f"{relative_target}_config.json"


def source_project_environment(env: dict[str, str]) -> dict[str, str]:
    setup_script = """
set -e
source hack/common.sh
source "kubevirtci/cluster-up/cluster/${KUBEVIRT_PROVIDER}/provider.sh"
source hack/config.sh
source hack/virt-template/default.sh
printf '%s\\0' \\
    KUBEVIRT_PROVIDER "${KUBEVIRT_PROVIDER-}" \\
    docker_prefix "${docker_prefix-}" \\
    image_prefix "${image_prefix-}" \\
    image_prefix_alt "${image_prefix_alt-}" \\
    virt_template_version "${virt_template_version-}" \\
    BAZEL_CS_CONFIG "${BAZEL_CS_CONFIG-}" \\
    ARCHITECTURE "${ARCHITECTURE-}"
"""
    result = command(["bash", "-c", setup_script], env=env)
    fields = result.stdout.split("\0")
    if fields[-1] == "":
        fields.pop()
    if len(fields) % 2:
        raise ResyncError("Could not read the KubeVirt provider configuration")
    sourced_env = dict(zip(fields[::2], fields[1::2]))
    if not SOURCE_ENV_KEYS.issubset(sourced_env):
        raise ResyncError("Could not read the KubeVirt provider configuration")
    return {**env, **sourced_env}


def kubectl(env: dict[str, str], *args: str, check: bool = True) -> str:
    result = command([KUBECTL, *args], env=env, check=check)
    return result.stdout


def kubectl_json(env: dict[str, str], *args: str, check: bool = True) -> dict:
    return parse_json(kubectl(env, *args, check=check), "kubectl")


def parse_json(output: str, source: str) -> dict:
    try:
        return json.loads(output)
    except json.JSONDecodeError as error:
        raise ResyncError(f"{source} returned invalid JSON: {error}") from error


def kubectl_env(env: dict[str, str]) -> tuple[str, str, str, str]:
    namespace = env.get("KUBEVIRT_INSTALLED_NAMESPACE", "kubevirt")
    operator_image = kubectl(
        env,
        "-n",
        namespace,
        "get",
        "deployment",
        "virt-operator",
        '-o=jsonpath={.spec.template.spec.containers[?(@.name=="virt-operator")].image}',
    ).strip()
    if not operator_image:
        raise ResyncError(f"No installed virt-operator in {namespace}")

    installed_tag = operator_image.rsplit(":", 1)[-1]
    tag = env.get("DOCKER_TAG") or installed_tag
    if tag != installed_tag:
        raise ResyncError(
            f"DOCKER_TAG={tag} differs from the installed operator tag {installed_tag}; "
            "regenerate and apply the manifests first"
        )

    template_setting = kubectl(
        env,
        "-n",
        namespace,
        "get",
        "kv",
        "kubevirt",
        "-o=jsonpath={.spec.configuration.virtTemplateDeployment.enabled}",
    ).strip()
    enabled_gates = kubectl(
        env,
        "-n",
        namespace,
        "get",
        "kv",
        "kubevirt",
        r'-o=jsonpath={range .spec.configuration.developerConfiguration.featureGates[*]}{.}{"\n"}{end}',
    )
    disabled_gates = kubectl(
        env,
        "-n",
        namespace,
        "get",
        "kv",
        "kubevirt",
        r'-o=jsonpath={range .spec.configuration.developerConfiguration.disabledFeatureGates[*]}{.}{"\n"}{end}',
    )
    templates_enabled = not (
        template_setting == "false"
        or ("Template" not in enabled_gates.splitlines() and "Template" in disabled_gates.splitlines())
    )
    env["FAST_VIRT_TEMPLATE_ENABLED"] = "true" if templates_enabled else "false"
    env["KUBEVIRT_ENABLED_GATES"] = enabled_gates
    template_tag = env.get("VIRT_TEMPLATE_VERSION") or env.get("virt_template_version", "v0.3.0")
    return namespace, tag, env.get("DOCKER_TAG_ALT", ""), template_tag


def tag_for_image(image: str, base_tag: str, template_tag: str) -> str:
    return template_tag if IMAGE_SPECS_BY_NAME[image].template else base_tag


def check_template_tags(env: dict[str, str], images: Iterable[str], namespace: str, template_tag: str) -> None:
    if env["FAST_VIRT_TEMPLATE_ENABLED"] != "true":
        return
    for image in images:
        if not IMAGE_SPECS_BY_NAME[image].template:
            continue
        installed = kubectl(
            env,
            "-n",
            namespace,
            "get",
            f"deployment/{image}",
            "-o=jsonpath={.spec.template.spec.containers[0].image}",
        ).strip()
        installed_tag = installed.rsplit(":", 1)[-1]
        if installed_tag != template_tag:
            raise ResyncError(
                f"{image} uses {installed_tag}; requested VirtTemplate tag is {template_tag}. "
                "Update the operator's embedded manifest first."
            )


def wait_kubevirt_ready(env: dict[str, str], namespace: str) -> None:
    for condition in ("Available=True", "Progressing=False", "Degraded=False"):
        kubectl(
            env,
            "-n",
            namespace,
            "wait",
            "kv/kubevirt",
            f"--for=condition={condition}",
            f"--timeout={ROLLOUT_TIMEOUT}",
        )


def cluster_node_architectures(env: dict[str, str]) -> dict[str, str]:
    nodes = kubectl_json(env, "get", "nodes", "-o", "json")
    node_architectures = {}
    for node in nodes.get("items", []):
        name = node.get("metadata", {}).get("name", "")
        architecture = node.get("status", {}).get("nodeInfo", {}).get("architecture", "")
        if architecture:
            node_architectures[name] = architecture
    if not node_architectures:
        raise ResyncError("Cannot check deployed images: the cluster has no nodes")
    return node_architectures


def registry_image_ref(env: dict[str, str], image: str, image_tag: str, repository_prefix: str = "") -> str:
    registry = env.get("DOCKER_PREFIX") or env.get("docker_prefix", "")
    if not registry:
        raise ResyncError(f"Cannot resolve {image}: registry prefix is empty")
    repository = "/".join(
        part.strip("/") for part in (repository_prefix, image) if part.strip("/")
    )
    return f"{registry.rstrip('/')}/{repository}:{image_tag}"


def resolve_registry_digest(
    env: dict[str, str],
    image: str,
    image_tag: str,
    architecture: str,
    repository_prefix: str = "",
    *,
    missing_ok: bool = False,
) -> str | None:
    image_ref = registry_image_ref(env, image, image_tag, repository_prefix)
    local_registry = image_ref.startswith(("localhost:", "127.0.0.1:"))
    args = ["skopeo", "inspect"]
    if local_registry:
        args.append("--tls-verify=false")
    args.extend((
        "--override-os",
        "linux",
        "--override-arch",
        architecture,
        "--format",
        "{{.Digest}}",
        f"docker://{image_ref}",
    ))
    result = subprocess.run(
        args,
        cwd=ROOT,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode:
        details = result.stderr.strip()
        missing_markers = ("manifest unknown", "name unknown", "manifest not found")
        if missing_ok and any(marker in details.lower() for marker in missing_markers):
            return None
        raise ResyncError(f"Cannot inspect registry image {image_ref} for {architecture}: {details}")
    digest = result.stdout.strip()
    if not DIGEST_RE.fullmatch(digest):
        raise ResyncError(f"Registry returned an invalid digest for {image_ref}: {digest}")
    return digest


def resolve_registry_digests(
    env: dict[str, str], image: str, image_tag: str, architectures: list[str]
) -> str:
    repository_prefix = env.get("IMAGE_PREFIX") or env.get("image_prefix", "")
    digests = []
    for architecture in architectures:
        digest = resolve_registry_digest(
            env, image, image_tag, architecture, repository_prefix
        )
        if digest is None:
            raise ResyncError(f"Registry image {image}:{image_tag} is missing for {architecture}")
        digests.append(f"{architecture}={digest}")
    return ";".join(digests)


def parse_image_ref(reference: str) -> str:
    return reference.split("@", 1)[0].rsplit("/", 1)[-1].split(":", 1)[0]


def pod_image_rows(pods: dict, image: str, pod_prefix: str) -> list[tuple[str, str, str]]:
    rows = []
    for pod in pods.get("items", []):
        status = pod.get("status", {})
        pod_name = pod.get("metadata", {}).get("name", "")
        if status.get("phase") != "Running" or not pod_name.startswith(pod_prefix + "-"):
            continue
        if not any(c.get("type") == "Ready" and c.get("status") == "True" for c in status.get("conditions", [])):
            continue
        spec = pod.get("spec", {})
        statuses = {
            item.get("name"): item
            for item in status.get("containerStatuses", []) + status.get("initContainerStatuses", [])
        }
        containers = spec.get("containers", []) + spec.get("initContainers", [])
        for container in containers:
            name = container.get("name")
            if parse_image_ref(container.get("image", "")) != image or name not in statuses:
                continue
            rows.append((
                spec.get("nodeName", ""),
                container.get("image", ""),
                statuses[name].get("imageID", ""),
            ))
    return rows


def workload_mapping(image: str, enabled_gates: set[str]) -> tuple[str, str] | None:
    spec = IMAGE_SPECS_BY_NAME.get(image)
    if spec is None:
        raise ResyncError(f"No workload mapping for image {image}")
    if spec.workload is None:
        return None
    if spec.feature_gate and spec.feature_gate not in enabled_gates:
        return None
    return spec.workload, spec.workload.rsplit("/", 1)[-1]


def workload_feature_gate(workload: str) -> str | None:
    return next(
        (spec.feature_gate for spec in IMAGE_SPECS if spec.workload == workload and spec.feature_gate),
        None,
    )


def image_matches_deployed_pods(
    env: dict[str, str],
    namespace: str,
    image: str,
    image_tag: str,
    current_digests: str,
    node_architectures: dict[str, str],
    enabled_gates: set[str],
) -> tuple[bool, str]:
    mapping = workload_mapping(image, enabled_gates)
    if mapping is None:
        return True, current_digests
    workload, pod_prefix = mapping
    workload_result = command(
        [
            KUBECTL,
            "-n",
            namespace,
            "get",
            workload,
            "-o",
            "json",
            "--ignore-not-found",
        ],
        env=env,
        check=False,
    )
    if workload_result.returncode:
        raise ResyncError(f"Unable to inspect workload {workload}")
    if not workload_result.stdout.strip():
        return False, current_digests
    workload_data = json.loads(workload_result.stdout)
    status = workload_data.get("status", {})
    if workload.startswith("deployment/"):
        expected = workload_data.get("spec", {}).get("replicas") or 1
        ready = status.get("readyReplicas", 0)
        updated = status.get("updatedReplicas", 0)
        observed = status.get("observedGeneration", 0)
        generation = workload_data.get("metadata", {}).get("generation", 0)
        if not (
            isinstance(expected, int)
            and expected > 0
            and ready == expected
            and updated == expected
            and observed >= generation
        ):
            return False, current_digests
    else:
        expected = status.get("desiredNumberScheduled", 0)
        if not (
            isinstance(expected, int)
            and expected > 0
            and status.get("numberReady", 0) == expected
            and status.get("updatedNumberScheduled", 0) == expected
        ):
            return False, current_digests

    pods = kubectl_json(env, "-n", namespace, "get", "pods", "-o", "json")
    rows = pod_image_rows(pods, image, pod_prefix)
    if len(rows) != expected:
        return False, current_digests
    expected_by_arch = dict(entry.split("=", 1) for entry in current_digests.split(";"))
    for node, reference, image_id in rows:
        if reference.rsplit("/", 1)[-1] != f"{image}:{image_tag}" or not node or not image_id:
            return False, current_digests
        arch = node_architectures.get(node, "")
        if arch not in expected_by_arch:
            raise ResyncError(f"No registry digest recorded for node architecture {arch}")
        actual_digest = image_id.rsplit("@", 1)[-1]
        actual_digest = actual_digest.removeprefix("containerd://").removeprefix("docker://")
        if actual_digest != expected_by_arch[arch]:
            return False, current_digests
    return True, current_digests


def build_images(env: dict[str, str], specs: list[tuple[str, str]]) -> None:
    targets = [target for _image, target in specs]
    build_script = r"""
set -e
source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh
read -r -a bazel_configs <<<"${BAZEL_CS_CONFIG:-}"
bazel build --config="$ARCHITECTURE" "${bazel_configs[@]}" \
    --define container_prefix= --define image_prefix= --define container_tag= "$@"
"""
    command(
        ["bash", "-c", build_script, "cluster-resync-build", *targets],
        env=env,
        capture=False,
    )
    for image, target in specs:
        digest_file = bazel_digest_path(target)
        if not digest_file.is_file():
            raise ResyncError(f"Missing Bazel digest: {digest_file.relative_to(ROOT)}")
        digest = digest_file.read_text().strip()
        if not DIGEST_RE.fullmatch(digest):
            raise ResyncError(f"Invalid Bazel image digest for {image}: {digest}")
        config_file = bazel_image_config_path(target)
        if not config_file.is_file():
            raise ResyncError(f"Missing Bazel image config: {config_file.relative_to(ROOT)}")
        image_config = parse_json(config_file.read_text(), f"Bazel image config for {image}")
        architecture = image_config.get("architecture", "")
        if not architecture:
            raise ResyncError(f"Bazel image config has no architecture for {image}")
        print(f"FAST_IMAGE_DIGEST={image}={digest}")
        print(f"FAST_IMAGE_ARCH={image}={architecture}")


def dockerized(env: dict[str, str], assignments: dict[str, str], script: str) -> str:
    prefix = " ".join(f"{key}={shlex.quote(value)}" for key, value in assignments.items())
    result = command([DOCKERIZED, f"{prefix} {script}"], env=env)
    return result.stdout


def parse_built_images(output: str) -> dict[str, BuiltImage]:
    digests: dict[str, str] = {}
    architectures: dict[str, str] = {}
    for line in output.splitlines():
        if line.startswith("FAST_IMAGE_DIGEST="):
            image, separator, digest = line.removeprefix("FAST_IMAGE_DIGEST=").partition("=")
            if separator:
                digests[image] = digest
        elif line.startswith("FAST_IMAGE_ARCH="):
            image, separator, architecture = line.removeprefix("FAST_IMAGE_ARCH=").partition("=")
            if separator:
                architectures[image] = architecture
    if digests.keys() != architectures.keys():
        raise ResyncError("Build returned incomplete image digest or architecture data")
    built_images = {
        image: BuiltImage(digest, architectures[image]) for image, digest in digests.items()
    }
    for image, built_image in built_images.items():
        if not DIGEST_RE.fullmatch(built_image.digest) or not built_image.architecture:
            raise ResyncError(f"Build returned invalid image metadata for {image}")
    return built_images


def validate_build_architectures(
    built_images: dict[str, BuiltImage], cluster_architectures: list[str]
) -> None:
    architectures = set(cluster_architectures)
    if len(architectures) == 1 and all(
        image.architecture in architectures for image in built_images.values()
    ):
        return
    built = ", ".join(sorted({image.architecture for image in built_images.values()}))
    cluster = ", ".join(sorted(architectures))
    raise ResyncError(
        "The local image build architectures do not match the cluster node architectures "
        f"(built: {built}; cluster: {cluster}). Build each cluster architecture before resync."
    )


def registry_destinations(
    env: dict[str, str], image: str, tag: str, alt_tag: str, template_tag: str
) -> list[tuple[str, str]]:
    image_spec = IMAGE_SPECS_BY_NAME[image]
    image_tag = tag_for_image(image, tag, template_tag)
    image_prefix = env.get("IMAGE_PREFIX") or env.get("image_prefix", "")
    image_prefix_alt = env.get("IMAGE_PREFIX_ALT") or env.get("image_prefix_alt", "")
    destinations = [(image_prefix, image_tag)]
    if alt_tag and not image_spec.template:
        destinations.append((image_prefix, alt_tag))
    if image_prefix_alt:
        destinations.append((image_prefix_alt, image_tag))
    return list(dict.fromkeys(destinations))


def plan_image_updates(
    env: dict[str, str],
    specs: list[tuple[str, str]],
    built_images: dict[str, BuiltImage],
    tag: str,
    alt_tag: str,
    template_tag: str,
) -> ImagePlan:
    plan = ImagePlan()
    for image, _target in specs:
        built_image = built_images.get(image)
        if built_image is None:
            raise ResyncError(f"Build returned no image metadata for {image}")
        needs_push = False
        destinations = registry_destinations(
            env, image, tag, alt_tag, template_tag
        )
        for destination_index, (repository_prefix, image_tag) in enumerate(destinations):
            registry_digest = resolve_registry_digest(
                env,
                image,
                image_tag,
                built_image.architecture,
                repository_prefix,
                missing_ok=True,
            )
            if destination_index == 0 and registry_digest is not None:
                plan.registry_digests[image] = f"{built_image.architecture}={registry_digest}"
            if registry_digest != built_image.digest:
                needs_push = True
        if needs_push:
            destination = plan.template_push if IMAGE_SPECS_BY_NAME[image].template else plan.core_push
            destination.append(image)
    return plan


def verify_published_images(
    env: dict[str, str],
    plan: ImagePlan,
    specs: list[tuple[str, str]],
    built_images: dict[str, BuiltImage],
    tag: str,
    alt_tag: str,
    template_tag: str,
) -> None:
    pushed_images = set(plan.core_push + plan.template_push)
    for image, _target in specs:
        if image not in pushed_images:
            continue
        built_image = built_images[image]
        for repository_prefix, image_tag in registry_destinations(
            env, image, tag, alt_tag, template_tag
        ):
            registry_digest = resolve_registry_digest(
                env,
                image,
                image_tag,
                built_image.architecture,
                repository_prefix,
            )
            if registry_digest != built_image.digest:
                raise ResyncError(
                    f"Registry image {image}:{image_tag} has digest {registry_digest}; "
                    f"expected built digest {built_image.digest}"
                )


def find_images_needing_rollout(
    env: dict[str, str],
    namespace: str,
    specs: list[tuple[str, str]],
    registry_digests_by_image: dict[str, str],
    tag: str,
    template_tag: str,
    architectures: list[str],
    node_architectures: dict[str, str],
    enabled_gates: set[str],
) -> set[str]:
    changed = set()
    for image, _target in specs:
        if workload_mapping(image, enabled_gates) is None:
            continue
        image_tag = tag_for_image(image, tag, template_tag)
        registry_digests = registry_digests_by_image.get(image)
        if registry_digests is None:
            registry_digests = resolve_registry_digests(env, image, image_tag, architectures)
        matches, _digests = image_matches_deployed_pods(
            env,
            namespace,
            image,
            image_tag,
            registry_digests,
            node_architectures,
            enabled_gates,
        )
        if not matches:
            changed.add(image)
    return changed


def verify_deployed_images(
    env: dict[str, str],
    namespace: str,
    specs: list[tuple[str, str]],
    tag: str,
    template_tag: str,
    architectures: list[str],
    node_architectures: dict[str, str],
    enabled_gates: set[str],
    changed: set[str],
) -> None:
    for image, _target in specs:
        image_tag = tag_for_image(image, tag, template_tag)
        registry_digests = resolve_registry_digests(env, image, image_tag, architectures)
        matches, _digests = image_matches_deployed_pods(
            env,
            namespace,
            image,
            image_tag,
            registry_digests,
            node_architectures,
            enabled_gates,
        )
        if not matches and workload_mapping(image, enabled_gates) is not None:
            stage = "after rollout" if image in changed else "during resync"
            raise ResyncError(f"{image} Pods do not match the current registry image digest {stage}")
        if image in changed:
            print(f"Verified deployed image digest for {image}.")


def push_images(
    env: dict[str, str],
    targets: list[str],
    tag: str,
    alt_tag: str,
    image_prefix: str,
    image_prefix_alt: str,
) -> None:
    assignments = {
        "PUSH_TARGETS": " ".join(targets),
        "BUILD_ARCH": env.get("BUILD_ARCH", ""),
        "DOCKER_PREFIX": env.get("DOCKER_PREFIX", ""),
        "DOCKER_TAG": tag,
        "DOCKER_TAG_ALT": alt_tag,
        "IMAGE_PREFIX": env.get("IMAGE_PREFIX", image_prefix),
        "IMAGE_PREFIX_ALT": env.get("IMAGE_PREFIX_ALT", image_prefix_alt),
        "KUBEVIRT_PROVIDER": env["KUBEVIRT_PROVIDER"],
    }
    dockerized(env, assignments, "./hack/bazel-push-images.sh")


def push_changed_images(
    env: dict[str, str],
    plan: ImagePlan,
    tag: str,
    alt_tag: str,
    template_tag: str,
) -> None:
    image_prefix = env.get("IMAGE_PREFIX") or env.get("image_prefix", "")
    image_prefix_alt = env.get("IMAGE_PREFIX_ALT") or env.get("image_prefix_alt", "")
    if plan.core_push:
        push_images(env, plan.core_push, tag, alt_tag, image_prefix, image_prefix_alt)
    if plan.template_push:
        push_images(env, plan.template_push, template_tag, "", image_prefix, image_prefix_alt)


def rollout_changed_workloads(env: dict[str, str], namespace: str, changed: set[str], enabled_gates: set[str]) -> None:
    for workload in rollout_targets(changed):
        feature_gate = workload_feature_gate(workload)
        if feature_gate and feature_gate not in enabled_gates:
            print(f"Skipping rollout for {workload}: {feature_gate} is not enabled")
            continue
        replace_workload_pods(env, namespace, workload)


def selector_and_expected(workload_data: dict) -> tuple[str, int]:
    selector_data = workload_data.get("spec", {}).get("selector", {})
    if selector_data.get("matchExpressions"):
        raise ResyncError("unsupported selector")
    selector = ",".join(f"{key}={value}" for key, value in selector_data.get("matchLabels", {}).items())
    if workload_data.get("kind") == "DaemonSet":
        expected = workload_data.get("status", {}).get("desiredNumberScheduled", 0)
    else:
        expected = workload_data.get("spec", {}).get("replicas", 0)
    if not selector:
        raise ResyncError("No pod selector for workload")
    if not isinstance(expected, int) or expected <= 0:
        raise ResyncError("No desired pods for workload")
    return selector, expected


def replace_workload_pods(env: dict[str, str], namespace: str, workload: str) -> None:
    workload_data = kubectl_json(env, "-n", namespace, "get", workload, "-o", "json")
    selector, expected = selector_and_expected(workload_data)
    pod_names = kubectl(env, "-n", namespace, "get", "pods", "-l", selector, "-o", "name")
    old_pods = [pod for pod in pod_names.splitlines() if pod.strip()]
    if old_pods:
        kubectl(env, "-n", namespace, "delete", *old_pods, "--wait=false")
    for pod in old_pods:
        kubectl(env, "-n", namespace, "wait", pod, "--for=delete", f"--timeout={ROLLOUT_TIMEOUT}")
    kubectl(env, "-n", namespace, "rollout", "status", workload, f"--timeout={ROLLOUT_TIMEOUT}")

    for _attempt in range(10):
        try:
            pods = kubectl_json(
                env,
                "-n",
                namespace,
                "get",
                "pods",
                "-l",
                selector,
                "-o",
                "json",
                check=False,
            )
        except ResyncError:
            pods = {}
        pod_items = pods.get("items", [])
        if len(pod_items) == expected and all(
            any(
                condition.get("type") == "Ready" and condition.get("status") == "True"
                for condition in pod.get("status", {}).get("conditions", [])
            )
            for pod in pod_items
        ):
            return
        time.sleep(1)
    raise ResyncError(f"{workload} did not replace all {expected} ready pods")


def resync(env: dict[str, str]) -> None:
    env = source_project_environment(env)
    namespace, tag, alt_tag, template_tag = kubectl_env(env)
    specs = select_images(env)
    check_template_tags(env, [image for image, _target in specs], namespace, template_tag)
    node_architectures = cluster_node_architectures(env)
    architectures = sorted(set(node_architectures.values()))
    enabled_gates = set(env.get("KUBEVIRT_ENABLED_GATES", "").splitlines())

    build_output = dockerized(
        env,
        {
            "KUBEVIRT_PROVIDER": env["KUBEVIRT_PROVIDER"],
            "BUILD_ARCH": env.get("BUILD_ARCH", ""),
            "PUSH_TARGETS": env.get("PUSH_TARGETS", ""),
            "USE_HINTS": env.get("USE_HINTS", "true"),
            "FAST_VIRT_TEMPLATE_ENABLED": env["FAST_VIRT_TEMPLATE_ENABLED"],
        },
        f"{SCRIPT_PATH} build",
    )
    print(build_output, end="")
    built_images = parse_built_images(build_output)
    validate_build_architectures(built_images, architectures)
    plan = plan_image_updates(
        env,
        specs,
        built_images,
        tag,
        alt_tag,
        template_tag,
    )

    push_changed_images(env, plan, tag, alt_tag, template_tag)
    verify_published_images(env, plan, specs, built_images, tag, alt_tag, template_tag)
    for image in plan.core_push + plan.template_push:
        plan.registry_digests[image] = (
            f"{built_images[image].architecture}={built_images[image].digest}"
        )
    plan.changed = find_images_needing_rollout(
        env,
        namespace,
        specs,
        plan.registry_digests,
        tag,
        template_tag,
        architectures,
        node_architectures,
        enabled_gates,
    )

    if not plan.changed:
        wait_kubevirt_ready(env, namespace)
        print("Selected images match the registry and deployed workloads.")
        return

    rollout_changed_workloads(env, namespace, plan.changed, enabled_gates)

    wait_kubevirt_ready(env, namespace)
    verify_published_images(env, plan, specs, built_images, tag, alt_tag, template_tag)
    verify_deployed_images(
        env,
        namespace,
        specs,
        tag,
        template_tag,
        architectures,
        node_architectures,
        enabled_gates,
        plan.changed,
    )

    print(f"Updated {' '.join(plan.core_push + plan.template_push)}")


def main(argv: list[str] | None = None) -> int:
    args = list(sys.argv[1:] if argv is None else argv)
    env = os.environ.copy()
    try:
        if args and args[0] == "list":
            for image, target in select_images(env):
                print(f"{image} {target}")
            return 0
        if args and args[0] == "plan":
            for workload in rollout_targets(set(args[1:])):
                print(workload)
            return 0
        if args and args[0] == "build":
            build_images(env, select_images(env))
            return 0
        resync(env)
        return 0
    except ResyncError as error:
        print(str(error), file=sys.stderr)
        return error.exit_code


if __name__ == "__main__":
    sys.exit(main())
