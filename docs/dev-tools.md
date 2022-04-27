# Developer Tools in the HCO

### tools/token.sh
Gather an OAUTH token for interacting with Quay.

```
make quay-token
```

### tools/quay-registry.sh
Setup an OperatorSource for Marketplace to pull content from.  This is useful
when a developer wants to test content from something other than the default
Quay Application Registries.

```
make bundleRegistry
```

### ./tools/operator-courier/push.sh
Validate the CSV in `deploy/olm-catalog/kubevirt-hyperconverged`, build, and push
a bundle to https://quay.io/application/kubevirt-hyperconverged/kubevirt-hyperconverged.

```
make bundle-push
```
