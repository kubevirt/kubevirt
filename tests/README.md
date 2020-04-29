# CI Tests
## kubevirtci tests
### Tests
The CI tests are defined in the [automation/test.sh](../automation/test.sh) file.
### CI Jobs
The CI job files are in the [project-infra](https://github.com/kubevirt/project-infra/tree/master/github/ci/prow/files/jobs/hyperconverged-cluster-operator) repository in github.
### Test tools
The test script is using a docker container in order to run the tests. The Dockerfile is here: [tests/build/Dockerfile](build/Dockerfile). 
The image uses an entry-point file from here: [tests/build/entrypoint.sh](build/entrypoint.sh).
The image contains golang and test utils, and in most cases, downloads the latest version of the tools.

To build the test-utils image, there is a post submit ci job that run after merge of a PR, and only if the content of the `tests/build` directory was modified in this PR.
Here is the [CI job](https://github.com/kubevirt/project-infra/blob/master/github/ci/prow/files/jobs/hyperconverged-cluster-operator/hyperconverged-cluster-operator-postsubmits.yaml) that performs the image build.
The job runs the [hack/build-in-docker.sh](../hack/build-in-docker.sh) script, that build the image and pushes it to docker.io, with a new tag.

The assumption is that there won't be frequent changes in the test-utils image. 

***Notice***: The CI test uses a hard coded tag for the test utils container. The meaning is that building a new image is not enough to use it. This is done in order to make sure the new image is valid before using it. 

To update the test utils image, use the following procedure:
1. Make the required changes in the [Dockerfile](build/Dockerfile) or the [entrypoint](build/entrypoint.sh). 
   
   **Note**: as most of the tools are just the latest version from the time of the image creation, it should be enough to `touch` the Dockerfile in order to upgrade most of the tools in the image. 
2. Commit the changes, push them, apply a github pull request and merge it.
3. After the PR is merged, look for the image build job here https://prow.apps.ovirt.org/. The job name is `publish-hco-test-utils-image`. 
Make sure the job was successfully done, get into its log and look for the line starting with `Successfully created and pushed new test utils image: `. The image tag is after the colon (`:`) in the image name, and will be in a format similar to `v20200427-db8c50b`.
4. In the [in-docker.sh](../hack/in-docker.sh) file, replace the default value of the `TAG` environment variable to the new tag from #3; It should look like this: `TAG=${TAG:-v20200427-db8c50b}`.
5. Commit and push the changes and apply a new PR. Make sure the kubevirtci jobs are still working, before merging the PR with the new image tag.