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
    "BAZEL_CS_CONFIG",
    "ARCHITECTURE",
}


@dataclass(frozen=True)
class ImageSpec:
    """Describe an image, its build target, and its rollout hints."""
    name: str
    target: str
    workload: str | None = None
    hints: tuple[str, ...] = ()
    feature_gate: str | None = None


IMAGE_SPECS = (
    # Keep virt-operator first so it can reconcile before its workloads are restarted.
    ImageSpec(
        "virt-operator",
        "//cmd/virt-operator:virt-operator-image",
        workload="deployment/virt-operator",
    ),
    ImageSpec(
        "virt-api",
        "//cmd/virt-api:virt-api-image",
        workload="deployment/virt-api",
    ),
    ImageSpec(
        "virt-controller",
        "//cmd/virt-controller:virt-controller-image",
        workload="deployment/virt-controller",
    ),
    ImageSpec(
        "virt-handler",
        "//cmd/virt-handler:virt-handler-image",
        workload="daemonset/virt-handler",
        hints=("virt-launcher",),
    ),
    ImageSpec(
        "virt-launcher",
        "//cmd/virt-launcher:virt-launcher-image",
        workload="daemonset/virt-handler",
    ),
    ImageSpec("virt-exportserver", "//cmd/virt-exportserver:virt-exportserver-image"),
    ImageSpec(
        "virt-exportproxy",
        "//cmd/virt-exportproxy:virt-exportproxy-image",
        workload="deployment/virt-exportproxy",
    ),
    ImageSpec(
        "virt-synchronization-controller",
        "//cmd/synchronization-controller:virt-synchronization-controller-image",
        workload="deployment/virt-synchronization-controller",
        feature_gate="DecentralizedLiveMigration",
    ),
)
IMAGE_SPECS_BY_NAME = {spec.name: spec for spec in IMAGE_SPECS}


class ResyncError(RuntimeError):
    """Report an expected resync failure and its process exit code."""

    def __init__(self, message: str, exit_code: int = 1):
        """Create a resync error with the status main should return."""
        super().__init__(message)
        self.exit_code = exit_code


@dataclass
class ImagePlan:
    """Hold the image pushes and workload rollouts selected for a run."""
    changed: set[str] = field(default_factory=set)
    push: list[str] = field(default_factory=list)
    registry_digests: dict[str, str] = field(default_factory=dict)


@dataclass(frozen=True)
class BuiltImage:
    """Store the digest and architecture reported by the image build."""
    digest: str
    architecture: str


def command(
    args: list[str],
    *,
    env: dict[str, str] | None = None,
    check: bool = True,
    capture: bool = True,
) -> subprocess.CompletedProcess[str]:
    """Run a repository command and raise ResyncError on failure."""
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


def select_images(env: dict[str, str]) -> list[tuple[str, str]]:
    """Select requested images and optionally add their deployment hints."""
    use_hints = env.get("USE_HINTS", "true")
    if use_hints not in {"true", "false"}:
        raise ResyncError("USE_HINTS must be true or false", exit_code=2)

    specs = [(spec.name, spec.target) for spec in IMAGE_SPECS]
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
        raise ResyncError(f"Unknown image targets: {' '.join(sorted(unknown))}", exit_code=2)
    return result


def rollout_targets(changed: set[str]) -> list[str]:
    """Return affected workloads once, in image-spec order."""
    workloads = []
    for spec in IMAGE_SPECS:
        if spec.name in changed and spec.workload and spec.workload not in workloads:
            workloads.append(spec.workload)
    return workloads


def bazel_digest_path(target: str) -> Path:
    """Return the Bazel output path for a target's image digest."""
    relative_target = target.removeprefix("//").replace(":", "/")
    return ROOT / "bazel-bin" / f"{relative_target}_digest"


def bazel_image_config_path(target: str) -> Path:
    """Return the Bazel output path for a target's image config."""
    relative_target = target.removeprefix("//").replace(":", "/")
    return ROOT / "bazel-bin" / f"{relative_target}_config.json"


