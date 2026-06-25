# Minikube provider

> **⚠️ EXPERIMENTAL**: This provider is in an experimental stage and is not production-ready.

Provides a pre-deployed Kubernetes cluster that runs using [Minikube](https://minikube.sigs.k8s.io/).
The cluster uses docker as the driver and is managed through the `kubevirtci` profile.

## Prerequisites

- Minikube installed on your system
- Podman installed and configured
- kubectl installed

## Bringing the cluster up

```bash
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME       STATUS   ROLES           AGE   VERSION
minikube   Ready    control-plane   1m    v1.x.x
```

## Bringing the cluster down

```bash
make cluster-down
```

This destroys the whole cluster.

## Configuration

The provider uses the following environment variables:

- `MINIKUBE`: Minikube command with profile (default: `minikube --profile=kubevirtci`)
- `BASE_PATH`: Base configuration path (default: `$KUBEVIRTCI_CONFIG_PATH` or `$PWD`)
- `KUBECONFIG`: Path to kubeconfig file (default: `${CI_CONFIG}/.kubeconfig`)

## Notes

- The cluster runs with docker as the default driver (`--driver=docker`)
- The cluster uses the `kubevirtci` profile to avoid conflicts with other Minikube instances
- kubectl binary is automatically copied to the configuration directory for convenience
