# Contributing to the CodeFlare Operator

Here are a few things to go over before getting started with CodeFlare Operator development:

## Environment setup

The following should be installed in your working environment:
 - Go 1.19.x
   - [Download release](https://go.dev/dl/)
   - [Install Instructions](https://go.dev/doc/install)
 - [Operator SDK](https://sdk.operatorframework.io/docs/installation/)
 - GCC

## Basic Overview
The main entrypoint for the operator is `main.go`

The MCAD and InstaScale custom resources are defined under the `api` dir:
 - See `mcad_types.go` and `instascale_types.go`

The MCAD and InstaScale resource templates can be found under `config/internal`:
 - Sorted under `mcad` and `instascale` subdirs

The code for MCAD/InstaScale resource reconcilliation can be found in the `controllers` dir:
 - See `mcad_controller.go` and `instascale_controller.go`

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
 - MCAD and InstaScale unit tests under `mcad_controller_test.go` and `instascale_controller_test.go` in the `controllers` dir
 - Unit test functions are defined in `suite_test.go` (with utils in `util/util.go`) in the `controllers dir`
 - Test cases defined under `controllers/testdata`