def source_project_environment(env: dict[str, str]) -> dict[str, str]:
    """Load provider and image settings from the project's shell config."""
    setup_script = """
set -e
source hack/common.sh
source "kubevirtci/cluster-up/cluster/${KUBEVIRT_PROVIDER}/provider.sh"
source hack/config.sh
printf '%s\\0' \\
    KUBEVIRT_PROVIDER "${KUBEVIRT_PROVIDER-}" \\
    docker_prefix "${docker_prefix-}" \\
    image_prefix "${image_prefix-}" \\
    image_prefix_alt "${image_prefix_alt-}" \\
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
    """Run kubectl and return its standard output."""
    result = command([KUBECTL, *args], env=env, check=check)
    return result.stdout


def kubectl_json(env: dict[str, str], *args: str, check: bool = True) -> dict:
    """Run kubectl and parse its JSON output."""
    return parse_json(kubectl(env, *args, check=check), "kubectl")


def parse_json(output: str, source: str) -> dict:
    """Parse JSON output with a source-specific error message."""
    try:
        return json.loads(output)
    except json.JSONDecodeError as error:
        raise ResyncError(f"{source} returned invalid JSON: {error}") from error


def kubectl_env(env: dict[str, str]) -> tuple[str, str, str]:
    """Read the installed operator tag and active feature gates."""
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

    kubevirt = kubectl_json(env, "-n", namespace, "get", "kv", "kubevirt", "-o=json")
    configuration = kubevirt.get("spec", {}).get("configuration", {}) or {}
    developer_configuration = configuration.get("developerConfiguration", {}) or {}
    enabled_gates = set(developer_configuration.get("featureGates", []) or [])
    env["KUBEVIRT_ENABLED_GATES"] = "\n".join(sorted(enabled_gates))
    return namespace, tag, env.get("DOCKER_TAG_ALT", "")


def wait_kubevirt_ready(env: dict[str, str], namespace: str) -> None:
    """Wait until the KubeVirt resource reports its ready conditions."""
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


def cluster_architecture(env: dict[str, str]) -> str:
    """Return the cluster architecture, rejecting mixed or missing values."""
    nodes = kubectl_json(env, "get", "nodes", "-o", "json")
    architectures = set()
    for node in nodes.get("items", []):
        architecture = node.get("status", {}).get("nodeInfo", {}).get("architecture", "")
        if not architecture:
            raise ResyncError("Cannot check deployed images: a cluster node has no architecture")
        architectures.add(architecture)
    if not architectures:
        raise ResyncError("Cannot check deployed images: the cluster has no nodes")
    if len(architectures) != 1:
        raise ResyncError("Cluster resync requires all nodes to use the same architecture")
    return next(iter(architectures))


def registry_image_ref(env: dict[str, str], image: str, image_tag: str, repository_prefix: str = "") -> str:
    """Build a fully qualified image reference for a registry destination."""
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
    repository_prefix: str = "",
    *,
    missing_ok: bool = False,
) -> str | None:
    """Inspect an image digest, optionally treating a missing tag as absent."""
    image_ref = registry_image_ref(env, image, image_tag, repository_prefix)
    local_registry = image_ref.startswith(("localhost:", "127.0.0.1:"))
    args = ["skopeo", "inspect"]
    if local_registry:
        args.append("--tls-verify=false")
    args.extend(("--format", "{{.Digest}}", f"docker://{image_ref}"))
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
        raise ResyncError(f"Cannot inspect registry image {image_ref}: {details}")
    digest = result.stdout.strip()
    if not DIGEST_RE.fullmatch(digest):
        raise ResyncError(f"Registry returned an invalid digest for {image_ref}: {digest}")
    return digest


def parse_image_ref(reference: str) -> str:
    """Extract an image name from a tagged or digest-qualified reference."""
    return reference.split("@", 1)[0].rsplit("/", 1)[-1].split(":", 1)[0]


def selector_matches(labels: dict, selector: dict) -> bool:
    """Return whether Pod labels satisfy a Kubernetes label selector."""
    if not selector.get("matchLabels") and not selector.get("matchExpressions"):
        return False
    if not all(labels.get(key) == value for key, value in selector.get("matchLabels", {}).items()):
        return False
    for expression in selector.get("matchExpressions", []):
        key = expression.get("key")
        operator = expression.get("operator")
        values = expression.get("values", [])
        if operator == "In" and (key not in labels or labels[key] not in values):
            return False
        if operator == "NotIn" and key in labels and labels[key] in values:
            return False
        if operator == "Exists" and key not in labels:
            return False
        if operator == "DoesNotExist" and key in labels:
            return False
        if operator not in {"In", "NotIn", "Exists", "DoesNotExist"}:
            return False
    return True


def pod_image_rows(pod: dict, image: str) -> list[tuple[str, str]]:
    """List every container and init-container image occurrence in one Pod."""
    rows = []
    status = pod.get("status", {})
    spec = pod.get("spec", {})
    statuses = {
        item.get("name"): item
        for item in status.get("containerStatuses", []) + status.get("initContainerStatuses", [])
    }
    containers = spec.get("containers", []) + spec.get("initContainers", [])
    for container in containers:
        name = container.get("name")
        if parse_image_ref(container.get("image", "")) != image:
            continue
        image_id = statuses.get(name, {}).get("imageID", "")
        rows.append((container.get("image", ""), image_id))
    return rows


def workload_mapping(image: str, enabled_gates: set[str]) -> tuple[str, str] | None:
    """Return an image's workload and Pod prefix when it is deployed."""
    spec = IMAGE_SPECS_BY_NAME.get(image)
    if spec is None:
        raise ResyncError(f"No workload mapping for image {image}")
    if spec.workload is None:
        return None
    if spec.feature_gate and spec.feature_gate not in enabled_gates:
        return None
    return spec.workload, spec.workload.rsplit("/", 1)[-1]


