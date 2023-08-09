# codeflare-operator

Operator for installation and lifecycle management of CodeFlare distributed workload stack, starting with MCAD and InstaScale

<!-- Don't delete these comments, they are used to generate Compatibility Matrix table for release automation -->
<!-- Compatibility Matrix start -->
CodeFlare Stack Compatibility Matrix

| Component                    | Version |
|------------------------------|---------|
| CodeFlare Operator           | v0.1.0  |
| Multi-Cluster App Dispatcher | v1.33.0 |
| CodeFlare-SDK                | v0.6.1  |
| InstaScale                   | v0.0.6  |
| KubeRay                      | v0.5.0  |
<!-- Compatibility Matrix end -->

## Development

### Testing

The e2e tests can be executed locally by running the following commands:

1. Use an existing cluster, or set up a test cluster, e.g.:

    ```bash
    # Create a KinD cluster
    $ make kind-e2e
    # Install the CRDs
    $ make install
    ```

   [!NOTE]
   Some e2e tests cover the access to services via Ingresses, as end-users would do, which requires access to the Ingress controller load balancer by its IP.
   For it to work on macOS, this requires installing [docker-mac-net-connect](https://github.com/chipmk/docker-mac-net-connect).

2. Start the operator locally:

    ```bash
    $ make run
    ```

   Alternatively, You can run the operator from your IDE / debugger.

3. Set up the test CodeFlare stack:

   ```bash
   $ make setup-e2e
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
