# vms-generator
This is the tool which generated the YAML files under: `cluster/examples`. Which, in turn, used for manual and integration testing.
To add new YAML there or to modify any of the existing ones, please edit: `tools/vms-generator/vms-generator.go` and run `make generate`, as all changes will be reverted, and all other content will be erased from `cluster/example`.