def workload_feature_gate(workload: str) -> str | None:
    """Return the feature gate that controls a workload, if any."""
    return next(
        (spec.feature_gate for spec in IMAGE_SPECS if spec.workload == workload and spec.feature_gate),
        None,
    )


def deployed_workload_state(env: dict[str, str], namespace: str) -> dict:
    """Read workload status and Pods once for a set of image checks."""
    resources = kubectl_json(
        env,
        "-n",
        namespace,
        "get",
        "deployments,daemonsets,pods",
        "-o",
        "json",
    )
    workloads = {}
    pods = []
    for resource in resources.get("items", []):
        kind = resource.get("kind")
        name = resource.get("metadata", {}).get("name")
        if kind == "Pod":
            pods.append(resource)
        elif kind in {"Deployment", "DaemonSet"} and name:
            workloads[f"{kind.lower()}/{name}"] = resource
    return {"workloads": workloads, "pods": {"items": pods}}


def image_matches_deployed_pods(
    env: dict[str, str],
    namespace: str,
    image: str,
    image_tag: str,
    current_digest: str,
    enabled_gates: set[str],
    deployed_state: dict | None = None,
) -> bool:
    """Check workload readiness and Pod image IDs against the registry digest."""
    mapping = workload_mapping(image, enabled_gates)
    if mapping is None:
        return True
    workload, _ = mapping
    deployed_state = deployed_state or deployed_workload_state(env, namespace)
    workload_data = deployed_state["workloads"].get(workload)
    if workload_data is None:
        return False
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
            return False
    else:
        expected = status.get("desiredNumberScheduled", 0)
        if not (
            isinstance(expected, int)
            and expected > 0
            and status.get("numberReady", 0) == expected
            and status.get("updatedNumberScheduled", 0) == expected
        ):
            return False

    selector = workload_data.get("spec", {}).get("selector", {})
    pods = [
        pod
        for pod in deployed_state["pods"].get("items", [])
        if selector_matches(pod.get("metadata", {}).get("labels", {}), selector)
    ]
    if len(pods) != expected:
        return False
    for pod in pods:
        status = pod.get("status", {})
        if status.get("phase") != "Running" or not any(
            condition.get("type") == "Ready" and condition.get("status") == "True"
            for condition in status.get("conditions", [])
        ):
            return False
        rows = pod_image_rows(pod, image)
        if not rows:
            return False
        for reference, image_id in rows:
            if reference.rsplit("/", 1)[-1] != f"{image}:{image_tag}" or not image_id:
                return False
            actual_digest = image_id.rsplit("@", 1)[-1]
            actual_digest = actual_digest.removeprefix("containerd://").removeprefix("docker://")
            if actual_digest != current_digest:
                return False
    return True


def build_images(env: dict[str, str], specs: list[tuple[str, str]]) -> None:
    """Build selected Bazel targets and print their digest metadata."""
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
    """Run a script through dockerized with the supplied environment values."""
    prefix = " ".join(f"{key}={shlex.quote(value)}" for key, value in assignments.items())
    result = command([DOCKERIZED, f"{prefix} {script}"], env=env)
    return result.stdout


def parse_built_images(output: str) -> dict[str, BuiltImage]:
    """Read and validate image digests and architectures from build output."""
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


def validate_build_architecture(
    built_images: dict[str, BuiltImage], cluster_architecture: str
) -> None:
    """Fail when the built images do not match the cluster architecture."""
    if all(image.architecture == cluster_architecture for image in built_images.values()):
        return
    built = ", ".join(sorted({image.architecture for image in built_images.values()}))
    raise ResyncError(
        "The local image build architecture does not match the cluster architecture "
        f"(built: {built}; cluster: {cluster_architecture})."
    )


