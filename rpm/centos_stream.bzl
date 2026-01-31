"""Macros for CentOS Stream version selection in RPM targets."""

load("@bazel_skylib//lib:selects.bzl", "selects")

def centos_stream_config_settings():
    """Define config_settings for CentOS Stream version selection."""
    native.config_setting(
        name = "is_cs9",
        define_values = {"centos_stream_version": "9"},
    )
    native.config_setting(
        name = "is_cs10",
        define_values = {"centos_stream_version": "10"},
    )

    # Platform-specific config settings for use in compound conditions
    native.config_setting(
        name = "linux_x86_64",
        constraint_values = ["@platforms//cpu:x86_64", "@platforms//os:linux"],
    )
    native.config_setting(
        name = "linux_arm64",
        constraint_values = ["@platforms//cpu:aarch64", "@platforms//os:linux"],
    )
    native.config_setting(
        name = "linux_s390x",
        constraint_values = ["@platforms//cpu:s390x", "@platforms//os:linux"],
    )

    # Compound config_setting_groups for platform + centos_stream combinations
    # x86_64 + CS versions
    selects.config_setting_group(
        name = "x86_64_cs9",
        match_all = [":linux_x86_64", ":is_cs9"],
    )
    selects.config_setting_group(
        name = "x86_64_cs10",
        match_all = [":linux_x86_64", ":is_cs10"],
    )

    # aarch64 + CS versions
    selects.config_setting_group(
        name = "aarch64_cs9",
        match_all = [":linux_arm64", ":is_cs9"],
    )
    selects.config_setting_group(
        name = "aarch64_cs10",
        match_all = [":linux_arm64", ":is_cs10"],
    )

    # s390x + CS versions
    selects.config_setting_group(
        name = "s390x_cs9",
        match_all = [":linux_s390x", ":is_cs9"],
    )
    selects.config_setting_group(
        name = "s390x_cs10",
        match_all = [":linux_s390x", ":is_cs10"],
    )

def centos_stream_alias(name, cs9_target, cs10_target, visibility = None):
    """Create an alias that selects between CS9 and CS10 targets.

    Args:
        name: The alias target name (unversioned, e.g., "launcherbase_x86_64")
        cs9_target: The CS9 target (e.g., ":launcherbase_x86_64_cs9")
        cs10_target: The CS10 target (e.g., ":launcherbase_x86_64_cs10")
        visibility: Target visibility
    """
    native.alias(
        name = name,
        actual = select({
            ":is_cs10": cs10_target,
            "//conditions:default": cs9_target,  # Default to CS9
        }),
        visibility = visibility,
    )
