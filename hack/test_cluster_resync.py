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

import json
import subprocess
import unittest
from unittest.mock import patch

from cluster_resync import (
    ROOT,
    SCRIPT_PATH,
    BuiltImage,
    ResyncError,
    bazel_digest_path,
    bazel_image_config_path,
    image_matches_deployed_pods,
    kubectl_env,
    parse_built_images,
    parse_image_ref,
    plan_image_updates,
    resolve_registry_digest,
    replace_workload_pods,
    resync,
    rollout_targets,
    select_images,
    validate_build_architectures,
    workload_mapping,
)


class SelectImagesTest(unittest.TestCase):
    def test_selects_only_requested_image_targets(self):
        specs = select_images({
            "FAST_VIRT_TEMPLATE_ENABLED": "true",
            "PUSH_TARGETS": "virt-controller,virt-launcher",
        })
        self.assertEqual(
            specs,
            [
                ("virt-controller", "//cmd/virt-controller:virt-controller-image"),
                ("virt-launcher", "//cmd/virt-launcher:virt-launcher-image"),
            ],
        )

    def test_controller_hint_selects_launcher(self):
        specs = select_images({"PUSH_TARGETS": "virt-handler"})
        self.assertEqual([image for image, _target in specs], ["virt-handler", "virt-launcher"])

    def test_hints_can_be_disabled(self):
        specs = select_images({"PUSH_TARGETS": "virt-handler", "USE_HINTS": "false"})
        self.assertEqual([image for image, _target in specs], ["virt-handler"])

    def test_rejects_unknown_and_disabled_targets(self):
        with self.assertRaisesRegex(ResyncError, "Unknown or disabled PUSH_TARGETS: virt-template-apiserver"):
            select_images({
                "FAST_VIRT_TEMPLATE_ENABLED": "false",
                "PUSH_TARGETS": "virt-template-apiserver",
            })

    def test_rejects_invalid_hint_setting(self):
        with self.assertRaisesRegex(ResyncError, "USE_HINTS must be true or false"):
            select_images({"USE_HINTS": "yes"})

    def test_rejects_nonempty_targets_that_contain_no_image_names(self):
        with self.assertRaisesRegex(ResyncError, "PUSH_TARGETS is empty"):
            select_images({"PUSH_TARGETS": " ,  , "})


class RolloutTargetsTest(unittest.TestCase):
    def test_launcher_and_controller_use_distinct_workloads(self):
        self.assertEqual(
            rollout_targets({"virt-controller", "virt-launcher"}),
            ["deployment/virt-controller", "daemonset/virt-handler"],
        )

    def test_launcher_and_exportserver_only_restart_handler(self):
        self.assertEqual(
            rollout_targets({"virt-launcher", "virt-exportserver"}),
            ["daemonset/virt-handler"],
        )

    def test_exportserver_has_no_standing_workload(self):
        self.assertEqual(rollout_targets({"virt-exportserver"}), [])

    def test_operator_is_first(self):
        self.assertEqual(
            rollout_targets({"virt-api", "virt-operator", "virt-handler"}),
            [
                "deployment/virt-operator",
                "deployment/virt-api",
                "daemonset/virt-handler",
            ],
        )


class BazelDigestPathTest(unittest.TestCase):
    def test_converts_target_to_digest_file(self):
        self.assertEqual(
            bazel_digest_path("//cmd/virt-api:virt-api-image"),
            ROOT / "bazel-bin/cmd/virt-api/virt-api-image_digest",
        )

    def test_converts_target_to_image_config_file(self):
        self.assertEqual(
            bazel_image_config_path("//cmd/virt-api:virt-api-image"),
            ROOT / "bazel-bin/cmd/virt-api/virt-api-image_config.json",
        )

    def test_builder_entry_point_exists(self):
        self.assertTrue((ROOT / SCRIPT_PATH).is_file())


