# codeflare-operator

Operator for installation and lifecycle management of CodeFlare distributed workload stack, starting with MCAD and InstaScale

<!-- Don't delete these comments, they are used to generate Compatibility Matrix table for release automation -->
<!-- Compatibility Matrix start -->
CodeFlare Stack Compatibility Matrix

| Component                    | Version |
|------------------------------|---------|
| CodeFlare Operator           | v0.0.5  |
| Multi-Cluster App Dispatcher | v1.32.0 |
| CodeFlare-SDK                | v0.5.0  |
| InstaScale                   | v0.0.5  |
| KubeRay                      | v0.5.0  |
<!-- Compatibility Matrix end -->

## Development

### Testing

The e2e tests can be executed locally by running the following commands:

1. Use an existing cluster, or set up a test cluster, e.g.:

    ```bash
    # Create a KinD cluster
    $ kind create cluster --image kindest/node:v1.25.8
    # Install the CRDs
    $ make install
    ```

2. Start the operator locally:

    ```bash
    $ make run
    ```

   Alternatively, You can run the operator from your IDE / debugger.

3. Set up the test CodeFlare stack:

   ```bash
   $ make setup-e2e
   ```

4. In a separate terminal, run the e2e suite:

    ```bash
    $ make test-e2e
    ```

   Alternatively, You can run the e2e test(s) from your IDE / debugger.

## Release

Prerequisite:
- Build and release [MCAD](https://github.com/project-codeflare/multi-cluster-app-dispatcher)
- Build and release [InstaScale](https://github.com/project-codeflare/instascale)
- Build and release [CodeFlare-SDK](https://github.com/project-codeflare/codeflare-sdk)

Release steps:
1. Invoke [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, this action will create a repository tag, build and push operator image.

2. Check result of [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, it should pass.

3. Verify that compatibility matrix in [README](https://github.com/project-codeflare/codeflare-operator/blob/main/README.md) was properly updated.

4. Verify that opened pull request to [OpenShift community operators repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod) has proper content.

5. Once PR is merged, update component stable tags to point at the latest image release.

6. Announce the new release in slack and mail lists, if any.

7. Update the Distributed Workloads component in ODH (also copy/update the compatibility matrix).
