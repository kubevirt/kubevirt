"""
Tier 2 End-to-End tests for CommonInstancetypesDeployment feature.

STP Reference: tests/CNV-61256/CNV-61256_test_plan.md
STD Reference: tests/CNV-61256/CNV-61256_test_description.yaml
Jira: https://issues.redhat.com/browse/CNV-61256

Feature: Disable common-instancetypes deployment from HCO

This test module validates the end-to-end behavior of the
CommonInstancetypesDeployment configuration in the HyperConverged CR,
including resource deployment/removal and lifecycle scenarios.

Phase 1: Design stubs with PSE docstrings for review
"""

# Exclude from test collection during Phase 1
__test__ = False

import pytest
from typing import Generator

from ocp_resources.hyperconverged import HyperConverged
from ocp_resources.virtual_machine_cluster_instancetype import VirtualMachineClusterInstancetype
from ocp_resources.virtual_machine_cluster_preference import VirtualMachineClusterPreference
from ocp_resources.virtual_machine import VirtualMachine


class TestCommonInstancetypesDeployment:
    """
    Test suite for CommonInstancetypesDeployment HCO configuration.

    Shared Preconditions:
        - OpenShift cluster with CNV 4.19+ installed
        - HyperConverged CR is healthy and accessible
        - Cluster admin privileges available
    """

    @pytest.fixture(autouse=True)
    def setup_teardown(self) -> Generator:
        """Setup and teardown for each test."""
        # TODO: Setup - save current HCO configuration
        yield
        # TODO: Teardown - restore HCO configuration to default

    @pytest.mark.tier2
    def test_disable_common_instancetypes_deployment(self) -> None:
        """
        Test ID: TS-CNV61256-001
        Tier: Tier 2
        Priority: P1
        Requirement: REQ-001

        Preconditions:
            - Common-instancetypes are deployed (default state)
            - VirtualMachineClusterInstancetype resources exist
            - VirtualMachineClusterPreference resources exist

        Steps:
            1. Verify common-instancetypes are deployed by default
            2. Set spec.commonInstancetypesDeployment.enabled to false in HCO CR
            3. Wait for KubeVirt reconciliation (up to 2 minutes)
            4. Verify no VirtualMachineClusterInstancetype resources exist
            5. Verify no VirtualMachineClusterPreference resources exist

        Expected:
            - All common-instancetypes resources are removed
            - No VirtualMachineClusterInstancetype resources in cluster
            - No VirtualMachineClusterPreference resources in cluster
        """
        pass

    @pytest.mark.tier2
    def test_enable_common_instancetypes_deployment(self) -> None:
        """
        Test ID: TS-CNV61256-002
        Tier: Tier 2
        Priority: P1
        Requirement: REQ-002

        Preconditions:
            - Common-instancetypes are disabled
            - No VirtualMachineClusterInstancetype resources exist
            - No VirtualMachineClusterPreference resources exist

        Steps:
            1. Verify common-instancetypes are not deployed
            2. Set spec.commonInstancetypesDeployment.enabled to true in HCO CR
            3. Wait for KubeVirt reconciliation (up to 2 minutes)
            4. Verify VirtualMachineClusterInstancetype resources are deployed
            5. Verify VirtualMachineClusterPreference resources are deployed

        Expected:
            - Common-instancetypes resources are deployed
            - Multiple VirtualMachineClusterInstancetype resources exist
            - Multiple VirtualMachineClusterPreference resources exist
        """
        pass

    @pytest.mark.tier2
    def test_default_behavior_common_instancetypes(self) -> None:
        """
        Test ID: TS-CNV61256-003
        Tier: Tier 2
        Priority: P1
        Requirement: REQ-003

        Preconditions:
            - Fresh HCO installation or HCO CR without commonInstancetypesDeployment field
            - spec.commonInstancetypesDeployment is nil/unset

        Steps:
            1. Remove commonInstancetypesDeployment field from HCO CR (if present)
            2. Wait for KubeVirt reconciliation
            3. Verify VirtualMachineClusterInstancetype resources exist
            4. Verify VirtualMachineClusterPreference resources exist

        Expected:
            - Default behavior is common-instancetypes enabled
            - Backward compatibility maintained
            - Existing installations continue to work without configuration
        """
        pass

    @pytest.mark.tier2
    def test_configuration_persists_after_reconciliation(self) -> None:
        """
        Test ID: TS-CNV61256-004
        Tier: Tier 2
        Priority: P2
        Requirement: REQ-005

        Preconditions:
            - HCO operator is running
            - HCO operator pod is accessible for restart

        Steps:
            1. Set commonInstancetypesDeployment.enabled to false
            2. Verify configuration is applied
            3. Delete HCO operator pod to trigger restart
            4. Wait for HCO operator to become ready
            5. Verify configuration unchanged (enabled: false)

        Expected:
            - Configuration persists across operator restarts
            - No manual intervention required
            - User configuration not overwritten by reconciliation
        """
        pass

    @pytest.mark.tier2
    def test_toggle_with_running_vms(self) -> None:
        """
        Test ID: TS-CNV61256-007
        Tier: Tier 2
        Priority: P2
        Requirement: REQ-008

        Preconditions:
            - Common-instancetypes are deployed
            - VM using common instance type is running

        Steps:
            1. Create VM using a common instance type
            2. Verify VM is running
            3. Disable common-instancetypes deployment
            4. Verify existing VM continues running
            5. Attempt to create new VM with common instance type
            6. Verify new VM creation behavior (may fail or warn)

        Expected:
            - Existing running VMs are not disrupted
            - New VMs referencing removed instance types fail gracefully
            - Clear error message when instance type not found
        """
        pass

    @pytest.mark.tier2
    @pytest.mark.upgrade
    def test_upgrade_preserves_default_behavior(self) -> None:
        """
        Test ID: TS-CNV61256-008
        Tier: Tier 2
        Priority: P2
        Requirement: REQ-007

        Preconditions:
            - CNV version without CommonInstancetypesDeployment feature
            - Common-instancetypes deployed in pre-upgrade state

        Steps:
            1. Verify common-instancetypes deployed before upgrade
            2. Upgrade CNV to version with feature
            3. Wait for upgrade to complete
            4. Verify common-instancetypes still deployed
            5. Verify HCO CR has no commonInstancetypesDeployment set

        Expected:
            - Upgrade preserves default behavior (enabled)
            - No disruption to existing common-instancetypes
            - Field is nil after upgrade (not explicitly set)
        """
        pass