def registry_destinations(
    env: dict[str, str], image: str, tag: str, alt_tag: str
) -> list[tuple[str, str]]:
    """List registry destinations and tags that must contain an image."""
    image_prefix = env.get("IMAGE_PREFIX") or env.get("image_prefix", "")
    image_prefix_alt = env.get("IMAGE_PREFIX_ALT") or env.get("image_prefix_alt", "")
    destinations = [(image_prefix, tag)]
    if alt_tag:
        destinations.append((image_prefix, alt_tag))
    if image_prefix_alt:
        destinations.append((image_prefix_alt, tag))
    return list(dict.fromkeys(destinations))


def plan_image_updates(
    env: dict[str, str],
    specs: list[tuple[str, str]],
    built_images: dict[str, BuiltImage],
    tag: str,
    alt_tag: str,
) -> ImagePlan:
    """Plan pushes by comparing built digests with all registry destinations."""
    plan = ImagePlan()
    for image, _target in specs:
        built_image = built_images.get(image)
        if built_image is None:
            raise ResyncError(f"Build returned no image metadata for {image}")
        needs_push = False
        for destination_index, (repository_prefix, image_tag) in enumerate(
            registry_destinations(env, image, tag, alt_tag)
        ):
            registry_digest = resolve_registry_digest(
                env,
                image,
                image_tag,
                repository_prefix,
                missing_ok=True,
            )
            if destination_index == 0 and registry_digest is not None:
                plan.registry_digests[image] = registry_digest
            if registry_digest != built_image.digest:
                needs_push = True
        if needs_push:
            plan.push.append(image)
    return plan


def verify_published_images(
    env: dict[str, str],
    plan: ImagePlan,
    specs: list[tuple[str, str]],
    built_images: dict[str, BuiltImage],
    tag: str,
    alt_tag: str,
) -> None:
    """Confirm each pushed image digest is present at every destination."""
    pushed_images = set(plan.push)
    for image, _target in specs:
        if image not in pushed_images:
            continue
        built_image = built_images[image]
        for repository_prefix, image_tag in registry_destinations(env, image, tag, alt_tag):
            registry_digest = resolve_registry_digest(
                env,
                image,
                image_tag,
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
    enabled_gates: set[str],
) -> set[str]:
    """Find selected deployed images whose Pods need a rollout."""
    changed = set()
    if not any(
        workload_mapping(image, enabled_gates) is not None for image, _target in specs
    ):
        return changed
    deployed_state = deployed_workload_state(env, namespace)
    for image, _target in specs:
        if workload_mapping(image, enabled_gates) is None:
            continue
        image_tag = tag
        registry_digest = registry_digests_by_image.get(image)
        if registry_digest is None:
            repository_prefix = env.get("IMAGE_PREFIX") or env.get("image_prefix", "")
            registry_digest = resolve_registry_digest(env, image, image_tag, repository_prefix)
        if registry_digest is None:
            raise ResyncError(f"Registry image {image}:{image_tag} is missing")
        matches = image_matches_deployed_pods(
            env,
            namespace,
            image,
            image_tag,
            registry_digest,
            enabled_gates,
            deployed_state,
        )
        if not matches:
            changed.add(image)
    return changed


def verify_deployed_images(
    env: dict[str, str],
    namespace: str,
    specs: list[tuple[str, str]],
    registry_digests_by_image: dict[str, str],
    tag: str,
    enabled_gates: set[str],
    images_to_verify: set[str],
) -> None:
    """Verify selected Pods against the digests recorded for this run."""
    selected_images = [image for image, _target in specs if image in images_to_verify]
    has_workload = any(
        workload_mapping(image, enabled_gates) is not None for image in selected_images
    )
    deployed_state = deployed_workload_state(env, namespace) if has_workload else None
    for image in selected_images:
        image_tag = tag
        registry_digest = registry_digests_by_image.get(image)
        if registry_digest is None:
            raise ResyncError(f"No recorded registry digest for {image}:{image_tag}")
        if workload_mapping(image, enabled_gates) is None:
            print(f"Verified registry image digest for {image}; no workload to check.")
            continue
        matches = image_matches_deployed_pods(
            env,
            namespace,
            image,
            image_tag,
            registry_digest,
            enabled_gates,
            deployed_state,
        )
        if not matches:
            raise ResyncError(
                f"{image} Pods do not match the recorded image digest at final verification"
            )
        print(f"Verified deployed image digest for {image}.")