class ImageReferenceTest(unittest.TestCase):
    def test_parses_tagged_and_digest_pinned_image_names(self):
        self.assertEqual(parse_image_ref("registry:5000/kubevirt/virt-handler:v1"), "virt-handler")
        self.assertEqual(parse_image_ref("registry/kubevirt/virt-api@sha256:abc"), "virt-api")

    def test_exportserver_has_no_standing_workload(self):
        self.assertIsNone(workload_mapping("virt-exportserver", set()))

    def test_sync_controller_requires_feature_gate_for_workload_mapping(self):
        image = "virt-synchronization-controller"
        self.assertIsNone(workload_mapping(image, set()))
        self.assertEqual(
            workload_mapping(image, {"DecentralizedLiveMigration"}),
            (
                "deployment/virt-synchronization-controller",
                "virt-synchronization-controller",
            ),
        )


class KubectlEnvironmentTest(unittest.TestCase):
    def setUp(self):
        self.responses = [
            "registry/kubevirt/virt-operator:devel",
            "true",
            "Template\nDecentralizedLiveMigration\n",
            "",
        ]

    @patch("cluster_resync.kubectl")
    def test_empty_docker_tag_uses_installed_operator_tag(self, run_kubectl):
        run_kubectl.side_effect = self.responses
        self.assertEqual(kubectl_env({"DOCKER_TAG": ""})[1], "devel")

    @patch("cluster_resync.kubectl")
    def test_unset_docker_tag_uses_installed_operator_tag(self, run_kubectl):
        run_kubectl.side_effect = self.responses
        self.assertEqual(kubectl_env({})[1], "devel")

    @patch("cluster_resync.kubectl")
    def test_explicit_docker_tag_must_match_operator_tag(self, run_kubectl):
        run_kubectl.side_effect = self.responses
        with self.assertRaisesRegex(ResyncError, "differs from the installed operator tag"):
            kubectl_env({"DOCKER_TAG": "other"})


class BuiltImagePlanTest(unittest.TestCase):
    digest = "sha256:" + "a" * 64

    def test_parses_manifest_digests_and_architectures(self):
        built_images = parse_built_images(
            "FAST_IMAGE_DIGEST=virt-api=sha256:" + "a" * 64 + "\n"
            "FAST_IMAGE_ARCH=virt-api=amd64\n"
        )

        self.assertEqual(built_images, {"virt-api": BuiltImage(self.digest, "amd64")})

    def test_rejects_cluster_architectures_without_matching_local_builds(self):
        with self.assertRaisesRegex(ResyncError, "Build each cluster architecture"):
            validate_build_architectures(
                {"virt-api": BuiltImage(self.digest, "amd64")}, ["amd64", "arm64"]
            )

    @patch("cluster_resync.resolve_registry_digest")
    def test_skips_push_when_built_manifest_matches_registry(self, registry_digest):
        registry_digest.return_value = self.digest

        plan = plan_image_updates(
            {"docker_prefix": "registry:5000"},
            [("virt-api", "//cmd/virt-api:virt-api-image")],
            {"virt-api": BuiltImage(self.digest, "amd64")},
            "devel",
            "",
            "v0.3.0",
        )

        self.assertEqual(plan.core_push, [])
        registry_digest.assert_called_once_with(
            {"docker_prefix": "registry:5000"}, "virt-api", "devel", "amd64", "", missing_ok=True
        )

    @patch("cluster_resync.resolve_registry_digest")
    def test_pushes_when_registry_manifest_differs(self, registry_digest):
        registry_digest.return_value = "sha256:" + "b" * 64

        plan = plan_image_updates(
            {"docker_prefix": "registry:5000"},
            [("virt-api", "//cmd/virt-api:virt-api-image")],
            {"virt-api": BuiltImage(self.digest, "amd64")},
            "devel",
            "",
            "v0.3.0",
        )

        self.assertEqual(plan.core_push, ["virt-api"])

    @patch("cluster_resync.resolve_registry_digest")
    def test_pushes_when_an_alternate_tag_is_missing_or_stale(self, registry_digest):
        registry_digest.side_effect = [self.digest, None]

        plan = plan_image_updates(
            {"docker_prefix": "registry:5000"},
            [("virt-api", "//cmd/virt-api:virt-api-image")],
            {"virt-api": BuiltImage(self.digest, "amd64")},
            "devel",
            "alt",
            "v0.3.0",
        )

        self.assertEqual(plan.core_push, ["virt-api"])

    @patch("cluster_resync.resolve_registry_digest")
    def test_template_uses_its_configured_tag(self, registry_digest):
        registry_digest.return_value = self.digest
        env = {"FAST_VIRT_TEMPLATE_ENABLED": "true", "docker_prefix": "registry:5000"}

        plan_image_updates(
            env,
            [("virt-template-controller", "//images/virt-template:virt-template-controller-image")],
            {"virt-template-controller": BuiltImage(self.digest, "amd64")},
            "devel",
            "alt",
            "v0.3.0",
        )

        registry_digest.assert_called_once_with(env, "virt-template-controller", "v0.3.0", "amd64", "", missing_ok=True)

    @patch("cluster_resync.resolve_registry_digest")
    def test_checks_alternate_repository_as_a_push_destination(self, registry_digest):
        registry_digest.side_effect = [self.digest, "sha256:" + "b" * 64]
        env = {
            "docker_prefix": "registry:5000",
            "image_prefix": "stable/",
            "image_prefix_alt": "test/",
        }

        plan = plan_image_updates(
            env,
            [("virt-api", "//cmd/virt-api:virt-api-image")],
            {"virt-api": BuiltImage(self.digest, "amd64")},
            "devel",
            "",
            "v0.3.0",
        )

        self.assertEqual(plan.core_push, ["virt-api"])
        self.assertEqual(registry_digest.call_args_list[1].args[4], "test/")

    @patch("cluster_resync.subprocess.run")
    def test_only_missing_manifest_errors_trigger_push(self, run_process):
        run_process.return_value = subprocess.CompletedProcess(
            [], 1, "", "manifest unknown"
        )
        self.assertIsNone(
            resolve_registry_digest(
                {"docker_prefix": "registry"}, "virt-api", "devel", "amd64", missing_ok=True
            )
        )

        run_process.return_value = subprocess.CompletedProcess([], 1, "", "connection refused")
        with self.assertRaisesRegex(ResyncError, "connection refused"):
            resolve_registry_digest(
                {"docker_prefix": "registry"}, "virt-api", "devel", "amd64", missing_ok=True
            )

    @patch("cluster_resync.subprocess.run")
    def test_reads_the_local_registry_manifest_digest(self, run_process):
        run_process.return_value = subprocess.CompletedProcess([], 0, self.digest, "")

        digest = resolve_registry_digest(
            {"docker_prefix": "localhost:5000"}, "virt-api", "devel", "amd64", "kubevirt"
        )

        self.assertEqual(digest, self.digest)
        args = run_process.call_args.args[0]
        self.assertIn("--tls-verify=false", args)
        self.assertEqual(args[args.index("--override-arch") + 1], "amd64")
        self.assertIn("docker://localhost:5000/kubevirt/virt-api:devel", args)


