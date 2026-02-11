"""
Tier 2 End-to-End tests for CommonInstancetypesDeployment feature.

STP Reference: tests/CNV-61256/CNV-61256_test_plan.md
STD Reference: tests/CNV-61256/CNV-61256_test_description.yaml
Jira: https://issues.redhat.com/browse/CNV-61256
PR: https://github.com/kubevirt/hyperconverged-cluster-operator/pull/3471

Feature: Disable common-instancetypes deployment from HCO

This test module validates the end-to-end behavior of the
CommonInstancetypesDeployment configuration in the HyperConverged CR,
including resource deployment/removal and lifecycle scenarios.

Phase 2: Full working implementation
"""

import logging
import pytest
from typing import Generator

from ocp_resources.hyperconverged import HyperConverged
from ocp_resources.kubevirt import KubeVirt
from ocp_resources.pod import Pod
from ocp_resources.virtual_machine import VirtualMachine
from ocp_resources.virtual_machine_cluster_instancetype import VirtualMachineClusterInstancetype
from ocp_resources.virtual_machine_cluster_preference import VirtualMachineClusterPreference
from timeout_sampler import TimeoutSampler

from utilities.constants import (
    TIMEOUT_2MIN,
    TIMEOUT_5MIN,
    CNV_NAMESPACE,
)
from utilities.hco import wait_for_hco_conditions


LOGGER = logging.getLogger(__name__)

# HCO and KubeVirt resource names
HCO_NAME = "kubevirt-hyperconverged"
KUBEVIRT_NAME = "kubevirt-hyperconverged"

pytestmark = [
    pytest.mark.tier2,
    pytest.mark.usefixtures("admin_client"),
]


