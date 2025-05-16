# codeflare-operator

The CodeFlare-Operator has embedded two controllers, a [RayCluster controller](https://github.com/project-codeflare/codeflare-operator/blob/main/pkg/controllers/raycluster_controller.go) which creates resources including secrets, ingress, routes, service, serviceaccounts, clusterrolebinding resources; all needed for the RayClusters created to work as expected.

There's an [AppWrapper Controller](https://github.com/project-codeflare/appwrapper/blob/main/internal/controller/appwrapper/appwrapper_controller.go), which is a flexible and workload-agnostic mechanism to enable Kueue to manage a group of Kubernetes resources as a single logical unit and to provide an additional level of automatic fault detection and recovery.

For each controller, there are webhooks in place that can be found [here](https://github.com/project-codeflare/codeflare-operator/tree/main/pkg/controllers).

<!-- Don't delete these comments, they are used to generate Compatibility Matrix table for release automation -->
<!-- Compatibility Matrix start -->
CodeFlare Stack Compatibility Matrix

| Component                    | Version                                                                                           |
|------------------------------|---------------------------------------------------------------------------------------------------|
| CodeFlare Operator           | [v1.15.0](https://github.com/project-codeflare/codeflare-operator/releases/tag/v1.15.0)             |
| CodeFlare-SDK                | [v0.28.1](https://github.com/project-codeflare/codeflare-sdk/releases/tag/v0.28.1)                |
| AppWrapper                   | [v1.1.2](https://github.com/project-codeflare/appwrapper/releases/tag/v1.1.2)                   |
| KubeRay                      | [v1.3.2](https://github.com/ray-project/kuberay/releases/tag/v1.3.2)                           |
| Kueue                        | [v0.11.4](https://github.com/kubernetes-sigs/kueue/releases/tag/v0.11.4)                             |
<!-- Compatibility Matrix end -->

## Development

Requirements:
- GNU sed - sed is used in several Makefile command. Using macOS default sed is incompatible, so GNU sed is needed for correct execution of these commands.
  When you have a version of the GNU sed installed on a macOS you may specify the binary using
  ```bash
  # brew install gnu-sed
  make install -e SED=/usr/local/bin/gsed
  ```
- Kind - Kind is used in the kind-e2e command in the Makefile. Follow these instructions for the kind setup <a href="https://kind.sigs.k8s.io/docs/user/quick-start/" target="_blank">here</a>

### Testing

The e2e tests can be executed locally by running the following commands:

1. Use an existing cluster, or set up a test cluster, e.g.:

    ```bash
    # Create a KinD cluster
    make kind-e2e
    ```

> [!NOTE]
   Some e2e tests cover the access to services via Ingresses, as end-users would do, which requires access to the Ingress controller load balancer by its IP.
   For it to work on macOS, this requires installing [docker-mac-net-connect](https://github.com/chipmk/docker-mac-net-connect).

2. Setup the rest of the CodeFlare stack.

   ```bash
   make setup-e2e
   ```
   
> [!NOTE]
   Kueue will only activate its Ray integration if KubeRay is installed before Kueue (as done by this make target).

> [!NOTE]
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

3.  In the /etc/hosts file add the following lines:
    ```bash
    127.0.0.1 ray-dashboard-raycluster-test-ns-1.kind
    127.0.0.1 ray-dashboard-raycluster-test-ns-2.kind
    ```

4.  Build, push and deploy the codeflare-operator image:
    ```bash
    make image-push IMG=<full-registry>:<tag>
    make deploy -e IMG=<full-registry>:<tag> -e ENV="e2e"
    ```

5.  To run the tests run the command
    ```bash
    make test-e2e
    ```

   Alternatively, You can run the e2e test(s) from your IDE / debugger.

#### Testing on disconnected cluster

To properly run e2e tests on disconnected cluster user has to provide additional environment variables to properly configure testing environment:

- `CODEFLARE_TEST_PYTORCH_IMAGE` - image tag for image used to run training job
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
5. Trigger the [auto-merge-sync workflow](https://github.com/red-hat-data-services/codeflare-operator/actions/workflows/auto-merge-sync.yaml) and verify it ran successfully. This will sync changes to the [ODH CodeFlare-Operator repo](https://github.com/opendatahub-io/codeflare-operator), and the [Red Hat CodeFlare Operator repo](https://github.com/red-hat-data-services/codeflare-operator). Please review the new merge-commit and commit history, and verify changes are also in the latest `rhoai` release branch. - If the auto-merge fails, conflicts must be resolved and force pushed manually to each downstream repository and release branch.
6. In ODH/CFO verify that the [Build and Push action](https://github.com/opendatahub-io/codeflare-operator/actions/workflows/build-and-push.yaml) was triggered and ran successfully.
7. Make sure that release automation created a PR updating CodeFlare SDK version in [ODH Notebooks repository](https://github.com/opendatahub-io/notebooks). Make sure the PR gets merged.
8. Run [ODH CodeFlare Operator release workflow](https://github.com/opendatahub-io/codeflare-operator/actions/workflows/odh-release.yml) to produce ODH CodeFlare Operator release.
9. Ensure that the version details in the `config/component_metadata.yaml` file are updated to reflect the latest upstream CodeFlare Operator release version

### Releases involving part of the stack

There may be instances in which a new CodeFlare stack release requires releases of only a subset of the stack components. Examples could be hotfixes for a specific component. In these instances:

1. Build updated components as needed:
    - Build and release [CodeFlare-SDK](https://github.com/project-codeflare/codeflare-sdk)

2. Invoke [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, this action will create a repository tag, build and push operator image.
3. Check result of [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, it should pass.
4. Verify that compatibility matrix in [README](https://github.com/project-codeflare/codeflare-operator/blob/main/README.md) was properly updated.
5. Follow the steps 3-6 from the previous section.