def push_images(
    env: dict[str, str],
    targets: list[str],
    tag: str,
    alt_tag: str,
    image_prefix: str,
    image_prefix_alt: str,
) -> None:
    """Run the image push script with the selected targets and tags."""
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
) -> None:
    """Push images whose registry digests differ from the build."""
    image_prefix = env.get("IMAGE_PREFIX") or env.get("image_prefix", "")
    image_prefix_alt = env.get("IMAGE_PREFIX_ALT") or env.get("image_prefix_alt", "")
    if plan.push:
        push_images(env, plan.push, tag, alt_tag, image_prefix, image_prefix_alt)


def rollout_changed_workloads(env: dict[str, str], namespace: str, changed: set[str], enabled_gates: set[str]) -> None:
    """Replace Pods for changed images in their configured rollout order."""
    for workload in rollout_targets(changed):
        feature_gate = workload_feature_gate(workload)
        if feature_gate and feature_gate not in enabled_gates:
            print(f"Skipping rollout for {workload}: {feature_gate} is not enabled")
            continue
        replace_workload_pods(env, namespace, workload)


def selector_and_expected(workload_data: dict) -> tuple[str, int]:
    """Return a supported Pod selector and expected replica count."""
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
    """Delete workload Pods and wait for the expected ready replacements."""
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
    """Build selected images, push digest changes, roll out, and verify."""
    env = source_project_environment(env)
    namespace, tag, alt_tag = kubectl_env(env)
    specs = select_images(env)
    architecture = cluster_architecture(env)
    enabled_gates = set(env.get("KUBEVIRT_ENABLED_GATES", "").splitlines())

    build_output = dockerized(
        env,
        {
            "KUBEVIRT_PROVIDER": env["KUBEVIRT_PROVIDER"],
            "BUILD_ARCH": env.get("BUILD_ARCH", ""),
            "PUSH_TARGETS": env.get("PUSH_TARGETS", ""),
            "USE_HINTS": env.get("USE_HINTS", "true"),
        },
        f"{SCRIPT_PATH} build",
    )
    print(build_output, end="")
    built_images = parse_built_images(build_output)
    validate_build_architecture(built_images, architecture)
    plan = plan_image_updates(
        env,
        specs,
        built_images,
        tag,
        alt_tag,
    )

    push_changed_images(env, plan, tag, alt_tag)
    verify_published_images(env, plan, specs, built_images, tag, alt_tag)
    for image in plan.push:
        plan.registry_digests[image] = built_images[image].digest
    plan.changed = find_images_needing_rollout(
        env,
        namespace,
        specs,
        plan.registry_digests,
        tag,
        enabled_gates,
    )

    if not plan.changed:
        wait_kubevirt_ready(env, namespace)
        mapped_images = {
            image
            for image, _target in specs
            if workload_mapping(image, enabled_gates) is not None
        }
        verify_deployed_images(
            env,
            namespace,
            specs,
            plan.registry_digests,
            tag,
            enabled_gates,
            mapped_images,
        )
        report_resync_result(plan, enabled_gates)
        return

    rollout_changed_workloads(env, namespace, plan.changed, enabled_gates)

    wait_kubevirt_ready(env, namespace)
    verify_deployed_images(
        env,
        namespace,
        specs,
        plan.registry_digests,
        tag,
        enabled_gates,
        plan.changed,
    )

    report_resync_result(plan, enabled_gates)


def report_resync_result(plan: ImagePlan, enabled_gates: set[str]) -> None:
    """Summarize pushed images and workloads rolled out by this run."""
    pushed_images = plan.push
    if pushed_images:
        print(f"Pushed images: {' '.join(pushed_images)}")
    images_without_workloads = [
        image for image in pushed_images if workload_mapping(image, enabled_gates) is None
    ]
    if images_without_workloads:
        print(f"No standing workload to roll out for: {' '.join(images_without_workloads)}")
    rolled_out_workloads = rollout_targets(plan.changed)
    if rolled_out_workloads:
        print(f"Rolled out workloads: {' '.join(rolled_out_workloads)}")
    if not pushed_images and not rolled_out_workloads:
        print("Selected images match the registry and deployed workloads.")


def main(argv: list[str] | None = None) -> int:
    """Handle resync subcommands and report expected failures."""
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
