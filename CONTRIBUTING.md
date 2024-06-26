# Contributing to the CodeFlare Operator

Here are a few things to go over before getting started with CodeFlare Operator development:

## Environment setup

The following should be installed in your working environment:
 - Go 1.22.x
   - [Download release](https://go.dev/dl/)
   - [Install Instructions](https://go.dev/doc/install)
 - [Operator SDK](https://sdk.operatorframework.io/docs/installation/)
 - GCC

## Basic Overview
The main entrypoint for the operator is `main.go`

## Building and Deployment
If changes are made in the `api` dir, run: `make manifests`
 - This will generate new CRDs and associated files

If changes are made to any Go code (like in the `controllers` dir for example), run: `make`
 - This will check and build/compile the modified code

For building and pushing a new version of the operator image:
 - `make image-build -e IMAGE_TAG_BASE=<image-repo/image-name> VERSION=<semver>`
 - `make image-push -e IMAGE_TAG_BASE=<image-repo/image-name> VERSION=<semver>`

For deploying onto a cluster:
 - First, either set `KUBECONFIG` or ensure you are logged into a cluster in your environment
 - `make install`
 - `make deploy -e IMG=<image-repo/image-name>`

For building and pushing a new version of the bundled operator image:
 - `make bundle-build -e IMAGE_TAG_BASE=<image-repo/image-name> VERSION=<new semver> PREVIOUS_VERSION=<semver to replace>`
 - `make bundle-push -e IMAGE_TAG_BASE=<image-repo/image-name> VERSION=<new semver> PREVIOUS_VERSION=<semver to replace>`

To create a new openshift-community-operator-release:
 - `make openshift-community-operator-release -e IMAGE_TAG_BASE=<image-repo/image-name> VERSION=<new semver> PREVIOUS_VERSION=<semver to replace> GH_TOKEN=<GitHub token for pushing bundle content to forked repository>`

## Testing
The CodeFlare Operator currently has unit tests and pre-commit checks
 - To enable and view pre-commit checks: `pre-commit install`
 - To run unit tests, run `make test-unit`
 - Note that both are required for CI to pass on pull requests

To write and inspect unit tests:
 - Unit test functions are defined in `suite_test.go` (with utils in `util/util.go`) in the `controllers dir`
 - Test cases defined under `controllers/testdata`

 ## Local debugging with VSCode
 Steps outlining how to run the operator locally.
 - Ensure you are authenticated to your Kubernetes/OpenShift Cluster.
 - Populate the [.vscode/launch.json](https://github.com/project-codeflare/codeflare-operator/tree/main/.vscode/launch.json) file with the location of your Kubernetes config file and desired namespace.
 - In VSCode on the activity bar click `Run and Debug` or `CTRL + SHIFT + D` to start a local debugging session of the CodeFlare Operator.
 The operator should be running as intended.


## Example dev workflow

I've made changes to `pkg/controllers/raycluster_controller.go` and `pkg/controllers/raycluster_controller.go`. I've
written unit tests and would like to test my changes using the unit tests as well as on an OpenShift cluster which I
have access to.

1. Ensure the unit tests you've written are passing and you haven't introduced any regressions
   1. run `make test-unit`
1. build and push image
   1. `make image-build -e IMG=<image-repo/image-name:image-tag>`
   1. `make image-push -e IMG=<image-repo/image-name:image-tag>`
1. Login to your OpenShift cluster via `oc login --token=... --server=...`
1. deploy ODH/RHOAI if necessary
   1. for the latest releases
      1. run `make delete-all-in-one -e USE_RHOAI=<true|false>` **if** you would like to run on a fresh ODH/RHOAI deployment
      1. run `make all-in-one -e USE_RHOAI=<true|false>` to deploy new instance of ODH/RHOAI
   1. otherwise follow the dev guides found in
    [ODH](https://github.com/opendatahub-io/opendatahub-operator?tab=readme-ov-file#deployment) and
    [RHOAI](https://gitlab.cee.redhat.com/data-hub/olminstall) (Red Hat internal only)
      1. run `make install-nfd-operator`, `make install-service-mesh-operator`, `make install-ai-platform-operator`, `make install-nvidia-operator` to ensure all dependent operators are also installed
1. navigate to the DataScienceCluster resource and set `.spec.components.codeflare.managementState` to `Removed`
1. run `make deploy -e IMG=<image-repo/image-name:image-tag>`

Your dev image should now be deployed in the cluster
