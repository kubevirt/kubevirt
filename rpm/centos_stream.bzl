"""Macros for CentOS Stream version selection in RPM targets."""

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
