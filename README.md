# codeflare-operator
Operator for installation and lifecycle management of CodeFlare distributed workload stack, starting with MCAD and InstaScale

CodeFlare Stack Compatibility Matrix

| Component                    | Version |
|------------------------------|---------|
| CodeFlare Operator           | v0.0.4  |
| Multi-Cluster App Dispatcher | v1.31.0 |
| CodeFlare-SDK                | v0.4.4  |
| InstaScale                   | v0.0.4  |
| KubeRay                      | v0.5.0  |

## Release process

Prerequisite:
- Build and release [MCAD](https://github.com/project-codeflare/multi-cluster-app-dispatcher)
- Build and release [InstaScale](https://github.com/project-codeflare/instascale)
- Build and release [CodeFlare-SDK](https://github.com/project-codeflare/codeflare-sdk)

Release steps:
1. Invoke [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, this action will create a repository tag, build and push operator image.

2. Check result of [tag-and-build.yml](https://github.com/project-codeflare/codeflare-operator/actions/workflows/tag-and-build.yml) GitHub action, it should pass.

3. Update CodeFlare Stack Compatibility Matrix in operator README.

4. Update InstaScale and MCAD versions:
- in [Makefile](https://github.com/project-codeflare/codeflare-operator/blob/02e14b535b4f7172b0b809bcae4025008a1a968b/Makefile#L12-L16).
- in [mcad/deployment.yaml.tmpl](https://github.com/project-codeflare/codeflare-operator/blob/main/config/internal/mcad/deployment.yaml.tmpl#L28)
- in [controllers/defaults.go](https://github.com/project-codeflare/codeflare-operator/blob/main/controllers/defaults.go) by running `make defaults`
- in [controllers/testdata/instascale_test_results/case_1/deployment.yaml](https://github.com/project-codeflare/codeflare-operator/blob/main/controllers/testdata/instascale_test_results/case_1/deployment.yaml) and [controllers/testdata/instascale_test_results/case_2/deployment.yaml](https://github.com/project-codeflare/codeflare-operator/blob/main/controllers/testdata/instascale_test_results/case_2/deployment.yaml)

5. Create a release in CodeFlare operator repository, release notes should include new support matrix.

6. Open a pull request to OpenShift community operators repository with latest bundle using make command, check that the created PR has proper content.
```
make openshift-community-operator-release
```

7. Once merged, update component stable tags to point at the latest image release.

8. Announce the new release in slack and mail lists, if any.

9. Update the Distributed Workloads component in ODH (also copy/update the compatibility matrix).
