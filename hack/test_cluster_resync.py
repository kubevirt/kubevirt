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
    ImagePlan,
    ResyncError,
    bazel_digest_path,
    bazel_image_config_path,
    cluster_architecture,
    find_images_needing_rollout,
    image_matches_deployed_pods,
    kubectl_env,
    parse_built_images,
    parse_image_ref,
    plan_image_updates,
    report_resync_result,
    resolve_registry_digest,
    replace_workload_pods,
    resync,
    rollout_targets,
    select_images,
    validate_build_architecture,
    verify_deployed_images,
    workload_mapping,
)


class SelectImagesTest(unittest.TestCase):
    def test_full_selection_contains_only_resync_supported_images(self):
        self.assertEqual(
            [image for image, _target in select_images({})],
            [
                "virt-operator",
                "virt-api",
                "virt-controller",
                "virt-handler",
                "virt-launcher",
                "virt-exportserver",
                "virt-exportproxy",
                "virt-synchronization-controller",
            ],
        )

    def test_selects_only_requested_image_targets(self):
        specs = select_images({
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

    def test_rejects_removed_template_image_targets(self):
        with self.assertRaisesRegex(ResyncError, "Unknown image targets: virt-template-apiserver"):
            select_images({
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
        self.operator_image = "registry/kubevirt/virt-operator:devel"
        self.kubevirt = {
            "spec": {
                "configuration": {
                    "developerConfiguration": {
                        "featureGates": ["DecentralizedLiveMigration"],
                    },
                }
            }
        }

    @patch("cluster_resync.kubectl_json")
    @patch("cluster_resync.kubectl")
    def test_empty_docker_tag_uses_installed_operator_tag(self, run_kubectl, run_kubectl_json):
        run_kubectl.return_value = self.operator_image
        run_kubectl_json.return_value = self.kubevirt
        self.assertEqual(kubectl_env({"DOCKER_TAG": ""})[1], "devel")

    @patch("cluster_resync.kubectl_json")
    @patch("cluster_resync.kubectl")
    def test_unset_docker_tag_uses_installed_operator_tag(self, run_kubectl, run_kubectl_json):
        run_kubectl.return_value = self.operator_image
        run_kubectl_json.return_value = self.kubevirt
        self.assertEqual(kubectl_env({})[1], "devel")

    @patch("cluster_resync.kubectl_json")
    @patch("cluster_resync.kubectl")
    def test_explicit_docker_tag_must_match_operator_tag(self, run_kubectl, _run_kubectl_json):
        run_kubectl.return_value = self.operator_image
        with self.assertRaisesRegex(ResyncError, "differs from the installed operator tag"):
            kubectl_env({"DOCKER_TAG": "other"})

    @patch("cluster_resync.kubectl_json")
    @patch("cluster_resync.kubectl")
    def test_enabled_feature_gates_are_exported(self, run_kubectl, run_kubectl_json):
        run_kubectl.return_value = self.operator_image
        run_kubectl_json.return_value = self.kubevirt
        env = {}

        kubectl_env(env)

        self.assertEqual(env["KUBEVIRT_ENABLED_GATES"].splitlines(), ["DecentralizedLiveMigration"])


class BuiltImagePlanTest(unittest.TestCase):
    digest = "sha256:" + "a" * 64

    def test_parses_manifest_digests_and_architectures(self):
        built_images = parse_built_images(
            "FAST_IMAGE_DIGEST=virt-api=sha256:" + "a" * 64 + "\n"
            "FAST_IMAGE_ARCH=virt-api=amd64\n"
        )

        self.assertEqual(built_images, {"virt-api": BuiltImage(self.digest, "amd64")})

    def test_rejects_build_architecture_that_differs_from_cluster(self):
        with self.assertRaisesRegex(ResyncError, "does not match the cluster architecture"):
            validate_build_architecture(
                {"virt-api": BuiltImage(self.digest, "amd64")}, "arm64"
            )

    def test_accepts_cluster_architecture_when_all_images_match(self):
        validate_build_architecture(
            {"virt-api": BuiltImage(self.digest, "amd64")}, "amd64"
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
        )

        self.assertEqual(plan.push, [])
        registry_digest.assert_called_once_with(
            {"docker_prefix": "registry:5000"}, "virt-api", "devel", "", missing_ok=True
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
        )

        self.assertEqual(plan.push, ["virt-api"])

    @patch("cluster_resync.resolve_registry_digest")
    def test_pushes_when_an_alternate_tag_is_missing_or_stale(self, registry_digest):
        registry_digest.side_effect = [self.digest, None]

        plan = plan_image_updates(
            {"docker_prefix": "registry:5000"},
            [("virt-api", "//cmd/virt-api:virt-api-image")],
            {"virt-api": BuiltImage(self.digest, "amd64")},
            "devel",
            "alt",
        )

        self.assertEqual(plan.push, ["virt-api"])

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
        )

        self.assertEqual(plan.push, ["virt-api"])
        self.assertEqual(registry_digest.call_args_list[1].args[3], "test/")

    @patch("cluster_resync.subprocess.run")
    def test_only_missing_manifest_errors_trigger_push(self, run_process):
        run_process.return_value = subprocess.CompletedProcess(
            [], 1, "", "manifest unknown"
        )
        self.assertIsNone(
            resolve_registry_digest(
                {"docker_prefix": "registry"}, "virt-api", "devel", missing_ok=True
            )
        )

        run_process.return_value = subprocess.CompletedProcess([], 1, "", "connection refused")
        with self.assertRaisesRegex(ResyncError, "connection refused"):
            resolve_registry_digest(
                {"docker_prefix": "registry"}, "virt-api", "devel", missing_ok=True
            )

    @patch("cluster_resync.subprocess.run")
    def test_reads_the_local_registry_manifest_digest(self, run_process):
        run_process.return_value = subprocess.CompletedProcess([], 0, self.digest, "")

        digest = resolve_registry_digest(
            {"docker_prefix": "localhost:5000"}, "virt-api", "devel", "kubevirt"
        )

        self.assertEqual(digest, self.digest)
        args = run_process.call_args.args[0]
        self.assertIn("--tls-verify=false", args)
        self.assertNotIn("--override-arch", args)
        self.assertIn("docker://localhost:5000/kubevirt/virt-api:devel", args)


class DeployedImageCheckTest(unittest.TestCase):
    def setUp(self):
        self.digest = "sha256:" + "a" * 64
        self.workload = {
            "kind": "Deployment",
            "metadata": {"name": "virt-api", "generation": 2},
            "spec": {
                "replicas": 1,
                "selector": {"matchLabels": {"app": "virt-api"}},
            },
            "status": {
                "readyReplicas": 1,
                "updatedReplicas": 1,
                "observedGeneration": 2,
            },
        }
        self.pods = {
            "items": [
                {
                    "kind": "Pod",
                    "metadata": {
                        "name": "virt-api-abc",
                        "labels": {"app": "virt-api"},
                    },
                    "spec": {
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

    @patch("cluster_resync.kubectl_json")
    def test_accepts_ready_pods_with_current_digest(self, run_kubectl):
        run_kubectl.return_value = {"items": [self.workload, *self.pods["items"]]}

        matches = image_matches_deployed_pods(
            {},
            "kubevirt",
            "virt-api",
            "devel",
            self.digest,
            set(),
        )

        self.assertTrue(matches)
        run_kubectl.assert_called_once_with(
            {}, "-n", "kubevirt", "get", "deployments,daemonsets,pods", "-o", "json"
        )

    def test_stale_running_pod_digest_invalidates_live_comparison(self):
        stale_pods = json.loads(json.dumps(self.pods))
        stale_pods["items"][0]["status"]["containerStatuses"][0]["imageID"] = (
            "containerd://sha256:" + "b" * 64
        )
        with patch("cluster_resync.kubectl_json", return_value={
            "items": [self.workload, *stale_pods["items"]],
        }):
            matches = image_matches_deployed_pods(
                {}, "kubevirt", "virt-api", "devel", self.digest, set(),
            )

        self.assertFalse(matches)

    def test_rollout_scan_reads_deployed_state_once_for_all_selected_images(self):
        digest = "sha256:" + "a" * 64
        specs = [
            ("virt-api", "//cmd/virt-api:virt-api-image"),
            ("virt-controller", "//cmd/virt-controller:virt-controller-image"),
        ]
        state = {"workloads": {}, "pods": {"items": []}}
        with (
            patch("cluster_resync.deployed_workload_state", return_value=state) as read_state,
            patch("cluster_resync.image_matches_deployed_pods", side_effect=[True, False]) as matches,
        ):
            changed = find_images_needing_rollout(
                {},
                "kubevirt",
                specs,
                {"virt-api": digest, "virt-controller": digest},
                "devel",
                set(),
            )

        self.assertEqual(changed, {"virt-controller"})
        read_state.assert_called_once_with({}, "kubevirt")
        self.assertEqual(matches.call_args_list[0].args[-1], state)
        self.assertEqual(matches.call_args_list[1].args[-1], state)

    def test_missing_ready_pod_invalidates_live_comparison(self):
        with patch("cluster_resync.kubectl_json", return_value={"items": [self.workload]}):

            matches = image_matches_deployed_pods(
                {},
                "kubevirt",
                "virt-api",
                "devel",
                self.digest,
                set(),
            )

        self.assertFalse(matches)

    def test_duplicate_image_occurrences_count_as_one_pod_and_all_are_checked(self):
        workload = {
            "kind": "DaemonSet",
            "spec": {"selector": {"matchLabels": {"app": "virt-handler"}}},
            "status": {
                "desiredNumberScheduled": 1,
                "numberReady": 1,
                "updatedNumberScheduled": 1,
            },
        }
        pod = {
            "kind": "Pod",
            "metadata": {"name": "virt-handler-node-a", "labels": {"app": "virt-handler"}},
            "spec": {
                "containers": [
                    {"name": "handler", "image": "registry/kubevirt/virt-handler:devel"},
                    {"name": "image-holder", "image": "registry/kubevirt/virt-launcher:devel"},
                ],
                "initContainers": [
                    {"name": "node-labeller", "image": "registry/kubevirt/virt-launcher:devel"},
                ],
            },
            "status": {
                "phase": "Running",
                "conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [
                    {"name": "handler", "imageID": f"containerd://{self.digest}"},
                    {"name": "image-holder", "imageID": f"containerd://{self.digest}"},
                ],
                "initContainerStatuses": [
                    {"name": "node-labeller", "imageID": f"containerd://{self.digest}"},
                ],
            },
        }
        deployed_state = {
            "workloads": {"daemonset/virt-handler": workload},
            "pods": {"items": [pod]},
        }

        self.assertTrue(image_matches_deployed_pods(
            {}, "kubevirt", "virt-launcher", "devel", self.digest, set(), deployed_state
        ))

        pod["status"]["containerStatuses"][1]["imageID"] = "containerd://sha256:" + "b" * 64
        self.assertFalse(image_matches_deployed_pods(
            {}, "kubevirt", "virt-launcher", "devel", self.digest, set(), deployed_state
        ))

        pod["status"]["containerStatuses"][1]["imageID"] = f"containerd://{self.digest}"
        pod["status"]["initContainerStatuses"] = []
        self.assertFalse(image_matches_deployed_pods(
            {}, "kubevirt", "virt-launcher", "devel", self.digest, set(), deployed_state
        ))


class ClusterArchitectureTest(unittest.TestCase):
    @patch("cluster_resync.kubectl_json")
    def test_returns_the_single_cluster_architecture(self, run_kubectl):
        run_kubectl.return_value = {
            "items": [
                {"status": {"nodeInfo": {"architecture": "amd64"}}},
                {"status": {"nodeInfo": {"architecture": "amd64"}}},
            ]
        }

        self.assertEqual(cluster_architecture({}), "amd64")

    @patch("cluster_resync.kubectl_json")
    def test_rejects_mixed_cluster_architectures(self, run_kubectl):
        run_kubectl.return_value = {
            "items": [
                {"status": {"nodeInfo": {"architecture": "amd64"}}},
                {"status": {"nodeInfo": {"architecture": "arm64"}}},
            ]
        }

        with self.assertRaisesRegex(ResyncError, "same architecture"):
            cluster_architecture({})

    @patch("cluster_resync.kubectl_json")
    def test_rejects_nodes_without_architecture(self, run_kubectl):
        run_kubectl.return_value = {"items": [{"status": {"nodeInfo": {}}}]}

        with self.assertRaisesRegex(ResyncError, "node has no architecture"):
            cluster_architecture({})


class VerificationMessageTest(unittest.TestCase):
    def test_no_workload_image_is_reported_as_registry_only(self):
        digest = "sha256:" + "a" * 64
        with (
            patch("cluster_resync.image_matches_deployed_pods", return_value=True),
            patch("cluster_resync.deployed_workload_state", return_value={"workloads": {}, "pods": {"items": []}}),
            patch("cluster_resync.resolve_registry_digest") as resolve_registry_digest,
            patch("builtins.print") as print_message,
        ):
            verify_deployed_images(
                {},
                "kubevirt",
                [("virt-exportserver", "//cmd/virt-exportserver:virt-exportserver-image")],
                {"virt-exportserver": digest},
                "devel",
                set(),
                {"virt-exportserver"},
            )

        resolve_registry_digest.assert_not_called()
        print_message.assert_called_once_with(
            "Verified registry image digest for virt-exportserver; no workload to check."
        )

    def test_post_rollout_verification_checks_only_changed_images(self):
        digest = "sha256:" + "a" * 64
        specs = [
            ("virt-api", "//cmd/virt-api:virt-api-image"),
            ("virt-controller", "//cmd/virt-controller:virt-controller-image"),
        ]
        with (
            patch("cluster_resync.image_matches_deployed_pods", return_value=True) as matches,
            patch("cluster_resync.deployed_workload_state", return_value={"workloads": {}, "pods": {"items": []}}),
            patch("cluster_resync.resolve_registry_digest") as resolve_registry_digest,
            patch("builtins.print"),
        ):
            verify_deployed_images(
                {},
                "kubevirt",
                specs,
                {"virt-api": digest, "virt-controller": digest},
                "devel",
                set(),
                {"virt-api"},
            )

        resolve_registry_digest.assert_not_called()
        matches.assert_called_once_with(
            {}, "kubevirt", "virt-api", "devel", digest, set(),
            {"workloads": {}, "pods": {"items": []}},
        )

    def test_resync_summary_separates_push_from_rollout(self):
        plan = ImagePlan(
            changed={"virt-api", "virt-exportserver"},
            push=["virt-api", "virt-exportserver"],
        )
        with patch("builtins.print") as print_message:
            report_resync_result(plan, set())

        self.assertEqual(
            [call.args[0] for call in print_message.call_args_list],
            [
                "Pushed images: virt-api virt-exportserver",
                "No standing workload to roll out for: virt-exportserver",
                "Rolled out workloads: deployment/virt-api",
            ],
        )

    def test_noop_resync_summary_reports_no_changes(self):
        with patch("builtins.print") as print_message:
            report_resync_result(ImagePlan(), set())

        print_message.assert_called_once_with(
            "Selected images match the registry and deployed workloads."
        )


class ResyncFlowTest(unittest.TestCase):
    def test_noop_rechecks_deployed_images_after_kubevirt_is_ready(self):
        digest = "sha256:" + "a" * 64
        env = {
            "KUBEVIRT_PROVIDER": "k8s-1.37",
            "PUSH_TARGETS": "virt-api",
        }
        specs = [("virt-api", "//cmd/virt-api:virt-api-image")]
        plan = ImagePlan(registry_digests={"virt-api": digest})
        events = []
        built_output = f"FAST_IMAGE_DIGEST=virt-api={digest}\nFAST_IMAGE_ARCH=virt-api=amd64\n"

        with (
            patch("cluster_resync.source_project_environment", return_value=env),
            patch("cluster_resync.kubectl_env", return_value=("kubevirt", "devel", "")),
            patch("cluster_resync.select_images", return_value=specs),
            patch("cluster_resync.cluster_architecture", return_value="amd64"),
            patch("cluster_resync.dockerized", return_value=built_output),
            patch("cluster_resync.plan_image_updates", return_value=plan),
            patch("cluster_resync.push_changed_images"),
            patch("cluster_resync.verify_published_images"),
            patch("cluster_resync.find_images_needing_rollout", return_value=set()),
            patch("cluster_resync.wait_kubevirt_ready", side_effect=lambda *_args: events.append("ready")),
            patch("cluster_resync.verify_deployed_images", side_effect=lambda *args: events.append(("verify", args))),
            patch("builtins.print"),
        ):
            resync(env)

        self.assertEqual(events[0], "ready")
        self.assertEqual(
            events[1],
            (
                "verify",
                (env, "kubevirt", specs, {"virt-api": digest}, "devel", set(), {"virt-api"}),
            ),
        )

    def test_registry_mismatch_pushes_and_pod_mismatch_rolls_out(self):
        build_digest = "sha256:" + "a" * 64
        old_registry_digest = "sha256:" + "b" * 64
        env = {
            "KUBEVIRT_PROVIDER": "k8s-1.37",
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
            patch("cluster_resync.kubectl_env", return_value=("kubevirt", "devel", "")),
            patch("cluster_resync.select_images", return_value=specs),
            patch("cluster_resync.cluster_architecture", return_value="amd64"),
            patch("cluster_resync.dockerized", return_value=built_output),
            patch("cluster_resync.resolve_registry_digest", return_value=old_registry_digest),
            patch("cluster_resync.push_changed_images") as push_images,
            patch("cluster_resync.verify_published_images") as verify_publish,
            patch("cluster_resync.deployed_workload_state", return_value={"workloads": {}, "pods": {"items": []}}),
            patch(
                "cluster_resync.image_matches_deployed_pods",
                side_effect=[False, True],
            ),
            patch("cluster_resync.rollout_changed_workloads") as rollout,
            patch("cluster_resync.wait_kubevirt_ready"),
            patch("builtins.print"),
        ):
            resync(env)

        push_plan = push_images.call_args.args[1]
        self.assertEqual(push_plan.push, ["virt-api"])
        rollout.assert_called_once_with(env, "kubevirt", {"virt-api"}, set())
        self.assertEqual(verify_publish.call_count, 1)


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
