# VM Log Checker

Analyze KubeVirt virt-launcher logs with color-coded error detection.

## Building

```bash
make vmlog-checker
```

## Usage

```bash
# Basic usage
./tools/vmlog-checker/vmlog-checker --log virt-launcher.log

# With options
./tools/vmlog-checker/vmlog-checker --log virt-launcher.log --no-color
./tools/vmlog-checker/vmlog-checker --log virt-launcher.log --all-levels
./tools/vmlog-checker/vmlog-checker --log virt-launcher.log --errors-only
./tools/vmlog-checker/vmlog-checker --log virt-launcher.log --errors-only --no-color
```

## Options

- `--log`: Log file to analyze (required)
- `--no-color`: Disable colored output
- `--all-levels`: Check all log levels (default: only ERROR level, matching test reporter)
- `--errors-only`: Print only unexpected errors (lines that need attention)

## Allowlist

Edit `tests/vmlogchecker/vmlog_checker.go` to add patterns:

```go
var VirtLauncherErrorAllowlist = []*regexp.Regexp{
    regexp.MustCompile(`"level":"error","msg":"pattern here"`),
}
```