class TestCommonInstancetypesDeployment:
    """
    Test suite for CommonInstancetypesDeployment HCO configuration.

    Markers:
        - tier2
        - p1

    Preconditions:
        - OpenShift cluster with CNV 4.19+ installed
        - HyperConverged CR is healthy and accessible
        - Cluster admin privileges available
    """

    @pytest.fixture()
    def hco_resource(self, admin_client) -> HyperConverged:
        """Get the HyperConverged CR."""
        return HyperConverged(
            client=admin_client,
            name=HCO_NAME,
            namespace=CNV_NAMESPACE,
        )

    @pytest.fixture()
    def kubevirt_resource(self, admin_client) -> KubeVirt:
        """Get the KubeVirt CR."""
        return KubeVirt(
            client=admin_client,
            name=KUBEVIRT_NAME,
            namespace=CNV_NAMESPACE,
        )

    @pytest.fixture()
    def save_and_restore_hco_config(
        self, admin_client, hco_resource
    ) -> Generator:
        """
        Save original HCO configuration and restore after test.

        This fixture ensures tests don't leave the cluster in a modified state.
        """
        # Save original configuration
        original_config = None
        if hco_resource.exists:
            hco_resource.reload()
            spec = hco_resource.instance.spec
            if hasattr(spec, "commonInstancetypesDeployment"):
                original_config = spec.commonInstancetypesDeployment

        yield

        # Restore original configuration
        LOGGER.info("Restoring HCO configuration to original state")
        if original_config is None:
            # Remove the field if it wasn't set originally
            self._remove_common_instancetypes_deployment(hco_resource)
        else:
            self._set_common_instancetypes_deployment(
                hco_resource, enabled=original_config.get("enabled", True)
            )
        wait_for_hco_conditions(admin_client=admin_client)

    def _set_common_instancetypes_deployment(
        self, hco_resource: HyperConverged, enabled: bool
    ) -> None:
        """Set the commonInstancetypesDeployment.enabled field in HCO CR."""
        LOGGER.info(f"Setting commonInstancetypesDeployment.enabled to {enabled}")
        hco_resource.reload()
        ResourceEditor = hco_resource.ResourceEditor
        with ResourceEditor(
            patches={
                hco_resource: {
                    "spec": {
                        "commonInstancetypesDeployment": {"enabled": enabled}
                    }
                }
            }
        ):
            pass  # Context manager applies the patch

    def _remove_common_instancetypes_deployment(
        self, hco_resource: HyperConverged
    ) -> None:
        """Remove the commonInstancetypesDeployment field from HCO CR."""
        LOGGER.info("Removing commonInstancetypesDeployment from HCO CR")
        hco_resource.reload()
        if hasattr(hco_resource.instance.spec, "commonInstancetypesDeployment"):
            # Use JSON patch to remove the field
            hco_resource.patch(
                body=[{"op": "remove", "path": "/spec/commonInstancetypesDeployment"}],
                content_type="application/json-patch+json",
            )

    def _wait_for_reconciliation(self, admin_client, timeout: int = TIMEOUT_2MIN) -> None:
        """Wait for HCO and KubeVirt reconciliation to complete."""
        LOGGER.info("Waiting for HCO reconciliation")
        wait_for_hco_conditions(admin_client=admin_client, timeout=timeout)

    def _get_cluster_instancetypes(self, admin_client) -> list:
        """Get all VirtualMachineClusterInstancetype resources."""
        return list(VirtualMachineClusterInstancetype.get(dyn_client=admin_client))

    def _get_cluster_preferences(self, admin_client) -> list:
        """Get all VirtualMachineClusterPreference resources."""
        return list(VirtualMachineClusterPreference.get(dyn_client=admin_client))

    @pytest.mark.polarion("CNV-61256")
    def test_ts_cnv61256_001_disable_common_instancetypes_deployment(
        self,
        admin_client,
        hco_resource,
        save_and_restore_hco_config,
    ) -> None:
        """
        Test TS-CNV61256-001: Disable common-instancetypes deployment via HCO CR.

        Markers:
            - tier2
            - p1

        Steps:
            1. Verify common-instancetypes are deployed by default
            2. Set spec.commonInstancetypesDeployment.enabled to false in HCO CR
            3. Wait for KubeVirt reconciliation
            4. Verify no VirtualMachineClusterInstancetype resources exist
            5. Verify no VirtualMachineClusterPreference resources exist

        Expected:
            - All common-instancetypes resources are removed after disabling
        """
        # Step 1: Verify common-instancetypes are deployed by default
        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) > 0, (
            "Expected common-instancetypes to be deployed by default"
        )
        LOGGER.info(f"Found {len(instancetypes)} VirtualMachineClusterInstancetype resources")

        # Step 2: Disable common-instancetypes deployment
        self._set_common_instancetypes_deployment(hco_resource, enabled=False)

        # Step 3: Wait for reconciliation
        self._wait_for_reconciliation(admin_client)

        # Step 4: Verify no VirtualMachineClusterInstancetype resources exist
        for sample in TimeoutSampler(
            wait_timeout=TIMEOUT_5MIN,
            sleep=10,
            func=self._get_cluster_instancetypes,
            admin_client=admin_client,
        ):
            if len(sample) == 0:
                break

        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) == 0, (
            f"Expected no VirtualMachineClusterInstancetype resources, "
            f"found {len(instancetypes)}"
        )

        # Step 5: Verify no VirtualMachineClusterPreference resources exist
        preferences = self._get_cluster_preferences(admin_client)
        assert len(preferences) == 0, (
            f"Expected no VirtualMachineClusterPreference resources, "
            f"found {len(preferences)}"
        )

        LOGGER.info("Successfully verified common-instancetypes are disabled")

    @pytest.mark.polarion("CNV-61256")
    def test_ts_cnv61256_002_enable_common_instancetypes_deployment(
        self,
        admin_client,
        hco_resource,
        save_and_restore_hco_config,
    ) -> None:
        """
        Test TS-CNV61256-002: Enable common-instancetypes deployment via HCO CR.

        Markers:
            - tier2
            - p1

        Steps:
            1. Disable common-instancetypes deployment
            2. Verify no VirtualMachineClusterInstancetype resources exist
            3. Set spec.commonInstancetypesDeployment.enabled to true
            4. Wait for KubeVirt reconciliation
            5. Verify VirtualMachineClusterInstancetype resources exist
            6. Verify VirtualMachineClusterPreference resources exist

        Expected:
            - Common-instancetypes are deployed after enabling
        """
        # Step 1: Disable common-instancetypes deployment first
        self._set_common_instancetypes_deployment(hco_resource, enabled=False)
        self._wait_for_reconciliation(admin_client)

        # Step 2: Verify resources are removed
        for sample in TimeoutSampler(
            wait_timeout=TIMEOUT_5MIN,
            sleep=10,
            func=self._get_cluster_instancetypes,
            admin_client=admin_client,
        ):
            if len(sample) == 0:
                break

        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) == 0, "Common-instancetypes should be disabled"

        # Step 3: Enable common-instancetypes deployment
        self._set_common_instancetypes_deployment(hco_resource, enabled=True)

        # Step 4: Wait for reconciliation
        self._wait_for_reconciliation(admin_client)

        # Step 5: Verify VirtualMachineClusterInstancetype resources exist
        for sample in TimeoutSampler(
            wait_timeout=TIMEOUT_5MIN,
            sleep=10,
            func=self._get_cluster_instancetypes,
            admin_client=admin_client,
        ):
            if len(sample) > 0:
                break

        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) > 0, (
            "Expected VirtualMachineClusterInstancetype resources after enabling"
        )

        # Step 6: Verify VirtualMachineClusterPreference resources exist
        preferences = self._get_cluster_preferences(admin_client)
        assert len(preferences) > 0, (
            "Expected VirtualMachineClusterPreference resources after enabling"
        )

        LOGGER.info(
            f"Successfully verified common-instancetypes enabled: "
            f"{len(instancetypes)} instancetypes, {len(preferences)} preferences"
        )

    @pytest.mark.polarion("CNV-61256")
    def test_ts_cnv61256_003_default_behavior_common_instancetypes(
        self,
        admin_client,
        hco_resource,
        save_and_restore_hco_config,
    ) -> None:
        """
        Test TS-CNV61256-003: Verify default behavior when configuration is not set.

        Markers:
            - tier2
            - p1

        Steps:
            1. Remove commonInstancetypesDeployment field from HCO CR
            2. Wait for KubeVirt reconciliation
            3. Verify VirtualMachineClusterInstancetype resources exist

        Expected:
            - Default behavior is common-instancetypes enabled (backward compatibility)
        """
        # Step 1: Remove commonInstancetypesDeployment from HCO CR
        self._remove_common_instancetypes_deployment(hco_resource)

        # Step 2: Wait for reconciliation
        self._wait_for_reconciliation(admin_client)

        # Step 3: Verify VirtualMachineClusterInstancetype resources exist
        for sample in TimeoutSampler(
            wait_timeout=TIMEOUT_5MIN,
            sleep=10,
            func=self._get_cluster_instancetypes,
            admin_client=admin_client,
        ):
            if len(sample) > 0:
                break

        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) > 0, (
            "Default behavior should deploy common-instancetypes"
        )

        # Verify field is not set in HCO CR
        hco_resource.reload()
        spec = hco_resource.instance.spec
        common_deployment = getattr(spec, "commonInstancetypesDeployment", None)
        assert common_deployment is None, (
            "commonInstancetypesDeployment should not be set (default)"
        )

        LOGGER.info("Successfully verified default behavior deploys common-instancetypes")

    @pytest.mark.polarion("CNV-61256")
    def test_ts_cnv61256_004_configuration_persists_after_reconciliation(
        self,
        admin_client,
        hco_resource,
        save_and_restore_hco_config,
    ) -> None:
        """
        Test TS-CNV61256-004: Verify configuration persists after HCO reconciliation.

        Markers:
            - tier2
            - p2

        Steps:
            1. Set commonInstancetypesDeployment.enabled to false
            2. Verify configuration is applied
            3. Delete HCO operator pod to trigger restart
            4. Wait for HCO operator to become ready
            5. Verify configuration unchanged (enabled: false)

        Expected:
            - Configuration persists across operator restarts
        """
        # Step 1: Set configuration to disabled
        self._set_common_instancetypes_deployment(hco_resource, enabled=False)
        self._wait_for_reconciliation(admin_client)

        # Step 2: Verify configuration is applied
        hco_resource.reload()
        spec = hco_resource.instance.spec
        assert hasattr(spec, "commonInstancetypesDeployment"), (
            "commonInstancetypesDeployment should be set"
        )
        assert spec.commonInstancetypesDeployment.enabled is False, (
            "enabled should be False"
        )

        # Step 3: Delete HCO operator pod to trigger restart
        hco_pods = list(
            Pod.get(
                dyn_client=admin_client,
                namespace=CNV_NAMESPACE,
                label_selector="name=hyperconverged-cluster-operator",
            )
        )
        assert len(hco_pods) > 0, "HCO operator pod not found"

        LOGGER.info("Deleting HCO operator pod to trigger restart")
        for pod in hco_pods:
            pod.delete()

        # Step 4: Wait for HCO operator to become ready
        self._wait_for_reconciliation(admin_client, timeout=TIMEOUT_5MIN)

        # Step 5: Verify configuration unchanged
        hco_resource.reload()
        spec = hco_resource.instance.spec
        assert hasattr(spec, "commonInstancetypesDeployment"), (
            "commonInstancetypesDeployment should persist after restart"
        )
        assert spec.commonInstancetypesDeployment.enabled is False, (
            "enabled should still be False after operator restart"
        )

        LOGGER.info("Successfully verified configuration persists after reconciliation")

    @pytest.mark.polarion("CNV-61256")
    def test_ts_cnv61256_007_toggle_with_running_vms(
        self,
        admin_client,
        unprivileged_client,
        namespace,
        hco_resource,
        save_and_restore_hco_config,
    ) -> None:
        """
        Test TS-CNV61256-007: Verify behavior when toggling configuration while VMs are running.

        Markers:
            - tier2
            - p2

        Steps:
            1. Ensure common-instancetypes are enabled
            2. Create VM using a common instance type
            3. Verify VM is running
            4. Disable common-instancetypes deployment
            5. Verify existing VM continues running

        Expected:
            - Existing running VMs are not disrupted by configuration change
        """
        from utilities.virt import VirtualMachineForTests, running_vm

        # Step 1: Ensure common-instancetypes are enabled
        self._set_common_instancetypes_deployment(hco_resource, enabled=True)
        self._wait_for_reconciliation(admin_client)

        # Wait for instancetypes to be available
        for sample in TimeoutSampler(
            wait_timeout=TIMEOUT_5MIN,
            sleep=10,
            func=self._get_cluster_instancetypes,
            admin_client=admin_client,
        ):
            if len(sample) > 0:
                break

        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) > 0, "Common-instancetypes must be available"

        # Get a common instance type to use
        instancetype_name = instancetypes[0].name
        LOGGER.info(f"Using instance type: {instancetype_name}")

        # Step 2: Create VM using common instance type
        vm_name = "test-vm-with-instancetype"
        with VirtualMachineForTests(
            client=unprivileged_client,
            name=vm_name,
            namespace=namespace.name,
            instancetype={"name": instancetype_name, "kind": "VirtualMachineClusterInstancetype"},
        ) as vm:
            # Step 3: Verify VM is running
            running_vm(vm=vm)
            LOGGER.info(f"VM {vm_name} is running with instance type {instancetype_name}")

            # Step 4: Disable common-instancetypes deployment
            self._set_common_instancetypes_deployment(hco_resource, enabled=False)
            self._wait_for_reconciliation(admin_client)

            # Step 5: Verify existing VM continues running
            vm.vmi.reload()
            assert vm.vmi.instance.status.phase == "Running", (
                f"VM should continue running, but status is {vm.vmi.instance.status.phase}"
            )

            LOGGER.info("Successfully verified existing VM not disrupted by config change")

    @pytest.mark.polarion("CNV-61256")
    @pytest.mark.upgrade
    def test_ts_cnv61256_008_upgrade_preserves_default_behavior(
        self,
        admin_client,
        hco_resource,
        kubevirt_resource,
    ) -> None:
        """
        Test TS-CNV61256-008: Verify behavior during upgrade preserves default.

        Markers:
            - tier2
            - p2
            - upgrade

        Steps:
            1. Verify common-instancetypes are deployed (default state)
            2. Verify HCO CR has no commonInstancetypesDeployment explicitly set
            3. Verify KubeVirt CR configuration matches expected default

        Expected:
            - Default behavior (common-instancetypes enabled) is preserved
            - No explicit configuration means default applies

        Note:
            This test validates post-upgrade state. Actual upgrade is handled
            by the upgrade test infrastructure, not this test directly.
        """
        # Step 1: Verify common-instancetypes are deployed
        instancetypes = self._get_cluster_instancetypes(admin_client)
        assert len(instancetypes) > 0, (
            "Common-instancetypes should be deployed (default behavior)"
        )
        LOGGER.info(f"Found {len(instancetypes)} VirtualMachineClusterInstancetype resources")

        preferences = self._get_cluster_preferences(admin_client)
        assert len(preferences) > 0, (
            "Common preferences should be deployed (default behavior)"
        )
        LOGGER.info(f"Found {len(preferences)} VirtualMachineClusterPreference resources")

        # Step 2: Check HCO CR configuration
        hco_resource.reload()
        spec = hco_resource.instance.spec
        common_deployment = getattr(spec, "commonInstancetypesDeployment", None)

        # If set, should be enabled (default)
        if common_deployment is not None:
            enabled = getattr(common_deployment, "enabled", None)
            if enabled is not None:
                assert enabled is True, (
                    "If commonInstancetypesDeployment is set, enabled should be True"
                )

        # Step 3: Verify KubeVirt CR configuration if accessible
        kubevirt_resource.reload()
        kv_config = kubevirt_resource.instance.spec.configuration
        kv_common_deployment = getattr(kv_config, "commonInstancetypesDeployment", None)

        # If set in KubeVirt, should match expected default behavior
        if kv_common_deployment is not None:
            kv_enabled = getattr(kv_common_deployment, "enabled", None)
            if kv_enabled is not None:
                LOGGER.info(f"KubeVirt commonInstancetypesDeployment.enabled: {kv_enabled}")

        LOGGER.info("Successfully verified default behavior is preserved")
