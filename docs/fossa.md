# FOSSA license check

## Background

[FOSSA] license scanning was introduced when the KubeVirt project entered the CNCF sandbox. It scans all relevant project dependencies for licensing issues. CNCF offers FOSSA usage [free of charge for all CNCF projects](https://www.cncf.io/services-for-projects/#legal-services).

## Scans performed

Currently, FOSSA license scanning happens in two situations: on every push to a PR branch and on every push to the `kubevirt/kubevirt` default branch.

Job definitions are located in `kubevirt/project-infra`:
* [GitHub search for FOSSA](https://github.com/kubevirt/project-infra/search?q=fossa+path%3Agithub%2Fci%2Fprow-deploy%2Ffiles%2Fjobs%2Fkubevirt%2Fkubevirt)

To view the scan results, you need to look at the job log file, where near the bottom of the file you'll find a direct link to the check results.

## Running the FOSSA license scanning locally

To run the license scan you need a FOSSA API key stored in a file. Please have a look at the [FOSSA CLI quick start guide](https://github.com/fossas/fossa-cli#quick-start).

This token you put into a file and start the analysis like this:

```bash
export FOSSA_TOKEN_FILE=<path to your token file>
./hack/fossa.sh
```

## Configuration

[Configuration of FOSSA license scan](https://app.fossa.com/projects?teamId=332) for KubeVirt project is allowed for KubeVirt FOSSA team members. Please ask one of the maintainers for access in case you need it.

[FOSSA]: https://fossa.com
