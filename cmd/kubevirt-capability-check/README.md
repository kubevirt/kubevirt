# KubeVirt Capability Check Tool

A command-line tool for listing supported and unsupported capabilities on specific KubeVirt platforms (hypervisor + architecture combinations).

## Usage

```bash
kubevirt-capability-check --hypervisor <hypervisor> --arch <arch> [options]
```

### Required Parameters

- `--hypervisor`: Target hypervisor (e.g., 'kvm', 'mshv')
- `--arch`: Target architecture (e.g., 'amd64', 'arm64', 's390x')

### Optional Parameters

- `--support-level`: Support level to filter by (default: "Unsupported")
  - `Unsupported`: Explicitly blocked capabilities on this platform
  - `Experimental`: Capabilities that require feature gates
  - `Deprecated`: Supported but discouraged capabilities
  - `Unregistered`: Capabilities without explicit platform registration
- `--output`: Output format (default: "keys")
  - `keys`: Capability keys only
  - `detailed`: Keys with messages and feature gate information
  - `json`: JSON formatted output
- `--list-all`: List all capabilities regardless of support level

### Examples

#### List unsupported capabilities for KVM on amd64
```bash
kubevirt-capability-check --hypervisor kvm --arch amd64
```

#### List experimental capabilities with detailed information
```bash
kubevirt-capability-check --hypervisor kvm --arch amd64 --support-level experimental --output detailed
```

#### Get all capabilities for MSHV in JSON format
```bash
kubevirt-capability-check --hypervisor mshv --arch amd64 --list-all --output json
```

#### List capabilities unsupported on S390X KVM
```bash
kubevirt-capability-check --hypervisor kvm --arch s390x --support-level unsupported --output detailed
```

## Output Formats

### Keys Format (Default)
Simple list of capability keys:
```
launchSecurity.tdx
devices.usb.redirection
```

### Detailed Format
Comprehensive information including messages and feature gates:
```
Capabilities for platform kvm/s390x:

Key: launchSecurity.tdx
  Level: Unsupported
  Message: Intel TDX is not supported on S390X architecture.

Key: devices.usb.redirection
  Level: Unsupported
  Message: USB redirection is not supported on S390X architecture.
```

### JSON Format
Machine-readable JSON output:
```json
{
  "platform": "kvm/s390x",
  "hypervisor": "kvm",
  "architecture": "s390x",
  "capabilities": [
    {
      "key": "launchSecurity.tdx",
      "level": 1,
      "message": "Intel TDX is not supported on S390X architecture."
    }
  ]
}
```

## Use Cases

1. **Platform Validation**: Check if specific capabilities are supported before deploying workloads
2. **CI/CD Integration**: Automatically validate platform capabilities in deployment pipelines
3. **Documentation**: Generate platform-specific capability matrices
4. **Debugging**: Identify why certain features are not working on specific platforms

## Integration with KubeVirt

This tool leverages the KubeVirt capabilities framework located in `pkg/capabilities/`. The capabilities system provides:

- **Capability Definitions**: What each capability represents and how to detect if it's required by a VMI
- **Platform Support Mapping**: Which platforms support, block, or experimentally support each capability
- **Feature Gate Integration**: Which capabilities require specific feature gates to be enabled

## Building

```bash
# Build the tool
bazel build //cmd/kubevirt-capability-check:kubevirt-capability-check

# Run the binary
bazel run //cmd/kubevirt-capability-check:kubevirt-capability-check -- --hypervisor kvm --arch amd64
```