class DeployedImageCheckTest(unittest.TestCase):
    def setUp(self):
        self.digest = "sha256:" + "a" * 64
        self.workload = {
            "metadata": {"generation": 2},
            "spec": {"replicas": 1},
            "status": {
                "readyReplicas": 1,
                "updatedReplicas": 1,
                "observedGeneration": 2,
            },
        }
        self.pods = {
            "items": [
                {
                    "metadata": {"name": "virt-api-abc"},
                    "spec": {
                        "nodeName": "node01",
                        "containers": [{"name": "api", "image": "registry/kubevirt/virt-api:devel"}],
                    },
                    "status": {
                        "phase": "Running",
                        "conditions": [{"type": "Ready", "status": "True"}],
                        "containerStatuses": [{"name": "api", "imageID": f"containerd://{self.digest}"}],
                    },
                }
            ],
        }

    @patch("cluster_resync.command")
    def test_accepts_ready_pods_with_current_architecture_digest(self, run_command):
        run_command.side_effect = [
            subprocess.CompletedProcess([], 0, json.dumps(self.workload)),
            subprocess.CompletedProcess([], 0, json.dumps(self.pods)),
        ]

        matches, digests = image_matches_deployed_pods(
            {},
            "kubevirt",
            "virt-api",
            "devel",
            f"amd64={self.digest}",
            {"node01": "amd64"},
            set(),
        )

        self.assertTrue(matches)
        self.assertEqual(digests, f"amd64={self.digest}")

    def test_stale_running_pod_digest_invalidates_live_comparison(self):
        stale_pods = json.loads(json.dumps(self.pods))
        stale_pods["items"][0]["status"]["containerStatuses"][0]["imageID"] = (
            "containerd://sha256:" + "b" * 64
        )
        with patch("cluster_resync.command") as run_command:
            run_command.side_effect = [
                subprocess.CompletedProcess([], 0, json.dumps(self.workload)),
                subprocess.CompletedProcess([], 0, json.dumps(stale_pods)),
            ]
            matches, _digests = image_matches_deployed_pods(
                {}, "kubevirt", "virt-api", "devel", f"amd64={self.digest}",
                {"node01": "amd64"}, set(),
            )

        self.assertFalse(matches)

    def test_missing_ready_pod_invalidates_live_comparison(self):
        with patch("cluster_resync.command") as run_command:
            run_command.side_effect = [
                subprocess.CompletedProcess([], 0, json.dumps(self.workload)),
                subprocess.CompletedProcess([], 0, json.dumps({"items": []})),
            ]

            matches, _digests = image_matches_deployed_pods(
                {},
                "kubevirt",
                "virt-api",
                "devel",
                f"amd64={self.digest}",
                {"node01": "amd64"},
                set(),
            )

        self.assertFalse(matches)


