# codeflare-operator

Operator for installation and lifecycle management of CodeFlare distributed workload stack, starting with MCAD and InstaScale

<!-- Don't delete these comments, they are used to generate Compatibility Matrix table for release automation -->
<!-- Compatibility Matrix start -->
CodeFlare Stack Compatibility Matrix

| Component                    | Version                                                                                           |
|------------------------------|---------------------------------------------------------------------------------------------------|
| CodeFlare Operator           | [v0.2.3](https://github.com/project-codeflare/codeflare-operator/releases/tag/v0.2.3)             |
| Multi-Cluster App Dispatcher | [v1.34.1](https://github.com/project-codeflare/multi-cluster-app-dispatcher/releases/tag/v1.34.1) |
| CodeFlare-SDK                | [v0.7.1](https://github.com/project-codeflare/codeflare-sdk/releases/tag/v0.7.1)                  |
| InstaScale                   | [v0.0.8](https://github.com/project-codeflare/instascale/releases/tag/v0.0.8)                     |
| KubeRay                      | [v0.5.0](https://github.com/ray-project/kuberay/releases/tag/v0.5.0)                              |
<!-- Compatibility Matrix end -->

## Development

Requirements:
- GNU sed - sed is used in several Makefile command. Using macOS default sed is incompatible, so GNU sed is needed for correct execution of these commands.

### Testing

The e2e tests can be executed locally by running the following commands:

1. Use an existing cluster, or set up a test cluster, e.g.:

    ```bash
    # Create a KinD cluster
    make kind-e2e
    # Install the CRDs
    make install
    ```

   [!NOTE]
   Some e2e tests cover the access to services via Ingresses, as end-users would do, which requires access to the Ingress controller load balancer by its IP.
   For it to work on macOS, this requires installing [docker-mac-net-connect](https://github.com/chipmk/docker-mac-net-connect).

2. Start the operator locally:

    ```bash
    make run
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

4. In a separate terminal, run the e2e suite:

    ```bash
    make test-e2e
    ```

   Alternatively, You can run the e2e test(s) from your IDE / debugger.

## Release

1. Invoke [project-codeflare-release.yaml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/project-codeflare-release.yml)
2. Once all jobs within the action are completed, verify that compatibility matrix in [README](https://github.com/project-codeflare/codeflare-operator/blob/main/README.md) was properly updated.
3. Verify that opened pull request to [OpenShift community operators repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod) has proper content.
4. Once PR is merged, announce the new release in slack and mail lists, if any.
5. Update the Distributed Workloads component in ODH (also copy/update the compatibility matrix). This may require yaml and test updates depending on the release. Make sure to create a tag + release in the Distributed Workloads repository that matches the project-codeflare release version.
6. Update the readme/markdown/yaml in odh-manifests as required.

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
