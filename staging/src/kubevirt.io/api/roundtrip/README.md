### Description:

1. This creates JSON and YAML files for all the API exposed by kubevirt in group-version "kubevirt.io/v1", 
   versioned by the release. The current version is in `HEAD` directory, previous versions are in `release-0.yy` release
   directory. APIs includes, more APIs can be added in the future:
    ```
    VirtualMachineInstance
    VirtualMachineInstanceList
    VirtualMachine
    VirtualMachineList
    KubeVirt
    KubeVirtList
    ```
2. Upon upgrade to API, the json and YAML files will be updated.
3. When KubeVirt cuts a new release of the project, the current version files will be copied to the release version and
   future development branch will add a unit test for past two releases:

Using this:
1. API reviewers can say if changes in current version will break older clients upon upgrade
2. During upgrades, vendors can check the API changes going into the upgrade using simple differ and get a better
   synopsis of what is failing during upgrade.