class ResyncFlowTest(unittest.TestCase):
    def test_registry_mismatch_pushes_and_pod_mismatch_rolls_out(self):
        build_digest = "sha256:" + "a" * 64
        old_registry_digest = "sha256:" + "b" * 64
        env = {
            "KUBEVIRT_PROVIDER": "k8s-1.37",
            "FAST_VIRT_TEMPLATE_ENABLED": "false",
            "USE_HINTS": "false",
            "docker_prefix": "registry:5000",
        }
        specs = [("virt-api", "//cmd/virt-api:virt-api-image")]
        built_output = (
            f"FAST_IMAGE_DIGEST=virt-api={build_digest}\n"
            "FAST_IMAGE_ARCH=virt-api=amd64\n"
        )

        with (
            patch("cluster_resync.source_project_environment", return_value=env),
            patch("cluster_resync.kubectl_env", return_value=("kubevirt", "devel", "", "v0.3.0")),
            patch("cluster_resync.select_images", return_value=specs),
            patch("cluster_resync.check_template_tags"),
            patch("cluster_resync.cluster_node_architectures", return_value={"node01": "amd64"}),
            patch("cluster_resync.dockerized", return_value=built_output),
            patch("cluster_resync.resolve_registry_digest", return_value=old_registry_digest),
            patch("cluster_resync.push_changed_images") as push_images,
            patch("cluster_resync.verify_published_images") as verify_publish,
            patch("cluster_resync.resolve_registry_digests", return_value=f"amd64={build_digest}"),
            patch(
                "cluster_resync.image_matches_deployed_pods",
                side_effect=[(False, f"amd64={build_digest}"), (True, f"amd64={build_digest}")],
            ),
            patch("cluster_resync.rollout_changed_workloads") as rollout,
            patch("cluster_resync.wait_kubevirt_ready"),
            patch("builtins.print"),
        ):
            resync(env)

        push_plan = push_images.call_args.args[1]
        self.assertEqual(push_plan.core_push, ["virt-api"])
        rollout.assert_called_once_with(env, "kubevirt", {"virt-api"}, set())
        self.assertEqual(verify_publish.call_count, 2)


class WorkloadRecoveryTest(unittest.TestCase):
    def test_waits_for_controller_when_no_pods_are_present(self):
        workload = {
            "kind": "Deployment",
            "spec": {"replicas": 1, "selector": {"matchLabels": {"app": "virt-api"}}},
        }
        ready_pods = {
            "items": [
                {
                    "status": {"conditions": [{"type": "Ready", "status": "True"}]},
                }
            ],
        }
        with (
            patch("cluster_resync.kubectl_json", side_effect=[workload, ready_pods]),
            patch("cluster_resync.kubectl", side_effect=["", ""]) as run_kubectl,
            patch("cluster_resync.time.sleep"),
        ):
            replace_workload_pods({}, "kubevirt", "deployment/virt-api")

        self.assertEqual(run_kubectl.call_count, 2)
        self.assertIn("rollout", run_kubectl.call_args_list[1].args)


if __name__ == "__main__":
    unittest.main()
