# codeflare-operator

Operator for installation and lifecycle management of CodeFlare distributed workload stack, starting with MCAD and InstaScale

<!-- Don't delete these comments, they are used to generate Compatibility Matrix table for release automation -->
<!-- Compatibility Matrix start -->
CodeFlare Stack Compatibility Matrix

| Component                    | Version                                                                                           |
|------------------------------|---------------------------------------------------------------------------------------------------|
| CodeFlare Operator           | [v1.3.0](https://github.com/project-codeflare/codeflare-operator/releases/tag/v1.3.0)             |
| Multi-Cluster App Dispatcher | [v1.40.0](https://github.com/project-codeflare/multi-cluster-app-dispatcher/releases/tag/v1.40.0) |
| CodeFlare-SDK                | [v0.15.0](https://github.com/project-codeflare/codeflare-sdk/releases/tag/v0.15.0)                  |
| InstaScale                   | [v0.4.0](https://github.com/project-codeflare/instascale/releases/tag/v0.4.0)                     |
| KubeRay                      | [v1.1.0](https://github.com/opendatahub-io/kuberay/releases/tag/v1.1.0)                              |
<!-- Compatibility Matrix end -->

## Development

Requirements:
- GNU sed - sed is used in several Makefile command. Using macOS default sed is incompatible, so GNU sed is needed for correct execution of these commands.
  When you have a version of the GNU sed installed on a macOS you may specify the binary using
  ```bash
  # brew install gnu-sed
  NAMESPACE=default make run
  ```

### Testing

The e2e tests can be executed locally by running the following commands:

1. Use an existing cluster, or set up a test cluster, e.g.:

    ```bash
    # Create a KinD cluster
    make kind-e2e
    ```

   [!NOTE]
   Some e2e tests cover the access to services via Ingresses, as end-users would do, which requires access to the Ingress controller load balancer by its IP.
   For it to work on macOS, this requires installing [docker-mac-net-connect](https://github.com/chipmk/docker-mac-net-connect).

2. Start the operator locally:

    ```bash
    NAMESPACE=default make run
    ```

   Alternatively, You can run the operator from your IDE / debugger.

3. Set up the test CodeFlare stack:

   ```bash
   make setup-e2e
   ```

   [!NOTE]
   In OpenShift the KubeRay operator pod gets random user assigned. This user is then used to run Ray cluster.
   However the random user assigned by OpenShift doesn't have rights to store dataset downloaded as part of test execution, causing tests to fail.
   To prevent this failure on OpenShift user should enforce user 1000 for KubeRay and Ray cluster by creating this SCC in KubeRay operator namespace (replace the namespace placeholder):

    ```yaml
    kind: SecurityContextConstraints
    apiVersion: security.openshift.io/v1
    metadata:
      name: run-as-ray-user
    seLinuxContext:
      type: MustRunAs
    runAsUser:
      type: MustRunAs
      uid: 1000
    users:
      - 'system:serviceaccount:$(namespace):kuberay-operator'
    ```

4.  In a separate terminal, set your output directory for test files, and run the e2e suite:
    ```bash
    export CODEFLARE_TEST_OUTPUT_DIR=<your_output_directory>
    ```

    ```bash
    make test-e2e
    ```

   Alternatively, You can run the e2e test(s) from your IDE / debugger.

#### Testing on disconnected cluster

To properly run e2e tests on disconnected cluster user has to provide additional environment variables to properly configure testing environment:

- `CODEFLARE_TEST_PYTORCH_IMAGE` - image tag for image used to run training job using MCAD
- `CODEFLARE_TEST_RAY_IMAGE` - image tag for Ray cluster image
- `MNIST_DATASET_URL` - URL where MNIST dataset is available
- `PIP_INDEX_URL` - URL where PyPI server with needed dependencies is running
- `PIP_TRUSTED_HOST` - PyPI server hostname

For ODH tests additional environment variables are needed:

- `NOTEBOOK_IMAGE_STREAM_NAME` - name of the ODH Notebook ImageStream to be used
- `ODH_NAMESPACE` - namespace where ODH is installed

## Release

1. Invoke [project-codeflare-release.yaml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/project-codeflare-release.yml)
2. Once all jobs within the action are completed, verify that compatibility matrix in [README](https://github.com/project-codeflare/codeflare-operator/blob/main/README.md) was properly updated.
3. Verify that opened pull request to [OpenShift community operators repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod) has proper content.
4. Once PR is merged, announce the new release in slack and mail lists, if any.
5. Release automation should open a PR with changes in [ODH CodeFlare operator repo](https://github.com/opendatahub-io/codeflare-operator). Review the changes proposed by automation. If all the changes are correct then manually cherrypick all `CARRY` and `PATCH` commits from the current main branch, push the result to a dedicated branch and ask on Slack channel for review of the result branch content. Once agreed then push the changes directly to the `main` branch (branch protection has to be temporarily disabled).
6. Build ODH/CFO image by triggering [Build and Push action](https://github.com/opendatahub-io/codeflare-operator/actions/workflows/build-and-push.yaml)
7. Create a release branch on [Red Hat CodeFlare operator repo](https://github.com/red-hat-data-services/codeflare-operator) for the next release if it doesn't exist yet.
8. Create a dedicated branch containing changes from [ODH CodeFlare operator repo](https://github.com/opendatahub-io/codeflare-operator). Cherrypick all relevant changes available in [Red Hat CodeFlare operator repo](https://github.com/red-hat-data-services/codeflare-operator) latest release branch which should be available also in the next release. Ask on Slack channel for review of the result branch content. Once agreed then push the changes directly to the release branch.
9. Make sure that release automation created a PR updating CodeFlare SDK version in [ODH Notebooks repository](https://github.com/opendatahub-io/notebooks). Make sure the PR gets merged.

### Releases involving part of the stack

There may be instances in which a new CodeFlare stack release requires releases of only a subset of the stack components. Examples could be hotfixes for a specific component. In these instances:

1. Build updated components as needed:
    - Build and release [MCAD](https://github.com/project-codeflare/multi-cluster-app-dispatcher)
    - Build and release [InstaScale](https://github.com/project-codeflare/instascale)
    - Build and release [CodeFlare-SDK](https://github.com/project-codeflare/codeflare-sdk)

2. Invoke [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, this action will create a repository tag, build and push operator image.
3. Check result of [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, it should pass.
4. Verify that compatibility matrix in [README](https://github.com/project-codeflare/codeflare-operator/blob/main/README.md) was properly updated.
5. Follow the steps 3-6 from the previous section.
