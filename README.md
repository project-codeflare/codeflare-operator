# codeflare-operator
Operator for installation and lifecycle management of CodeFlare distributed workload stack, starting with MCAD and InstaScale

<!-- Don't delete these comments, they are used to generate Compatibility Matrix table for release automation -->
<!-- Compatibility Matrix start -->
CodeFlare Stack Compatibility Matrix

| Component                    | Version |
|------------------------------|---------|
| CodeFlare Operator           | v0.0.4  |
| Multi-Cluster App Dispatcher | v1.31.0 |
| CodeFlare-SDK                | v0.4.4  |
| InstaScale                   | v0.0.4  |
| KubeRay                      | v0.5.0  |
<!-- Compatibility Matrix end -->

## Release process

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
